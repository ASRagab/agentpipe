package core

import (
	"time"

	"github.com/google/uuid"
)

// ConversationStatus represents the current state of a conversation.
type ConversationStatus string

const (
	// ConversationStatusActive indicates the conversation is ongoing.
	ConversationStatusActive ConversationStatus = "active"
	// ConversationStatusPaused indicates the conversation is temporarily halted.
	ConversationStatusPaused ConversationStatus = "paused"
	// ConversationStatusCompleted indicates the conversation has ended normally.
	ConversationStatusCompleted ConversationStatus = "completed"
	// ConversationStatusError indicates the conversation ended due to an error.
	ConversationStatusError ConversationStatus = "error"
)

// Conversation represents a multi-agent conversation session.
type Conversation struct {
	// ID is the unique identifier for this conversation.
	ID string `json:"id"`
	// Messages is the ordered list of messages in the conversation.
	Messages []Message `json:"messages"`
	// Agents is the list of agents participating in the conversation.
	Agents []Agent `json:"agents"`
	// Started is when the conversation began.
	Started time.Time `json:"started"`
	// Updated is when the conversation was last modified.
	Updated time.Time `json:"updated"`
	// Status is the current state of the conversation.
	Status ConversationStatus `json:"status"`
	// Metadata contains optional additional information.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewConversation creates a new conversation with the given agents.
func NewConversation(agents []Agent) *Conversation {
	now := time.Now()
	return &Conversation{
		ID:       uuid.New().String(),
		Messages: make([]Message, 0),
		Agents:   agents,
		Started:  now,
		Updated:  now,
		Status:   ConversationStatusActive,
		Metadata: make(map[string]interface{}),
	}
}

// AddMessage adds a message to the conversation and updates the timestamp.
func (c *Conversation) AddMessage(msg Message) {
	c.Messages = append(c.Messages, msg)
	c.Updated = time.Now()
}

// GetMessages returns a copy of all messages in the conversation.
func (c *Conversation) GetMessages() []Message {
	result := make([]Message, len(c.Messages))
	copy(result, c.Messages)
	return result
}

// GetLastMessage returns the most recent message, or nil if empty.
func (c *Conversation) GetLastMessage() *Message {
	if len(c.Messages) == 0 {
		return nil
	}
	return &c.Messages[len(c.Messages)-1]
}

// GetMessagesByAgent returns all messages from a specific agent.
func (c *Conversation) GetMessagesByAgent(agentID string) []Message {
	var result []Message
	for _, msg := range c.Messages {
		if msg.AgentID == agentID {
			result = append(result, msg)
		}
	}
	return result
}

// GetAgentByID returns the agent with the given ID, or nil if not found.
func (c *Conversation) GetAgentByID(id string) *Agent {
	for i, agent := range c.Agents {
		if agent.ID == id {
			return &c.Agents[i]
		}
	}
	return nil
}

// GetAgentByName returns the agent with the given name, or nil if not found.
func (c *Conversation) GetAgentByName(name string) *Agent {
	for i, agent := range c.Agents {
		if agent.Name == name {
			return &c.Agents[i]
		}
	}
	return nil
}

// MessageCount returns the total number of messages.
func (c *Conversation) MessageCount() int {
	return len(c.Messages)
}

// AgentCount returns the number of participating agents.
func (c *Conversation) AgentCount() int {
	return len(c.Agents)
}

// Duration returns how long the conversation has been running.
func (c *Conversation) Duration() time.Duration {
	return time.Since(c.Started)
}

// SetStatus updates the conversation status.
func (c *Conversation) SetStatus(status ConversationStatus) {
	c.Status = status
	c.Updated = time.Now()
}

// Complete marks the conversation as completed.
func (c *Conversation) Complete() {
	c.SetStatus(ConversationStatusCompleted)
}

// Pause marks the conversation as paused.
func (c *Conversation) Pause() {
	c.SetStatus(ConversationStatusPaused)
}

// Resume marks the conversation as active (from paused).
func (c *Conversation) Resume() {
	c.SetStatus(ConversationStatusActive)
}

// SetError marks the conversation as errored.
func (c *Conversation) SetError() {
	c.SetStatus(ConversationStatusError)
}

// SetMetadata sets a metadata value.
func (c *Conversation) SetMetadata(key string, value interface{}) {
	if c.Metadata == nil {
		c.Metadata = make(map[string]interface{})
	}
	c.Metadata[key] = value
	c.Updated = time.Now()
}

// GetMetadata retrieves a metadata value.
func (c *Conversation) GetMetadata(key string) (interface{}, bool) {
	if c.Metadata == nil {
		return nil, false
	}
	val, ok := c.Metadata[key]
	return val, ok
}

// TotalTokens calculates the total token usage across all messages.
func (c *Conversation) TotalTokens() int {
	var total int
	for _, msg := range c.Messages {
		if msg.Metrics != nil {
			total += msg.Metrics.TotalTokens
		}
	}
	return total
}

// TotalCost calculates the total cost across all messages.
func (c *Conversation) TotalCost() float64 {
	var total float64
	for _, msg := range c.Messages {
		if msg.Metrics != nil {
			total += msg.Metrics.Cost
		}
	}
	return total
}

// Summary returns a summary of the conversation.
type ConversationSummary struct {
	ID           string             `json:"id"`
	Status       ConversationStatus `json:"status"`
	AgentCount   int                `json:"agent_count"`
	MessageCount int                `json:"message_count"`
	TotalTokens  int                `json:"total_tokens"`
	TotalCost    float64            `json:"total_cost"`
	Duration     time.Duration      `json:"duration"`
	Started      time.Time          `json:"started"`
	Updated      time.Time          `json:"updated"`
}

// Summary returns a summary of the conversation.
func (c *Conversation) Summary() ConversationSummary {
	return ConversationSummary{
		ID:           c.ID,
		Status:       c.Status,
		AgentCount:   c.AgentCount(),
		MessageCount: c.MessageCount(),
		TotalTokens:  c.TotalTokens(),
		TotalCost:    c.TotalCost(),
		Duration:     c.Duration(),
		Started:      c.Started,
		Updated:      c.Updated,
	}
}
