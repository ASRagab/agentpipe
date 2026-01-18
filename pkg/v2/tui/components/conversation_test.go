package components

import (
	"testing"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/v2/core"
)

func TestNewConversationModel(t *testing.T) {
	model := NewConversationModel()

	if model.messages == nil {
		t.Error("Expected messages to be initialized")
	}
	if model.streamingMessages == nil {
		t.Error("Expected streamingMessages to be initialized")
	}
	if model.agentIndex == nil {
		t.Error("Expected agentIndex to be initialized")
	}
	if !model.cursorVisible {
		t.Error("Expected cursorVisible to be true by default")
	}
	if !model.autoScroll {
		t.Error("Expected autoScroll to be true by default")
	}
	if model.userScrolledUp {
		t.Error("Expected userScrolledUp to be false by default")
	}
}

func TestStreamingChunks(t *testing.T) {
	model := NewConversationModel()

	// Start streaming
	model.StartStreaming("msg-1", "agent-1", "Claude")

	if !model.HasStreamingMessages() {
		t.Error("Expected HasStreamingMessages to return true")
	}
	if model.GetStreamingMessageCount() != 1 {
		t.Errorf("Expected 1 streaming message, got %d", model.GetStreamingMessageCount())
	}

	// Verify streaming message was created
	sm := model.streamingMessages["msg-1"]
	if sm == nil {
		t.Fatal("Expected streaming message to be created")
	}
	if sm.AgentID != "agent-1" {
		t.Errorf("Expected AgentID 'agent-1', got '%s'", sm.AgentID)
	}
	if sm.AgentName != "Claude" {
		t.Errorf("Expected AgentName 'Claude', got '%s'", sm.AgentName)
	}
	if sm.Content != "" {
		t.Errorf("Expected empty content initially, got '%s'", sm.Content)
	}

	// Append chunks
	chunk1 := core.MessageChunk{
		MessageID: "msg-1",
		AgentID:   "agent-1",
		AgentName: "Claude",
		Content:   "Hello",
		Index:     0,
		IsFinal:   false,
	}
	model.AppendChunk(chunk1)

	if sm.Content != "Hello" {
		t.Errorf("Expected content 'Hello', got '%s'", sm.Content)
	}
	if sm.ChunkCount != 1 {
		t.Errorf("Expected ChunkCount 1, got %d", sm.ChunkCount)
	}

	// Append another chunk
	chunk2 := core.MessageChunk{
		MessageID: "msg-1",
		AgentID:   "agent-1",
		AgentName: "Claude",
		Content:   " World!",
		Index:     1,
		IsFinal:   false,
	}
	model.AppendChunk(chunk2)

	if sm.Content != "Hello World!" {
		t.Errorf("Expected content 'Hello World!', got '%s'", sm.Content)
	}
	if sm.ChunkCount != 2 {
		t.Errorf("Expected ChunkCount 2, got %d", sm.ChunkCount)
	}
}

func TestMultipleAgentsStreaming(t *testing.T) {
	model := NewConversationModel()

	// Start streaming from multiple agents
	model.StartStreaming("msg-1", "agent-1", "Claude")
	model.StartStreaming("msg-2", "agent-2", "Gemini")

	if model.GetStreamingMessageCount() != 2 {
		t.Errorf("Expected 2 streaming messages, got %d", model.GetStreamingMessageCount())
	}

	// Append chunks to different agents
	model.AppendChunk(core.MessageChunk{
		MessageID: "msg-1",
		AgentID:   "agent-1",
		AgentName: "Claude",
		Content:   "From Claude",
	})
	model.AppendChunk(core.MessageChunk{
		MessageID: "msg-2",
		AgentID:   "agent-2",
		AgentName: "Gemini",
		Content:   "From Gemini",
	})

	// Verify both have correct content
	if model.streamingMessages["msg-1"].Content != "From Claude" {
		t.Errorf("Expected Claude content, got '%s'", model.streamingMessages["msg-1"].Content)
	}
	if model.streamingMessages["msg-2"].Content != "From Gemini" {
		t.Errorf("Expected Gemini content, got '%s'", model.streamingMessages["msg-2"].Content)
	}
}

