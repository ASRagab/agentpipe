package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kevinelliott/agentpipe/pkg/v2/core"
	"github.com/kevinelliott/agentpipe/pkg/v2/tui/styles"
)

// ConversationModel manages the conversation view panel.
type ConversationModel struct {
	messages   []core.Message
	viewport   viewport.Model
	width      int
	height     int
	focused    bool
	ready      bool
	agentIndex map[string]int // Maps agent ID to color index
}

// NewConversationModel creates a new conversation model.
func NewConversationModel() ConversationModel {
	return ConversationModel{
		messages:   make([]core.Message, 0),
		agentIndex: make(map[string]int),
	}
}

// Init initializes the conversation model.
func (m ConversationModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the conversation view.
func (m ConversationModel) Update(msg tea.Msg) (ConversationModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.focused {
			switch msg.String() {
			case "pgup":
				m.viewport.ViewUp()
			case "pgdown":
				m.viewport.ViewDown()
			case "up", "k":
				m.viewport.LineUp(1)
			case "down", "j":
				m.viewport.LineDown(1)
			case "home":
				m.viewport.GotoTop()
			case "end":
				m.viewport.GotoBottom()
			}
		}

	case tea.WindowSizeMsg:
		// Handle resize
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height
		}
	}

	if m.ready {
		m.viewport, cmd = m.viewport.Update(msg)
	}

	return m, cmd
}

// View renders the conversation view.
func (m ConversationModel) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Build content
	content := m.renderMessages()
	m.viewport.SetContent(content)

	// Apply panel border
	borderStyle := styles.PanelBorderStyle()
	if m.focused {
		borderStyle = styles.SelectedPanelBorderStyle()
	}

	// Title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Render("Conversation")

	header := title + "\n" + strings.Repeat("─", m.width-4) + "\n"

	return borderStyle.
		Width(m.width - 2).
		Height(m.height - 2).
		Render(header + m.viewport.View())
}

// renderMessages renders all messages to a string.
func (m ConversationModel) renderMessages() string {
	if len(m.messages) == 0 {
		return styles.PlaceholderStyle().Render("No messages yet. Start typing to begin the conversation.")
	}

	var b strings.Builder

	for _, msg := range m.messages {
		m.renderMessage(&b, msg)
		b.WriteString("\n")
	}

	return b.String()
}

// renderMessage renders a single message.
func (m ConversationModel) renderMessage(b *strings.Builder, msg core.Message) {
	timestamp := msg.Timestamp.Format("15:04:05")

	switch msg.Role {
	case core.RoleUser:
		// User messages - prefixed with "You:"
		header := fmt.Sprintf("[%s] You:", timestamp)
		headerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Bold(true)
		b.WriteString(headerStyle.Render(header))
		b.WriteString("\n")
		b.WriteString(styles.UserMessageStyle().Render(msg.Content))

	case core.RoleAgent:
		// Agent messages with color-coded name
		colorIndex := m.getAgentColorIndex(msg.AgentID)
		color := styles.AgentColorForIndex(colorIndex)

		header := fmt.Sprintf("[%s] %s:", timestamp, msg.AgentName)
		headerStyle := styles.AgentNameStyle(color)
		b.WriteString(headerStyle.Render(header))

		// Show metrics if available
		if msg.Metrics != nil {
			metrics := m.formatMetrics(msg.Metrics)
			b.WriteString(" ")
			b.WriteString(styles.MetricsStyle().Render(metrics))
		}

		b.WriteString("\n")
		b.WriteString(styles.AgentMessageStyle(color).Render(msg.Content))

	case core.RoleSystem:
		// System messages - centered and gray
		header := fmt.Sprintf("[%s] System:", timestamp)
		b.WriteString(styles.SystemMessageStyle().Render(header))
		b.WriteString("\n")
		b.WriteString(styles.SystemMessageStyle().Render(msg.Content))
	}

	b.WriteString("\n")
}

// formatMetrics formats metrics for inline display.
func (m ConversationModel) formatMetrics(metrics *core.Metrics) string {
	if metrics == nil {
		return ""
	}

	parts := make([]string, 0, 3)

	if metrics.Duration > 0 {
		parts = append(parts, fmt.Sprintf("%dms", metrics.Duration.Milliseconds()))
	}

	if metrics.TotalTokens > 0 {
		parts = append(parts, fmt.Sprintf("%d tokens", metrics.TotalTokens))
	}

	if metrics.Cost > 0 {
		parts = append(parts, fmt.Sprintf("$%.4f", metrics.Cost))
	}

	if len(parts) == 0 {
		return ""
	}

	return "[" + strings.Join(parts, " | ") + "]"
}

// getAgentColorIndex returns the color index for an agent.
func (m *ConversationModel) getAgentColorIndex(agentID string) int {
	if idx, ok := m.agentIndex[agentID]; ok {
		return idx
	}
	// Assign new index
	idx := len(m.agentIndex)
	m.agentIndex[agentID] = idx
	return idx
}

// SetSize updates the dimensions of the conversation view.
func (m *ConversationModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	if m.ready {
		m.viewport.Width = width - 4  // Account for borders
		m.viewport.Height = height - 6 // Account for borders and header
	}
}

// SetFocused sets the focused state of the conversation view.
func (m *ConversationModel) SetFocused(focused bool) {
	m.focused = focused
}

// IsFocused returns whether the conversation view is focused.
func (m *ConversationModel) IsFocused() bool {
	return m.focused
}

// AddMessage adds a message to the conversation.
func (m *ConversationModel) AddMessage(msg core.Message) {
	m.messages = append(m.messages, msg)
	// Auto-scroll to bottom
	if m.ready {
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
	}
}

// SetMessages replaces all messages.
func (m *ConversationModel) SetMessages(messages []core.Message) {
	m.messages = messages
	if m.ready {
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
	}
}

// GetMessages returns all messages.
func (m *ConversationModel) GetMessages() []core.Message {
	return m.messages
}

// MessageCount returns the number of messages.
func (m *ConversationModel) MessageCount() int {
	return len(m.messages)
}

// ScrollToBottom scrolls the viewport to the bottom.
func (m *ConversationModel) ScrollToBottom() {
	if m.ready {
		m.viewport.GotoBottom()
	}
}

// InitViewport initializes the viewport with the given dimensions.
func (m *ConversationModel) InitViewport(width, height int) {
	if !m.ready {
		m.viewport = viewport.New(width-4, height-6)
		m.ready = true
		m.width = width
		m.height = height
	}
}

// SetAgentIndex sets the agent color index mapping.
func (m *ConversationModel) SetAgentIndex(index map[string]int) {
	m.agentIndex = index
}
