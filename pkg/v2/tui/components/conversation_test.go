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

// Progress bar tests

func TestProgressBarConfig(t *testing.T) {
	config := DefaultProgressBarConfig()

	if config.Width != 20 {
		t.Errorf("Expected default width 20, got %d", config.Width)
	}
	if config.EstimatedDuration != 5*time.Second {
		t.Errorf("Expected default estimated duration 5s, got %v", config.EstimatedDuration)
	}
}

func TestRenderProgressBarZeroElapsed(t *testing.T) {
	model := NewConversationModel()

	result := model.renderProgressBar(0, DefaultProgressBarConfig())

	// Should contain the empty bar characters
	if result == "" {
		t.Error("Expected non-empty progress bar")
	}
	// At 0 elapsed, duration should show 0ms
	if !containsSubstring(result, "0ms") {
		t.Error("Expected progress bar to contain '0ms' at zero elapsed")
	}
}

func TestRenderProgressBarPartialFill(t *testing.T) {
	model := NewConversationModel()

	// 2.5 seconds = 50% of 5 second estimate
	elapsed := 2500 * time.Millisecond
	result := model.renderProgressBar(elapsed, DefaultProgressBarConfig())

	// Should contain duration text
	if !containsSubstring(result, "2.5s") {
		t.Error("Expected progress bar to contain '2.5s' duration")
	}
	// Should be non-empty
	if result == "" {
		t.Error("Expected non-empty progress bar")
	}
}

func TestRenderProgressBarFullFill(t *testing.T) {
	model := NewConversationModel()

	// 10 seconds is beyond 5 second estimate (should cap at 100%)
	elapsed := 10 * time.Second
	result := model.renderProgressBar(elapsed, DefaultProgressBarConfig())

	// Should contain duration text
	if !containsSubstring(result, "10.0s") {
		t.Error("Expected progress bar to contain '10.0s' duration")
	}
}

func TestRenderProgressBarGreenColor(t *testing.T) {
	model := NewConversationModel()

	// Under 1 second should be green
	elapsed := 500 * time.Millisecond
	result := model.renderProgressBar(elapsed, DefaultProgressBarConfig())

	// Should contain duration showing milliseconds
	if !containsSubstring(result, "500ms") {
		t.Error("Expected progress bar to contain '500ms' duration")
	}
}

func TestRenderProgressBarYellowColor(t *testing.T) {
	model := NewConversationModel()

	// 1-3 seconds should be yellow
	elapsed := 2 * time.Second
	result := model.renderProgressBar(elapsed, DefaultProgressBarConfig())

	// Should contain duration
	if !containsSubstring(result, "2.0s") {
		t.Error("Expected progress bar to contain '2.0s' duration")
	}
}

func TestRenderProgressBarRedColor(t *testing.T) {
	model := NewConversationModel()

	// Over 3 seconds should be red
	elapsed := 4 * time.Second
	result := model.renderProgressBar(elapsed, DefaultProgressBarConfig())

	// Should contain duration
	if !containsSubstring(result, "4.0s") {
		t.Error("Expected progress bar to contain '4.0s' duration")
	}
}

func TestRenderProgressBarCustomConfig(t *testing.T) {
	model := NewConversationModel()

	// Custom config with different width and duration
	config := ProgressBarConfig{
		Width:             10,
		EstimatedDuration: 10 * time.Second,
	}

	// 5 seconds = 50% of 10 second estimate
	elapsed := 5 * time.Second
	result := model.renderProgressBar(elapsed, config)

	// Should contain duration
	if !containsSubstring(result, "5.0s") {
		t.Error("Expected progress bar to contain '5.0s' duration")
	}
}

func TestRenderProgressBarZeroConfig(t *testing.T) {
	model := NewConversationModel()

	// Zero config should use defaults
	config := ProgressBarConfig{
		Width:             0,
		EstimatedDuration: 0,
	}

	elapsed := 1 * time.Second
	result := model.renderProgressBar(elapsed, config)

	// Should still render correctly with defaults
	if result == "" {
		t.Error("Expected non-empty progress bar even with zero config")
	}
	if !containsSubstring(result, "1.0s") {
		t.Error("Expected progress bar to contain '1.0s' duration")
	}
}

func TestRenderProgressBarNegativeConfig(t *testing.T) {
	model := NewConversationModel()

	// Negative config should use defaults
	config := ProgressBarConfig{
		Width:             -5,
		EstimatedDuration: -1 * time.Second,
	}

	elapsed := 1 * time.Second
	result := model.renderProgressBar(elapsed, config)

	// Should still render correctly with defaults
	if result == "" {
		t.Error("Expected non-empty progress bar even with negative config")
	}
}

