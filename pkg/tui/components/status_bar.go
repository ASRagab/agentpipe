package components

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ASRagab/agentpipe/pkg/core"
	"github.com/ASRagab/agentpipe/pkg/tui/styles"
)

// StatusBarModel manages the status bar at the top of the TUI.
// It displays conversation status, turn/message counts, and running totals.
type StatusBarModel struct {
	// Conversation state
	status      core.ConversationStatus
	startTime   time.Time
	turnCount   int
	msgCount    int
	totalTokens int
	totalCost   float64

	// Agent counts
	totalAgents     int
	connectedAgents int

	// Dimensions
	width int
}

// NewStatusBarModel creates a new status bar model.
func NewStatusBarModel() StatusBarModel {
	return StatusBarModel{
		status:    core.ConversationStatusActive,
		startTime: time.Now(),
	}
}

// Init initializes the status bar model.
func (m StatusBarModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the status bar.
func (m StatusBarModel) Update(msg tea.Msg) (StatusBarModel, tea.Cmd) {
	// Status bar is primarily driven by external updates,
	// but we handle window resize here
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	}
	return m, nil
}

// View renders the status bar.
func (m StatusBarModel) View() string {
	if m.width == 0 {
		return ""
	}

	// Left section: Status indicator
	left := m.renderStatusSection()

	// Center section: Turn and message counts + agent count
	center := m.renderCenterSection()

	// Right section: Totals (tokens, cost, elapsed)
	right := m.renderTotalsSection()

	// Calculate spacing to distribute sections
	leftWidth := lipgloss.Width(left)
	centerWidth := lipgloss.Width(center)
	rightWidth := lipgloss.Width(right)

	// Calculate available space for padding
	totalContentWidth := leftWidth + centerWidth + rightWidth
	availableSpace := m.width - totalContentWidth

	// Distribute spacing evenly between sections
	leftPad := availableSpace / 3
	rightPad := availableSpace / 3
	if leftPad < 1 {
		leftPad = 1
	}
	if rightPad < 1 {
		rightPad = 1
	}

	// Build the status bar with spacing
	var b strings.Builder
	b.WriteString(left)
	b.WriteString(strings.Repeat(" ", leftPad))
	b.WriteString(center)
	b.WriteString(strings.Repeat(" ", rightPad))
	b.WriteString(right)

	// Apply status bar background style
	return styles.StatusBarContainerStyle().
		Width(m.width).
		Render(b.String())
}

// renderStatusSection renders the conversation status indicator.
func (m StatusBarModel) renderStatusSection() string {
	var icon string
	var statusText string
	var style lipgloss.Style

	switch m.status {
	case core.ConversationStatusActive:
		icon = "●"
		statusText = "Active"
		style = styles.StatusActiveStyle()
	case core.ConversationStatusPaused:
		icon = "◐"
		statusText = "Paused"
		style = styles.StatusPausedStyle()
	case core.ConversationStatusCompleted:
		icon = "✓"
		statusText = "Completed"
		style = styles.StatusCompletedStyle()
	case core.ConversationStatusError:
		icon = "✗"
		statusText = "Error"
		style = styles.StatusErrorStyle()
	default:
		icon = "○"
		statusText = "Unknown"
		style = styles.StatusBarStyle()
	}

	return style.Render(fmt.Sprintf("%s %s", icon, statusText))
}

// renderCenterSection renders turn count, message count, and agent count.
func (m StatusBarModel) renderCenterSection() string {
	// Format: "Turn: 5 | Messages: 12 | Connected: 3/3 agents"
	parts := make([]string, 0, 3)

	// Turn count (simplified - every 2 messages is roughly a turn)
	if m.turnCount > 0 {
		parts = append(parts, fmt.Sprintf("Turn: %d", m.turnCount))
	}

	// Message count
	parts = append(parts, fmt.Sprintf("Msgs: %d", m.msgCount))

	// Agent count
	if m.totalAgents > 0 {
		parts = append(parts, fmt.Sprintf("Agents: %d/%d", m.connectedAgents, m.totalAgents))
	}

	return styles.StatusBarCenterStyle().Render(strings.Join(parts, " │ "))
}

// renderTotalsSection renders the running totals (tokens, cost, elapsed time).
func (m StatusBarModel) renderTotalsSection() string {
	parts := make([]string, 0, 3)

	// Total tokens
	if m.totalTokens > 0 {
		parts = append(parts, fmt.Sprintf("%dt", m.totalTokens))
	}

	// Total cost
	if m.totalCost > 0 {
		parts = append(parts, m.formatCost(m.totalCost))
	}

	// Elapsed time
	elapsed := time.Since(m.startTime)
	parts = append(parts, m.formatElapsed(elapsed))

	return styles.StatusBarTotalsStyle().Render(strings.Join(parts, " │ "))
}

