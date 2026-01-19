package tui

// Layout contains the calculated dimensions for each panel.
type Layout struct {
	// AgentList dimensions
	AgentListWidth  int
	AgentListHeight int

	// Conversation dimensions
	ConversationWidth  int
	ConversationHeight int

	// Input dimensions
	InputWidth  int
	InputHeight int

	// StatusBar dimensions
	StatusBarWidth  int
	StatusBarHeight int

	// Total dimensions
	TotalWidth  int
	TotalHeight int
}

const (
	MinAgentListWidth     = 20
	AgentListWidthPercent = 0.20
	InputHeight           = 3
	StatusBarHeight       = 1
	LogoHeight            = 6
	LogoVisibleWidth      = 73
	BorderPadding         = 2

	// Minimum terminal dimensions
	MinTerminalWidth  = 80
	MinTerminalHeight = 24
)

// CalculateLayout computes panel dimensions based on terminal size.
func CalculateLayout(width, height int) Layout {
	// Calculate agent list width (20% of total, minimum 20 chars)
	agentListWidth := int(float64(width) * AgentListWidthPercent)
	if agentListWidth < MinAgentListWidth {
		agentListWidth = MinAgentListWidth
	}
	// Cap at reasonable max
	if agentListWidth > 40 {
		agentListWidth = 40
	}

	// Conversation takes remaining width
	conversationWidth := width - agentListWidth - BorderPadding

	// Height calculations:
	// - Logo at top: 6 lines
	// - Status bar below logo: 1 line
	// - Input area at bottom: 3 lines
	// - Remaining goes to agent list and conversation
	availableHeight := height - LogoHeight - StatusBarHeight - InputHeight - BorderPadding*2

	// Agent list and conversation share the available height
	agentListHeight := availableHeight
	conversationHeight := availableHeight

	return Layout{
		AgentListWidth:     agentListWidth,
		AgentListHeight:    agentListHeight,
		ConversationWidth:  conversationWidth,
		ConversationHeight: conversationHeight,
		InputWidth:         width - BorderPadding,
		InputHeight:        InputHeight,
		StatusBarWidth:     width,
		StatusBarHeight:    StatusBarHeight,
		TotalWidth:         width,
		TotalHeight:        height,
	}
}

// RecalculateOnResize updates layout when terminal is resized.
func RecalculateOnResize(oldLayout Layout, newWidth, newHeight int) Layout {
	return CalculateLayout(newWidth, newHeight)
}

// IsMinimumSize checks if the terminal meets minimum size requirements.
func IsMinimumSize(width, height int) bool {
	return width >= MinTerminalWidth && height >= MinTerminalHeight
}

// MinimumSizeMessage returns a message to display if terminal is too small.
func MinimumSizeMessage() string {
	return "Terminal too small. Please resize to at least 80x24."
}
