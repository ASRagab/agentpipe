package core

import (
	"time"

	"github.com/google/uuid"
)

// EventType defines the category of events in the system.
type EventType string

const (
	// EventMessageCreated is emitted when a new message is added to the conversation.
	EventMessageCreated EventType = "message.created"
	// EventMessageChunk is emitted when a streaming chunk is received.
	EventMessageChunk EventType = "message.chunk"
	// EventAgentTyping is emitted when an agent starts generating a response.
	EventAgentTyping EventType = "agent.typing"
	// EventAgentDone is emitted when an agent finishes generating a response.
	EventAgentDone EventType = "agent.done"
	// EventAgentError is emitted when an agent encounters an error.
	EventAgentError EventType = "agent.error"
	// EventAgentCancelled is emitted when an agent's request is cancelled.
	EventAgentCancelled EventType = "agent.cancelled"
	// EventConversationStarted is emitted when a conversation begins.
	EventConversationStarted EventType = "conversation.started"
	// EventConversationSaved is emitted when a conversation is persisted.
	EventConversationSaved EventType = "conversation.saved"
	// EventConversationCompleted is emitted when a conversation ends normally.
	EventConversationCompleted EventType = "conversation.completed"
	// EventConversationError is emitted when a conversation ends due to an error.
	EventConversationError EventType = "conversation.error"
	// EventAgentHealthy is emitted when an agent passes a health check.
	EventAgentHealthy EventType = "agent.healthy"
	// EventAgentUnhealthy is emitted when an agent fails a health check.
	EventAgentUnhealthy EventType = "agent.unhealthy"
	// EventPreflightCompleted is emitted when preflight health checks complete.
	EventPreflightCompleted EventType = "preflight.completed"
)

// Event represents a system event that can be published and subscribed to.
type Event struct {
	// ID is the unique identifier for this event.
	ID string `json:"id"`
	// Type is the category of the event.
	Type EventType `json:"type"`
	// Timestamp is when the event occurred.
	Timestamp time.Time `json:"timestamp"`
	// Data is the event-specific payload.
	Data interface{} `json:"data"`
}

// NewEvent creates a new event with the given type and data.
func NewEvent(eventType EventType, data interface{}) Event {
	return Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}
}

// MessageChunk represents a partial message received during streaming.
type MessageChunk struct {
	// MessageID is the ID of the message being streamed.
	MessageID string `json:"message_id"`
	// AgentID is the ID of the agent sending the chunk.
	AgentID string `json:"agent_id"`
	// AgentName is the name of the agent sending the chunk.
	AgentName string `json:"agent_name"`
	// Content is the chunk content.
	Content string `json:"content"`
	// Index is the chunk sequence number.
	Index int `json:"index"`
	// IsFinal indicates if this is the last chunk.
	IsFinal bool `json:"is_final"`
}

// AgentTypingData contains information about an agent starting to type.
type AgentTypingData struct {
	// AgentID is the ID of the typing agent.
	AgentID string `json:"agent_id"`
	// AgentName is the name of the typing agent.
	AgentName string `json:"agent_name"`
}

// AgentDoneData contains information about an agent completing a response.
type AgentDoneData struct {
	// AgentID is the ID of the agent.
	AgentID string `json:"agent_id"`
	// AgentName is the name of the agent.
	AgentName string `json:"agent_name"`
	// Message is the completed message.
	Message Message `json:"message"`
}

// AgentErrorData contains information about an agent error.
type AgentErrorData struct {
	// AgentID is the ID of the agent.
	AgentID string `json:"agent_id"`
	// AgentName is the name of the agent.
	AgentName string `json:"agent_name"`
	// Error is the error message.
	Error string `json:"error"`
	// ErrorType categorizes the error (timeout, rate_limit, network, auth, etc.).
	ErrorType string `json:"error_type,omitempty"`
	// Recoverable indicates if the error can be retried.
	Recoverable bool `json:"recoverable,omitempty"`
	// RetryAfter is the suggested wait time before retrying (for rate limits).
	RetryAfter time.Duration `json:"retry_after,omitempty"`
	// RetryHint provides actionable guidance for the user.
	RetryHint string `json:"retry_hint,omitempty"`
}

// AgentCancelledData contains information about an agent cancellation.
type AgentCancelledData struct {
	// AgentID is the ID of the cancelled agent.
	AgentID string `json:"agent_id"`
	// AgentName is the name of the cancelled agent.
	AgentName string `json:"agent_name"`
	// Reason is the cancellation reason.
	Reason string `json:"reason"`
}

// ConversationStartedData contains information about a conversation starting.
type ConversationStartedData struct {
	// ConversationID is the ID of the conversation.
	ConversationID string `json:"conversation_id"`
	// Agents is the list of participating agents.
	Agents []Agent `json:"agents"`
}

// ConversationSavedData contains information about a conversation being saved.
type ConversationSavedData struct {
	// ConversationID is the ID of the conversation.
	ConversationID string `json:"conversation_id"`
	// FilePath is where the conversation was saved.
	FilePath string `json:"file_path"`
}

// ConversationCompletedData contains information about a conversation ending.
type ConversationCompletedData struct {
	// ConversationID is the ID of the conversation.
	ConversationID string `json:"conversation_id"`
	// Summary contains the final conversation statistics.
	Summary ConversationSummary `json:"summary"`
}

