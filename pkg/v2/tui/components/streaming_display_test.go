package components

import (
	"strings"
	"testing"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/v2/core"
)

// =============================================================================
// Streaming Display Tests
// These tests verify the visual display updates when streaming messages.
// =============================================================================

// TestStreamingChunksDisplayUpdate verifies that streaming chunks update the display.
// This tests the full flow: chunks arrive -> display updates with new content.
func TestStreamingChunksDisplayUpdate(t *testing.T) {
	model := NewConversationModel()
	model.InitViewport(80, 24)
	model.width = 80
	model.height = 24
	model.cursorVisible = true

	// Start streaming
	model.StartStreaming("msg-1", "agent-1", "Claude")

	// Verify initial state shows agent name in display
	content1 := model.renderMessages()
	if !containsSubstring(content1, "Claude") {
		t.Error("Expected display to contain agent name 'Claude' after starting streaming")
	}

	// Append first chunk
	model.AppendChunk(core.MessageChunk{
		MessageID: "msg-1",
		AgentID:   "agent-1",
		AgentName: "Claude",
		Content:   "Hello ",
		Index:     0,
	})

	// Verify display updated with chunk content
	content2 := model.renderMessages()
	if !containsSubstring(content2, "Hello") {
		t.Error("Expected display to contain 'Hello' after first chunk")
	}

	// Append second chunk
	model.AppendChunk(core.MessageChunk{
		MessageID: "msg-1",
		AgentID:   "agent-1",
		AgentName: "Claude",
		Content:   "World!",
		Index:     1,
	})

	// Verify display contains accumulated content
	content3 := model.renderMessages()
	if !containsSubstring(content3, "Hello World!") {
		t.Error("Expected display to contain accumulated 'Hello World!' after second chunk")
	}

	// Verify blinking cursor is shown (when cursorVisible is true)
	if !containsSubstring(content3, "▌") {
		t.Error("Expected display to contain blinking cursor '▌'")
	}

	// Verify streaming indicator is shown
	if !containsSubstring(content3, "streaming") {
		t.Error("Expected display to contain 'streaming' indicator")
	}

	// Toggle cursor off and verify cursor is hidden
	model.ToggleCursor()
	content4 := model.renderMessages()
	if containsSubstring(content4, "▌") {
		t.Error("Expected cursor to be hidden after toggle")
	}
}

// TestMultipleAgentsStreamingDisplay verifies that two agents streaming simultaneously
// are both displayed correctly with proper ordering and identification.
func TestMultipleAgentsStreamingDisplay(t *testing.T) {
	model := NewConversationModel()
	model.InitViewport(80, 24)
	model.width = 80
	model.height = 24
	model.cursorVisible = true

	// Start streaming from two agents
	model.StartStreaming("msg-1", "agent-1", "Claude")
	time.Sleep(10 * time.Millisecond) // Small delay to ensure ordering
	model.StartStreaming("msg-2", "agent-2", "Gemini")

	// Append chunks to both agents
	model.AppendChunk(core.MessageChunk{
		MessageID: "msg-1",
		AgentID:   "agent-1",
		AgentName: "Claude",
		Content:   "Response from Claude",
	})
	model.AppendChunk(core.MessageChunk{
		MessageID: "msg-2",
		AgentID:   "agent-2",
		AgentName: "Gemini",
		Content:   "Response from Gemini",
	})

	// Render and verify both agents are displayed
	content := model.renderMessages()

	// Verify both agent names are displayed
	if !containsSubstring(content, "Claude") {
		t.Error("Expected display to contain 'Claude'")
	}
	if !containsSubstring(content, "Gemini") {
		t.Error("Expected display to contain 'Gemini'")
	}

	// Verify both responses are displayed
	if !containsSubstring(content, "Response from Claude") {
		t.Error("Expected display to contain 'Response from Claude'")
	}
	if !containsSubstring(content, "Response from Gemini") {
		t.Error("Expected display to contain 'Response from Gemini'")
	}

	// Verify order: Claude should appear before Gemini (started first)
	claudePos := strings.Index(content, "Claude")
	geminiPos := strings.Index(content, "Gemini")
	if claudePos > geminiPos {
		t.Error("Expected Claude to appear before Gemini in display (based on start time)")
	}

	// Verify both have streaming indicators
	streamingCount := strings.Count(content, "streaming")
	if streamingCount < 2 {
		t.Errorf("Expected at least 2 'streaming' indicators, got %d", streamingCount)
	}

	// Verify both have progress bars
	progressBarCount := strings.Count(content, "━") + strings.Count(content, "─")
	if progressBarCount < 2 {
		t.Error("Expected progress bar characters in display for both agents")
	}
}

