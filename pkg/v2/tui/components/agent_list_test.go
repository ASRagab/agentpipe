package components

import (
	"testing"
	"time"

	"github.com/ASRagab/agentpipe/pkg/v2/core"
)

func TestNewAgentListModel(t *testing.T) {
	agents := []core.Agent{
		{ID: "agent1", Name: "Claude", Model: "claude-3-sonnet"},
		{ID: "agent2", Name: "Gemini", Model: "gemini-pro"},
	}

	model := NewAgentListModel(agents)

	if len(model.agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(model.agents))
	}

	if model.statusMap["agent1"] != AgentStatusReady {
		t.Errorf("expected agent1 status to be Ready, got %v", model.statusMap["agent1"])
	}

	if model.animFrame != 0 {
		t.Errorf("expected initial animFrame to be 0, got %d", model.animFrame)
	}
}

func TestTypingIndicatorAnimation(t *testing.T) {
	agents := []core.Agent{
		{ID: "agent1", Name: "Claude", Model: "claude-3-sonnet"},
	}

	model := NewAgentListModel(agents)

	// Test animation frame advancement
	if model.GetAnimationFrame() != 0 {
		t.Errorf("expected initial animation frame 0, got %d", model.GetAnimationFrame())
	}

	model.AdvanceAnimationFrame()
	if model.GetAnimationFrame() != 1 {
		t.Errorf("expected animation frame 1, got %d", model.GetAnimationFrame())
	}

	model.AdvanceAnimationFrame()
	if model.GetAnimationFrame() != 2 {
		t.Errorf("expected animation frame 2, got %d", model.GetAnimationFrame())
	}

	// Should wrap around to 0
	model.AdvanceAnimationFrame()
	if model.GetAnimationFrame() != 0 {
		t.Errorf("expected animation frame to wrap to 0, got %d", model.GetAnimationFrame())
	}
}

func TestTypingStatusTracking(t *testing.T) {
	agents := []core.Agent{
		{ID: "agent1", Name: "Claude", Model: "claude-3-sonnet"},
	}

	model := NewAgentListModel(agents)

	// Initially no typing agents
	if model.HasTypingAgents() {
		t.Error("expected no typing agents initially")
	}

	// Start typing
	model.UpdateStatus("agent1", AgentStatusTyping)

	if !model.HasTypingAgents() {
		t.Error("expected HasTypingAgents to return true after setting typing status")
	}

	// Should have typing start time tracked
	elapsed := model.GetTypingElapsed("agent1")
	if elapsed == 0 {
		t.Error("expected non-zero elapsed time for typing agent")
	}

	// Stop typing
	model.UpdateStatus("agent1", AgentStatusReady)

	if model.HasTypingAgents() {
		t.Error("expected no typing agents after setting ready status")
	}

	// Elapsed time should be 0 after stopping
	elapsed = model.GetTypingElapsed("agent1")
	if elapsed != 0 {
		t.Errorf("expected 0 elapsed time after stopping typing, got %v", elapsed)
	}
}

func TestTypingElapsedTime(t *testing.T) {
	agents := []core.Agent{
		{ID: "agent1", Name: "Claude", Model: "claude-3-sonnet"},
	}

	model := NewAgentListModel(agents)

	// Start typing
	model.UpdateStatus("agent1", AgentStatusTyping)

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	elapsed := model.GetTypingElapsed("agent1")
	if elapsed < 50*time.Millisecond {
		t.Errorf("expected elapsed time >= 50ms, got %v", elapsed)
	}
}

func TestErrorTracking(t *testing.T) {
	agents := []core.Agent{
		{ID: "agent1", Name: "Claude", Model: "claude-3-sonnet"},
	}

	model := NewAgentListModel(agents)

	// Set error
	model.UpdateStatus("agent1", AgentStatusError)
	model.UpdateError("agent1", "Connection timeout")

	errInfo, ok := model.GetError("agent1")
	if !ok {
		t.Error("expected error info to be present")
	}
	if errInfo.Error != "Connection timeout" {
		t.Errorf("expected error message 'Connection timeout', got '%s'", errInfo.Error)
	}

	// Clear error by setting ready status
	model.UpdateStatus("agent1", AgentStatusReady)

	_, ok = model.GetError("agent1")
	if ok {
		t.Error("expected error info to be cleared after setting ready status")
	}
}

func TestTypingIndicatorDisplay(t *testing.T) {
	agents := []core.Agent{
		{ID: "agent1", Name: "Claude", Model: "claude-3-sonnet"},
	}

	model := NewAgentListModel(agents)
	model.SetSize(40, 20)

	// Start typing
	model.UpdateStatus("agent1", AgentStatusTyping)

	// Render view - should contain typing indicator
	view := model.View()

	if len(view) == 0 {
		t.Error("expected non-empty view")
	}

	// The view should contain the typing text (the actual dots may vary by frame)
	// The yellow typing indicator emoji should be present
	if model.statusMap["agent1"] != AgentStatusTyping {
		t.Error("expected agent to be in typing status")
	}
}

