// Package components provides TUI sub-components for the AgentPipe v2 interface.
package components

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kevinelliott/agentpipe/pkg/v2/core"
	"github.com/kevinelliott/agentpipe/pkg/v2/tui/styles"
)

// AgentStatus represents the display status of an agent.
type AgentStatus string

const (
	AgentStatusReady  AgentStatus = "ready"
	AgentStatusTyping AgentStatus = "typing"
	AgentStatusError  AgentStatus = "error"
)

// AgentMetrics contains response metrics for display.
type AgentMetrics struct {
	Duration time.Duration
	Tokens   int
	Cost     float64
}

// AgentErrorInfo contains error information for display.
type AgentErrorInfo struct {
	Error     string
	Timestamp time.Time
}

// AgentListModel manages the agent list panel.
type AgentListModel struct {
	agents         []core.Agent
	statusMap      map[string]AgentStatus
	metricsMap     map[string]AgentMetrics
	typingStartMap map[string]time.Time  // When each agent started typing
	errorMap       map[string]AgentErrorInfo // Error info for each agent
	animFrame      int                   // Current frame for typing animation (0-2)
	selectedIndex  int
	width          int
	height         int
	focused        bool
}

// NewAgentListModel creates a new agent list model.
func NewAgentListModel(agents []core.Agent) AgentListModel {
	statusMap := make(map[string]AgentStatus)
	metricsMap := make(map[string]AgentMetrics)
	typingStartMap := make(map[string]time.Time)
	errorMap := make(map[string]AgentErrorInfo)

	for _, agent := range agents {
		statusMap[agent.ID] = AgentStatusReady
	}

	return AgentListModel{
		agents:         agents,
		statusMap:      statusMap,
		metricsMap:     metricsMap,
		typingStartMap: typingStartMap,
		errorMap:       errorMap,
		selectedIndex:  0,
		focused:        false,
		animFrame:      0,
	}
}

// Init initializes the agent list model.
func (m AgentListModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the agent list.
func (m AgentListModel) Update(msg tea.Msg) (AgentListModel, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selectedIndex > 0 {
				m.selectedIndex--
			}
		case "down", "j":
			if m.selectedIndex < len(m.agents)-1 {
				m.selectedIndex++
			}
		case "home":
			m.selectedIndex = 0
		case "end":
			m.selectedIndex = len(m.agents) - 1
		}
	}

	return m, nil
}

// View renders the agent list.
func (m AgentListModel) View() string {
	if len(m.agents) == 0 {
		return styles.PlaceholderStyle().Render("No agents")
	}

	var b strings.Builder

	// Title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Render("Agents")
	b.WriteString(title)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.width-2))
	b.WriteString("\n")

	for i, agent := range m.agents {
		selected := i == m.selectedIndex
		color := styles.AgentColorForIndex(i)

		// Status indicator
		status := m.statusMap[agent.ID]
		var indicator styles.StatusIndicator
		switch status {
		case AgentStatusTyping:
			indicator = styles.StatusTyping
		case AgentStatusError:
			indicator = styles.StatusError
		default:
			indicator = styles.StatusReady
		}

		// Agent name line
		nameStyle := styles.AgentListItemStyle(selected, color)
		nameLine := fmt.Sprintf("%s %s", indicator, agent.Name)
		b.WriteString(nameStyle.Render(nameLine))
		b.WriteString("\n")

		// Model name (smaller, indented)
		modelStyle := styles.AgentModelStyle()
		modelLine := agent.Model
		if len(modelLine) > m.width-4 {
			modelLine = modelLine[:m.width-7] + "..."
		}
		b.WriteString(modelStyle.Render(modelLine))
		b.WriteString("\n")

		// Show typing indicator with animation and elapsed time
		if status == AgentStatusTyping {
			typingLine := m.renderTypingIndicator(agent.ID)
			b.WriteString(typingLine)
			b.WriteString("\n")
		} else if status == AgentStatusError {
			// Show error info if available
			if errInfo, ok := m.errorMap[agent.ID]; ok {
				errorStyle := styles.ErrorStyle()
				errorLine := fmt.Sprintf("  ⚠ %s", truncateString(errInfo.Error, m.width-8))
				b.WriteString(errorStyle.Render(errorLine))
				b.WriteString("\n")
			}
		} else if metrics, ok := m.metricsMap[agent.ID]; ok {
			// Show metrics if available (only when not typing/error)
			metricsStyle := styles.MetricsStyle()
			metricsLine := fmt.Sprintf("  %dms | %d tok", metrics.Duration.Milliseconds(), metrics.Tokens)
			b.WriteString(metricsStyle.Render(metricsLine))
			b.WriteString("\n")
		}

		// Add spacing between agents
		if i < len(m.agents)-1 {
			b.WriteString("\n")
		}
	}

	// Apply panel border
	borderStyle := styles.PanelBorderStyle()
	if m.focused {
		borderStyle = styles.SelectedPanelBorderStyle()
	}

	content := b.String()
	return borderStyle.
		Width(m.width - 2).
		Height(m.height - 2).
		Render(content)
}

