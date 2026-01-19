package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ASRagab/agentpipe/pkg/tui/styles"
	"github.com/ASRagab/agentpipe/pkg/utils"
)

// InputModel manages the user input panel.
type InputModel struct {
	textarea          textarea.Model
	width             int
	height            int
	focused           bool
	placeholder       string
	maxChars          int
	showTokenEstimate bool    // Whether to show token/cost estimate
	targetModel       string  // Model used for cost estimation
	lastTokenEstimate int     // Cached token estimate
	lastCostEstimate  float64 // Cached cost estimate
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
	ta.Focus() // Start focused so typing works immediately

	return InputModel{
		textarea:          ta,
		placeholder:       "Type your message...",
		maxChars:          4096,
		focused:           true, // Start focused
		showTokenEstimate: true,
		targetModel:       "", // Will use a default for estimation
		lastTokenEstimate: 0,
		lastCostEstimate:  0.0,
	}
}

// Init initializes the input model.
func (m InputModel) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles messages for the input panel.
func (m InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
	var cmds []tea.Cmd

	// Save previous content to detect changes
	prevContent := m.textarea.Value()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.focused {
			keyStr := msg.String()

			// Handle our special keys FIRST, before textarea gets them
			switch keyStr {
			case "enter":
				// Submit the message on Enter
				// (Use Shift+Enter for newlines in the textarea)
				content := strings.TrimSpace(m.textarea.Value())
				if content != "" {
					savedContent := content // capture for closure
					m.textarea.Reset()
					// Clear estimates on submit
					m.lastTokenEstimate = 0
					m.lastCostEstimate = 0.0
					return m, func() tea.Msg {
						return InputSubmittedMsg{Content: savedContent}
					}
				}
				return m, nil // Don't pass to textarea
			case "esc":
				// Clear input
				m.textarea.Reset()
				// Clear estimates on clear
				m.lastTokenEstimate = 0
				m.lastCostEstimate = 0.0
				return m, nil
			}

			// Pass other keys to textarea
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			cmds = append(cmds, cmd)
		}
	default:
		// Pass non-key messages to textarea
		if m.focused {
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Update token/cost estimate if content changed
	newContent := m.textarea.Value()
	if newContent != prevContent && m.showTokenEstimate {
		m.updateTokenEstimate()
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
		Render("Message (Enter to send, Esc to clear)")

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

	// Add token/cost estimate footer if enabled and there's content
	if m.showTokenEstimate && charCount > 0 {
		footer := m.renderTokenEstimateFooter()
		if footer != "" {
			b.WriteString("\n")
			b.WriteString(footer)
		}
	}

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

// SetShowTokenEstimate enables or disables the token/cost estimate display.
func (m *InputModel) SetShowTokenEstimate(show bool) {
	m.showTokenEstimate = show
}

// IsShowTokenEstimate returns whether token estimation is enabled.
func (m *InputModel) IsShowTokenEstimate() bool {
	return m.showTokenEstimate
}

// SetTargetModel sets the model used for cost estimation.
func (m *InputModel) SetTargetModel(model string) {
	m.targetModel = model
	m.updateTokenEstimate()
}

// GetTargetModel returns the model used for cost estimation.
func (m *InputModel) GetTargetModel() string {
	return m.targetModel
}

// GetTokenEstimate returns the current token estimate.
func (m *InputModel) GetTokenEstimate() int {
	return m.lastTokenEstimate
}

// GetCostEstimate returns the current cost estimate.
func (m *InputModel) GetCostEstimate() float64 {
	return m.lastCostEstimate
}

// EstimateInputTokens estimates the token count for the given text.
// Uses simple word count * 1.3 as specified in requirements.
func EstimateInputTokens(text string) int {
	if text == "" {
		return 0
	}
	words := len(strings.Fields(text))
	// If no words (whitespace only), return 0
	if words == 0 {
		return 0
	}
	// Simple estimation: word count * 1.3
	tokens := int(float64(words) * 1.3)
	if tokens == 0 && words > 0 {
		// At minimum, return 1 token for non-empty text with words
		tokens = 1
	}
	return tokens
}

// EstimateInputCost estimates the cost for the given token count and model.
// If model is empty, uses a default model for estimation.
func EstimateInputCost(tokens int, model string) float64 {
	if tokens == 0 {
		return 0.0
	}
	// Use utils.EstimateCost for accurate pricing
	// For user input, we estimate input tokens only (output is 0)
	if model == "" {
		// Default to claude-3.7-sonnet for estimation if no model specified
		model = "anthropic/claude-3.7-sonnet"
	}
	return utils.EstimateCost(model, tokens, 0)
}

// updateTokenEstimate recalculates the token and cost estimate from current content.
func (m *InputModel) updateTokenEstimate() {
	content := m.textarea.Value()
	m.lastTokenEstimate = EstimateInputTokens(content)
	m.lastCostEstimate = EstimateInputCost(m.lastTokenEstimate, m.targetModel)
}

// renderTokenEstimateFooter renders the token/cost estimate footer line.
func (m InputModel) renderTokenEstimateFooter() string {
	if m.lastTokenEstimate == 0 {
		return ""
	}

	// Format: "~15t | ~$0.0001"
	var parts []string

	// Token estimate with ~ prefix to indicate approximation
	parts = append(parts, fmt.Sprintf("~%dt", m.lastTokenEstimate))

	// Cost estimate
	if m.lastCostEstimate > 0 {
		parts = append(parts, formatEstimateCost(m.lastCostEstimate))
	}

	if len(parts) == 0 {
		return ""
	}

	estimate := strings.Join(parts, " | ")
	return styles.TokenEstimateStyle().Render(estimate)
}

// formatEstimateCost formats a cost estimate for display with ~ prefix.
func formatEstimateCost(cost float64) string {
	if cost >= 1.0 {
		return fmt.Sprintf("~$%.2f", cost)
	} else if cost >= 0.01 {
		return fmt.Sprintf("~$%.3f", cost)
	} else if cost >= 0.0001 {
		return fmt.Sprintf("~$%.4f", cost)
	}
	return fmt.Sprintf("~$%.5f", cost)
}
