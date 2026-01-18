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

// AgentListModel manages the agent list panel.
type AgentListModel struct {
	agents        []core.Agent
	statusMap     map[string]AgentStatus
	metricsMap    map[string]AgentMetrics
	selectedIndex int
	width         int
	height        int
	focused       bool
}

// NewAgentListModel creates a new agent list model.
func NewAgentListModel(agents []core.Agent) AgentListModel {
	statusMap := make(map[string]AgentStatus)
	metricsMap := make(map[string]AgentMetrics)

	for _, agent := range agents {
		statusMap[agent.ID] = AgentStatusReady
	}

	return AgentListModel{
		agents:        agents,
		statusMap:     statusMap,
		metricsMap:    metricsMap,
		selectedIndex: 0,
		focused:       false,
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

		// Show metrics if available
		if metrics, ok := m.metricsMap[agent.ID]; ok {
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
	m.statusMap[agentID] = status
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