// TestScrollBehaviorAutoScroll verifies auto-scroll when at bottom.
func TestScrollBehaviorAutoScroll(t *testing.T) {
	model := NewConversationModel()
	model.InitViewport(80, 24)
	model.width = 80
	model.height = 24

	// Add enough messages to require scrolling
	for i := 0; i < 50; i++ {
		model.AddMessage(core.Message{
			ID:        "msg-" + string(rune('a'+i)),
			Role:      core.RoleAgent,
			AgentID:   "agent-1",
			AgentName: "Claude",
			Content:   "Message content line " + string(rune('a'+i)),
			Timestamp: time.Now(),
		})
	}

	// Initially should be at bottom (auto-scrolled)
	if model.IsUserScrolledUp() {
		t.Error("Expected to be at bottom initially (auto-scroll)")
	}

	// Add new message - should auto-scroll to show it
	model.AddMessage(core.Message{
		ID:        "msg-new",
		Role:      core.RoleAgent,
		AgentID:   "agent-1",
		AgentName: "Claude",
		Content:   "New message that should be visible",
		Timestamp: time.Now(),
	})

	// Verify still at bottom
	if model.IsUserScrolledUp() {
		t.Error("Expected to still be at bottom after adding message")
	}

	// Verify new messages indicator is NOT shown when at bottom
	if model.HasNewMessages() {
		t.Error("Expected no new messages indicator when at bottom")
	}
}

// TestScrollBehaviorUserScrolledUp verifies no auto-scroll when user scrolled up.
func TestScrollBehaviorUserScrolledUp(t *testing.T) {
	model := NewConversationModel()
	model.InitViewport(80, 24)
	model.width = 80
	model.height = 24

	// Add enough messages to require scrolling
	for i := 0; i < 50; i++ {
		model.AddMessage(core.Message{
			ID:        "msg-" + string(rune('a'+i)),
			Role:      core.RoleAgent,
			AgentID:   "agent-1",
			AgentName: "Claude",
			Content:   "Message content line " + string(rune('a'+i)),
			Timestamp: time.Now(),
		})
	}

	// Scroll to top (simulate user scrolling up)
	model.viewport.GotoTop()
	model.userScrolledUp = true

	// Add new message
	model.StartStreaming("msg-new", "agent-2", "Gemini")
	model.AppendChunk(core.MessageChunk{
		MessageID: "msg-new",
		Content:   "New content while scrolled up",
	})

	// Verify user is still scrolled up (position preserved)
	if !model.IsUserScrolledUp() {
		t.Error("Expected userScrolledUp to remain true after new message")
	}

	// Verify new messages indicator IS shown when scrolled up
	if !model.HasNewMessages() {
		t.Error("Expected new messages indicator when scrolled up and new message arrives")
	}

	// Verify indicator appears in view
	view := model.View()
	if !containsSubstring(view, "New messages below") {
		t.Error("Expected 'New messages below' indicator in view when scrolled up")
	}
}

// TestScrollBehaviorJumpToBottom verifies End key jumps to bottom and clears indicators.
func TestScrollBehaviorJumpToBottom(t *testing.T) {
	model := NewConversationModel()
	model.InitViewport(80, 24)
	model.width = 80
	model.height = 24

	// Add messages and simulate scrolled state
	for i := 0; i < 50; i++ {
		model.AddMessage(core.Message{
			ID:      "msg-" + string(rune('a'+i)),
			Content: "Message " + string(rune('a'+i)),
		})
	}

	// Set scrolled up state
	model.viewport.GotoTop()
	model.userScrolledUp = true
	model.hasNewMessages = true

	// Jump to bottom (simulates End key)
	model.JumpToBottom()

	// Verify state cleared
	if model.IsUserScrolledUp() {
		t.Error("Expected userScrolledUp to be false after JumpToBottom")
	}
	if model.HasNewMessages() {
		t.Error("Expected hasNewMessages to be false after JumpToBottom")
	}

	// Verify indicator is not in view
	view := model.View()
	if containsSubstring(view, "New messages below") {
		t.Error("Expected no 'New messages below' indicator after JumpToBottom")
	}
}

