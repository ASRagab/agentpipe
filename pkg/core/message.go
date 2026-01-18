// Package core provides the foundational data structures for the v2 AgentPipe architecture.
// It defines messages, agents, conversations, and events that form the backbone of
// multi-agent communication orchestration.
package core

import (
	"time"

	"github.com/google/uuid"
)

// MessageStatus represents the current state of a message.
type MessageStatus string

const (
	// MessageStatusPending indicates the message is awaiting processing.
	MessageStatusPending MessageStatus = "pending"
	// MessageStatusStreaming indicates the message content is currently being streamed.
	MessageStatusStreaming MessageStatus = "streaming"
	// MessageStatusComplete indicates the message has been fully received/sent.
	MessageStatusComplete MessageStatus = "complete"
	// MessageStatusError indicates an error occurred while processing the message.
	MessageStatusError MessageStatus = "error"
)

// MessageRole represents who sent the message.
type MessageRole string

const (
	// RoleUser indicates the message is from a human user.
	RoleUser MessageRole = "user"
	// RoleAgent indicates the message is from an AI agent.
	RoleAgent MessageRole = "agent"
	// RoleSystem indicates the message is from the system/orchestrator.
	RoleSystem MessageRole = "system"
)

// Metrics contains performance and cost information for a message.
type Metrics struct {
	// Duration is how long it took to generate/receive the message.
	Duration time.Duration `json:"duration"`
	// InputTokens is the number of tokens in the input prompt.
	InputTokens int `json:"input_tokens"`
	// OutputTokens is the number of tokens in the response.
	OutputTokens int `json:"output_tokens"`
	// TotalTokens is InputTokens + OutputTokens.
	TotalTokens int `json:"total_tokens"`
	// Model is the specific AI model used.
	Model string `json:"model"`
	// Cost is the estimated monetary cost in USD.
	Cost float64 `json:"cost"`
}

// Message represents a single message in a conversation.
type Message struct {
	// ID is the unique identifier for this message.
	ID string `json:"id"`
	// Timestamp is when the message was created.
	Timestamp time.Time `json:"timestamp"`
	// Role indicates who sent the message (user, agent, or system).
	Role MessageRole `json:"role"`
	// AgentID is the unique identifier of the agent (if role is agent).
	AgentID string `json:"agent_id,omitempty"`
	// AgentName is the display name of the agent (if role is agent).
	AgentName string `json:"agent_name,omitempty"`
	// Content is the actual message text.
	Content string `json:"content"`
	// Status is the current processing state of the message.
	Status MessageStatus `json:"status"`
	// Metrics contains performance and cost data (optional).
	Metrics *Metrics `json:"metrics,omitempty"`
}

// NewUserMessage creates a new message from a human user.
func NewUserMessage(content string) Message {
	return Message{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Role:      RoleUser,
		Content:   content,
		Status:    MessageStatusComplete,
	}
}

// NewAgentMessage creates a new message from an AI agent.
func NewAgentMessage(agentID, agentName, content string, metrics *Metrics) Message {
	return Message{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Role:      RoleAgent,
		AgentID:   agentID,
		AgentName: agentName,
		Content:   content,
		Status:    MessageStatusComplete,
		Metrics:   metrics,
	}
}

// NewSystemMessage creates a new message from the system/orchestrator.
func NewSystemMessage(content string) Message {
	return Message{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Role:      RoleSystem,
		Content:   content,
		Status:    MessageStatusComplete,
	}
}

// NewPendingAgentMessage creates a placeholder message for an agent that is currently responding.
func NewPendingAgentMessage(agentID, agentName string) Message {
	return Message{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Role:      RoleAgent,
		AgentID:   agentID,
		AgentName: agentName,
		Status:    MessageStatusPending,
	}
}

// Complete marks the message as complete and sets its content and metrics.
func (m *Message) Complete(content string, metrics *Metrics) {
	m.Content = content
	m.Metrics = metrics
	m.Status = MessageStatusComplete
}

// SetError marks the message as having encountered an error.
func (m *Message) SetError(errContent string) {
	m.Content = errContent
	m.Status = MessageStatusError
}

// SetStreaming marks the message as currently streaming.
func (m *Message) SetStreaming() {
	m.Status = MessageStatusStreaming
}

// AppendContent adds content to the message (used during streaming).
func (m *Message) AppendContent(chunk string) {
	m.Content += chunk
}