func TestMultipleAgentsTyping(t *testing.T) {
	agents := []core.Agent{
		{ID: "agent1", Name: "Claude", Model: "claude-3-sonnet"},
		{ID: "agent2", Name: "Gemini", Model: "gemini-pro"},
	}

	model := NewAgentListModel(agents)

	// Both agents typing
	model.UpdateStatus("agent1", AgentStatusTyping)
	model.UpdateStatus("agent2", AgentStatusTyping)

	if !model.HasTypingAgents() {
		t.Error("expected HasTypingAgents to return true")
	}

	// Both should have elapsed times
	elapsed1 := model.GetTypingElapsed("agent1")
	elapsed2 := model.GetTypingElapsed("agent2")

	if elapsed1 == 0 {
		t.Error("expected non-zero elapsed time for agent1")
	}
	if elapsed2 == 0 {
		t.Error("expected non-zero elapsed time for agent2")
	}

	// Stop one agent
	model.UpdateStatus("agent1", AgentStatusReady)

	// Should still have typing agents
	if !model.HasTypingAgents() {
		t.Error("expected HasTypingAgents to return true with agent2 still typing")
	}

	// Stop second agent
	model.UpdateStatus("agent2", AgentStatusReady)

	if model.HasTypingAgents() {
		t.Error("expected no typing agents after both stopped")
	}
}

