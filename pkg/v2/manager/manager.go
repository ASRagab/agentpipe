// Package manager provides the ConversationManager for orchestrating multi-agent conversations.
package manager

import (
	"context"
	"sync"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/log"
	"github.com/kevinelliott/agentpipe/pkg/v2/adapters"
	"github.com/kevinelliott/agentpipe/pkg/v2/core"
	"github.com/kevinelliott/agentpipe/pkg/v2/events"
	"github.com/kevinelliott/agentpipe/pkg/v2/persistence"
	"github.com/kevinelliott/agentpipe/pkg/v2/pool"
)

// PersistenceConfig contains configuration for auto-save functionality.
type PersistenceConfig struct {
	// Enabled enables auto-save functionality.
	Enabled bool
	// SaveDir is the directory for saving conversations.
	SaveDir string
	// SaveInterval is how often to auto-save (0 means only save on agent responses).
	SaveInterval time.Duration
}

// Config contains configuration for the ConversationManager.
type Config struct {
	// Timeout is the maximum time to wait for agent responses.
	Timeout time.Duration
	// SaveDir is the directory for saving conversation logs (deprecated, use Persistence.SaveDir).
	SaveDir string
	// Persistence contains auto-save configuration.
	Persistence PersistenceConfig
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		Timeout: 60 * time.Second,
		SaveDir: "",
		Persistence: PersistenceConfig{
			Enabled:      false,
			SaveDir:      persistence.DefaultSaveDir(),
			SaveInterval: 0,
		},
	}
}

// ConversationManager orchestrates conversations between multiple AI agents.
type ConversationManager struct {
	config       Config
	conversation *core.Conversation
	agentPool    *pool.Pool
	eventBus     *events.Bus
	mu           sync.RWMutex

	// Auto-save fields
	lastSave     time.Time
	saveDebounce time.Duration
	stopAutoSave chan struct{}
	autoSaveWg   sync.WaitGroup
	resumed      bool
}

// NewConversationManager creates a new conversation manager.
func NewConversationManager(config Config, agents []core.Agent, eventBus *events.Bus) (*ConversationManager, error) {
	if eventBus == nil {
		eventBus = events.NewBus()
	}

	// Create agent pool
	agentPool := pool.NewPool(eventBus, config.Timeout)

	// Initialize adapters and add agents to pool
	for _, agent := range agents {
		adapter, err := adapters.Get(agent.AdapterName)
		if err != nil {
			log.WithFields(map[string]interface{}{
				"agent_id":   agent.ID,
				"agent_name": agent.Name,
				"adapter":    agent.AdapterName,
			}).WithError(err).Error("failed to get adapter")
			return nil, err
		}

		if err := adapter.Initialize(agent); err != nil {
			log.WithFields(map[string]interface{}{
				"agent_id":   agent.ID,
				"agent_name": agent.Name,
				"adapter":    agent.AdapterName,
			}).WithError(err).Error("failed to initialize adapter")
			return nil, err
		}

		agentPool.AddAgent(agent, adapter)
	}

	// Create conversation
	conversation := core.NewConversation(agents)

	m := &ConversationManager{
		config:       config,
		conversation: conversation,
		agentPool:    agentPool,
		eventBus:     eventBus,
		saveDebounce: 2 * time.Second,
		stopAutoSave: make(chan struct{}),
	}

	// Set up auto-save if enabled
	if config.Persistence.Enabled {
		m.setupAutoSave()
	}

	return m, nil
}

// setupAutoSave sets up the auto-save functionality.
func (m *ConversationManager) setupAutoSave() {
	// Subscribe to agent done events to trigger saves
	m.eventBus.Subscribe(core.EventAgentDone, func(event core.Event) {
		m.triggerAutoSave()
	})

	// Start interval-based auto-save if configured
	if m.config.Persistence.SaveInterval > 0 {
		m.autoSaveWg.Add(1)
		go m.autoSaveLoop()
	}
}

// autoSaveLoop runs the interval-based auto-save.
func (m *ConversationManager) autoSaveLoop() {
	defer m.autoSaveWg.Done()

	ticker := time.NewTicker(m.config.Persistence.SaveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.triggerAutoSave()
		case <-m.stopAutoSave:
			return
		}
	}
}

