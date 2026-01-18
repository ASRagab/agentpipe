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
	Error       string
	Timestamp   time.Time
	ErrorType   string        // Error category (timeout, rate_limit, network, auth, etc.)
	Recoverable bool          // Whether the error can be retried
	RetryAfter  time.Duration // Suggested wait time before retrying (for rate limits)
	RetryHint   string        // Actionable hint for the user
}

// HealthStatus represents the health state of an agent in the TUI.
type HealthStatus string

const (
	// HealthStatusUnknown means health has not been checked yet.
	HealthStatusUnknown HealthStatus = "unknown"
	// HealthStatusHealthy means the agent is responding normally.
	HealthStatusHealthy HealthStatus = "healthy"
	// HealthStatusUnhealthy means the agent has failed recent health checks.
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	// HealthStatusDegraded means the agent is slow but responding.
	HealthStatusDegraded HealthStatus = "degraded"
	// HealthStatusChecking means a health check is in progress.
	HealthStatusChecking HealthStatus = "checking"
)

// AgentHealthInfo contains health information for display.
type AgentHealthInfo struct {
	Status       HealthStatus
	LastCheck    time.Time
	ResponseTime time.Duration
	Error        string
}

// AgentListModel manages the agent list panel.
type AgentListModel struct {
	agents         []core.Agent
	statusMap      map[string]AgentStatus
	metricsMap     map[string]AgentMetrics
	typingStartMap map[string]time.Time       // When each agent started typing
	errorMap       map[string]AgentErrorInfo  // Error info for each agent
	healthMap      map[string]AgentHealthInfo // Health info for each agent
	animFrame      int                        // Current frame for typing animation (0-2)
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
	healthMap := make(map[string]AgentHealthInfo)

	for _, agent := range agents {
		statusMap[agent.ID] = AgentStatusReady
		healthMap[agent.ID] = AgentHealthInfo{Status: HealthStatusUnknown}
	}

	return AgentListModel{
		agents:         agents,
		statusMap:      statusMap,
		metricsMap:     metricsMap,
		typingStartMap: typingStartMap,
		errorMap:       errorMap,
		healthMap:      healthMap,
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
		healthInfo := m.healthMap[agent.ID]
		var indicator styles.StatusIndicator
		switch status {
		case AgentStatusTyping:
			indicator = styles.StatusTyping
		case AgentStatusError:
			indicator = styles.StatusError
		default:
			// If ready but unhealthy, show warning indicator
			if healthInfo.Status == HealthStatusUnhealthy {
				indicator = styles.StatusError // Reuse error indicator for unhealthy
			} else {
				indicator = styles.StatusReady
			}
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
				// Show error message
				errorStyle := styles.ErrorStyle()
				errorLine := fmt.Sprintf("  ⚠ %s", truncateString(errInfo.Error, m.width-8))
				b.WriteString(errorStyle.Render(errorLine))
				b.WriteString("\n")

				// Show retry countdown if rate limited
				if countdown := m.GetRetryCountdown(agent.ID); countdown > 0 {
					retryLine := m.renderRetryCountdown(countdown)
					b.WriteString(retryLine)
					b.WriteString("\n")
				} else if errInfo.Recoverable && errInfo.RetryHint != "" {
					// Show retry hint for recoverable errors
					hintLine := fmt.Sprintf("  → %s", truncateString(errInfo.RetryHint, m.width-10))
					b.WriteString(styles.RetryHintStyle().Render(hintLine))
					b.WriteString("\n")
				}
			}
		} else if healthInfo.Status == HealthStatusUnhealthy {
			// Show unhealthy agent warning
			healthLine := m.renderHealthStatus(agent.ID)
			if healthLine != "" {
				b.WriteString(healthLine)
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

// UpdateErrorWithDetails updates the error info with full details for a specific agent.
func (m *AgentListModel) UpdateErrorWithDetails(agentID string, info AgentErrorInfo) {
	if info.Timestamp.IsZero() {
		info.Timestamp = time.Now()
	}
	m.errorMap[agentID] = info
}

// GetRetryCountdown returns the remaining retry countdown for a rate-limited agent.
// Returns 0 if there's no active countdown.
func (m *AgentListModel) GetRetryCountdown(agentID string) time.Duration {
	errInfo, ok := m.errorMap[agentID]
	if !ok || errInfo.RetryAfter == 0 {
		return 0
	}

	// Calculate remaining time
	elapsed := time.Since(errInfo.Timestamp)
	remaining := errInfo.RetryAfter - elapsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// HasActiveRetryCountdown returns true if any agent has an active retry countdown.
func (m *AgentListModel) HasActiveRetryCountdown() bool {
	for agentID := range m.errorMap {
		if m.GetRetryCountdown(agentID) > 0 {
			return true
		}
	}
	return false
}

// GetAgentsWithErrors returns a list of agent IDs that currently have errors.
func (m *AgentListModel) GetAgentsWithErrors() []string {
	result := make([]string, 0, len(m.errorMap))
	for agentID := range m.errorMap {
		result = append(result, agentID)
	}
	return result
}

// GetRecoverableAgents returns agent IDs with recoverable (retryable) errors.
func (m *AgentListModel) GetRecoverableAgents() []string {
	result := make([]string, 0)
	for agentID, info := range m.errorMap {
		if info.Recoverable {
			result = append(result, agentID)
		}
	}
	return result
}

// ClearError clears the error for a specific agent.
func (m *AgentListModel) ClearError(agentID string) {
	delete(m.errorMap, agentID)
}

// RenderErrorDetailsModal renders a detailed error information modal for the selected agent.
// Returns empty string if no error or no agent selected.
func (m *AgentListModel) RenderErrorDetailsModal(width int) string {
	agent := m.GetSelectedAgent()
	if agent == nil {
		return ""
	}

	errInfo, ok := m.errorMap[agent.ID]
	if !ok {
		return ""
	}

	var b strings.Builder

	// Modal header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196"))
	b.WriteString(headerStyle.Render(fmt.Sprintf("⚠ Error Details: %s", agent.Name)))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", width-4))
	b.WriteString("\n\n")

	// Error type badge
	if errInfo.ErrorType != "" {
		badgeStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("196")).
			Bold(true).
			Padding(0, 1)
		b.WriteString("Type: ")
		b.WriteString(badgeStyle.Render(strings.ToUpper(errInfo.ErrorType)))
		b.WriteString("\n\n")
	}

	// Error message
	messageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	b.WriteString("Message:\n")
	b.WriteString(messageStyle.Render("  " + errInfo.Error))
	b.WriteString("\n\n")

	// Timestamp
	timestampStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true)
	b.WriteString("Time: ")
	b.WriteString(timestampStyle.Render(errInfo.Timestamp.Format("15:04:05")))
	b.WriteString("\n\n")

	// Retry information
	if errInfo.Recoverable {
		recoverableStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("34")). // Green
			Bold(true)
		b.WriteString("Status: ")
		b.WriteString(recoverableStyle.Render("Recoverable"))
		b.WriteString("\n")

		// Show countdown or retry hint
		if countdown := m.GetRetryCountdown(agent.ID); countdown > 0 {
			countdownStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("220")). // Yellow
				Italic(true)
			b.WriteString("Retry in: ")
			b.WriteString(countdownStyle.Render(fmt.Sprintf("%.0f seconds", countdown.Seconds())))
			b.WriteString("\n")
		}

		if errInfo.RetryHint != "" {
			hintStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				Italic(true)
			b.WriteString("\n")
			b.WriteString(hintStyle.Render("→ " + errInfo.RetryHint))
			b.WriteString("\n")
		}
	} else {
		nonRecoverableStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")). // Red
			Bold(true)
		b.WriteString("Status: ")
		b.WriteString(nonRecoverableStyle.Render("Non-recoverable"))
		b.WriteString("\n")
	}

	// Help footer
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
	if errInfo.Recoverable {
		b.WriteString(helpStyle.Render("Press 'r' to retry | 'esc' to close"))
	} else {
		b.WriteString(helpStyle.Render("Press 'esc' to close"))
	}

	// Apply modal border style
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(1, 2).
		Width(width - 4)

	return modalStyle.Render(b.String())
}

