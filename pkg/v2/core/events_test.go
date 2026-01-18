package core

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewEvent(t *testing.T) {
	data := map[string]string{"key": "value"}
	event := NewEvent(EventMessageCreated, data)

	if event.ID == "" {
		t.Error("expected non-empty event ID")
	}
	if event.Type != EventMessageCreated {
		t.Errorf("expected type %q, got %q", EventMessageCreated, event.Type)
	}
	if time.Since(event.Timestamp) > time.Second {
		t.Error("Timestamp should be recent")
	}
	if event.Data == nil {
		t.Error("expected non-nil Data")
	}
}

func TestNewEvent_UniqueIDs(t *testing.T) {
	event1 := NewEvent(EventMessageCreated, nil)
	event2 := NewEvent(EventMessageCreated, nil)

	if event1.ID == event2.ID {
		t.Error("expected unique event IDs")
	}
}

func TestEventType_Constants_Unique(t *testing.T) {
	eventTypes := []EventType{
		EventMessageCreated,
		EventMessageChunk,
		EventAgentTyping,
		EventAgentDone,
		EventAgentError,
		EventConversationStarted,
		EventConversationSaved,
		EventConversationCompleted,
		EventConversationError,
	}

	seen := make(map[EventType]bool)
	for _, et := range eventTypes {
		if seen[et] {
			t.Errorf("duplicate event type constant: %q", et)
		}
		seen[et] = true
	}
}

func TestNewMessageCreatedEvent(t *testing.T) {
	msg := NewUserMessage("Hello")
	event := NewMessageCreatedEvent(msg)

	if event.Type != EventMessageCreated {
		t.Errorf("expected type %q, got %q", EventMessageCreated, event.Type)
	}

	data, ok := event.Data.(Message)
	if !ok {
		t.Fatal("expected Data to be Message")
	}
	if data.Content != "Hello" {
		t.Errorf("expected content 'Hello', got %q", data.Content)
	}
}

func TestNewMessageChunkEvent(t *testing.T) {
	chunk := MessageChunk{
		MessageID: "msg-1",
		AgentID:   "agent-1",
		AgentName: "Agent",
		Content:   "Hello ",
		Index:     0,
		IsFinal:   false,
	}
	event := NewMessageChunkEvent(chunk)

	if event.Type != EventMessageChunk {
		t.Errorf("expected type %q, got %q", EventMessageChunk, event.Type)
	}

	data, ok := event.Data.(MessageChunk)
	if !ok {
		t.Fatal("expected Data to be MessageChunk")
	}
	if data.MessageID != "msg-1" {
		t.Errorf("expected MessageID 'msg-1', got %q", data.MessageID)
	}
	if data.Content != "Hello " {
		t.Errorf("expected Content 'Hello ', got %q", data.Content)
	}
	if data.Index != 0 {
		t.Errorf("expected Index=0, got %d", data.Index)
	}
	if data.IsFinal {
		t.Error("expected IsFinal=false")
	}
}

func TestNewAgentTypingEvent(t *testing.T) {
	event := NewAgentTypingEvent("agent-1", "Agent 1")

	if event.Type != EventAgentTyping {
		t.Errorf("expected type %q, got %q", EventAgentTyping, event.Type)
	}

	data, ok := event.Data.(AgentTypingData)
	if !ok {
		t.Fatal("expected Data to be AgentTypingData")
	}
	if data.AgentID != "agent-1" {
		t.Errorf("expected AgentID 'agent-1', got %q", data.AgentID)
	}
	if data.AgentName != "Agent 1" {
		t.Errorf("expected AgentName 'Agent 1', got %q", data.AgentName)
	}
}