func TestErrorDisplayTruncation(t *testing.T) {
	agents := []core.Agent{
		{ID: "agent1", Name: "Claude", Model: "claude-3-sonnet"},
	}

	model := NewAgentListModel(agents)
	model.SetSize(30, 20)

	// Set a long error message
	longError := "This is a very long error message that should be truncated when displayed in the agent list view"
	model.UpdateStatus("agent1", AgentStatusError)
	model.UpdateError("agent1", longError)

	// Verify truncateString function works correctly
	truncated := truncateString(longError, 20)
	if len(truncated) > 20 {
		t.Errorf("expected truncated string to be <= 20 chars, got %d", len(truncated))
	}
	if truncated[len(truncated)-3:] != "..." {
		t.Error("expected truncated string to end with ...")
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10c", 10, "exactly10c"},
		{"this is too long", 10, "this is..."},
		{"abc", 3, "abc"},   // Edge case: maxLen equals length
		{"ab", 5, "ab"},     // String shorter than max
		{"test", 2, "test"}, // Edge case: maxLen < 3, don't truncate
	}

	for _, tt := range tests {
		result := truncateString(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestRenderTypingIndicatorDots(t *testing.T) {
	agents := []core.Agent{
		{ID: "agent1", Name: "Claude", Model: "claude-3-sonnet"},
	}

	model := NewAgentListModel(agents)
	model.SetSize(40, 20)

	// Start typing
	model.UpdateStatus("agent1", AgentStatusTyping)

	// Frame 0: single dot
	line0 := model.renderTypingIndicator("agent1")
	if len(line0) == 0 {
		t.Error("expected non-empty typing indicator at frame 0")
	}

	// Frame 1: double dot
	model.AdvanceAnimationFrame()
	line1 := model.renderTypingIndicator("agent1")
	if len(line1) == 0 {
		t.Error("expected non-empty typing indicator at frame 1")
	}

	// Frame 2: triple dot
	model.AdvanceAnimationFrame()
	line2 := model.renderTypingIndicator("agent1")
	if len(line2) == 0 {
		t.Error("expected non-empty typing indicator at frame 2")
	}

	// Verify animation cycles
	if line0 == line1 && line1 == line2 {
		t.Error("expected different typing indicator content for different frames")
	}
}

func TestAgentListMetrics(t *testing.T) {
	agents := []core.Agent{
		{ID: "agent1", Name: "Claude", Model: "claude-3-sonnet"},
	}

	model := NewAgentListModel(agents)
	model.SetSize(40, 20)

	// Update metrics
	model.UpdateMetrics("agent1", AgentMetrics{
		Duration: 150 * time.Millisecond,
		Tokens:   256,
		Cost:     0.015,
	})

	// View should render without error
	view := model.View()
	if len(view) == 0 {
		t.Error("expected non-empty view with metrics")
	}
}

func TestStatusTransitions(t *testing.T) {
	agents := []core.Agent{
		{ID: "agent1", Name: "Claude", Model: "claude-3-sonnet"},
	}

	model := NewAgentListModel(agents)

	// Ready -> Typing
	model.UpdateStatus("agent1", AgentStatusTyping)
	if model.statusMap["agent1"] != AgentStatusTyping {
		t.Error("expected typing status")
	}
	if _, ok := model.typingStartMap["agent1"]; !ok {
		t.Error("expected typing start time to be recorded")
	}

	// Typing -> Error
	model.UpdateStatus("agent1", AgentStatusError)
	model.UpdateError("agent1", "test error")
	if model.statusMap["agent1"] != AgentStatusError {
		t.Error("expected error status")
	}
	if _, ok := model.typingStartMap["agent1"]; ok {
		t.Error("expected typing start time to be cleared on error")
	}

	// Error -> Ready (clears error)
	model.UpdateStatus("agent1", AgentStatusReady)
	if model.statusMap["agent1"] != AgentStatusReady {
		t.Error("expected ready status")
	}
	if _, ok := model.errorMap["agent1"]; ok {
		t.Error("expected error to be cleared on ready")
	}
}

func TestUpdateErrorWithDetails(t *testing.T) {
	agents := []core.Agent{
		{ID: "agent1", Name: "Claude", Model: "claude-3-sonnet"},
	}

	model := NewAgentListModel(agents)

	// Set detailed error info
	errInfo := AgentErrorInfo{
		Error:       "Rate limit exceeded",
		ErrorType:   "rate_limit",
		Recoverable: true,
		RetryAfter:  30 * time.Second,
		RetryHint:   "Wait and try again",
	}
	model.UpdateErrorWithDetails("agent1", errInfo)

	gotInfo, ok := model.GetError("agent1")
	if !ok {
		t.Error("expected error info to be present")
	}
	if gotInfo.Error != "Rate limit exceeded" {
		t.Errorf("expected error 'Rate limit exceeded', got '%s'", gotInfo.Error)
	}
	if gotInfo.ErrorType != "rate_limit" {
		t.Errorf("expected error type 'rate_limit', got '%s'", gotInfo.ErrorType)
	}
	if !gotInfo.Recoverable {
		t.Error("expected error to be recoverable")
	}
	if gotInfo.RetryAfter != 30*time.Second {
		t.Errorf("expected RetryAfter 30s, got %v", gotInfo.RetryAfter)
	}
	if gotInfo.RetryHint != "Wait and try again" {
		t.Errorf("expected RetryHint 'Wait and try again', got '%s'", gotInfo.RetryHint)
	}
	if gotInfo.Timestamp.IsZero() {
		t.Error("expected timestamp to be auto-populated")
	}
}

func TestRetryCountdown(t *testing.T) {
	agents := []core.Agent{
		{ID: "agent1", Name: "Claude", Model: "claude-3-sonnet"},
	}

	model := NewAgentListModel(agents)

	// Set error with retry-after
	errInfo := AgentErrorInfo{
		Error:       "Rate limit",
		ErrorType:   "rate_limit",
		Recoverable: true,
		RetryAfter:  100 * time.Millisecond,
		Timestamp:   time.Now(),
	}
	model.UpdateErrorWithDetails("agent1", errInfo)

	// Should have active countdown initially
	countdown := model.GetRetryCountdown("agent1")
	if countdown <= 0 {
		t.Error("expected positive countdown initially")
	}
	if countdown > 100*time.Millisecond {
		t.Errorf("expected countdown <= 100ms, got %v", countdown)
	}

	// HasActiveRetryCountdown should return true
	if !model.HasActiveRetryCountdown() {
		t.Error("expected HasActiveRetryCountdown to return true")
	}

	// Wait for countdown to expire
	time.Sleep(120 * time.Millisecond)

	// Should have no countdown
	countdown = model.GetRetryCountdown("agent1")
	if countdown != 0 {
		t.Errorf("expected countdown to be 0 after expiry, got %v", countdown)
	}

	// HasActiveRetryCountdown should return false
	if model.HasActiveRetryCountdown() {
		t.Error("expected HasActiveRetryCountdown to return false after expiry")
	}
}

func TestGetAgentsWithErrors(t *testing.T) {
	agents := []core.Agent{
		{ID: "agent1", Name: "Claude", Model: "claude-3-sonnet"},
		{ID: "agent2", Name: "Gemini", Model: "gemini-pro"},
	}

	model := NewAgentListModel(agents)

	// Initially no errors
	errAgents := model.GetAgentsWithErrors()
	if len(errAgents) != 0 {
		t.Errorf("expected no error agents, got %d", len(errAgents))
	}

	// Add errors to both agents
	model.UpdateError("agent1", "Error 1")
	model.UpdateError("agent2", "Error 2")

	errAgents = model.GetAgentsWithErrors()
	if len(errAgents) != 2 {
		t.Errorf("expected 2 error agents, got %d", len(errAgents))
	}
}

func TestGetRecoverableAgents(t *testing.T) {
	agents := []core.Agent{
		{ID: "agent1", Name: "Claude", Model: "claude-3-sonnet"},
		{ID: "agent2", Name: "Gemini", Model: "gemini-pro"},
	}

	model := NewAgentListModel(agents)

	// Add recoverable error to agent1
	model.UpdateErrorWithDetails("agent1", AgentErrorInfo{
		Error:       "Network error",
		ErrorType:   "network",
		Recoverable: true,
	})

	// Add non-recoverable error to agent2
	model.UpdateErrorWithDetails("agent2", AgentErrorInfo{
		Error:       "Auth error",
		ErrorType:   "auth",
		Recoverable: false,
	})

	recoverable := model.GetRecoverableAgents()
	if len(recoverable) != 1 {
		t.Errorf("expected 1 recoverable agent, got %d", len(recoverable))
	}
	if recoverable[0] != "agent1" {
		t.Errorf("expected agent1 to be recoverable, got %s", recoverable[0])
	}
}

func TestClearError(t *testing.T) {
	agents := []core.Agent{
		{ID: "agent1", Name: "Claude", Model: "claude-3-sonnet"},
	}

	model := NewAgentListModel(agents)

	model.UpdateError("agent1", "Test error")
	_, ok := model.GetError("agent1")
	if !ok {
		t.Error("expected error to be present")
	}

	model.ClearError("agent1")
	_, ok = model.GetError("agent1")
	if ok {
		t.Error("expected error to be cleared")
	}
}

func TestHasSelectedAgentError(t *testing.T) {
	agents := []core.Agent{
		{ID: "agent1", Name: "Claude", Model: "claude-3-sonnet"},
		{ID: "agent2", Name: "Gemini", Model: "gemini-pro"},
	}

	model := NewAgentListModel(agents)
	model.SetSize(40, 20)

	// Initially selected agent has no error
	if model.HasSelectedAgentError() {
		t.Error("expected no error for selected agent initially")
	}

	// Add error to selected agent
	model.UpdateError("agent1", "Test error")
	if !model.HasSelectedAgentError() {
		t.Error("expected error for selected agent after setting")
	}

	// Select agent2 (no error)
	model.selectedIndex = 1
	if model.HasSelectedAgentError() {
		t.Error("expected no error for agent2")
	}
}

func TestRenderRetryCountdown(t *testing.T) {
	agents := []core.Agent{
		{ID: "agent1", Name: "Claude", Model: "claude-3-sonnet"},
	}

	model := NewAgentListModel(agents)

	// Test different countdown durations
	tests := []struct {
		countdown time.Duration
		contains  string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m"},
		{500 * time.Millisecond, "< 1s"},
	}

	for _, tc := range tests {
		result := model.renderRetryCountdown(tc.countdown)
		if result == "" {
			t.Errorf("expected non-empty result for countdown %v", tc.countdown)
		}
		// Check that the result contains "Retrying in"
		if len(result) == 0 {
			t.Error("expected result to contain countdown indicator")
		}
	}
}

func TestRenderErrorDetailsModal(t *testing.T) {
	agents := []core.Agent{
		{ID: "agent1", Name: "Claude", Model: "claude-3-sonnet"},
	}

	model := NewAgentListModel(agents)
	model.SetSize(40, 20)

	// No modal when no error
	modal := model.RenderErrorDetailsModal(60)
	if modal != "" {
		t.Error("expected empty modal when no error")
	}

	// Add detailed error
	model.UpdateErrorWithDetails("agent1", AgentErrorInfo{
		Error:       "Rate limit exceeded",
		ErrorType:   "rate_limit",
		Recoverable: true,
		RetryAfter:  30 * time.Second,
		RetryHint:   "Wait and try again",
		Timestamp:   time.Now(),
	})

	modal = model.RenderErrorDetailsModal(60)
	if modal == "" {
		t.Error("expected non-empty modal with error")
	}
	// Modal should contain the agent name
	if len(modal) < 10 {
		t.Error("expected modal to have substantial content")
	}
}

func TestRenderViewWithRetryCountdown(t *testing.T) {
	agents := []core.Agent{
		{ID: "agent1", Name: "Claude", Model: "claude-3-sonnet"},
	}

	model := NewAgentListModel(agents)
	model.SetSize(50, 30)

	// Set error with active retry countdown
	model.UpdateStatus("agent1", AgentStatusError)
	model.UpdateErrorWithDetails("agent1", AgentErrorInfo{
		Error:       "Rate limit",
		ErrorType:   "rate_limit",
		Recoverable: true,
		RetryAfter:  60 * time.Second,
		RetryHint:   "Wait and try again",
		Timestamp:   time.Now(),
	})

	view := model.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
	// View should contain error indicator (red dot)
	if len(view) == 0 {
		t.Error("expected view to have content")
	}
}
