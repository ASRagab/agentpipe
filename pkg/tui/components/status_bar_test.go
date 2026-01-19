package components

import (
	"testing"
	"time"

	"github.com/ASRagab/agentpipe/pkg/core"
)

func TestNewStatusBarModel(t *testing.T) {
	m := NewStatusBarModel()

	if m.status != core.ConversationStatusActive {
		t.Errorf("expected status Active, got %v", m.status)
	}

	if m.turnCount != 0 {
		t.Errorf("expected turnCount 0, got %d", m.turnCount)
	}

	if m.msgCount != 0 {
		t.Errorf("expected msgCount 0, got %d", m.msgCount)
	}

	if m.totalTokens != 0 {
		t.Errorf("expected totalTokens 0, got %d", m.totalTokens)
	}

	if m.totalCost != 0 {
		t.Errorf("expected totalCost 0, got %f", m.totalCost)
	}
}

func TestStatusBarSetters(t *testing.T) {
	m := NewStatusBarModel()

	// Test SetStatus
	m.SetStatus(core.ConversationStatusPaused)
	if m.GetStatus() != core.ConversationStatusPaused {
		t.Errorf("expected Paused status, got %v", m.GetStatus())
	}

	// Test SetStartTime
	testTime := time.Now().Add(-5 * time.Minute)
	m.SetStartTime(testTime)
	if !m.GetStartTime().Equal(testTime) {
		t.Errorf("expected start time %v, got %v", testTime, m.GetStartTime())
	}

	// Test SetTurnCount
	m.SetTurnCount(5)
	if m.GetTurnCount() != 5 {
		t.Errorf("expected turn count 5, got %d", m.GetTurnCount())
	}

	// Test SetMessageCount
	m.SetMessageCount(10)
	if m.GetMessageCount() != 10 {
		t.Errorf("expected message count 10, got %d", m.GetMessageCount())
	}

	// Test IncrementMessageCount
	m.IncrementMessageCount()
	if m.GetMessageCount() != 11 {
		t.Errorf("expected message count 11, got %d", m.GetMessageCount())
	}

	// Test SetTotalTokens
	m.SetTotalTokens(1000)
	if m.GetTotalTokens() != 1000 {
		t.Errorf("expected tokens 1000, got %d", m.GetTotalTokens())
	}

	// Test AddTokens
	m.AddTokens(500)
	if m.GetTotalTokens() != 1500 {
		t.Errorf("expected tokens 1500, got %d", m.GetTotalTokens())
	}

	// Test SetTotalCost
	m.SetTotalCost(0.05)
	if m.GetTotalCost() != 0.05 {
		t.Errorf("expected cost 0.05, got %f", m.GetTotalCost())
	}

	// Test AddCost
	m.AddCost(0.03)
	if m.GetTotalCost() != 0.08 {
		t.Errorf("expected cost 0.08, got %f", m.GetTotalCost())
	}
}

func TestStatusBarAgentCounts(t *testing.T) {
	m := NewStatusBarModel()

	m.SetAgentCounts(3, 5)
	connected, total := m.GetAgentCounts()

	if connected != 3 {
		t.Errorf("expected connected 3, got %d", connected)
	}

	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
}

func TestStatusBarReset(t *testing.T) {
	m := NewStatusBarModel()

	// Set some values
	m.SetStatus(core.ConversationStatusCompleted)
	m.SetTurnCount(10)
	m.SetMessageCount(50)
	m.SetTotalTokens(5000)
	m.SetTotalCost(1.5)
	m.SetAgentCounts(3, 4)

	// Reset
	m.Reset()

	// Verify reset
	if m.GetStatus() != core.ConversationStatusActive {
		t.Errorf("expected Active after reset, got %v", m.GetStatus())
	}

	if m.GetTurnCount() != 0 {
		t.Errorf("expected turnCount 0 after reset, got %d", m.GetTurnCount())
	}

	if m.GetMessageCount() != 0 {
		t.Errorf("expected msgCount 0 after reset, got %d", m.GetMessageCount())
	}

	if m.GetTotalTokens() != 0 {
		t.Errorf("expected tokens 0 after reset, got %d", m.GetTotalTokens())
	}

	if m.GetTotalCost() != 0 {
		t.Errorf("expected cost 0 after reset, got %f", m.GetTotalCost())
	}
}

func TestStatusBarUpdateFromConversation(t *testing.T) {
	m := NewStatusBarModel()
	m.SetWidth(100)

	// Create a conversation with messages
	agents := []core.Agent{
		{ID: "agent-1", Name: "Claude"},
		{ID: "agent-2", Name: "Gemini"},
	}
	conv := core.NewConversation(agents)
	conv.SetStatus(core.ConversationStatusActive)

	// Add some messages
	userMsg := core.NewUserMessage("Hello")
	conv.AddMessage(userMsg)

	agentMsg := core.NewAgentMessage("agent-1", "Claude", "Hi there!", &core.Metrics{
		TotalTokens: 100,
		Cost:        0.01,
	})
	conv.AddMessage(agentMsg)

	userMsg2 := core.NewUserMessage("How are you?")
	conv.AddMessage(userMsg2)

	// Update from conversation
	m.UpdateFromConversation(conv)

	// Verify
	if m.GetStatus() != core.ConversationStatusActive {
		t.Errorf("expected Active, got %v", m.GetStatus())
	}

	if m.GetMessageCount() != 3 {
		t.Errorf("expected 3 messages, got %d", m.GetMessageCount())
	}

	if m.GetTurnCount() != 2 {
		t.Errorf("expected 2 turns (user messages), got %d", m.GetTurnCount())
	}

	connected, total := m.GetAgentCounts()
	if total != 2 {
		t.Errorf("expected 2 total agents, got %d", total)
	}
	if connected != 2 {
		t.Errorf("expected 2 connected agents, got %d", connected)
	}

	if m.GetTotalTokens() != 100 {
		t.Errorf("expected 100 tokens, got %d", m.GetTotalTokens())
	}

	if m.GetTotalCost() != 0.01 {
		t.Errorf("expected 0.01 cost, got %f", m.GetTotalCost())
	}
}