func TestStreamingMessageOrder(t *testing.T) {
	model := NewConversationModel()

	// Start streaming in specific order with slight delays
	model.StartStreaming("msg-1", "agent-1", "First")
	time.Sleep(10 * time.Millisecond)
	model.StartStreaming("msg-2", "agent-2", "Second")
	time.Sleep(10 * time.Millisecond)
	model.StartStreaming("msg-3", "agent-3", "Third")

	// Get sorted messages
	sorted := model.getSortedStreamingMessages()

	if len(sorted) != 3 {
		t.Fatalf("Expected 3 sorted messages, got %d", len(sorted))
	}

	// Verify order by start time
	if sorted[0].MessageID != "msg-1" {
		t.Errorf("Expected first message to be msg-1, got %s", sorted[0].MessageID)
	}
	if sorted[1].MessageID != "msg-2" {
		t.Errorf("Expected second message to be msg-2, got %s", sorted[1].MessageID)
	}
	if sorted[2].MessageID != "msg-3" {
		t.Errorf("Expected third message to be msg-3, got %s", sorted[2].MessageID)
	}
}

func TestCompleteStreaming(t *testing.T) {
	model := NewConversationModel()

	// Start streaming
	model.StartStreaming("msg-1", "agent-1", "Claude")
	model.AppendChunk(core.MessageChunk{
		MessageID: "msg-1",
		AgentID:   "agent-1",
		AgentName: "Claude",
		Content:   "Complete message",
	})

	if model.GetStreamingMessageCount() != 1 {
		t.Errorf("Expected 1 streaming message before completion, got %d", model.GetStreamingMessageCount())
	}
	if model.MessageCount() != 0 {
		t.Errorf("Expected 0 completed messages before completion, got %d", model.MessageCount())
	}

	// Complete the streaming message
	finalMsg := core.Message{
		ID:        "msg-1",
		Role:      core.RoleAgent,
		AgentID:   "agent-1",
		AgentName: "Claude",
		Content:   "Complete message",
		Timestamp: time.Now(),
	}
	model.CompleteStreaming("msg-1", finalMsg)

	if model.GetStreamingMessageCount() != 0 {
		t.Errorf("Expected 0 streaming messages after completion, got %d", model.GetStreamingMessageCount())
	}
	if model.MessageCount() != 1 {
		t.Errorf("Expected 1 completed message after completion, got %d", model.MessageCount())
	}
}

func TestCancelStreaming(t *testing.T) {
	model := NewConversationModel()

	// Start streaming
	model.StartStreaming("msg-1", "agent-1", "Claude")
	model.AppendChunk(core.MessageChunk{
		MessageID: "msg-1",
		Content:   "Partial content",
	})

	if model.GetStreamingMessageCount() != 1 {
		t.Errorf("Expected 1 streaming message, got %d", model.GetStreamingMessageCount())
	}

	// Cancel the streaming
	model.CancelStreaming("msg-1")

	if model.GetStreamingMessageCount() != 0 {
		t.Errorf("Expected 0 streaming messages after cancel, got %d", model.GetStreamingMessageCount())
	}
	if model.MessageCount() != 0 {
		t.Errorf("Expected 0 completed messages after cancel, got %d", model.MessageCount())
	}
}

func TestCursorToggle(t *testing.T) {
	model := NewConversationModel()

	// Initially visible
	if !model.cursorVisible {
		t.Error("Expected cursor to be visible initially")
	}

	// Toggle off
	model.ToggleCursor()
	if model.cursorVisible {
		t.Error("Expected cursor to be invisible after first toggle")
	}

	// Toggle on
	model.ToggleCursor()
	if !model.cursorVisible {
		t.Error("Expected cursor to be visible after second toggle")
	}
}

func TestAutoStartStreamingOnChunk(t *testing.T) {
	model := NewConversationModel()

	// Append chunk without starting streaming first
	chunk := core.MessageChunk{
		MessageID: "msg-auto",
		AgentID:   "agent-1",
		AgentName: "AutoAgent",
		Content:   "Auto-started",
	}
	model.AppendChunk(chunk)

	// Should auto-start streaming
	if !model.HasStreamingMessages() {
		t.Error("Expected streaming to auto-start on chunk")
	}
	if model.streamingMessages["msg-auto"] == nil {
		t.Fatal("Expected streaming message to be auto-created")
	}
	if model.streamingMessages["msg-auto"].Content != "Auto-started" {
		t.Errorf("Expected content 'Auto-started', got '%s'", model.streamingMessages["msg-auto"].Content)
	}
}

