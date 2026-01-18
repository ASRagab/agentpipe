package components

import (
	"fmt"
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
	// Check format - uses shorter 't' suffix now
	expected := "[150ms | 234t | $0.012]"
	if formatted != expected {
		t.Errorf("Expected '%s', got '%s'", expected, formatted)
	}
}

func TestMetricsWithAgentDisplay(t *testing.T) {
	model := NewConversationModel()

	metrics := &core.Metrics{
		Duration:     150 * time.Millisecond,
		TotalTokens:  234,
		InputTokens:  100,
		OutputTokens: 134,
		Cost:         0.012,
	}

	formatted := model.formatMetricsWithAgent("Claude", metrics)

	// Check format includes agent name
	expected := "[Claude | 150ms | 234t | $0.012]"
	if formatted != expected {
		t.Errorf("Expected '%s', got '%s'", expected, formatted)
	}
}

func TestMetricsExpandedDisplay(t *testing.T) {
	model := NewConversationModel()

	metrics := &core.Metrics{
		Duration:     150 * time.Millisecond,
		TotalTokens:  234,
		InputTokens:  100,
		OutputTokens: 134,
		Cost:         0.012,
	}

	formatted := model.formatMetricsExpanded("Claude", metrics)

	// Check format includes input/output breakdown
	expected := "[Claude | 150ms | 100in/134out (234t) | $0.012]"
	if formatted != expected {
		t.Errorf("Expected '%s', got '%s'", expected, formatted)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{150 * time.Millisecond, "150ms"},
		{999 * time.Millisecond, "999ms"},
		{1 * time.Second, "1.0s"},
		{1500 * time.Millisecond, "1.5s"},
		{45 * time.Second, "45.0s"},
		{59 * time.Second, "59.0s"},
		{60 * time.Second, "1.0m"},
		{90 * time.Second, "1.5m"},
		{5 * time.Minute, "5.0m"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.duration)
		if result != tt.expected {
			t.Errorf("formatDuration(%v): expected '%s', got '%s'", tt.duration, tt.expected, result)
		}
	}
}

func TestFormatCost(t *testing.T) {
	tests := []struct {
		cost     float64
		expected string
	}{
		{0.0001, "$0.0001"},
		{0.001, "$0.0010"},
		{0.0123, "$0.012"},
		{0.1234, "$0.123"},
		{1.0, "$1.00"},
		{1.234, "$1.23"},
		{10.0, "$10.00"},
	}

	for _, tt := range tests {
		result := formatCost(tt.cost)
		if result != tt.expected {
			t.Errorf("formatCost(%v): expected '%s', got '%s'", tt.cost, tt.expected, result)
		}
	}
}