// TestMetricsDisplayFormatting verifies metrics are formatted correctly in display.
func TestMetricsDisplayFormatting(t *testing.T) {
	model := NewConversationModel()
	model.InitViewport(80, 24)
	model.width = 80
	model.height = 24

	// Add message with metrics
	msg := core.Message{
		ID:        "msg-1",
		Role:      core.RoleAgent,
		AgentID:   "agent-1",
		AgentName: "Claude",
		Content:   "This is a response with metrics",
		Timestamp: time.Now(),
		Metrics: &core.Metrics{
			Duration:     1500 * time.Millisecond,
			TotalTokens:  234,
			InputTokens:  100,
			OutputTokens: 134,
			Cost:         0.012,
		},
	}
	model.AddMessage(msg)

	// Render and verify metrics display
	content := model.renderMessages()

	// Verify metrics badge format: [1.5s | 234t | $0.012]
	if !containsSubstring(content, "1.5s") {
		t.Error("Expected metrics to contain duration '1.5s'")
	}
	if !containsSubstring(content, "234t") {
		t.Error("Expected metrics to contain token count '234t'")
	}
	if !containsSubstring(content, "$0.012") {
		t.Error("Expected metrics to contain cost '$0.012'")
	}
}

// TestMetricsDisplayVariousFormats verifies different metric value formats.
func TestMetricsDisplayVariousFormats(t *testing.T) {
	model := NewConversationModel()

	testCases := []struct {
		name     string
		metrics  *core.Metrics
		expected []string // Substrings that should appear
	}{
		{
			name: "milliseconds duration",
			metrics: &core.Metrics{
				Duration:    150 * time.Millisecond,
				TotalTokens: 100,
				Cost:        0.001,
			},
			expected: []string{"150ms", "100t", "$0.0010"},
		},
		{
			name: "seconds duration",
			metrics: &core.Metrics{
				Duration:    2500 * time.Millisecond,
				TotalTokens: 500,
				Cost:        0.05,
			},
			expected: []string{"2.5s", "500t", "$0.050"},
		},
		{
			name: "minutes duration",
			metrics: &core.Metrics{
				Duration:    90 * time.Second,
				TotalTokens: 10000,
				Cost:        1.5,
			},
			expected: []string{"1.5m", "10000t", "$1.50"},
		},
		{
			name: "high cost",
			metrics: &core.Metrics{
				Duration:    1 * time.Second,
				TotalTokens: 50000,
				Cost:        12.50,
			},
			expected: []string{"1.0s", "50000t", "$12.50"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			formatted := model.formatMetrics(tc.metrics)
			for _, exp := range tc.expected {
				if !containsSubstring(formatted, exp) {
					t.Errorf("Expected formatted metrics to contain '%s', got '%s'", exp, formatted)
				}
			}
		})
	}
}

// TestStatusBarUpdateAfterMessage verifies status bar totals update after each message.
func TestStatusBarUpdateAfterMessage(t *testing.T) {
	statusBar := NewStatusBarModel()
	statusBar.SetWidth(100)

	// Initial state
	if statusBar.GetTotalTokens() != 0 {
		t.Error("Expected initial tokens to be 0")
	}
	if statusBar.GetTotalCost() != 0 {
		t.Error("Expected initial cost to be 0")
	}
	if statusBar.GetMessageCount() != 0 {
		t.Error("Expected initial message count to be 0")
	}

	// Simulate first message completion with metrics
	statusBar.IncrementMessageCount()
	statusBar.UpdateFromMetrics(&core.Metrics{
		TotalTokens: 100,
		Cost:        0.01,
	})

	// Verify totals updated
	if statusBar.GetTotalTokens() != 100 {
		t.Errorf("Expected 100 tokens after first message, got %d", statusBar.GetTotalTokens())
	}
	if statusBar.GetTotalCost() != 0.01 {
		t.Errorf("Expected 0.01 cost after first message, got %f", statusBar.GetTotalCost())
	}
	if statusBar.GetMessageCount() != 1 {
		t.Errorf("Expected 1 message after first increment, got %d", statusBar.GetMessageCount())
	}

	// Simulate second message
	statusBar.IncrementMessageCount()
	statusBar.UpdateFromMetrics(&core.Metrics{
		TotalTokens: 200,
		Cost:        0.02,
	})

	// Verify totals accumulated
	if statusBar.GetTotalTokens() != 300 {
		t.Errorf("Expected 300 tokens after second message, got %d", statusBar.GetTotalTokens())
	}
	if statusBar.GetTotalCost() != 0.03 {
		t.Errorf("Expected 0.03 cost after second message, got %f", statusBar.GetTotalCost())
	}
	if statusBar.GetMessageCount() != 2 {
		t.Errorf("Expected 2 messages, got %d", statusBar.GetMessageCount())
	}

	// Verify view contains updated totals
	view := statusBar.View()
	if !containsSubstring(view, "300t") {
		t.Error("Expected status bar view to contain '300t'")
	}
	if !containsSubstring(view, "Msgs: 2") {
		t.Error("Expected status bar view to contain 'Msgs: 2'")
	}
}

