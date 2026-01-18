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
	"github.com/kevinelliott/agentpipe/pkg/v2/pool"
)

// Config contains configuration for the ConversationManager.
type Config struct {
	// Timeout is the maximum time to wait for agent responses.
	Timeout time.Duration
	// SaveDir is the directory for saving conversation logs.
	SaveDir string
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		Timeout: 60 * time.Second,
		SaveDir: "",
	}
}

// ConversationManager orchestrates conversations between multiple AI agents.
type ConversationManager struct {
	config       Config
	conversation *core.Conversation
	agentPool    *pool.Pool
	eventBus     *events.Bus
	mu           sync.RWMutex
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

	return &ConversationManager{
		config:       config,
		conversation: conversation,
		agentPool:    agentPool,
		eventBus:     eventBus,
	}, nil
}

// Start begins the conversation and emits the started event.
func (m *ConversationManager) Start() {
	m.mu.Lock()
	agents := m.conversation.Agents
	conversationID := m.conversation.ID
	m.mu.Unlock()

	m.eventBus.Publish(core.NewConversationStartedEvent(conversationID, agents))

	log.WithFields(map[string]interface{}{
		"conversation_id": conversationID,
		"agent_count":     len(agents),
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
	m.eventBus.Close()
}

// Summary returns the current conversation summary.
func (m *ConversationManager) Summary() core.ConversationSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.conversation.Summary()
}
