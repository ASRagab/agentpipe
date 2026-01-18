package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kevinelliott/agentpipe/pkg/agent"
)

// Preview tile styles
var (
	previewTileStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(0, 1)

	previewTileActiveStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("82")).
				Padding(0, 1)

	previewHeaderStyle = lipgloss.NewStyle().
				Bold(true)

	previewContentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244"))
)

// renderPreviewTile renders a preview tile for an agent
func renderPreviewTile(agentName string, messages []agent.Message, lines int, width int, color lipgloss.Color, isActive bool) string {
	var b strings.Builder

	// Header with agent name and activity indicator
	indicator := "○" // Grey circle for inactive
	indicatorColor := lipgloss.Color("240")
	if isActive {
		indicator = "●"                       // Filled circle for active
		indicatorColor = lipgloss.Color("82") // Green
	}

	nameStyle := previewHeaderStyle.Foreground(color)
	indicatorStyle := lipgloss.NewStyle().Foreground(indicatorColor)

	// Calculate spacing for right-aligned indicator
	nameLen := len(agentName)
	availableWidth := width - 4 // Account for padding and border
	if availableWidth < 1 {
		availableWidth = 1
	}
	spacing := availableWidth - nameLen - 2
	if spacing < 1 {
		spacing = 1
	}

	header := nameStyle.Render(agentName) + strings.Repeat(" ", spacing) + indicatorStyle.Render(indicator)
	b.WriteString(header)
	b.WriteString("\n")

	// Get last N lines of content
	content := getLastNLines(messages, lines, availableWidth)
	if content == "" {
		content = previewContentStyle.Render("(no messages)")
	} else {
		content = previewContentStyle.Render(content)
	}
	b.WriteString(content)

	// Apply tile style
	tileStyle := previewTileStyle
	if isActive {
		tileStyle = previewTileActiveStyle
	}

	return tileStyle.Width(width).Render(b.String())
}

// getLastNLines extracts the last N lines from agent messages
func getLastNLines(messages []agent.Message, n int, width int) string {
	if len(messages) == 0 {
		return ""
	}

	// Collect all content lines
	var allLines []string
	for _, msg := range messages {
		if msg.Content == "" {
			continue
		}
		wrapped := wrapText(msg.Content, width)
		lines := strings.Split(wrapped, "\n")
		allLines = append(allLines, lines...)
	}

	if len(allLines) == 0 {
		return ""
	}

	// Take last N lines
	start := len(allLines) - n
	if start < 0 {
		start = 0
	}

	return strings.Join(allLines[start:], "\n")
}

// renderPreviewSidebar renders all preview tiles for non-selected agents
func renderPreviewSidebar(m *EnhancedModel, width int, height int) string {
	var tiles []string

	for i, agentName := range m.agentOrder {
		// Skip the currently selected agent
		if i == m.selectedAgentIndex {
			continue
		}

		messages := m.agentMessages[agentName]
		color := m.agentColors[agentName]
		isActive := m.activeAgent == agentName

		tile := renderPreviewTile(agentName, messages, m.previewLines, width-2, color, isActive)
		tiles = append(tiles, tile)
	}

	if len(tiles) == 0 {
		return lipgloss.NewStyle().
			Width(width).
			Height(height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("244")).
			Render("No other agents")
	}

	return lipgloss.JoinVertical(lipgloss.Top, tiles...)
}