// triggerAutoSave triggers an auto-save with debouncing.
func (m *ConversationManager) triggerAutoSave() {
	m.mu.Lock()
	if time.Since(m.lastSave) < m.saveDebounce {
		m.mu.Unlock()
		return
	}
	m.lastSave = time.Now()
	conv := m.conversation
	m.mu.Unlock()

	// Save in background to avoid blocking
	go func() {
		saveDir := m.config.Persistence.SaveDir
		if saveDir == "" {
			saveDir = persistence.DefaultSaveDir()
		}

		filePath, err := persistence.SaveConversation(conv, saveDir)
		if err != nil {
			log.WithFields(map[string]interface{}{
				"conversation_id": conv.ID,
			}).WithError(err).Warn("auto-save failed")
			return
		}

		m.eventBus.Publish(core.NewConversationSavedEvent(conv.ID, filePath))
	}()
}

// Resume restores a conversation from a file or conversation ID.
// If pathOrID is "latest", loads the most recent conversation.
// If pathOrID is a file path (contains "/" or ".json"), loads from that file.
// Otherwise, treats pathOrID as a conversation ID prefix.
func (m *ConversationManager) Resume(pathOrID string) error {
	saveDir := m.config.Persistence.SaveDir
	if saveDir == "" {
		saveDir = persistence.DefaultSaveDir()
	}

	var filePath string
	var err error

	if pathOrID == "latest" {
		// Load latest conversation
		conv, loadErr := persistence.LoadLatest(saveDir)
		if loadErr != nil {
			return loadErr
		}
		return m.restoreConversation(conv)
	} else if isFilePath(pathOrID) {
		// Load from file path
		filePath = pathOrID
	} else {
		// Find by ID prefix
		filePath, err = persistence.FindConversationByID(saveDir, pathOrID)
		if err != nil {
			return err
		}
	}

	conv, err := persistence.LoadConversation(filePath)
	if err != nil {
		return err
	}

	return m.restoreConversation(conv)
}

// restoreConversation restores a loaded conversation into the manager.
func (m *ConversationManager) restoreConversation(conv *core.Conversation) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Re-initialize adapters for loaded agents
	for _, agent := range conv.Agents {
		adapter, err := adapters.Get(agent.AdapterName)
		if err != nil {
			log.WithFields(map[string]interface{}{
				"agent_id":     agent.ID,
				"agent_name":   agent.Name,
				"adapter_name": agent.AdapterName,
			}).WithError(err).Warn("failed to get adapter for resumed agent")
			continue
		}

		if err := adapter.Initialize(agent); err != nil {
			log.WithFields(map[string]interface{}{
				"agent_id":   agent.ID,
				"agent_name": agent.Name,
			}).WithError(err).Warn("failed to initialize adapter for resumed agent")
			continue
		}

		m.agentPool.AddAgent(agent, adapter)
	}

	// Restore conversation state
	m.conversation = conv
	m.conversation.SetStatus(core.ConversationStatusActive)
	m.resumed = true

	log.WithFields(map[string]interface{}{
		"conversation_id": conv.ID,
		"message_count":   len(conv.Messages),
		"agent_count":     len(conv.Agents),
	}).Info("conversation resumed")

	return nil
}

// isFilePath checks if a string looks like a file path.
func isFilePath(s string) bool {
	return len(s) > 0 && (s[0] == '/' || s[0] == '.' || len(s) > 5 && s[len(s)-5:] == ".json")
}

// IsResumed returns true if the conversation was resumed from a saved state.
func (m *ConversationManager) IsResumed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.resumed
}

// Save manually saves the current conversation.
func (m *ConversationManager) Save() (string, error) {
	m.mu.RLock()
	conv := m.conversation
	m.mu.RUnlock()

	saveDir := m.config.Persistence.SaveDir
	if saveDir == "" {
		saveDir = persistence.DefaultSaveDir()
	}

	filePath, err := persistence.SaveConversation(conv, saveDir)
	if err != nil {
		return "", err
	}

	m.eventBus.Publish(core.NewConversationSavedEvent(conv.ID, filePath))
	return filePath, nil
}

// ExportToMarkdown exports the conversation to a Markdown file.
func (m *ConversationManager) ExportToMarkdown(outputPath string) error {
	m.mu.RLock()
	conv := m.conversation
	m.mu.RUnlock()

	return persistence.SaveAsMarkdown(conv, outputPath)
}