// TestStatusBarRealTimeUpdate verifies status bar updates in real-time.
func TestStatusBarRealTimeUpdate(t *testing.T) {
	statusBar := NewStatusBarModel()
	statusBar.SetWidth(100)
	statusBar.SetStartTime(time.Now().Add(-30 * time.Second))

	// Set various values
	statusBar.SetStatus(core.ConversationStatusActive)
	statusBar.SetTurnCount(5)
	statusBar.SetMessageCount(15)
	statusBar.SetAgentCounts(3, 3)
	statusBar.SetTotalTokens(500)
	statusBar.SetTotalCost(0.05)

	// Render view
	view := statusBar.View()

	// Verify all sections are present
	if !containsSubstring(view, "Active") {
		t.Error("Expected view to contain status 'Active'")
	}
	if !containsSubstring(view, "Turn: 5") {
		t.Error("Expected view to contain 'Turn: 5'")
	}
	if !containsSubstring(view, "Msgs: 15") {
		t.Error("Expected view to contain 'Msgs: 15'")
	}
	if !containsSubstring(view, "Agents: 3/3") {
		t.Error("Expected view to contain 'Agents: 3/3'")
	}
	if !containsSubstring(view, "500t") {
		t.Error("Expected view to contain '500t'")
	}
	if !containsSubstring(view, "$0.050") {
		t.Error("Expected view to contain '$0.050'")
	}
}

// TestStreamingWithProgressBar verifies progress bar appears during streaming.
func TestStreamingWithProgressBar(t *testing.T) {
	model := NewConversationModel()
	model.InitViewport(80, 24)
	model.width = 80
	model.height = 24
	model.cursorVisible = true

	// Start streaming
	model.StartStreaming("msg-1", "agent-1", "Claude")
	model.AppendChunk(core.MessageChunk{
		MessageID: "msg-1",
		Content:   "Streaming content...",
	})

	// Wait a bit to have elapsed time
	time.Sleep(50 * time.Millisecond)

	// Render content
	content := model.renderMessages()

	// Verify progress bar characters are present
	hasProgressBar := containsSubstring(content, "━") || containsSubstring(content, "─")
	if !hasProgressBar {
		t.Error("Expected progress bar characters in streaming message display")
	}
}

// TestStreamingCompletionClearsProgressBar verifies progress bar is removed after completion.
func TestStreamingCompletionClearsProgressBar(t *testing.T) {
	model := NewConversationModel()
	model.InitViewport(80, 24)
	model.width = 80
	model.height = 24

	// Start and complete streaming
	model.StartStreaming("msg-1", "agent-1", "Claude")
	model.AppendChunk(core.MessageChunk{
		MessageID: "msg-1",
		Content:   "Complete content",
	})

	// Complete the streaming
	finalMsg := core.Message{
		ID:        "msg-1",
		Role:      core.RoleAgent,
		AgentID:   "agent-1",
		AgentName: "Claude",
		Content:   "Complete content",
		Timestamp: time.Now(),
		Metrics: &core.Metrics{
			Duration:    500 * time.Millisecond,
			TotalTokens: 50,
		},
	}
	model.CompleteStreaming("msg-1", finalMsg)

	// Verify no streaming messages remain
	if model.HasStreamingMessages() {
		t.Error("Expected no streaming messages after completion")
	}

	// Render and verify no streaming indicator
	content := model.renderMessages()
	if containsSubstring(content, "streaming...") {
		t.Error("Expected no 'streaming...' indicator after completion")
	}

	// Verify completed message has metrics instead
	if !containsSubstring(content, "500ms") {
		t.Error("Expected completed message to show duration metrics")
	}
}