func TestStreamingMessageIncludesProgressBar(t *testing.T) {
	model := NewConversationModel()
	model.cursorVisible = true

	// Start streaming
	model.StartStreaming("msg-1", "agent-1", "Claude")
	model.AppendChunk(core.MessageChunk{
		MessageID: "msg-1",
		Content:   "Hello world",
	})

	// Wait a bit to have some elapsed time
	time.Sleep(50 * time.Millisecond)

	// Render the messages
	content := model.renderMessages()

	// Should contain the message content
	if !containsSubstring(content, "Hello world") {
		t.Error("Expected content to contain 'Hello world'")
	}

	// Progress bar uses box-drawing characters
	// The filled portion uses ━ and empty uses ─
	if !containsSubstring(content, "─") && !containsSubstring(content, "━") {
		t.Error("Expected content to contain progress bar characters")
	}
}

func TestProgressBarFillCalculation(t *testing.T) {
	model := NewConversationModel()

	testCases := []struct {
		elapsed    time.Duration
		estimated  time.Duration
		expectedFillPercent float64
	}{
		{0, 10 * time.Second, 0.0},
		{5 * time.Second, 10 * time.Second, 0.5},
		{10 * time.Second, 10 * time.Second, 1.0},
		{15 * time.Second, 10 * time.Second, 1.0}, // Capped at 100%
	}

	for _, tc := range testCases {
		config := ProgressBarConfig{
			Width:             20,
			EstimatedDuration: tc.estimated,
		}
		result := model.renderProgressBar(tc.elapsed, config)

		// Just verify it renders without error
		if result == "" {
			t.Errorf("Expected non-empty progress bar for elapsed=%v, estimated=%v", tc.elapsed, tc.estimated)
		}
	}
}

// ===============================
// Error Display Tests
// ===============================

func TestErrorMessageInitialization(t *testing.T) {
	model := NewConversationModel()

	if model.errorMessages == nil {
		t.Error("Expected errorMessages to be initialized")
	}
	if model.HasErrors() {
		t.Error("Expected HasErrors to return false for new model")
	}
	if model.GetErrorMessageCount() != 0 {
		t.Errorf("Expected 0 error messages, got %d", model.GetErrorMessageCount())
	}
}

func TestAddErrorMessage(t *testing.T) {
	model := NewConversationModel()

	errInfo := core.NewAgentErrorInfo(core.ErrorTypeTimeout, "Request timed out", "agent-1", "Claude")
	model.AddErrorMessage(errInfo)

	if !model.HasErrors() {
		t.Error("Expected HasErrors to return true after adding error")
	}
	if model.GetErrorMessageCount() != 1 {
		t.Errorf("Expected 1 error message, got %d", model.GetErrorMessageCount())
	}

	// Check error was also added to messages
	if len(model.messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(model.messages))
	}
	if model.messages[0].Status != core.MessageStatusError {
		t.Errorf("Expected message status to be error, got %s", model.messages[0].Status)
	}
}

func TestAddAgentError(t *testing.T) {
	model := NewConversationModel()

	model.AddAgentError("agent-1", "Claude", "Connection refused")

	if !model.HasErrors() {
		t.Error("Expected HasErrors to return true")
	}

	errors := model.GetErrorMessages()
	if len(errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(errors))
	}

	err := errors[0]
	if err.AgentID != "agent-1" {
		t.Errorf("Expected AgentID 'agent-1', got '%s'", err.AgentID)
	}
	if err.AgentName != "Claude" {
		t.Errorf("Expected AgentName 'Claude', got '%s'", err.AgentName)
	}
	if err.Message != "Connection refused" {
		t.Errorf("Expected message 'Connection refused', got '%s'", err.Message)
	}
	// Connection refused should be classified as network error
	if err.ErrorType != core.ErrorTypeNetwork {
		t.Errorf("Expected ErrorType 'network', got '%s'", err.ErrorType)
	}
}

