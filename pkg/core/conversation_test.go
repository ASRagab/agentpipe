package core

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewConversation(t *testing.T) {
	agents := []Agent{
		NewAgent("agent-1", "openrouter", "Agent 1", "gpt-4", "openrouter"),
		NewAgent("agent-2", "claude-api", "Agent 2", "claude-3", "claude-api"),
	}

	conv := NewConversation(agents)

	if conv.ID == "" {
		t.Error("expected non-empty conversation ID")
	}
	if len(conv.Agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(conv.Agents))
	}
	if len(conv.Messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(conv.Messages))
	}
	if conv.Status != ConversationStatusActive {
		t.Errorf("expected status %q, got %q", ConversationStatusActive, conv.Status)
	}
	if time.Since(conv.Started) > time.Second {
		t.Error("Started timestamp should be recent")
	}
	if time.Since(conv.Updated) > time.Second {
		t.Error("Updated timestamp should be recent")
	}
	if conv.Metadata == nil {
		t.Error("expected non-nil Metadata map")
	}
}

func TestConversation_AddMessage(t *testing.T) {
	conv := NewConversation(nil)
	initialUpdated := conv.Updated

	// Small delay to ensure timestamp differs
	time.Sleep(1 * time.Millisecond)

	msg := NewUserMessage("Hello")
	conv.AddMessage(msg)

	if len(conv.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(conv.Messages))
	}
	if conv.Messages[0].Content != "Hello" {
		t.Errorf("expected content 'Hello', got %q", conv.Messages[0].Content)
	}
	if !conv.Updated.After(initialUpdated) {
		t.Error("Updated timestamp should be updated after AddMessage")
	}
}

func TestConversation_GetMessages(t *testing.T) {
	conv := NewConversation(nil)
	conv.AddMessage(NewUserMessage("Hello"))
	conv.AddMessage(NewAgentMessage("agent-1", "Agent", "Hi there", nil))

	messages := conv.GetMessages()

	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}

	// Verify it's a copy - modifying shouldn't affect original
	messages[0].Content = "Modified"
	if conv.Messages[0].Content == "Modified" {
		t.Error("GetMessages should return a copy, not the original")
	}
}

func TestConversation_GetLastMessage(t *testing.T) {
	conv := NewConversation(nil)

	// Empty conversation
	if conv.GetLastMessage() != nil {
		t.Error("expected nil for empty conversation")
	}

	conv.AddMessage(NewUserMessage("First"))
	conv.AddMessage(NewUserMessage("Second"))

	last := conv.GetLastMessage()
	if last == nil {
		t.Fatal("expected non-nil last message")
	}
	if last.Content != "Second" {
		t.Errorf("expected last message 'Second', got %q", last.Content)
	}
}

func TestConversation_GetMessagesByAgent(t *testing.T) {
	conv := NewConversation(nil)
	conv.AddMessage(NewUserMessage("Hi"))
	conv.AddMessage(NewAgentMessage("agent-1", "Agent 1", "Hello from 1", nil))
	conv.AddMessage(NewAgentMessage("agent-2", "Agent 2", "Hello from 2", nil))
	conv.AddMessage(NewAgentMessage("agent-1", "Agent 1", "Second from 1", nil))

	agent1Messages := conv.GetMessagesByAgent("agent-1")
	if len(agent1Messages) != 2 {
		t.Errorf("expected 2 messages from agent-1, got %d", len(agent1Messages))
	}

	agent2Messages := conv.GetMessagesByAgent("agent-2")
	if len(agent2Messages) != 1 {
		t.Errorf("expected 1 message from agent-2, got %d", len(agent2Messages))
	}

	unknownMessages := conv.GetMessagesByAgent("unknown")
	if len(unknownMessages) != 0 {
		t.Errorf("expected 0 messages from unknown agent, got %d", len(unknownMessages))
	}
}

func TestConversation_GetAgentByID(t *testing.T) {
	agents := []Agent{
		NewAgent("agent-1", "openrouter", "Agent 1", "gpt-4", "openrouter"),
		NewAgent("agent-2", "claude-api", "Agent 2", "claude-3", "claude-api"),
	}
	conv := NewConversation(agents)

	agent := conv.GetAgentByID("agent-1")
	if agent == nil {
		t.Fatal("expected to find agent-1")
	}
	if agent.Name != "Agent 1" {
		t.Errorf("expected Name 'Agent 1', got %q", agent.Name)
	}

	unknown := conv.GetAgentByID("unknown")
	if unknown != nil {
		t.Error("expected nil for unknown agent ID")
	}
}