func TestMetricsDisplay(t *testing.T) {
	model := NewConversationModel()

	metrics := &core.Metrics{
		Duration:     150 * time.Millisecond,
		TotalTokens:  234,
		InputTokens:  100,
		OutputTokens: 134,
		Cost:         0.012,
	}

	formatted := model.formatMetrics(metrics)

	// Should contain duration
	if formatted == "" {
		t.Error("Expected non-empty metrics string")
	}
	// Check format
	expected := "[150ms | 234 tokens | $0.0120]"
	if formatted != expected {
		t.Errorf("Expected '%s', got '%s'", expected, formatted)
	}
}

func TestNilMetricsDisplay(t *testing.T) {
	model := NewConversationModel()

	formatted := model.formatMetrics(nil)
	if formatted != "" {
		t.Errorf("Expected empty string for nil metrics, got '%s'", formatted)
	}
}

func TestEmptyMetricsDisplay(t *testing.T) {
	model := NewConversationModel()

	metrics := &core.Metrics{}
	formatted := model.formatMetrics(metrics)
	if formatted != "" {
		t.Errorf("Expected empty string for empty metrics, got '%s'", formatted)
	}
}

func TestTotalMessageCount(t *testing.T) {
	model := NewConversationModel()

	// No messages
	if model.TotalMessageCount() != 0 {
		t.Errorf("Expected 0 total messages, got %d", model.TotalMessageCount())
	}

	// Add completed message
	model.AddMessage(core.Message{
		ID:      "msg-1",
		Content: "Hello",
	})
	if model.TotalMessageCount() != 1 {
		t.Errorf("Expected 1 total message, got %d", model.TotalMessageCount())
	}

	// Add streaming message
	model.StartStreaming("msg-2", "agent-1", "Claude")
	if model.TotalMessageCount() != 2 {
		t.Errorf("Expected 2 total messages, got %d", model.TotalMessageCount())
	}
}

func TestUserScrolledUp(t *testing.T) {
	model := NewConversationModel()

	// Initially false
	if model.IsUserScrolledUp() {
		t.Error("Expected userScrolledUp to be false initially")
	}

	// Set to true
	model.SetUserScrolledUp(true)
	if !model.IsUserScrolledUp() {
		t.Error("Expected userScrolledUp to be true after setting")
	}

	// Set back to false
	model.SetUserScrolledUp(false)
	if model.IsUserScrolledUp() {
		t.Error("Expected userScrolledUp to be false after clearing")
	}
}

func TestAgentColorIndex(t *testing.T) {
	model := NewConversationModel()

	// Get color for new agents
	idx1 := model.getAgentColorIndex("agent-1")
	idx2 := model.getAgentColorIndex("agent-2")
	idx3 := model.getAgentColorIndex("agent-1") // Same agent

	// Different agents should get different indices
	if idx1 == idx2 {
		t.Error("Expected different indices for different agents")
	}

	// Same agent should get same index
	if idx1 != idx3 {
		t.Errorf("Expected same index for same agent, got %d and %d", idx1, idx3)
	}
}

func TestSetAgentIndex(t *testing.T) {
	model := NewConversationModel()

	// Set custom agent index
	customIndex := map[string]int{
		"agent-1": 5,
		"agent-2": 3,
	}
	model.SetAgentIndex(customIndex)

	// Verify indices are used
	if model.getAgentColorIndex("agent-1") != 5 {
		t.Errorf("Expected index 5 for agent-1, got %d", model.getAgentColorIndex("agent-1"))
	}
	if model.getAgentColorIndex("agent-2") != 3 {
		t.Errorf("Expected index 3 for agent-2, got %d", model.getAgentColorIndex("agent-2"))
	}
}

func TestRenderMessagesEmpty(t *testing.T) {
	model := NewConversationModel()

	content := model.renderMessages()
	if content == "" {
		t.Error("Expected placeholder text for empty messages")
	}
}

func TestRenderMessagesWithStreaming(t *testing.T) {
	model := NewConversationModel()
	model.cursorVisible = true

	// Add a streaming message
	model.StartStreaming("msg-1", "agent-1", "Claude")
	model.AppendChunk(core.MessageChunk{
		MessageID: "msg-1",
		Content:   "Streaming content",
	})

	content := model.renderMessages()

	// Should contain agent name and content
	if content == "" {
		t.Error("Expected non-empty content")
	}
	// Content should include the streaming text
	if !containsSubstring(content, "Streaming content") {
		t.Error("Expected content to contain 'Streaming content'")
	}
}

// Helper function to check if string contains substring
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