func TestNewAgentDoneEvent(t *testing.T) {
	msg := NewAgentMessage("agent-1", "Agent", "Hello", nil)
	event := NewAgentDoneEvent("agent-1", "Agent", msg)

	if event.Type != EventAgentDone {
		t.Errorf("expected type %q, got %q", EventAgentDone, event.Type)
	}

	data, ok := event.Data.(AgentDoneData)
	if !ok {
		t.Fatal("expected Data to be AgentDoneData")
	}
	if data.AgentID != "agent-1" {
		t.Errorf("expected AgentID 'agent-1', got %q", data.AgentID)
	}
	if data.Message.Content != "Hello" {
		t.Errorf("expected Message.Content 'Hello', got %q", data.Message.Content)
	}
}

func TestNewAgentErrorEvent(t *testing.T) {
	event := NewAgentErrorEvent("agent-1", "Agent", "Connection failed")

	if event.Type != EventAgentError {
		t.Errorf("expected type %q, got %q", EventAgentError, event.Type)
	}

	data, ok := event.Data.(AgentErrorData)
	if !ok {
		t.Fatal("expected Data to be AgentErrorData")
	}
	if data.AgentID != "agent-1" {
		t.Errorf("expected AgentID 'agent-1', got %q", data.AgentID)
	}
	if data.Error != "Connection failed" {
		t.Errorf("expected Error 'Connection failed', got %q", data.Error)
	}
}

func TestNewConversationStartedEvent(t *testing.T) {
	agents := []Agent{
		NewAgent("agent-1", "openrouter", "Agent 1", "gpt-4", "openrouter"),
	}
	event := NewConversationStartedEvent("conv-1", agents)

	if event.Type != EventConversationStarted {
		t.Errorf("expected type %q, got %q", EventConversationStarted, event.Type)
	}

	data, ok := event.Data.(ConversationStartedData)
	if !ok {
		t.Fatal("expected Data to be ConversationStartedData")
	}
	if data.ConversationID != "conv-1" {
		t.Errorf("expected ConversationID 'conv-1', got %q", data.ConversationID)
	}
	if len(data.Agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(data.Agents))
	}
}

func TestNewConversationSavedEvent(t *testing.T) {
	event := NewConversationSavedEvent("conv-1", "/path/to/file.json")

	if event.Type != EventConversationSaved {
		t.Errorf("expected type %q, got %q", EventConversationSaved, event.Type)
	}

	data, ok := event.Data.(ConversationSavedData)
	if !ok {
		t.Fatal("expected Data to be ConversationSavedData")
	}
	if data.ConversationID != "conv-1" {
		t.Errorf("expected ConversationID 'conv-1', got %q", data.ConversationID)
	}
	if data.FilePath != "/path/to/file.json" {
		t.Errorf("expected FilePath '/path/to/file.json', got %q", data.FilePath)
	}
}

func TestNewConversationCompletedEvent(t *testing.T) {
	summary := ConversationSummary{
		ID:           "conv-1",
		Status:       ConversationStatusCompleted,
		AgentCount:   2,
		MessageCount: 10,
		TotalTokens:  500,
		TotalCost:    0.05,
		Duration:     5 * time.Minute,
	}
	event := NewConversationCompletedEvent("conv-1", summary)

	if event.Type != EventConversationCompleted {
		t.Errorf("expected type %q, got %q", EventConversationCompleted, event.Type)
	}

	data, ok := event.Data.(ConversationCompletedData)
	if !ok {
		t.Fatal("expected Data to be ConversationCompletedData")
	}
	if data.ConversationID != "conv-1" {
		t.Errorf("expected ConversationID 'conv-1', got %q", data.ConversationID)
	}
	if data.Summary.TotalTokens != 500 {
		t.Errorf("expected TotalTokens=500, got %d", data.Summary.TotalTokens)
	}
}

func TestNewConversationErrorEvent(t *testing.T) {
	event := NewConversationErrorEvent("conv-1", "Something went wrong")

	if event.Type != EventConversationError {
		t.Errorf("expected type %q, got %q", EventConversationError, event.Type)
	}

	data, ok := event.Data.(ConversationErrorData)
	if !ok {
		t.Fatal("expected Data to be ConversationErrorData")
	}
	if data.ConversationID != "conv-1" {
		t.Errorf("expected ConversationID 'conv-1', got %q", data.ConversationID)
	}
	if data.Error != "Something went wrong" {
		t.Errorf("expected Error 'Something went wrong', got %q", data.Error)
	}
}

