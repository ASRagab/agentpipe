package core

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewUserMessage(t *testing.T) {
	content := "Hello, world!"
	msg := NewUserMessage(content)

	// Check UUID is valid (non-empty)
	if msg.ID == "" {
		t.Error("expected non-empty message ID")
	}

	// Check timestamp is recent
	if time.Since(msg.Timestamp) > time.Second {
		t.Error("timestamp should be recent")
	}

	// Check role
	if msg.Role != RoleUser {
		t.Errorf("expected role %q, got %q", RoleUser, msg.Role)
	}

	// Check content
	if msg.Content != content {
		t.Errorf("expected content %q, got %q", content, msg.Content)
	}

	// Check status
	if msg.Status != MessageStatusComplete {
		t.Errorf("expected status %q, got %q", MessageStatusComplete, msg.Status)
	}

	// Agent fields should be empty for user messages
	if msg.AgentID != "" {
		t.Errorf("expected empty AgentID, got %q", msg.AgentID)
	}
	if msg.AgentName != "" {
		t.Errorf("expected empty AgentName, got %q", msg.AgentName)
	}
	if msg.Metrics != nil {
		t.Error("expected nil Metrics for user message")
	}
}

func TestNewUserMessage_UniqueIDs(t *testing.T) {
	msg1 := NewUserMessage("test1")
	msg2 := NewUserMessage("test2")

	if msg1.ID == msg2.ID {
		t.Error("expected unique message IDs")
	}
}

func TestNewAgentMessage(t *testing.T) {
	agentID := "agent-1"
	agentName := "Test Agent"
	content := "Agent response"
	metrics := &Metrics{
		Duration:     100 * time.Millisecond,
		InputTokens:  10,
		OutputTokens: 20,
		TotalTokens:  30,
		Model:        "test-model",
		Cost:         0.001,
	}

	msg := NewAgentMessage(agentID, agentName, content, metrics)

	// Check UUID
	if msg.ID == "" {
		t.Error("expected non-empty message ID")
	}

	// Check timestamp
	if time.Since(msg.Timestamp) > time.Second {
		t.Error("timestamp should be recent")
	}

	// Check role
	if msg.Role != RoleAgent {
		t.Errorf("expected role %q, got %q", RoleAgent, msg.Role)
	}

	// Check agent fields
	if msg.AgentID != agentID {
		t.Errorf("expected AgentID %q, got %q", agentID, msg.AgentID)
	}
	if msg.AgentName != agentName {
		t.Errorf("expected AgentName %q, got %q", agentName, msg.AgentName)
	}

	// Check content
	if msg.Content != content {
		t.Errorf("expected content %q, got %q", content, msg.Content)
	}

	// Check status
	if msg.Status != MessageStatusComplete {
		t.Errorf("expected status %q, got %q", MessageStatusComplete, msg.Status)
	}

	// Check metrics
	if msg.Metrics == nil {
		t.Fatal("expected non-nil Metrics")
	}
	if msg.Metrics.TotalTokens != 30 {
		t.Errorf("expected TotalTokens=30, got %d", msg.Metrics.TotalTokens)
	}
	if msg.Metrics.Cost != 0.001 {
		t.Errorf("expected Cost=0.001, got %f", msg.Metrics.Cost)
	}
}

func TestNewAgentMessage_NilMetrics(t *testing.T) {
	msg := NewAgentMessage("agent-1", "Agent", "response", nil)
	if msg.Metrics != nil {
		t.Error("expected nil Metrics when nil is passed")
	}
}

func TestNewSystemMessage(t *testing.T) {
	content := "System notification"
	msg := NewSystemMessage(content)

	if msg.ID == "" {
		t.Error("expected non-empty message ID")
	}
	if msg.Role != RoleSystem {
		t.Errorf("expected role %q, got %q", RoleSystem, msg.Role)
	}
	if msg.Content != content {
		t.Errorf("expected content %q, got %q", content, msg.Content)
	}
	if msg.Status != MessageStatusComplete {
		t.Errorf("expected status %q, got %q", MessageStatusComplete, msg.Status)
	}
}

func TestNewPendingAgentMessage(t *testing.T) {
	agentID := "agent-1"
	agentName := "Test Agent"
	msg := NewPendingAgentMessage(agentID, agentName)

	if msg.ID == "" {
		t.Error("expected non-empty message ID")
	}
	if msg.Role != RoleAgent {
		t.Errorf("expected role %q, got %q", RoleAgent, msg.Role)
	}
	if msg.AgentID != agentID {
		t.Errorf("expected AgentID %q, got %q", agentID, msg.AgentID)
	}
	if msg.AgentName != agentName {
		t.Errorf("expected AgentName %q, got %q", agentName, msg.AgentName)
	}
	if msg.Status != MessageStatusPending {
		t.Errorf("expected status %q, got %q", MessageStatusPending, msg.Status)
	}
	if msg.Content != "" {
		t.Errorf("expected empty content, got %q", msg.Content)
	}
}