func TestConversation_GetAgentByName(t *testing.T) {
	agents := []Agent{
		NewAgent("agent-1", "openrouter", "Agent 1", "gpt-4", "openrouter"),
		NewAgent("agent-2", "claude-api", "Agent 2", "claude-3", "claude-api"),
	}
	conv := NewConversation(agents)

	agent := conv.GetAgentByName("Agent 2")
	if agent == nil {
		t.Fatal("expected to find Agent 2")
	}
	if agent.ID != "agent-2" {
		t.Errorf("expected ID 'agent-2', got %q", agent.ID)
	}

	unknown := conv.GetAgentByName("Unknown Agent")
	if unknown != nil {
		t.Error("expected nil for unknown agent name")
	}
}

func TestConversation_MessageCount(t *testing.T) {
	conv := NewConversation(nil)

	if conv.MessageCount() != 0 {
		t.Errorf("expected 0 messages, got %d", conv.MessageCount())
	}

	conv.AddMessage(NewUserMessage("1"))
	conv.AddMessage(NewUserMessage("2"))
	conv.AddMessage(NewUserMessage("3"))

	if conv.MessageCount() != 3 {
		t.Errorf("expected 3 messages, got %d", conv.MessageCount())
	}
}

func TestConversation_AgentCount(t *testing.T) {
	agents := []Agent{
		NewAgent("agent-1", "openrouter", "Agent 1", "gpt-4", "openrouter"),
		NewAgent("agent-2", "claude-api", "Agent 2", "claude-3", "claude-api"),
	}
	conv := NewConversation(agents)

	if conv.AgentCount() != 2 {
		t.Errorf("expected 2 agents, got %d", conv.AgentCount())
	}

	emptyConv := NewConversation(nil)
	if emptyConv.AgentCount() != 0 {
		t.Errorf("expected 0 agents, got %d", emptyConv.AgentCount())
	}
}

func TestConversation_Duration(t *testing.T) {
	conv := NewConversation(nil)

	// Duration should be very small
	duration := conv.Duration()
	if duration > time.Second {
		t.Errorf("expected duration < 1s, got %v", duration)
	}
}

func TestConversation_StatusTransitions(t *testing.T) {
	conv := NewConversation(nil)

	// Initial status
	if conv.Status != ConversationStatusActive {
		t.Errorf("expected initial status %q, got %q", ConversationStatusActive, conv.Status)
	}

	// Pause
	conv.Pause()
	if conv.Status != ConversationStatusPaused {
		t.Errorf("expected status %q, got %q", ConversationStatusPaused, conv.Status)
	}

	// Resume
	conv.Resume()
	if conv.Status != ConversationStatusActive {
		t.Errorf("expected status %q, got %q", ConversationStatusActive, conv.Status)
	}

	// Complete
	conv.Complete()
	if conv.Status != ConversationStatusCompleted {
		t.Errorf("expected status %q, got %q", ConversationStatusCompleted, conv.Status)
	}
}

func TestConversation_SetError(t *testing.T) {
	conv := NewConversation(nil)
	conv.SetError()

	if conv.Status != ConversationStatusError {
		t.Errorf("expected status %q, got %q", ConversationStatusError, conv.Status)
	}
}

func TestConversation_SetStatus(t *testing.T) {
	conv := NewConversation(nil)
	initialUpdated := conv.Updated

	time.Sleep(1 * time.Millisecond)
	conv.SetStatus(ConversationStatusPaused)

	if conv.Status != ConversationStatusPaused {
		t.Errorf("expected status %q, got %q", ConversationStatusPaused, conv.Status)
	}
	if !conv.Updated.After(initialUpdated) {
		t.Error("Updated timestamp should be updated after SetStatus")
	}
}

func TestConversation_Metadata(t *testing.T) {
	conv := NewConversation(nil)

	// Set metadata
	conv.SetMetadata("key1", "value1")
	conv.SetMetadata("key2", 42)

	// Get metadata
	val1, ok := conv.GetMetadata("key1")
	if !ok {
		t.Error("expected to find key1")
	}
	if val1 != "value1" {
		t.Errorf("expected value1, got %v", val1)
	}

	val2, ok := conv.GetMetadata("key2")
	if !ok {
		t.Error("expected to find key2")
	}
	if val2 != 42 {
		t.Errorf("expected 42, got %v", val2)
	}

	// Get unknown key
	_, ok = conv.GetMetadata("unknown")
	if ok {
		t.Error("expected not to find unknown key")
	}
}

func TestConversation_GetMetadata_NilMap(t *testing.T) {
	conv := &Conversation{}

	_, ok := conv.GetMetadata("key")
	if ok {
		t.Error("expected false for nil Metadata map")
	}
}