func TestEvent_JSONSerialization(t *testing.T) {
	original := NewAgentTypingEvent("agent-1", "Agent")

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	var restored Event
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if restored.ID != original.ID {
		t.Errorf("ID mismatch: expected %q, got %q", original.ID, restored.ID)
	}
	if restored.Type != original.Type {
		t.Errorf("Type mismatch: expected %q, got %q", original.Type, restored.Type)
	}
}

func TestMessageChunk_Fields(t *testing.T) {
	chunk := MessageChunk{
		MessageID: "msg-1",
		AgentID:   "agent-1",
		AgentName: "Test Agent",
		Content:   "Hello",
		Index:     3,
		IsFinal:   true,
	}

	if chunk.MessageID != "msg-1" {
		t.Errorf("expected MessageID 'msg-1', got %q", chunk.MessageID)
	}
	if chunk.Index != 3 {
		t.Errorf("expected Index=3, got %d", chunk.Index)
	}
	if !chunk.IsFinal {
		t.Error("expected IsFinal=true")
	}
}

func TestAgentTypingData_Fields(t *testing.T) {
	data := AgentTypingData{
		AgentID:   "agent-1",
		AgentName: "Test Agent",
	}

	if data.AgentID != "agent-1" {
		t.Errorf("expected AgentID 'agent-1', got %q", data.AgentID)
	}
	if data.AgentName != "Test Agent" {
		t.Errorf("expected AgentName 'Test Agent', got %q", data.AgentName)
	}
}

func TestAgentDoneData_Fields(t *testing.T) {
	msg := NewAgentMessage("agent-1", "Agent", "Response", nil)
	data := AgentDoneData{
		AgentID:   "agent-1",
		AgentName: "Agent",
		Message:   msg,
	}

	if data.AgentID != "agent-1" {
		t.Errorf("expected AgentID 'agent-1', got %q", data.AgentID)
	}
	if data.Message.Content != "Response" {
		t.Errorf("expected Message.Content 'Response', got %q", data.Message.Content)
	}
}

func TestAgentErrorData_Fields(t *testing.T) {
	data := AgentErrorData{
		AgentID:   "agent-1",
		AgentName: "Agent",
		Error:     "Connection timeout",
	}

	if data.Error != "Connection timeout" {
		t.Errorf("expected Error 'Connection timeout', got %q", data.Error)
	}
}

func TestConversationStartedData_Fields(t *testing.T) {
	agents := []Agent{
		NewAgent("agent-1", "openrouter", "Agent 1", "gpt-4", "openrouter"),
	}
	data := ConversationStartedData{
		ConversationID: "conv-1",
		Agents:         agents,
	}

	if data.ConversationID != "conv-1" {
		t.Errorf("expected ConversationID 'conv-1', got %q", data.ConversationID)
	}
	if len(data.Agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(data.Agents))
	}
}

func TestConversationSavedData_Fields(t *testing.T) {
	data := ConversationSavedData{
		ConversationID: "conv-1",
		FilePath:       "/tmp/conv.json",
	}

	if data.FilePath != "/tmp/conv.json" {
		t.Errorf("expected FilePath '/tmp/conv.json', got %q", data.FilePath)
	}
}

func TestConversationCompletedData_Fields(t *testing.T) {
	summary := ConversationSummary{
		MessageCount: 10,
	}
	data := ConversationCompletedData{
		ConversationID: "conv-1",
		Summary:        summary,
	}

	if data.Summary.MessageCount != 10 {
		t.Errorf("expected MessageCount=10, got %d", data.Summary.MessageCount)
	}
}

func TestConversationErrorData_Fields(t *testing.T) {
	data := ConversationErrorData{
		ConversationID: "conv-1",
		Error:          "Fatal error",
	}

	if data.Error != "Fatal error" {
		t.Errorf("expected Error 'Fatal error', got %q", data.Error)
	}
}