func TestMessage_Complete(t *testing.T) {
	msg := NewPendingAgentMessage("agent-1", "Agent")
	metrics := &Metrics{
		TotalTokens: 50,
		Cost:        0.005,
	}

	msg.Complete("Completed response", metrics)

	if msg.Status != MessageStatusComplete {
		t.Errorf("expected status %q, got %q", MessageStatusComplete, msg.Status)
	}
	if msg.Content != "Completed response" {
		t.Errorf("expected content 'Completed response', got %q", msg.Content)
	}
	if msg.Metrics == nil {
		t.Fatal("expected non-nil Metrics")
	}
	if msg.Metrics.TotalTokens != 50 {
		t.Errorf("expected TotalTokens=50, got %d", msg.Metrics.TotalTokens)
	}
}

func TestMessage_SetError(t *testing.T) {
	msg := NewPendingAgentMessage("agent-1", "Agent")
	errContent := "Something went wrong"

	msg.SetError(errContent)

	if msg.Status != MessageStatusError {
		t.Errorf("expected status %q, got %q", MessageStatusError, msg.Status)
	}
	if msg.Content != errContent {
		t.Errorf("expected content %q, got %q", errContent, msg.Content)
	}
}

func TestMessage_SetStreaming(t *testing.T) {
	msg := NewPendingAgentMessage("agent-1", "Agent")

	msg.SetStreaming()

	if msg.Status != MessageStatusStreaming {
		t.Errorf("expected status %q, got %q", MessageStatusStreaming, msg.Status)
	}
}

func TestMessage_AppendContent(t *testing.T) {
	msg := NewPendingAgentMessage("agent-1", "Agent")
	msg.SetStreaming()

	msg.AppendContent("Hello ")
	msg.AppendContent("world")
	msg.AppendContent("!")

	if msg.Content != "Hello world!" {
		t.Errorf("expected content 'Hello world!', got %q", msg.Content)
	}
}

func TestMessage_JSONSerialization(t *testing.T) {
	original := NewAgentMessage("agent-1", "Test Agent", "response", &Metrics{
		Duration:     100 * time.Millisecond,
		InputTokens:  10,
		OutputTokens: 20,
		TotalTokens:  30,
		Model:        "test-model",
		Cost:         0.001,
	})

	// Serialize
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal message: %v", err)
	}

	// Deserialize
	var restored Message
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}

	// Verify fields
	if restored.ID != original.ID {
		t.Errorf("ID mismatch: expected %q, got %q", original.ID, restored.ID)
	}
	if restored.Role != original.Role {
		t.Errorf("Role mismatch: expected %q, got %q", original.Role, restored.Role)
	}
	if restored.AgentID != original.AgentID {
		t.Errorf("AgentID mismatch: expected %q, got %q", original.AgentID, restored.AgentID)
	}
	if restored.Content != original.Content {
		t.Errorf("Content mismatch: expected %q, got %q", original.Content, restored.Content)
	}
	if restored.Status != original.Status {
		t.Errorf("Status mismatch: expected %q, got %q", original.Status, restored.Status)
	}
	if restored.Metrics == nil {
		t.Fatal("expected non-nil Metrics after deserialization")
	}
	if restored.Metrics.TotalTokens != original.Metrics.TotalTokens {
		t.Errorf("TotalTokens mismatch: expected %d, got %d", original.Metrics.TotalTokens, restored.Metrics.TotalTokens)
	}
}

func TestMessageStatus_Constants(t *testing.T) {
	// Verify status constants are distinct
	statuses := []MessageStatus{
		MessageStatusPending,
		MessageStatusStreaming,
		MessageStatusComplete,
		MessageStatusError,
	}

	seen := make(map[MessageStatus]bool)
	for _, s := range statuses {
		if seen[s] {
			t.Errorf("duplicate status constant: %q", s)
		}
		seen[s] = true
	}
}

func TestMessageRole_Constants(t *testing.T) {
	// Verify role constants are distinct
	roles := []MessageRole{
		RoleUser,
		RoleAgent,
		RoleSystem,
	}

	seen := make(map[MessageRole]bool)
	for _, r := range roles {
		if seen[r] {
			t.Errorf("duplicate role constant: %q", r)
		}
		seen[r] = true
	}
}

func TestMetrics_JSONSerialization(t *testing.T) {
	original := Metrics{
		Duration:     150 * time.Millisecond,
		InputTokens:  100,
		OutputTokens: 200,
		TotalTokens:  300,
		Model:        "gpt-4",
		Cost:         0.05,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal metrics: %v", err)
	}

	var restored Metrics
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("failed to unmarshal metrics: %v", err)
	}

	if restored.InputTokens != original.InputTokens {
		t.Errorf("InputTokens mismatch: expected %d, got %d", original.InputTokens, restored.InputTokens)
	}
	if restored.OutputTokens != original.OutputTokens {
		t.Errorf("OutputTokens mismatch: expected %d, got %d", original.OutputTokens, restored.OutputTokens)
	}
	if restored.Model != original.Model {
		t.Errorf("Model mismatch: expected %q, got %q", original.Model, restored.Model)
	}
}