// Start begins the conversation and emits the started event.
func (m *ConversationManager) Start() {
	m.mu.Lock()
	agents := m.conversation.Agents
	conversationID := m.conversation.ID
	resumed := m.resumed
	m.mu.Unlock()

	// Emit conversation started event
	event := core.NewConversationStartedEvent(conversationID, agents)
	// Add resumed flag to event data
	if data, ok := event.Data.(core.ConversationStartedData); ok {
		if resumed {
			m.conversation.SetMetadata("resumed", true)
		}
		event.Data = data
	}
	m.eventBus.Publish(event)

	log.WithFields(map[string]interface{}{
		"conversation_id": conversationID,
		"agent_count":     len(agents),
		"resumed":         resumed,
	}).Info("conversation started")
}

// SendUserMessage sends a user message and waits for all agent responses.
func (m *ConversationManager) SendUserMessage(ctx context.Context, content string) ([]core.Message, error) {
	// Create and add user message
	userMsg := core.NewUserMessage(content)

	m.mu.Lock()
	m.conversation.AddMessage(userMsg)
	messages := m.conversation.GetMessages()
	m.mu.Unlock()

	// Emit message created event
	m.eventBus.Publish(core.NewMessageCreatedEvent(userMsg))

	log.WithFields(map[string]interface{}{
		"conversation_id": m.conversation.ID,
		"message_id":      userMsg.ID,
		"content_length":  len(content),
	}).Debug("user message sent")

	// Execute all agents in parallel
	responses := m.agentPool.ExecuteParallel(ctx, messages)

	// Collect and add agent messages
	agentMessages := make([]core.Message, 0, len(responses))
	for _, resp := range responses {
		if resp.Error != nil {
			// Create error message for failed agents
			errMsg := core.NewAgentMessage(resp.AgentID, resp.AgentName, "", nil)
			errMsg.SetError(resp.Error.Error())

			m.mu.Lock()
			m.conversation.AddMessage(errMsg)
			m.mu.Unlock()

			agentMessages = append(agentMessages, errMsg)
			continue
		}

		if resp.Message != nil {
			m.mu.Lock()
			m.conversation.AddMessage(*resp.Message)
			m.mu.Unlock()

			// Emit message created event
			m.eventBus.Publish(core.NewMessageCreatedEvent(*resp.Message))

			agentMessages = append(agentMessages, *resp.Message)
		}
	}

	return agentMessages, nil
}

// GetMessages returns a copy of all messages in the conversation.
func (m *ConversationManager) GetMessages() []core.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.conversation.GetMessages()
}

// GetConversation returns the conversation state.
func (m *ConversationManager) GetConversation() *core.Conversation {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.conversation
}

// GetAgents returns the list of participating agents.
func (m *ConversationManager) GetAgents() []core.Agent {
	return m.agentPool.GetAgents()
}

// GetAgentStatus returns the status of all agents.
func (m *ConversationManager) GetAgentStatus() map[string]core.AgentStatus {
	return m.agentPool.GetStatus()
}

// Subscribe registers a handler for a specific event type.
func (m *ConversationManager) Subscribe(eventType core.EventType, handler events.Handler) func() {
	return m.eventBus.Subscribe(eventType, handler)
}

// SubscribeAll registers a handler for all events.
func (m *ConversationManager) SubscribeAll(handler events.Handler) func() {
	return m.eventBus.SubscribeAll(handler)
}

// Complete marks the conversation as completed.
func (m *ConversationManager) Complete() {
	m.mu.Lock()
	m.conversation.Complete()
	summary := m.conversation.Summary()
	conversationID := m.conversation.ID
	m.mu.Unlock()

	m.eventBus.Publish(core.NewConversationCompletedEvent(conversationID, summary))

	log.WithFields(map[string]interface{}{
		"conversation_id": conversationID,
		"message_count":   summary.MessageCount,
		"total_tokens":    summary.TotalTokens,
		"total_cost":      summary.TotalCost,
		"duration":        summary.Duration.String(),
	}).Info("conversation completed")
}

// Close shuts down the manager and event bus.
func (m *ConversationManager) Close() {
	// Stop auto-save
	close(m.stopAutoSave)
	m.autoSaveWg.Wait()

	m.eventBus.Close()
}

// Summary returns the current conversation summary.
func (m *ConversationManager) Summary() core.ConversationSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.conversation.Summary()
}