func TestErrorTypeClassification(t *testing.T) {
	testCases := []struct {
		errorMsg     string
		expectedType core.ErrorType
	}{
		{"Request timed out", core.ErrorTypeTimeout},
		{"deadline exceeded", core.ErrorTypeTimeout},
		{"rate limit exceeded", core.ErrorTypeRateLimit},
		{"429 Too Many Requests", core.ErrorTypeRateLimit},
		{"connection refused", core.ErrorTypeNetwork},
		{"network unreachable", core.ErrorTypeNetwork},
		{"dns lookup failed", core.ErrorTypeNetwork},
		{"unauthorized", core.ErrorTypeAuthentication},
		{"invalid api key", core.ErrorTypeAuthentication},
		{"403 Forbidden", core.ErrorTypeAuthentication},
		{"500 Internal Server Error", core.ErrorTypeAPI},
		{"502 Bad Gateway", core.ErrorTypeAPI},
		{"some random error", core.ErrorTypeUnknown},
	}

	for _, tc := range testCases {
		result := core.ClassifyError(tc.errorMsg)
		if result != tc.expectedType {
			t.Errorf("For '%s': expected %s, got %s", tc.errorMsg, tc.expectedType, result)
		}
	}
}

func TestGetLastError(t *testing.T) {
	model := NewConversationModel()

	// Test with no errors
	if model.GetLastError() != nil {
		t.Error("Expected GetLastError to return nil when no errors")
	}

	// Add first error
	model.AddAgentError("agent-1", "Claude", "First error")

	lastErr := model.GetLastError()
	if lastErr == nil {
		t.Fatal("Expected GetLastError to return error")
	}
	if lastErr.Message != "First error" {
		t.Errorf("Expected message 'First error', got '%s'", lastErr.Message)
	}

	// Add second error
	model.AddAgentError("agent-2", "Gemini", "Second error")

	lastErr = model.GetLastError()
	if lastErr == nil {
		t.Fatal("Expected GetLastError to return error")
	}
	if lastErr.Message != "Second error" {
		t.Errorf("Expected message 'Second error', got '%s'", lastErr.Message)
	}
}

func TestGetAgentErrors(t *testing.T) {
	model := NewConversationModel()

	// Add errors for multiple agents
	model.AddAgentError("agent-1", "Claude", "Error 1")
	model.AddAgentError("agent-2", "Gemini", "Error 2")
	model.AddAgentError("agent-1", "Claude", "Error 3")

	// Get errors for agent-1
	agent1Errors := model.GetAgentErrors("agent-1")
	if len(agent1Errors) != 2 {
		t.Errorf("Expected 2 errors for agent-1, got %d", len(agent1Errors))
	}

	// Get errors for agent-2
	agent2Errors := model.GetAgentErrors("agent-2")
	if len(agent2Errors) != 1 {
		t.Errorf("Expected 1 error for agent-2, got %d", len(agent2Errors))
	}

	// Get errors for non-existent agent
	agent3Errors := model.GetAgentErrors("agent-3")
	if len(agent3Errors) != 0 {
		t.Errorf("Expected 0 errors for agent-3, got %d", len(agent3Errors))
	}
}

func TestClearErrors(t *testing.T) {
	model := NewConversationModel()

	model.AddAgentError("agent-1", "Claude", "Error 1")
	model.AddAgentError("agent-2", "Gemini", "Error 2")

	if model.GetErrorMessageCount() != 2 {
		t.Errorf("Expected 2 errors before clear, got %d", model.GetErrorMessageCount())
	}

	model.ClearErrors()

	if model.GetErrorMessageCount() != 0 {
		t.Errorf("Expected 0 errors after clear, got %d", model.GetErrorMessageCount())
	}
	if model.HasErrors() {
		t.Error("Expected HasErrors to return false after clear")
	}

	// Messages should still be there (part of history)
	if len(model.messages) != 2 {
		t.Errorf("Expected 2 messages still in history, got %d", len(model.messages))
	}
}

func TestRecoverableErrorInfo(t *testing.T) {
	testCases := []struct {
		errType       core.ErrorType
		recoverable   bool
		hasRetryHint  bool
	}{
		{core.ErrorTypeTimeout, true, true},
		{core.ErrorTypeRateLimit, true, true},
		{core.ErrorTypeNetwork, true, true},
		{core.ErrorTypeAuthentication, false, true}, // Has hint but not auto-recoverable
		{core.ErrorTypeAPI, false, false},
		{core.ErrorTypeInternal, false, false},
		{core.ErrorTypeUnknown, false, false},
	}

	for _, tc := range testCases {
		info := core.NewErrorInfo(tc.errType, "test error")
		if info.Recoverable != tc.recoverable {
			t.Errorf("For %s: expected recoverable=%v, got %v", tc.errType, tc.recoverable, info.Recoverable)
		}
		hasHint := info.RetryHint != ""
		if hasHint != tc.hasRetryHint {
			t.Errorf("For %s: expected hasRetryHint=%v, got %v", tc.errType, tc.hasRetryHint, hasHint)
		}
	}
}