func TestStatusBarUpdateFromMetrics(t *testing.T) {
	m := NewStatusBarModel()

	// Start with some values
	m.SetTotalTokens(100)
	m.SetTotalCost(0.01)

	// Update from metrics
	metrics := &core.Metrics{
		TotalTokens: 50,
		Cost:        0.005,
	}
	m.UpdateFromMetrics(metrics)

	// Verify values are added
	if m.GetTotalTokens() != 150 {
		t.Errorf("expected 150 tokens, got %d", m.GetTotalTokens())
	}

	if m.GetTotalCost() != 0.015 {
		t.Errorf("expected 0.015 cost, got %f", m.GetTotalCost())
	}

	// Nil metrics should be handled gracefully
	m.UpdateFromMetrics(nil)
	if m.GetTotalTokens() != 150 {
		t.Errorf("expected 150 tokens after nil update, got %d", m.GetTotalTokens())
	}
}

func TestStatusBarFormatCost(t *testing.T) {
	m := NewStatusBarModel()

	tests := []struct {
		cost     float64
		expected string
	}{
		{1.50, "$1.50"},
		{0.05, "$0.050"},
		{0.001, "$0.0010"},
		{10.00, "$10.00"},
	}

	for _, tt := range tests {
		result := m.formatCost(tt.cost)
		if result != tt.expected {
			t.Errorf("formatCost(%f) = %s, expected %s", tt.cost, result, tt.expected)
		}
	}
}

func TestStatusBarFormatElapsed(t *testing.T) {
	m := NewStatusBarModel()

	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m30s"},
		{3600 * time.Second, "1h00m"},
		{3660 * time.Second, "1h01m"},
	}

	for _, tt := range tests {
		result := m.formatElapsed(tt.duration)
		if result != tt.expected {
			t.Errorf("formatElapsed(%v) = %s, expected %s", tt.duration, result, tt.expected)
		}
	}
}

func TestStatusBarView(t *testing.T) {
	m := NewStatusBarModel()
	m.SetWidth(100)
	m.SetStatus(core.ConversationStatusActive)
	m.SetTurnCount(5)
	m.SetMessageCount(15)
	m.SetAgentCounts(3, 3)
	m.SetTotalTokens(500)
	m.SetTotalCost(0.05)

	view := m.View()

	// Check that view is not empty
	if len(view) == 0 {
		t.Error("expected non-empty view")
	}

	// Check that view contains expected elements
	if !containsAny(view, "Active") {
		t.Error("expected view to contain 'Active'")
	}

	if !containsAny(view, "Msgs:") {
		t.Error("expected view to contain 'Msgs:'")
	}

	if !containsAny(view, "Agents:") {
		t.Error("expected view to contain 'Agents:'")
	}
}

func TestStatusBarViewZeroWidth(t *testing.T) {
	m := NewStatusBarModel()
	// Width is 0 by default

	view := m.View()

	// Should return empty string when width is 0
	if view != "" {
		t.Errorf("expected empty view for zero width, got %s", view)
	}
}

func TestStatusBarStatusIndicators(t *testing.T) {
	m := NewStatusBarModel()
	m.SetWidth(100)

	// Test all status types
	statuses := []core.ConversationStatus{
		core.ConversationStatusActive,
		core.ConversationStatusPaused,
		core.ConversationStatusCompleted,
		core.ConversationStatusError,
	}

	expectedTexts := []string{
		"Active",
		"Paused",
		"Completed",
		"Error",
	}

	for i, status := range statuses {
		m.SetStatus(status)
		view := m.View()

		if !containsAny(view, expectedTexts[i]) {
			t.Errorf("expected view to contain '%s' for status %v", expectedTexts[i], status)
		}
	}
}

func TestStatusBarGetElapsed(t *testing.T) {
	m := NewStatusBarModel()

	// Set start time to 5 seconds ago
	m.SetStartTime(time.Now().Add(-5 * time.Second))

	// Allow some tolerance for execution time
	elapsed := m.GetElapsed()
	if elapsed < 4*time.Second || elapsed > 6*time.Second {
		t.Errorf("expected elapsed around 5s, got %v", elapsed)
	}
}

func TestStatusBarUpdateFromNilConversation(t *testing.T) {
	m := NewStatusBarModel()
	m.SetTurnCount(5)
	m.SetMessageCount(10)

	// Should not panic with nil conversation
	m.UpdateFromConversation(nil)

	// Values should remain unchanged
	if m.GetTurnCount() != 5 {
		t.Errorf("expected turn count 5, got %d", m.GetTurnCount())
	}

	if m.GetMessageCount() != 10 {
		t.Errorf("expected message count 10, got %d", m.GetMessageCount())
	}
}

// containsAny checks if the string contains any of the given substrings.
func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if contains(s, substr) {
			return true
		}
	}
	return false
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