// formatCost formats a cost value for display.
func (m StatusBarModel) formatCost(cost float64) string {
	if cost >= 1.0 {
		return fmt.Sprintf("$%.2f", cost)
	} else if cost >= 0.01 {
		return fmt.Sprintf("$%.3f", cost)
	}
	return fmt.Sprintf("$%.4f", cost)
}

// formatElapsed formats elapsed time in a human-readable format.
func (m StatusBarModel) formatElapsed(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%02ds", mins, secs)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%02dm", hours, mins)
}

// SetWidth sets the width of the status bar.
func (m *StatusBarModel) SetWidth(width int) {
	m.width = width
}

// SetStatus sets the conversation status.
func (m *StatusBarModel) SetStatus(status core.ConversationStatus) {
	m.status = status
}

// GetStatus returns the current conversation status.
func (m *StatusBarModel) GetStatus() core.ConversationStatus {
	return m.status
}

// SetStartTime sets the conversation start time.
func (m *StatusBarModel) SetStartTime(t time.Time) {
	m.startTime = t
}

// GetStartTime returns the conversation start time.
func (m *StatusBarModel) GetStartTime() time.Time {
	return m.startTime
}

// SetTurnCount sets the current turn count.
func (m *StatusBarModel) SetTurnCount(count int) {
	m.turnCount = count
}

// GetTurnCount returns the current turn count.
func (m *StatusBarModel) GetTurnCount() int {
	return m.turnCount
}

// SetMessageCount sets the total message count.
func (m *StatusBarModel) SetMessageCount(count int) {
	m.msgCount = count
}

// GetMessageCount returns the total message count.
func (m *StatusBarModel) GetMessageCount() int {
	return m.msgCount
}

// IncrementMessageCount increments the message count by 1.
func (m *StatusBarModel) IncrementMessageCount() {
	m.msgCount++
}

// SetTotalTokens sets the total token count.
func (m *StatusBarModel) SetTotalTokens(tokens int) {
	m.totalTokens = tokens
}

// GetTotalTokens returns the total token count.
func (m *StatusBarModel) GetTotalTokens() int {
	return m.totalTokens
}

// AddTokens adds tokens to the running total.
func (m *StatusBarModel) AddTokens(tokens int) {
	m.totalTokens += tokens
}

// SetTotalCost sets the total cost.
func (m *StatusBarModel) SetTotalCost(cost float64) {
	m.totalCost = cost
}

// GetTotalCost returns the total cost.
func (m *StatusBarModel) GetTotalCost() float64 {
	return m.totalCost
}

// AddCost adds cost to the running total.
func (m *StatusBarModel) AddCost(cost float64) {
	m.totalCost += cost
}

// SetAgentCounts sets the agent counts.
func (m *StatusBarModel) SetAgentCounts(connected, total int) {
	m.connectedAgents = connected
	m.totalAgents = total
}

// GetAgentCounts returns the connected and total agent counts.
func (m *StatusBarModel) GetAgentCounts() (connected, total int) {
	return m.connectedAgents, m.totalAgents
}

// UpdateFromConversation updates the status bar from a conversation.
func (m *StatusBarModel) UpdateFromConversation(conv *core.Conversation) {
	if conv == nil {
		return
	}

	m.status = conv.Status
	m.startTime = conv.Started
	m.msgCount = len(conv.Messages)
	m.totalAgents = len(conv.Agents)
	m.connectedAgents = len(conv.Agents) // Assume all connected initially

	// Calculate turn count (user messages = turns)
	turnCount := 0
	for _, msg := range conv.Messages {
		if msg.Role == core.RoleUser {
			turnCount++
		}
	}
	m.turnCount = turnCount

	// Calculate totals from message metrics
	m.totalTokens = conv.TotalTokens()
	m.totalCost = conv.TotalCost()
}

// UpdateFromMetrics updates running totals from message metrics.
func (m *StatusBarModel) UpdateFromMetrics(metrics *core.Metrics) {
	if metrics == nil {
		return
	}

	m.totalTokens += metrics.TotalTokens
	m.totalCost += metrics.Cost
}

// GetElapsed returns the elapsed time since conversation start.
func (m *StatusBarModel) GetElapsed() time.Duration {
	return time.Since(m.startTime)
}

// Reset resets the status bar to initial state.
func (m *StatusBarModel) Reset() {
	m.status = core.ConversationStatusActive
	m.startTime = time.Now()
	m.turnCount = 0
	m.msgCount = 0
	m.totalTokens = 0
	m.totalCost = 0
	m.totalAgents = 0
	m.connectedAgents = 0
}