func TestConversation_SetMetadata_NilMap(t *testing.T) {
	conv := &Conversation{}

	conv.SetMetadata("key", "value")

	if conv.Metadata == nil {
		t.Error("expected Metadata map to be initialized")
	}
	val, ok := conv.GetMetadata("key")
	if !ok || val != "value" {
		t.Error("expected to find set value")
	}
}

func TestConversation_TotalTokens(t *testing.T) {
	conv := NewConversation(nil)

	// No messages
	if conv.TotalTokens() != 0 {
		t.Errorf("expected 0 tokens, got %d", conv.TotalTokens())
	}

	// Add messages with metrics
	conv.AddMessage(NewUserMessage("Hi")) // No metrics
	conv.AddMessage(NewAgentMessage("agent-1", "Agent", "Hello", &Metrics{TotalTokens: 50}))
	conv.AddMessage(NewAgentMessage("agent-2", "Agent", "Hi", &Metrics{TotalTokens: 30}))

	if conv.TotalTokens() != 80 {
		t.Errorf("expected 80 tokens, got %d", conv.TotalTokens())
	}
}

func TestConversation_TotalCost(t *testing.T) {
	conv := NewConversation(nil)

	// No messages
	if conv.TotalCost() != 0 {
		t.Errorf("expected 0 cost, got %f", conv.TotalCost())
	}

	// Add messages with metrics
	conv.AddMessage(NewAgentMessage("agent-1", "Agent", "Hello", &Metrics{Cost: 0.01}))
	conv.AddMessage(NewAgentMessage("agent-2", "Agent", "Hi", &Metrics{Cost: 0.02}))

	if conv.TotalCost() != 0.03 {
		t.Errorf("expected 0.03 cost, got %f", conv.TotalCost())
	}
}

func TestConversation_Summary(t *testing.T) {
	agents := []Agent{
		NewAgent("agent-1", "openrouter", "Agent 1", "gpt-4", "openrouter"),
	}
	conv := NewConversation(agents)
	conv.AddMessage(NewUserMessage("Hi"))
	conv.AddMessage(NewAgentMessage("agent-1", "Agent", "Hello", &Metrics{TotalTokens: 50, Cost: 0.01}))

	summary := conv.Summary()

	if summary.ID != conv.ID {
		t.Errorf("ID mismatch: expected %q, got %q", conv.ID, summary.ID)
	}
	if summary.Status != ConversationStatusActive {
		t.Errorf("expected status %q, got %q", ConversationStatusActive, summary.Status)
	}
	if summary.AgentCount != 1 {
		t.Errorf("expected AgentCount=1, got %d", summary.AgentCount)
	}
	if summary.MessageCount != 2 {
		t.Errorf("expected MessageCount=2, got %d", summary.MessageCount)
	}
	if summary.TotalTokens != 50 {
		t.Errorf("expected TotalTokens=50, got %d", summary.TotalTokens)
	}
	if summary.TotalCost != 0.01 {
		t.Errorf("expected TotalCost=0.01, got %f", summary.TotalCost)
	}
}

func TestConversationStatus_Constants(t *testing.T) {
	statuses := []ConversationStatus{
		ConversationStatusActive,
		ConversationStatusPaused,
		ConversationStatusCompleted,
		ConversationStatusError,
	}

	seen := make(map[ConversationStatus]bool)
	for _, s := range statuses {
		if seen[s] {
			t.Errorf("duplicate status constant: %q", s)
		}
		seen[s] = true
	}
}

func TestConversation_JSONSerialization(t *testing.T) {
	agents := []Agent{
		NewAgent("agent-1", "openrouter", "Agent 1", "gpt-4", "openrouter"),
	}
	original := NewConversation(agents)
	original.AddMessage(NewUserMessage("Hello"))
	original.SetMetadata("test", "value")

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal conversation: %v", err)
	}

	var restored Conversation
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("failed to unmarshal conversation: %v", err)
	}

	if restored.ID != original.ID {
		t.Errorf("ID mismatch: expected %q, got %q", original.ID, restored.ID)
	}
	if restored.Status != original.Status {
		t.Errorf("Status mismatch: expected %q, got %q", original.Status, restored.Status)
	}
	if len(restored.Messages) != len(original.Messages) {
		t.Errorf("Message count mismatch: expected %d, got %d", len(original.Messages), len(restored.Messages))
	}
	if len(restored.Agents) != len(original.Agents) {
		t.Errorf("Agent count mismatch: expected %d, got %d", len(original.Agents), len(restored.Agents))
	}
}