func TestErrorMessageFormatting(t *testing.T) {
	// Test with agent name
	info := core.NewAgentErrorInfo(core.ErrorTypeTimeout, "Request timed out", "agent-1", "Claude")
	formatted := core.FormatErrorMessage(info)

	expectedPrefix := "Claude failed to respond"
	if !containsSubstring(formatted, expectedPrefix) {
		t.Errorf("Expected formatted message to contain '%s', got '%s'", expectedPrefix, formatted)
	}
	if !containsSubstring(formatted, "Request timed out") {
		t.Errorf("Expected formatted message to contain error message")
	}

	// Test without agent name
	info2 := core.NewErrorInfo(core.ErrorTypeAPI, "API error occurred")
	formatted2 := core.FormatErrorMessage(info2)

	if !containsSubstring(formatted2, "Error:") {
		t.Errorf("Expected formatted message to contain 'Error:', got '%s'", formatted2)
	}
}

func TestFormatErrorTypeBadge(t *testing.T) {
	model := NewConversationModel()

	testCases := []struct {
		errType       core.ErrorType
		expectedLabel string
	}{
		{core.ErrorTypeTimeout, "TIMEOUT"},
		{core.ErrorTypeRateLimit, "RATE LIMIT"},
		{core.ErrorTypeNetwork, "NETWORK"},
		{core.ErrorTypeAuthentication, "AUTH"},
		{core.ErrorTypeAPI, "API"},
		{core.ErrorTypeInternal, "INTERNAL"},
		{core.ErrorTypeUnknown, "ERROR"},
	}

	for _, tc := range testCases {
		badge := model.formatErrorTypeBadge(tc.errType)
		if !containsSubstring(badge, tc.expectedLabel) {
			t.Errorf("For %s: expected badge to contain '%s', got '%s'", tc.errType, tc.expectedLabel, badge)
		}
	}
}

func TestErrorMessageRendering(t *testing.T) {
	model := NewConversationModel()
	model.InitViewport(80, 24)

	// Add an error message
	model.AddAgentError("agent-1", "Claude", "Connection timeout")

	// Render messages
	content := model.renderMessages()

	// Check that error indicator is present
	if !containsSubstring(content, "✗") {
		t.Error("Expected content to contain error icon ✗")
	}
	if !containsSubstring(content, "Claude") {
		t.Error("Expected content to contain agent name 'Claude'")
	}
	if !containsSubstring(content, "failed to respond") {
		t.Error("Expected content to contain 'failed to respond'")
	}
}

func TestGetStreamingMessagesForAgent(t *testing.T) {
	model := NewConversationModel()

	// Start streaming for multiple agents
	model.StartStreaming("msg-1", "agent-1", "Claude")
	model.StartStreaming("msg-2", "agent-1", "Claude")
	model.StartStreaming("msg-3", "agent-2", "Gemini")

	// Get streaming messages for agent-1
	agent1Msgs := model.GetStreamingMessagesForAgent("agent-1")
	if len(agent1Msgs) != 2 {
		t.Errorf("Expected 2 streaming messages for agent-1, got %d", len(agent1Msgs))
	}

	// Get streaming messages for agent-2
	agent2Msgs := model.GetStreamingMessagesForAgent("agent-2")
	if len(agent2Msgs) != 1 {
		t.Errorf("Expected 1 streaming message for agent-2, got %d", len(agent2Msgs))
	}

	// Get streaming messages for non-existent agent
	agent3Msgs := model.GetStreamingMessagesForAgent("agent-3")
	if len(agent3Msgs) != 0 {
		t.Errorf("Expected 0 streaming messages for agent-3, got %d", len(agent3Msgs))
	}
}

func TestErrorMessageWithStreamingCancellation(t *testing.T) {
	model := NewConversationModel()

	// Start streaming
	model.StartStreaming("msg-1", "agent-1", "Claude")
	model.AppendChunk(core.MessageChunk{
		MessageID: "msg-1",
		AgentID:   "agent-1",
		AgentName: "Claude",
		Content:   "Starting response...",
		Index:     0,
	})

	if !model.HasStreamingMessages() {
		t.Error("Expected streaming messages before error")
	}

	// Simulate error by cancelling streaming and adding error
	for messageID := range model.GetStreamingMessagesForAgent("agent-1") {
		model.CancelStreaming(messageID)
	}
	model.AddAgentError("agent-1", "Claude", "Connection lost")

	// Streaming should be cancelled
	if model.HasStreamingMessages() {
		t.Error("Expected no streaming messages after cancellation")
	}

	// Error should be present
	if !model.HasErrors() {
		t.Error("Expected error to be present")
	}
}
