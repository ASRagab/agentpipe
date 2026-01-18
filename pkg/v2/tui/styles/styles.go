// Package styles provides the styling definitions for the AgentPipe v2 TUI.
package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// AgentColors provides a palette of distinct colors for agents.
var AgentColors = []lipgloss.Color{
	lipgloss.Color("33"),  // Blue
	lipgloss.Color("34"),  // Green
	lipgloss.Color("178"), // Yellow/Gold
	lipgloss.Color("133"), // Magenta
	lipgloss.Color("37"),  // Cyan
	lipgloss.Color("208"), // Orange
	lipgloss.Color("175"), // Pink
	lipgloss.Color("99"),  // Purple
}

// AgentColorForIndex returns a consistent color for the given agent index.
func AgentColorForIndex(index int) lipgloss.Color {
	return AgentColors[index%len(AgentColors)]
}

// UserMessageStyle returns the style for user messages.
func UserMessageStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Bold(true).
		PaddingLeft(2)
}

// AgentMessageStyle returns the style for agent messages with the given color.
func AgentMessageStyle(color lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(color).
		PaddingLeft(2)
}

// AgentNameStyle returns the style for agent name badges.
func AgentNameStyle(color lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(color).
		Bold(true)
}

// SystemMessageStyle returns the style for system messages.
func SystemMessageStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Italic(true).
		PaddingLeft(2)
}

// MetricsStyle returns the style for metrics display.
func MetricsStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)
}

// ErrorStyle returns the style for error messages.
func ErrorStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)
}

// TitleStyle returns the style for the title bar.
func TitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("63")).
		Padding(0, 1)
}

// StatusBarStyle returns the style for the status bar.
func StatusBarStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("236")).
		Padding(0, 1)
}

// HelpStyle returns the style for help text.
func HelpStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
}

// PanelBorderStyle returns the style for panel borders.
func PanelBorderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))
}

// SelectedPanelBorderStyle returns the style for selected panel borders.
func SelectedPanelBorderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63"))
}

// AgentListItemStyle returns the style for an agent list item.
func AgentListItemStyle(selected bool, color lipgloss.Color) lipgloss.Style {
	style := lipgloss.NewStyle().
		Foreground(color).
		PaddingLeft(1)

	if selected {
		style = style.
			Bold(true).
			Background(lipgloss.Color("237"))
	}

	return style
}

// AgentModelStyle returns the style for agent model text.
func AgentModelStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true).
		PaddingLeft(3)
}

// PlaceholderStyle returns the style for placeholder text.
func PlaceholderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true)
}

// CharCountStyle returns the style for character count display.
func CharCountStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
}

// StatusIndicator represents agent status emoji indicators.
type StatusIndicator string

const (
	StatusReady  StatusIndicator = "🟢"
	StatusTyping StatusIndicator = "🟡"
	StatusError  StatusIndicator = "🔴"
)

// HelpOverlayStyle returns the style for the help overlay.
func HelpOverlayStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2).
		Background(lipgloss.Color("235"))
}

// HelpKeyStyle returns the style for help key bindings.
func HelpKeyStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true)
}

// HelpDescStyle returns the style for help descriptions.
func HelpDescStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
}

// TypingIndicatorStyle returns the style for typing indicators.
func TypingIndicatorStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("226")). // Yellow for visibility
		Italic(true).
		PaddingLeft(1)
}

// MetricsBadgeStyle returns the style for metrics badges.
// Designed for subtle inline display with right-alignment support.
func MetricsBadgeStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")). // Subtle gray
		Italic(true)
}

// MetricsBadgeExpandedStyle returns the style for expanded metrics with more detail.
func MetricsBadgeExpandedStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("249")). // Slightly brighter for expanded view
		Italic(true)
}

// CostHighStyle returns the style for high-cost indicators.
func CostHighStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("208")). // Orange for attention
		Bold(true)
}

// CostLowStyle returns the style for low-cost indicators.
func CostLowStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("34")). // Green for good
		Italic(true)
}

// StatusBarContainerStyle returns the container style for the status bar.
func StatusBarContainerStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("252")).
		Padding(0, 1)
}

// StatusBarCenterStyle returns the style for the center section of the status bar.
func StatusBarCenterStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("250"))
}

// StatusBarTotalsStyle returns the style for the totals section of the status bar.
func StatusBarTotalsStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true)
}

// StatusActiveStyle returns the style for active conversation status.
func StatusActiveStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("34")). // Green
		Bold(true)
}

// StatusPausedStyle returns the style for paused conversation status.
func StatusPausedStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("226")). // Yellow
		Bold(true)
}

// StatusCompletedStyle returns the style for completed conversation status.
func StatusCompletedStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")). // Blue
		Bold(true)
}

// StatusErrorStyle returns the style for error conversation status.
func StatusErrorStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")). // Red
		Bold(true)
}

// NewMessagesIndicatorStyle returns the style for the "new messages below" indicator.
// Uses a visually distinct style to draw attention without being distracting.
func NewMessagesIndicatorStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")). // Blue
		Background(lipgloss.Color("235")). // Dark background
		Bold(true).
		Padding(0, 1)
}

// ScrollIndicatorStyle returns the style for scroll position indicators.
func ScrollIndicatorStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")). // Subtle gray
		Italic(true)
}
