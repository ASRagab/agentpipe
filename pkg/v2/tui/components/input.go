package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kevinelliott/agentpipe/pkg/v2/tui/styles"
)

// InputModel manages the user input panel.
type InputModel struct {
	textarea    textarea.Model
	width       int
	height      int
	focused     bool
	placeholder string
	maxChars    int
}

// InputSubmittedMsg is sent when the user submits input.
type InputSubmittedMsg struct {
	Content string
}

// NewInputModel creates a new input model.
func NewInputModel() InputModel {
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.ShowLineNumbers = false
	ta.CharLimit = 4096
	ta.SetHeight(3)

	return InputModel{
		textarea:    ta,
		placeholder: "Type your message...",
		maxChars:    4096,
		focused:     false,
	}
}

// Init initializes the input model.
func (m InputModel) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles messages for the input panel.
func (m InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.focused {
			switch msg.String() {
			case "ctrl+enter":
				// Submit the message
				content := strings.TrimSpace(m.textarea.Value())
				if content != "" {
					m.textarea.Reset()
					return m, func() tea.Msg {
						return InputSubmittedMsg{Content: content}
					}
				}
			case "esc":
				// Clear input
				m.textarea.Reset()
				return m, nil
			}
		}
	}

	if m.focused {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the input panel.
func (m InputModel) View() string {
	var b strings.Builder

	// Input area with border
	borderStyle := styles.PanelBorderStyle()
	if m.focused {
		borderStyle = styles.SelectedPanelBorderStyle()
	}

	// Header
	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("Message (Ctrl+Enter to send, Esc to clear)")

	// Character count
	charCount := len(m.textarea.Value())
	charCountStr := fmt.Sprintf("%d/%d", charCount, m.maxChars)
	charCountStyle := styles.CharCountStyle()
	if charCount > m.maxChars*9/10 {
		// Warn when approaching limit
		charCountStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	}

	// Combine header and char count
	headerWidth := m.width - 4 - len(charCountStr) - 2
	if headerWidth < 0 {
		headerWidth = 0
	}
	paddedHeader := header + strings.Repeat(" ", headerWidth-len(header)+2) + charCountStyle.Render(charCountStr)

	b.WriteString(paddedHeader)
	b.WriteString("\n")
	b.WriteString(m.textarea.View())

	return borderStyle.
		Width(m.width - 2).
		Render(b.String())
}

// SetSize updates the dimensions of the input panel.
func (m *InputModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.textarea.SetWidth(width - 4)
	m.textarea.SetHeight(height - 2)
}

// SetFocused sets the focused state of the input panel.
func (m *InputModel) SetFocused(focused bool) {
	m.focused = focused
	if focused {
		m.textarea.Focus()
	} else {
		m.textarea.Blur()
	}
}

// IsFocused returns whether the input panel is focused.
func (m *InputModel) IsFocused() bool {
	return m.focused
}

// Focus sets focus on the textarea.
func (m *InputModel) Focus() tea.Cmd {
	m.focused = true
	return m.textarea.Focus()
}

// Blur removes focus from the textarea.
func (m *InputModel) Blur() {
	m.focused = false
	m.textarea.Blur()
}

// GetValue returns the current input value.
func (m *InputModel) GetValue() string {
	return m.textarea.Value()
}

// SetValue sets the input value.
func (m *InputModel) SetValue(value string) {
	m.textarea.SetValue(value)
}

// Reset clears the input.
func (m *InputModel) Reset() {
	m.textarea.Reset()
}

// SetPlaceholder sets the placeholder text.
func (m *InputModel) SetPlaceholder(placeholder string) {
	m.placeholder = placeholder
	m.textarea.Placeholder = placeholder
}

// SetCharLimit sets the character limit.
func (m *InputModel) SetCharLimit(limit int) {
	m.maxChars = limit
	m.textarea.CharLimit = limit
}

// CharCount returns the current character count.
func (m *InputModel) CharCount() int {
	return len(m.textarea.Value())
}