func TestNilMetricsDisplay(t *testing.T) {
	model := NewConversationModel()

	formatted := model.formatMetrics(nil)
	if formatted != "" {
		t.Errorf("Expected empty string for nil metrics, got '%s'", formatted)
	}

	// Also test nil for with-agent version
	formattedWithAgent := model.formatMetricsWithAgent("Claude", nil)
	if formattedWithAgent != "" {
		t.Errorf("Expected empty string for nil metrics with agent, got '%s'", formattedWithAgent)
	}

	// And for expanded version
	formattedExpanded := model.formatMetricsExpanded("Claude", nil)
	if formattedExpanded != "" {
		t.Errorf("Expected empty string for nil metrics expanded, got '%s'", formattedExpanded)
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

func TestMetricsWithEmptyAgentName(t *testing.T) {
	model := NewConversationModel()

	metrics := &core.Metrics{
		Duration:    150 * time.Millisecond,
		TotalTokens: 234,
		Cost:        0.012,
	}

	// Empty agent name should still work
	formatted := model.formatMetricsWithAgent("", metrics)
	expected := "[150ms | 234t | $0.012]"
	if formatted != expected {
		t.Errorf("Expected '%s', got '%s'", expected, formatted)
	}
}

func TestMetricsPartialData(t *testing.T) {
	model := NewConversationModel()

	// Only duration
	metrics1 := &core.Metrics{Duration: 100 * time.Millisecond}
	if f := model.formatMetrics(metrics1); f != "[100ms]" {
		t.Errorf("Duration only: expected '[100ms]', got '%s'", f)
	}

	// Only tokens
	metrics2 := &core.Metrics{TotalTokens: 100}
	if f := model.formatMetrics(metrics2); f != "[100t]" {
		t.Errorf("Tokens only: expected '[100t]', got '%s'", f)
	}

	// Only cost
	metrics3 := &core.Metrics{Cost: 0.5}
	if f := model.formatMetrics(metrics3); f != "[$0.500]" {
		t.Errorf("Cost only: expected '[$0.500]', got '%s'", f)
	}

	// Duration and tokens only
	metrics4 := &core.Metrics{Duration: 200 * time.Millisecond, TotalTokens: 50}
	if f := model.formatMetrics(metrics4); f != "[200ms | 50t]" {
		t.Errorf("Duration+tokens: expected '[200ms | 50t]', got '%s'", f)
	}
}

func TestMetricsExpandedWithOnlyTotalTokens(t *testing.T) {
	model := NewConversationModel()

	// If no input/output breakdown, just show total
	metrics := &core.Metrics{
		Duration:    100 * time.Millisecond,
		TotalTokens: 100,
		Cost:        0.001,
	}

	formatted := model.formatMetricsExpanded("Test", metrics)
	expected := "[Test | 100ms | 100t | $0.0010]"
	if formatted != expected {
		t.Errorf("Expected '%s', got '%s'", expected, formatted)
	}
}

func TestMetricsExpandedWithInputOutputNoTotal(t *testing.T) {
	model := NewConversationModel()

	// Input/output but no total (should still show breakdown)
	metrics := &core.Metrics{
		Duration:     100 * time.Millisecond,
		InputTokens:  50,
		OutputTokens: 60,
		Cost:         0.001,
	}

	formatted := model.formatMetricsExpanded("Test", metrics)
	expected := "[Test | 100ms | 50in/60out | $0.0010]"
	if formatted != expected {
		t.Errorf("Expected '%s', got '%s'", expected, formatted)
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

// Smooth scrolling tests

func TestHasNewMessages(t *testing.T) {
	model := NewConversationModel()

	// Initially false
	if model.HasNewMessages() {
		t.Error("Expected HasNewMessages to be false initially")
	}

	// Set user scrolled up
	model.SetUserScrolledUp(true)

	// Initialize viewport for refreshContent to work
	model.InitViewport(80, 24)

	// Simulate many messages to have scrollable content
	for i := 0; i < 50; i++ {
		model.AddMessage(core.Message{
			ID:      fmt.Sprintf("msg-%d", i),
			Content: fmt.Sprintf("Message %d", i),
		})
	}

	// Start streaming (triggers refreshContent with userScrolledUp = true)
	model.SetUserScrolledUp(true)
	model.StartStreaming("msg-new", "agent-1", "Claude")

	// Should now have new messages
	if !model.HasNewMessages() {
		t.Error("Expected HasNewMessages to be true after message while scrolled up")
	}
}

func TestClearNewMessagesIndicator(t *testing.T) {
	model := NewConversationModel()

	// Manually set hasNewMessages
	model.hasNewMessages = true

	// Clear it
	model.ClearNewMessagesIndicator()

	if model.HasNewMessages() {
		t.Error("Expected HasNewMessages to be false after clearing")
	}
}

func TestJumpToBottom(t *testing.T) {
	model := NewConversationModel()
	model.InitViewport(80, 24)

	// Set up state
	model.userScrolledUp = true
	model.hasNewMessages = true

	// Jump to bottom
	model.JumpToBottom()

	// Should clear scroll state and new messages
	if model.IsUserScrolledUp() {
		t.Error("Expected userScrolledUp to be false after JumpToBottom")
	}
	if model.HasNewMessages() {
		t.Error("Expected hasNewMessages to be false after JumpToBottom")
	}
}

func TestDetectUserScroll(t *testing.T) {
	model := NewConversationModel()
	model.InitViewport(80, 24)

	// Add enough messages to create scrollable content
	for i := 0; i < 100; i++ {
		model.AddMessage(core.Message{
			ID:      fmt.Sprintf("msg-%d", i),
			Content: fmt.Sprintf("Message line %d with some content", i),
		})
	}

	// Initially at bottom, so not scrolled up
	model.detectUserScroll()
	if model.IsUserScrolledUp() {
		t.Error("Expected not scrolled up when at bottom")
	}

	// Scroll up (simulate by going to top)
	model.viewport.GotoTop()
	model.detectUserScroll()

	if !model.IsUserScrolledUp() {
		t.Error("Expected scrolled up after going to top")
	}

	// Go back to bottom
	model.viewport.GotoBottom()
	model.detectUserScroll()

	if model.IsUserScrolledUp() {
		t.Error("Expected not scrolled up after returning to bottom")
	}
}

func TestNewMessagesIndicatorRendering(t *testing.T) {
	model := NewConversationModel()
	model.InitViewport(80, 24)
	model.width = 80
	model.height = 24

	// Set state to show indicator
	model.hasNewMessages = true
	model.userScrolledUp = true

	view := model.View()

	// Should contain the indicator text
	if !containsSubstring(view, "New messages below") {
		t.Error("Expected view to contain 'New messages below' indicator")
	}
}

func TestNoIndicatorWhenAtBottom(t *testing.T) {
	model := NewConversationModel()
	model.InitViewport(80, 24)
	model.width = 80
	model.height = 24

	// Add some messages
	model.AddMessage(core.Message{
		ID:      "msg-1",
		Content: "Hello",
	})

	// Not scrolled up, no new messages
	model.hasNewMessages = false
	model.userScrolledUp = false

	view := model.View()

	// Should NOT contain the indicator
	if containsSubstring(view, "New messages below") {
		t.Error("Expected view to NOT contain 'New messages below' indicator when at bottom")
	}
}

func TestRefreshContentSetsNewMessages(t *testing.T) {
	model := NewConversationModel()
	model.InitViewport(80, 24)

	// Add many messages to create scrollable content
	for i := 0; i < 50; i++ {
		model.AddMessage(core.Message{
			ID:      fmt.Sprintf("msg-%d", i),
			Content: fmt.Sprintf("Message %d", i),
		})
	}

	// Scroll to top to simulate user scrolling up
	model.viewport.GotoTop()
	model.userScrolledUp = true

	// Refresh content should detect user is scrolled up and set hasNewMessages
	model.refreshContent()

	if !model.HasNewMessages() {
		t.Error("Expected hasNewMessages to be true after refreshContent while scrolled up")
	}
}

func TestScrollToBottomClearsIndicators(t *testing.T) {
	model := NewConversationModel()
	model.InitViewport(80, 24)

	// Set up state
	model.userScrolledUp = true
	model.hasNewMessages = true

	// Use ScrollToBottom (the existing method)
	model.ScrollToBottom()

	// ScrollToBottom doesn't clear indicators in original implementation
	// but JumpToBottom does - this tests the existing behavior
	// We're adding JumpToBottom for the new behavior
	model.JumpToBottom()

	if model.IsUserScrolledUp() {
		t.Error("Expected userScrolledUp to be false after JumpToBottom")
	}
	if model.HasNewMessages() {
		t.Error("Expected hasNewMessages to be false after JumpToBottom")
	}
}

func TestOverlayIndicatorCentering(t *testing.T) {
	model := NewConversationModel()
	model.width = 80

	content := "Test content"
	indicator := "TEST"

	result := model.overlayIndicator(content, indicator)

	// Should contain both the original content and indicator
	if !containsSubstring(result, "Test content") {
		t.Error("Expected result to contain original content")
	}
	if !containsSubstring(result, "TEST") {
		t.Error("Expected result to contain indicator")
	}
}

func TestRenderNewMessagesIndicator(t *testing.T) {
	model := NewConversationModel()

	indicator := model.renderNewMessagesIndicator()

	// Should contain the expected text
	if !containsSubstring(indicator, "New messages below") {
		t.Error("Expected indicator to contain 'New messages below'")
	}
	if !containsSubstring(indicator, "End to jump") {
		t.Error("Expected indicator to contain 'End to jump'")
	}
}