// HasSelectedAgentError returns true if the currently selected agent has an error.
func (m *AgentListModel) HasSelectedAgentError() bool {
	agent := m.GetSelectedAgent()
	if agent == nil {
		return false
	}
	_, ok := m.errorMap[agent.ID]
	return ok
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

// renderRetryCountdown renders the retry countdown indicator.
func (m AgentListModel) renderRetryCountdown(countdown time.Duration) string {
	var countdownStr string
	if countdown >= time.Minute {
		countdownStr = fmt.Sprintf("%.0fm%.0fs", countdown.Minutes(), countdown.Seconds()-countdown.Truncate(time.Minute).Seconds())
	} else if countdown >= time.Second {
		countdownStr = fmt.Sprintf("%.0fs", countdown.Seconds())
	} else {
		countdownStr = "< 1s"
	}

	retryLine := fmt.Sprintf("  ⏳ Retrying in %s...", countdownStr)
	return styles.RetryCountdownStyle().Render(retryLine)
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

// renderHealthStatus renders the health status for an agent.
func (m AgentListModel) renderHealthStatus(agentID string) string {
	healthInfo, ok := m.healthMap[agentID]
	if !ok {
		return ""
	}

	var healthLine string
	switch healthInfo.Status {
	case HealthStatusUnhealthy:
		if healthInfo.Error != "" {
			healthLine = fmt.Sprintf("  ⚠ Unhealthy: %s", truncateString(healthInfo.Error, m.width-16))
		} else {
			healthLine = "  ⚠ Unhealthy"
		}
		return styles.ErrorStyle().Render(healthLine)
	case HealthStatusDegraded:
		healthLine = fmt.Sprintf("  ⚡ Slow (%dms)", healthInfo.ResponseTime.Milliseconds())
		return styles.RetryHintStyle().Render(healthLine)
	case HealthStatusChecking:
		healthLine = "  ↻ Checking health..."
		return styles.MetricsStyle().Render(healthLine)
	default:
		return ""
	}
}

// UpdateHealth updates the health status for a specific agent.
func (m *AgentListModel) UpdateHealth(agentID string, status HealthStatus, responseTime time.Duration, errMsg string) {
	m.healthMap[agentID] = AgentHealthInfo{
		Status:       status,
		LastCheck:    time.Now(),
		ResponseTime: responseTime,
		Error:        errMsg,
	}
}

// UpdateHealthInfo updates the full health info for a specific agent.
func (m *AgentListModel) UpdateHealthInfo(agentID string, info AgentHealthInfo) {
	if info.LastCheck.IsZero() {
		info.LastCheck = time.Now()
	}
	m.healthMap[agentID] = info
}

// GetHealth returns the health info for a specific agent.
func (m *AgentListModel) GetHealth(agentID string) (AgentHealthInfo, bool) {
	info, ok := m.healthMap[agentID]
	return info, ok
}

// GetUnhealthyAgents returns a list of agent IDs that are currently unhealthy.
func (m *AgentListModel) GetUnhealthyAgents() []string {
	result := make([]string, 0)
	for agentID, info := range m.healthMap {
		if info.Status == HealthStatusUnhealthy {
			result = append(result, agentID)
		}
	}
	return result
}

// GetHealthyAgents returns a list of agent IDs that are currently healthy.
func (m *AgentListModel) GetHealthyAgents() []string {
	result := make([]string, 0)
	for agentID, info := range m.healthMap {
		if info.Status == HealthStatusHealthy || info.Status == HealthStatusUnknown {
			result = append(result, agentID)
		}
	}
	return result
}

// HasUnhealthyAgents returns true if any agent is unhealthy.
func (m *AgentListModel) HasUnhealthyAgents() bool {
	for _, info := range m.healthMap {
		if info.Status == HealthStatusUnhealthy {
			return true
		}
	}
	return false
}

// ResetHealth resets the health status for a specific agent to unknown.
func (m *AgentListModel) ResetHealth(agentID string) {
	if _, ok := m.healthMap[agentID]; ok {
		m.healthMap[agentID] = AgentHealthInfo{Status: HealthStatusUnknown}
	}
}

// GetSelectedAgentHealth returns the health info for the currently selected agent.
func (m *AgentListModel) GetSelectedAgentHealth() (AgentHealthInfo, bool) {
	agent := m.GetSelectedAgent()
	if agent == nil {
		return AgentHealthInfo{}, false
	}
	return m.GetHealth(agent.ID)
}

// RenderHealthDetailsTooltip renders a health details tooltip for the selected agent.
func (m *AgentListModel) RenderHealthDetailsTooltip(width int) string {
	agent := m.GetSelectedAgent()
	if agent == nil {
		return ""
	}

	healthInfo, ok := m.healthMap[agent.ID]
	if !ok {
		return ""
	}

	var b strings.Builder

	// Status header
	var statusStr string
	var statusColor string
	switch healthInfo.Status {
	case HealthStatusHealthy:
		statusStr = "✓ Healthy"
		statusColor = "34" // Green
	case HealthStatusUnhealthy:
		statusStr = "✗ Unhealthy"
		statusColor = "196" // Red
	case HealthStatusDegraded:
		statusStr = "⚡ Degraded"
		statusColor = "220" // Yellow
	case HealthStatusChecking:
		statusStr = "↻ Checking"
		statusColor = "245" // Gray
	default:
		statusStr = "? Unknown"
		statusColor = "245" // Gray
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(statusColor))
	b.WriteString(headerStyle.Render(statusStr))
	b.WriteString("\n")

	// Last check time
	if !healthInfo.LastCheck.IsZero() {
		checkStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Italic(true)
		b.WriteString("Last check: ")
		b.WriteString(checkStyle.Render(healthInfo.LastCheck.Format("15:04:05")))
		b.WriteString("\n")
	}

	// Response time
	if healthInfo.ResponseTime > 0 {
		b.WriteString(fmt.Sprintf("Response: %dms\n", healthInfo.ResponseTime.Milliseconds()))
	}

	// Error message
	if healthInfo.Error != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))
		b.WriteString("Error: ")
		b.WriteString(errorStyle.Render(truncateString(healthInfo.Error, width-10)))
		b.WriteString("\n")
	}

	// Apply tooltip style
	tooltipStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(width - 4)

	return tooltipStyle.Render(b.String())
}