// ConversationErrorData contains information about a conversation error.
type ConversationErrorData struct {
	// ConversationID is the ID of the conversation.
	ConversationID string `json:"conversation_id"`
	// Error is the error message.
	Error string `json:"error"`
}

// NewMessageCreatedEvent creates an event for a new message.
func NewMessageCreatedEvent(msg Message) Event {
	return NewEvent(EventMessageCreated, msg)
}

// NewMessageChunkEvent creates an event for a streaming chunk.
func NewMessageChunkEvent(chunk MessageChunk) Event {
	return NewEvent(EventMessageChunk, chunk)
}

// NewAgentTypingEvent creates an event for an agent starting to type.
func NewAgentTypingEvent(agentID, agentName string) Event {
	return NewEvent(EventAgentTyping, AgentTypingData{
		AgentID:   agentID,
		AgentName: agentName,
	})
}

// NewAgentDoneEvent creates an event for an agent completing a response.
func NewAgentDoneEvent(agentID, agentName string, msg Message) Event {
	return NewEvent(EventAgentDone, AgentDoneData{
		AgentID:   agentID,
		AgentName: agentName,
		Message:   msg,
	})
}

// NewAgentErrorEvent creates an event for an agent error.
func NewAgentErrorEvent(agentID, agentName, errMsg string) Event {
	return NewEvent(EventAgentError, AgentErrorData{
		AgentID:   agentID,
		AgentName: agentName,
		Error:     errMsg,
	})
}

// NewAgentCancelledEvent creates an event for an agent cancellation.
func NewAgentCancelledEvent(agentID, agentName, reason string) Event {
	return NewEvent(EventAgentCancelled, AgentCancelledData{
		AgentID:   agentID,
		AgentName: agentName,
		Reason:    reason,
	})
}

// NewConversationStartedEvent creates an event for a conversation starting.
func NewConversationStartedEvent(conversationID string, agents []Agent) Event {
	return NewEvent(EventConversationStarted, ConversationStartedData{
		ConversationID: conversationID,
		Agents:         agents,
	})
}

// NewConversationSavedEvent creates an event for a conversation being saved.
func NewConversationSavedEvent(conversationID, filePath string) Event {
	return NewEvent(EventConversationSaved, ConversationSavedData{
		ConversationID: conversationID,
		FilePath:       filePath,
	})
}

// NewConversationCompletedEvent creates an event for a conversation ending.
func NewConversationCompletedEvent(conversationID string, summary ConversationSummary) Event {
	return NewEvent(EventConversationCompleted, ConversationCompletedData{
		ConversationID: conversationID,
		Summary:        summary,
	})
}

// NewConversationErrorEvent creates an event for a conversation error.
func NewConversationErrorEvent(conversationID, errMsg string) Event {
	return NewEvent(EventConversationError, ConversationErrorData{
		ConversationID: conversationID,
		Error:          errMsg,
	})
}

// AgentHealthyData contains information about an agent passing a health check.
type AgentHealthyData struct {
	// AgentID is the ID of the agent.
	AgentID string `json:"agent_id"`
	// AgentName is the name of the agent.
	AgentName string `json:"agent_name"`
	// ResponseTime is how long the health check took.
	ResponseTime time.Duration `json:"response_time"`
}

// AgentUnhealthyData contains information about an agent failing a health check.
type AgentUnhealthyData struct {
	// AgentID is the ID of the agent.
	AgentID string `json:"agent_id"`
	// AgentName is the name of the agent.
	AgentName string `json:"agent_name"`
	// Error is the health check error message.
	Error string `json:"error"`
}

// PreflightCompletedData contains information about preflight health checks.
type PreflightCompletedData struct {
	// AllHealthy is true if all agents passed health checks.
	AllHealthy bool `json:"all_healthy"`
	// HealthyCount is the number of healthy agents.
	HealthyCount int `json:"healthy_count"`
	// UnhealthyCount is the number of unhealthy agents.
	UnhealthyCount int `json:"unhealthy_count"`
	// Duration is how long the preflight check took.
	Duration time.Duration `json:"duration"`
	// Warnings contains any warning messages.
	Warnings []string `json:"warnings,omitempty"`
}

// NewAgentHealthyEvent creates an event for an agent passing a health check.
func NewAgentHealthyEvent(agentID, agentName string, responseTime time.Duration) Event {
	return NewEvent(EventAgentHealthy, AgentHealthyData{
		AgentID:      agentID,
		AgentName:    agentName,
		ResponseTime: responseTime,
	})
}

// NewAgentUnhealthyEvent creates an event for an agent failing a health check.
func NewAgentUnhealthyEvent(agentID, agentName, errMsg string) Event {
	return NewEvent(EventAgentUnhealthy, AgentUnhealthyData{
		AgentID:   agentID,
		AgentName: agentName,
		Error:     errMsg,
	})
}

// NewPreflightCompletedEvent creates an event for preflight checks completing.
func NewPreflightCompletedEvent(allHealthy bool, healthyCount, unhealthyCount int, duration time.Duration, warnings []string) Event {
	return NewEvent(EventPreflightCompleted, PreflightCompletedData{
		AllHealthy:     allHealthy,
		HealthyCount:   healthyCount,
		UnhealthyCount: unhealthyCount,
		Duration:       duration,
		Warnings:       warnings,
	})
}