// SetSize updates the dimensions of the agent list.
func (m *AgentListModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetFocused sets the focused state of the agent list.
func (m *AgentListModel) SetFocused(focused bool) {
	m.focused = focused
}

// IsFocused returns whether the agent list is focused.
func (m *AgentListModel) IsFocused() bool {
	return m.focused
}

// UpdateStatus updates the status for a specific agent.
func (m *AgentListModel) UpdateStatus(agentID string, status AgentStatus) {
	previousStatus := m.statusMap[agentID]
	m.statusMap[agentID] = status

	// Track typing start time when transitioning to typing
	if status == AgentStatusTyping && previousStatus != AgentStatusTyping {
		m.typingStartMap[agentID] = time.Now()
	} else if status != AgentStatusTyping {
		// Clear typing start time when no longer typing
		delete(m.typingStartMap, agentID)
	}

	// Clear error when status changes to ready
	if status == AgentStatusReady {
		delete(m.errorMap, agentID)
	}
}

// UpdateMetrics updates the metrics for a specific agent.
func (m *AgentListModel) UpdateMetrics(agentID string, metrics AgentMetrics) {
	m.metricsMap[agentID] = metrics
}

// GetSelectedAgent returns the currently selected agent.
func (m *AgentListModel) GetSelectedAgent() *core.Agent {
	if m.selectedIndex >= 0 && m.selectedIndex < len(m.agents) {
		return &m.agents[m.selectedIndex]
	}
	return nil
}

// GetSelectedIndex returns the current selection index.
func (m *AgentListModel) GetSelectedIndex() int {
	return m.selectedIndex
}

// SetAgents updates the list of agents.
func (m *AgentListModel) SetAgents(agents []core.Agent) {
	m.agents = agents
	// Initialize status for new agents
	for _, agent := range agents {
		if _, ok := m.statusMap[agent.ID]; !ok {
			m.statusMap[agent.ID] = AgentStatusReady
		}
	}
	// Adjust selection if needed
	if m.selectedIndex >= len(agents) {
		m.selectedIndex = len(agents) - 1
	}
	if m.selectedIndex < 0 {
		m.selectedIndex = 0
	}
}

// AgentCount returns the number of agents.
func (m *AgentListModel) AgentCount() int {
	return len(m.agents)
}

// UpdateError updates the error info for a specific agent.
func (m *AgentListModel) UpdateError(agentID string, errorMsg string) {
	m.errorMap[agentID] = AgentErrorInfo{
		Error:     errorMsg,
		Timestamp: time.Now(),
	}
}

// GetError returns the error info for a specific agent.
func (m *AgentListModel) GetError(agentID string) (AgentErrorInfo, bool) {
	info, ok := m.errorMap[agentID]
	return info, ok
}

// AdvanceAnimationFrame advances the typing animation frame.
// Should be called on each frame tick (every 200ms) to create cycling dots animation.
func (m *AgentListModel) AdvanceAnimationFrame() {
	m.animFrame = (m.animFrame + 1) % 3
}

// GetAnimationFrame returns the current animation frame.
func (m *AgentListModel) GetAnimationFrame() int {
	return m.animFrame
}

// HasTypingAgents returns true if any agent is currently typing.
func (m *AgentListModel) HasTypingAgents() bool {
	for _, status := range m.statusMap {
		if status == AgentStatusTyping {
			return true
		}
	}
	return false
}

// GetTypingElapsed returns the elapsed time since an agent started typing.
func (m *AgentListModel) GetTypingElapsed(agentID string) time.Duration {
	if startTime, ok := m.typingStartMap[agentID]; ok {
		return time.Since(startTime)
	}
	return 0
}

// renderTypingIndicator renders the animated typing indicator with elapsed time.
func (m AgentListModel) renderTypingIndicator(agentID string) string {
	// Cycling dots animation based on animation frame
	dots := []string{".", "..", "..."}
	indicator := dots[m.animFrame]

	// Calculate elapsed time
	elapsed := m.GetTypingElapsed(agentID)
	var elapsedStr string
	if elapsed < time.Second {
		elapsedStr = fmt.Sprintf("%dms", elapsed.Milliseconds())
	} else if elapsed < time.Minute {
		elapsedStr = fmt.Sprintf("%.1fs", elapsed.Seconds())
	} else {
		elapsedStr = fmt.Sprintf("%.1fm", elapsed.Minutes())
	}

	typingLine := fmt.Sprintf("  typing%s %s", indicator, elapsedStr)
	return styles.TypingIndicatorStyle().Render(typingLine)
}

// truncateString truncates a string to the specified length, adding "..." if truncated.
func truncateString(s string, maxLen int) string {
	if maxLen <= 3 {
		return s
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
