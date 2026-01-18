# Multi-Window TUI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the single conversation panel with per-agent windows showing real-time streaming output.

**Architecture:** Refactor `EnhancedModel` to store messages per-agent in `map[string][]agent.Message`, add per-agent viewports, implement tabbed navigation with preview sidebar. Keep existing message channel architecture, logo, status bar, and styling.

**Tech Stack:** Go 1.24, Bubble Tea (bubbletea), Bubbles (viewport, textarea, list), Lipgloss

---

## Task 1: Add Multi-Window State Fields to EnhancedModel

**Files:**
- Modify: `pkg/tui/enhanced.go:35-85` (EnhancedModel struct)

**Step 1: Add new fields to EnhancedModel struct**

Add these fields after line 85 (after `agentColors`):

```go
	// Multi-window state
	agentMessages      map[string][]agent.Message  // Per-agent message buffers
	agentViewports     map[string]viewport.Model   // Per-agent viewports
	agentOrder         []string                    // Ordered list of agent names
	selectedAgentIndex int                         // Which agent is in main view
	autoFollow         bool                        // Toggle for following active agent
	previewLines       int                         // Lines to show in previews (default: 3)
```

**Step 2: Run tests to verify no regressions**

Run: `go test ./pkg/tui/... -v -count=1`
Expected: All existing tests pass

**Step 3: Commit**

```bash
git add pkg/tui/enhanced.go
git commit -m "feat(tui): add multi-window state fields to EnhancedModel"
```

---

## Task 2: Initialize Multi-Window State in RunEnhanced

**Files:**
- Modify: `pkg/tui/enhanced.go:272-421` (RunEnhanced function)

**Step 1: Initialize new maps in RunEnhanced**

After line 400 (after `configPath: configPath,`), add initialization:

```go
		agentMessages:      make(map[string][]agent.Message),
		agentViewports:     make(map[string]viewport.Model),
		agentOrder:         make([]string, 0),
		selectedAgentIndex: 0,
		autoFollow:         true,  // Default to auto-follow enabled
		previewLines:       3,
```

**Step 2: Build agent order from config**

After the agent color assignment loop (around line 287), capture agent order:

```go
	// Build agent order for multi-window navigation
	agentOrderList := make([]string, 0, len(agents))
	for _, a := range agents {
		agentOrderList = append(agentOrderList, a.GetName())
	}
```

Then pass it to the model initialization.

**Step 3: Run tests**

Run: `go test ./pkg/tui/... -v -count=1`
Expected: All tests pass

**Step 4: Commit**

```bash
git add pkg/tui/enhanced.go
git commit -m "feat(tui): initialize multi-window state in RunEnhanced"
```

---

## Task 3: Create Preview Tile Rendering

**Files:**
- Create: `pkg/tui/preview.go`
- Create: `pkg/tui/preview_test.go`

**Step 1: Write the failing test**

Create `pkg/tui/preview_test.go`:

```go
package tui

import (
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ASRagab/agentpipe/pkg/agent"
)

func TestRenderPreviewTile(t *testing.T) {
	messages := []agent.Message{
		{Content: "First message from agent", Timestamp: time.Now().Unix()},
		{Content: "Second message with more content here", Timestamp: time.Now().Unix()},
		{Content: "Third and final message", Timestamp: time.Now().Unix()},
	}

	tile := renderPreviewTile("Claude", messages, 3, 25, lipgloss.Color("63"), false)

	if tile == "" {
		t.Error("Expected non-empty tile")
	}
	if !containsString(tile, "Claude") {
		t.Error("Expected tile to contain agent name")
	}
}

func TestRenderPreviewTile_Empty(t *testing.T) {
	tile := renderPreviewTile("Claude", []agent.Message{}, 3, 25, lipgloss.Color("63"), false)

	if tile == "" {
		t.Error("Expected non-empty tile even with no messages")
	}
	if !containsString(tile, "Claude") {
		t.Error("Expected tile to contain agent name")
	}
}

func TestRenderPreviewTile_ActiveAgent(t *testing.T) {
	messages := []agent.Message{
		{Content: "Some message", Timestamp: time.Now().Unix()},
	}

	tile := renderPreviewTile("Claude", messages, 3, 25, lipgloss.Color("63"), true)

	// Active agent should have green indicator
	if !containsString(tile, "Claude") {
		t.Error("Expected tile to contain agent name")
	}
}

func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/tui/... -v -run TestRenderPreviewTile`
Expected: FAIL - `renderPreviewTile` not defined

**Step 3: Write the implementation**

Create `pkg/tui/preview.go`:

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ASRagab/agentpipe/pkg/agent"
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
		indicator = "●" // Filled circle for active
		indicatorColor = lipgloss.Color("82") // Green
	}

	nameStyle := previewHeaderStyle.Foreground(color)
	indicatorStyle := lipgloss.NewStyle().Foreground(indicatorColor)

	// Calculate spacing for right-aligned indicator
	nameLen := len(agentName)
	availableWidth := width - 4 // Account for padding and border
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

		// Calculate tile height based on number of agents
		numOtherAgents := len(m.agentOrder) - 1
		if numOtherAgents < 1 {
			numOtherAgents = 1
		}
		tileHeight := (height - 2) / numOtherAgents
		if tileHeight < 4 {
			tileHeight = 4
		}

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
```

**Step 4: Run tests to verify they pass**

Run: `go test ./pkg/tui/... -v -run TestRenderPreviewTile`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/tui/preview.go pkg/tui/preview_test.go
git commit -m "feat(tui): add preview tile rendering for multi-window view"
```

---

## Task 4: Add Tab Navigation Key Handlers

**Files:**
- Modify: `pkg/tui/enhanced.go:555-897` (Update function)

**Step 1: Write the failing test**

Add to `pkg/tui/enhanced_test.go`:

```go
func TestEnhancedModel_TabNavigation(t *testing.T) {
	cfg := &config.Config{
		Orchestrator: config.OrchestratorConfig{
			Mode:     "round-robin",
			MaxTurns: 10,
		},
	}

	m := createTestEnhancedModel(cfg, conversationPanel, false)
	m.agentOrder = []string{"Agent1", "Agent2", "Agent3"}
	m.selectedAgentIndex = 0
	m.agentMessages = make(map[string][]agent.Message)
	m.agentViewports = make(map[string]viewport.Model)

	// Test right arrow navigation
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	updated := newModel.(EnhancedModel)
	if updated.selectedAgentIndex != 1 {
		t.Errorf("Expected selectedAgentIndex=1, got %d", updated.selectedAgentIndex)
	}

	// Test left arrow navigation
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyLeft})
	updated = newModel.(EnhancedModel)
	if updated.selectedAgentIndex != 0 {
		t.Errorf("Expected selectedAgentIndex=0, got %d", updated.selectedAgentIndex)
	}

	// Test wrap-around right
	m.selectedAgentIndex = 2
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	updated = newModel.(EnhancedModel)
	if updated.selectedAgentIndex != 0 {
		t.Errorf("Expected wrap to 0, got %d", updated.selectedAgentIndex)
	}
}

func TestEnhancedModel_AutoFollowToggle(t *testing.T) {
	cfg := &config.Config{
		Orchestrator: config.OrchestratorConfig{
			Mode: "round-robin",
		},
	}

	m := createTestEnhancedModel(cfg, conversationPanel, false)
	m.autoFollow = true

	// Press 'f' to toggle
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	updated := newModel.(EnhancedModel)
	if updated.autoFollow != false {
		t.Error("Expected autoFollow to be toggled off")
	}

	// Press 'f' again to toggle back
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	updated = newModel.(EnhancedModel)
	if updated.autoFollow != true {
		t.Error("Expected autoFollow to be toggled on")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./pkg/tui/... -v -run TestEnhancedModel_TabNavigation`
Expected: FAIL (fields don't exist or navigation not implemented)

**Step 3: Add navigation handlers to Update function**

In `Update()` function, add these cases inside the `switch msg.String()` block (around line 567):

```go
		case "left", "h":
			// Previous agent tab (when not in input panel)
			if m.activePanel != inputPanel && len(m.agentOrder) > 0 {
				m.selectedAgentIndex--
				if m.selectedAgentIndex < 0 {
					m.selectedAgentIndex = len(m.agentOrder) - 1
				}
			}

		case "right", "l":
			// Next agent tab (when not in input panel)
			if m.activePanel != inputPanel && len(m.agentOrder) > 0 {
				m.selectedAgentIndex = (m.selectedAgentIndex + 1) % len(m.agentOrder)
			}

		case "f":
			// Toggle auto-follow
			m.autoFollow = !m.autoFollow
```

**Step 4: Run tests to verify they pass**

Run: `go test ./pkg/tui/... -v -run "TestEnhancedModel_TabNavigation|TestEnhancedModel_AutoFollowToggle"`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/tui/enhanced.go pkg/tui/enhanced_test.go
git commit -m "feat(tui): add tab navigation and auto-follow toggle handlers"
```

---

## Task 5: Route Messages to Per-Agent Buffers

**Files:**
- Modify: `pkg/tui/enhanced.go:745-795` (messageUpdate handling in Update)

**Step 1: Write the failing test**

Add to `pkg/tui/enhanced_test.go`:

```go
func TestEnhancedModel_MessageRouting(t *testing.T) {
	cfg := &config.Config{
		Orchestrator: config.OrchestratorConfig{
			Mode: "round-robin",
		},
	}

	m := createTestEnhancedModel(cfg, conversationPanel, false)
	m.agentOrder = []string{"Agent1", "Agent2"}
	m.agentMessages = make(map[string][]agent.Message)
	m.agentViewports = make(map[string]viewport.Model)
	m.running = true

	// Simulate message from Agent1
	msg := messageUpdate{
		message: agent.Message{
			AgentID:   "agent1",
			AgentName: "Agent1",
			Content:   "Hello from Agent1",
			Role:      "agent",
		},
	}

	newModel, _ := m.Update(msg)
	updated := newModel.(EnhancedModel)

	// Check message was routed to Agent1's buffer
	if len(updated.agentMessages["Agent1"]) != 1 {
		t.Errorf("Expected 1 message for Agent1, got %d", len(updated.agentMessages["Agent1"]))
	}
	if len(updated.agentMessages["Agent2"]) != 0 {
		t.Errorf("Expected 0 messages for Agent2, got %d", len(updated.agentMessages["Agent2"]))
	}
}

func TestEnhancedModel_AutoFollowOnActive(t *testing.T) {
	cfg := &config.Config{
		Orchestrator: config.OrchestratorConfig{
			Mode: "round-robin",
		},
	}

	m := createTestEnhancedModel(cfg, conversationPanel, false)
	m.agentOrder = []string{"Agent1", "Agent2"}
	m.agentMessages = make(map[string][]agent.Message)
	m.agentViewports = make(map[string]viewport.Model)
	m.selectedAgentIndex = 0
	m.autoFollow = true
	m.running = true

	// Simulate "active" message from Agent2
	msg := messageUpdate{
		message: agent.Message{
			AgentID:   "agent2",
			AgentName: "Agent2",
			Role:      "active",
		},
	}

	newModel, _ := m.Update(msg)
	updated := newModel.(EnhancedModel)

	// Should auto-switch to Agent2
	if updated.selectedAgentIndex != 1 {
		t.Errorf("Expected selectedAgentIndex=1 (Agent2), got %d", updated.selectedAgentIndex)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./pkg/tui/... -v -run "TestEnhancedModel_MessageRouting|TestEnhancedModel_AutoFollowOnActive"`
Expected: FAIL

**Step 3: Update messageUpdate handling in Update function**

Replace the `case messageUpdate:` block (around line 745) with:

```go
	case messageUpdate:
		// Message received - waiter completed, spawn next one
		m.waitingForMessage = false
		if m.running {
			m.waitingForMessage = true
			cmds = append(cmds, m.waitForMessage())
		}

		if msg.message.Role == "active" {
			// This is just an indicator that an agent is actively typing
			m.activeAgent = msg.message.AgentName

			// Auto-follow if enabled
			if m.autoFollow {
				for i, name := range m.agentOrder {
					if name == msg.message.AgentName {
						m.selectedAgentIndex = i
						break
					}
				}
			}
		} else {
			// Route message to per-agent buffer
			agentName := msg.message.AgentName
			if agentName == "" {
				agentName = "System"
			}

			// Initialize buffer if needed
			if m.agentMessages == nil {
				m.agentMessages = make(map[string][]agent.Message)
			}
			m.agentMessages[agentName] = append(m.agentMessages[agentName], msg.message)

			// Also keep in legacy messages array for compatibility
			m.messages = append(m.messages, msg.message)

			// Log the message if logging is enabled
			if m.chatLogger != nil {
				m.chatLogger.LogMessage(msg.message)
			}

			// Track turn count and cost for agent messages
			if msg.message.Role == "agent" {
				m.turnCount++
				if msg.message.AgentName == m.activeAgent {
					m.activeAgent = ""
				}
				if msg.message.Metrics != nil {
					if msg.message.Metrics.Cost > 0 {
						m.totalCost += msg.message.Metrics.Cost
					}
					if msg.message.Metrics.Duration > 0 {
						m.totalTime += msg.message.Metrics.Duration
					}
				}
			}

			// Update viewport for this agent
			if m.ready && m.agentViewports != nil {
				if vp, exists := m.agentViewports[agentName]; exists {
					vp.SetContent(m.renderAgentMessages(agentName))
					vp.GotoBottom()
					m.agentViewports[agentName] = vp
				}
			}

			// Handle conversation state messages
			if strings.Contains(msg.message.Content, "Starting AgentPipe conversation") {
				m.running = true
			}
			if strings.Contains(msg.message.Content, "Conversation ended") {
				m.running = false
			}

			// Also update legacy conversation viewport
			m.conversation.SetContent(m.renderConversation())
			m.conversation.GotoBottom()
		}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./pkg/tui/... -v -run "TestEnhancedModel_MessageRouting|TestEnhancedModel_AutoFollowOnActive"`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/tui/enhanced.go pkg/tui/enhanced_test.go
git commit -m "feat(tui): route messages to per-agent buffers with auto-follow"
```

---

## Task 6: Add Per-Agent Message Rendering

**Files:**
- Modify: `pkg/tui/enhanced.go` (add new renderAgentMessages method)

**Step 1: Add renderAgentMessages method**

Add after `renderConversation()` (around line 1365):

```go
// renderAgentMessages renders messages for a specific agent
func (m *EnhancedModel) renderAgentMessages(agentName string) string {
	var b strings.Builder

	messages, exists := m.agentMessages[agentName]
	if !exists || len(messages) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Render("(no messages yet)")
	}

	// Get viewport width for text wrapping
	textWidth := 80
	if m.ready {
		textWidth = m.width - 50 // Account for sidebar
		if textWidth < 40 {
			textWidth = 40
		}
	}

	for i, msg := range messages {
		timestamp := time.Unix(msg.Timestamp, 0).Format("15:04:05")

		// Get color for this agent
		color := m.agentColors[agentName]
		if color == "" {
			color = lipgloss.Color("244")
		}

		// Header with timestamp
		headerStyle := lipgloss.NewStyle().Foreground(color).Bold(true)
		b.WriteString(fmt.Sprintf("[%s] ", timestamp))
		b.WriteString(headerStyle.Render(agentName))

		// Add metrics if available
		if m.config.Logging.ShowMetrics && msg.Metrics != nil {
			metricsStr := fmt.Sprintf(" (%.1fs, %d tokens, $%.4f)",
				msg.Metrics.Duration.Seconds(),
				msg.Metrics.TotalTokens,
				msg.Metrics.Cost)
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(metricsStr))
		}
		b.WriteString("\n")

		// Message content
		wrappedContent := wrapText(msg.Content, textWidth)
		b.WriteString(wrappedContent)

		if i < len(messages)-1 {
			b.WriteString("\n\n")
		}
	}

	return b.String()
}
```

**Step 2: Run tests**

Run: `go test ./pkg/tui/... -v -count=1`
Expected: All tests pass

**Step 3: Commit**

```bash
git add pkg/tui/enhanced.go
git commit -m "feat(tui): add per-agent message rendering method"
```

---

## Task 7: Refactor View for Multi-Window Layout

**Files:**
- Modify: `pkg/tui/enhanced.go:899-1044` (View function)

**Step 1: Add renderTabIndicator helper**

Add before the View function:

```go
// renderTabIndicator renders the tab navigation bar
func (m *EnhancedModel) renderTabIndicator() string {
	if len(m.agentOrder) == 0 {
		return ""
	}

	currentAgent := "(none)"
	if m.selectedAgentIndex < len(m.agentOrder) {
		currentAgent = m.agentOrder[m.selectedAgentIndex]
	}

	color := m.agentColors[currentAgent]
	if color == "" {
		color = lipgloss.Color("99")
	}

	// Build indicator
	leftArrow := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render("◀ ")
	rightArrow := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(" ▶")
	agentName := lipgloss.NewStyle().Foreground(color).Bold(true).Render(currentAgent)

	// Position indicator
	position := fmt.Sprintf(" (%d/%d)", m.selectedAgentIndex+1, len(m.agentOrder))
	posStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	// Auto-follow indicator
	followIndicator := ""
	if m.autoFollow {
		followIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render(" [auto-follow]")
	}

	return leftArrow + agentName + rightArrow + posStyle.Render(position) + followIndicator
}
```

**Step 2: Update View function to use multi-window layout**

Replace the main layout section in View() (starting around line 910) with the new layout. The key changes:

1. Main window shows selected agent's messages (not interleaved conversation)
2. Right sidebar shows preview tiles for other agents
3. Tab indicator bar below main content

This is a significant refactor - replace the body of View() after the modal check:

```go
	// Calculate panel dimensions
	sidebarWidth := 35                        // Preview sidebar
	mainWidth := m.width - sidebarWidth - 10  // Main agent window

	// Render logo panel
	logoView := m.renderLogo()

	// Get selected agent name
	selectedAgent := ""
	if m.selectedAgentIndex < len(m.agentOrder) {
		selectedAgent = m.agentOrder[m.selectedAgentIndex]
	}

	// Render main agent window
	mainPanelStyle := activePanelStyle
	mainContent := ""
	if selectedAgent != "" {
		if vp, exists := m.agentViewports[selectedAgent]; exists {
			mainContent = vp.View()
		} else {
			mainContent = m.renderAgentMessages(selectedAgent)
		}
	} else {
		mainContent = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Render("No agent selected. Waiting for agents to initialize...")
	}

	// Calculate main window height
	mainHeight := m.height - 22 // Account for logo, tab bar, status, input

	mainWindow := mainPanelStyle.
		Width(mainWidth).
		Height(mainHeight).
		Render(mainContent)

	// Render preview sidebar
	previewSidebar := renderPreviewSidebar(m, sidebarWidth, mainHeight)

	// Render tab indicator
	tabIndicator := m.renderTabIndicator()
	tabBar := lipgloss.NewStyle().
		Width(m.width - 8).
		Align(lipgloss.Center).
		Padding(0, 1).
		Render(tabIndicator)

	// Render input panel
	inputPanelStyle := inactiveInputPanelStyle
	if m.activePanel == inputPanel {
		inputPanelStyle = activeInputPanelStyle
	}
	inputContent := m.userInput.View()
	if strings.TrimSpace(inputContent) == "" || inputContent == "> " {
		inputContent = "> \n"
	}
	inputView := inputPanelStyle.
		Width(m.width - 8).
		Height(2).
		Render(inputContent)

	// Render status bar with updated help
	statusBar := m.renderMultiWindowStatusBar()

	// Combine layout
	mainRow := lipgloss.JoinHorizontal(lipgloss.Top, mainWindow, previewSidebar)

	return lipgloss.NewStyle().
		MaxWidth(m.width - 6).
		MaxHeight(m.height - 1).
		PaddingLeft(1).
		Render(lipgloss.JoinVertical(lipgloss.Top,
			logoView,
			mainRow,
			tabBar,
			inputView,
			statusBar,
		))
```

**Step 3: Add renderMultiWindowStatusBar helper**

```go
func (m *EnhancedModel) renderMultiWindowStatusBar() string {
	help := []string{
		helpKeyStyle.Render("←→") + helpDescStyle.Render(" Switch agent"),
		helpKeyStyle.Render("↑↓") + helpDescStyle.Render(" Scroll"),
		helpKeyStyle.Render("f") + helpDescStyle.Render(" Auto-follow"),
		helpKeyStyle.Render("Tab") + helpDescStyle.Render(" Focus input"),
		helpKeyStyle.Render("Enter") + helpDescStyle.Render(" Send"),
		helpKeyStyle.Render("Q") + helpDescStyle.Render(" Quit"),
	}

	return statusBarStyle.
		Width(m.width).
		Render(strings.Join(help, " • "))
}
```

**Step 4: Run build to check for errors**

Run: `go build ./...`
Expected: Build succeeds

**Step 5: Run tests**

Run: `go test ./pkg/tui/... -v -count=1`
Expected: All tests pass (some may need updates)

**Step 6: Commit**

```bash
git add pkg/tui/enhanced.go
git commit -m "feat(tui): refactor View for multi-window layout with preview sidebar"
```

---

## Task 8: Initialize Per-Agent Viewports on Window Size

**Files:**
- Modify: `pkg/tui/enhanced.go` (WindowSizeMsg handler around line 635)

**Step 1: Update WindowSizeMsg handler**

In the `case tea.WindowSizeMsg:` block, after initializing the main viewport, add:

```go
		// Initialize per-agent viewports
		if m.agentViewports == nil {
			m.agentViewports = make(map[string]viewport.Model)
		}
		for _, agentName := range m.agentOrder {
			if _, exists := m.agentViewports[agentName]; !exists {
				vp := viewport.New(mainWidth-2, mainHeight-2)
				vp.SetContent(m.renderAgentMessages(agentName))
				m.agentViewports[agentName] = vp
			} else {
				vp := m.agentViewports[agentName]
				vp.Width = mainWidth - 2
				vp.Height = mainHeight - 2
				m.agentViewports[agentName] = vp
			}
		}
```

**Step 2: Run tests**

Run: `go test ./pkg/tui/... -v -count=1`
Expected: All tests pass

**Step 3: Commit**

```bash
git add pkg/tui/enhanced.go
git commit -m "feat(tui): initialize per-agent viewports on window resize"
```

---

## Task 9: Update Agent Initialization to Set Order

**Files:**
- Modify: `pkg/tui/enhanced.go` (agentInitComplete handler around line 699)

**Step 1: Update agentInitComplete handler**

In the `case agentInitComplete:` block, after setting up agent colors, add:

```go
		// Set up agent order for multi-window navigation
		m.agentOrder = make([]string, len(m.agents))
		for i, a := range m.agents {
			m.agentOrder[i] = a.GetName()
		}

		// Initialize per-agent message buffers
		if m.agentMessages == nil {
			m.agentMessages = make(map[string][]agent.Message)
		}
		for _, name := range m.agentOrder {
			if _, exists := m.agentMessages[name]; !exists {
				m.agentMessages[name] = make([]agent.Message, 0)
			}
		}
```

**Step 2: Run tests**

Run: `go test ./pkg/tui/... -v -count=1`
Expected: All tests pass

**Step 3: Commit**

```bash
git add pkg/tui/enhanced.go
git commit -m "feat(tui): set up agent order and buffers on initialization"
```

---

## Task 10: Update Test Helper and Fix Broken Tests

**Files:**
- Modify: `pkg/tui/enhanced_test.go`

**Step 1: Update createTestEnhancedModel helper**

Update the helper to initialize all new fields:

```go
func createTestEnhancedModel(cfg *config.Config, activePanel panel, showModal bool) EnhancedModel {
	// ... existing setup ...

	m := EnhancedModel{
		ctx:                context.Background(),
		config:             cfg,
		agentList:          agentList,
		userInput:          ta,
		ready:              true,
		activePanel:        activePanel,
		showModal:          showModal,
		agentColors:        make(map[string]lipgloss.Color),
		agentMessages:      make(map[string][]agent.Message),
		agentViewports:     make(map[string]viewport.Model),
		agentOrder:         make([]string, 0),
		selectedAgentIndex: 0,
		autoFollow:         true,
		previewLines:       3,
	}

	return m
}
```

**Step 2: Run all tests**

Run: `go test ./pkg/tui/... -v -count=1`
Expected: All tests pass

**Step 3: Run full test suite**

Run: `go test -race ./... 2>&1 | grep -E "(^ok|^FAIL|^---)" | head -30`
Expected: All packages pass (except the artifact folder issue)

**Step 4: Commit**

```bash
git add pkg/tui/enhanced_test.go
git commit -m "test(tui): update test helper for multi-window state"
```

---

## Task 11: Manual Testing and Polish

**Files:**
- Various minor fixes as needed

**Step 1: Build the binary**

Run: `go build -o agentpipe .`
Expected: Build succeeds

**Step 2: Run with a test config**

Run: `./agentpipe run -t -c examples/brainstorm.yaml` (or similar config)
Expected: TUI launches with new multi-window layout

**Step 3: Test navigation**

- Press `←` and `→` to switch between agents
- Press `f` to toggle auto-follow
- Verify preview tiles update
- Verify scrolling works in main window

**Step 4: Fix any visual issues found**

Make adjustments as needed for spacing, colors, etc.

**Step 5: Final commit**

```bash
git add -A
git commit -m "feat(tui): polish multi-window layout"
```

---

## Task 12: Run Linting and Final Verification

**Step 1: Run linter**

Run: `golangci-lint run --timeout=5m ./...`
Expected: No new linting errors

**Step 2: Run full test suite**

Run: `go test -race ./...`
Expected: All tests pass

**Step 3: Final commit if needed**

```bash
git add -A
git commit -m "chore: fix linting issues"
```

---

## Summary

| Task | Description | Estimated Complexity |
|------|-------------|---------------------|
| 1 | Add state fields | Low |
| 2 | Initialize state | Low |
| 3 | Create preview tiles | Medium |
| 4 | Tab navigation handlers | Low |
| 5 | Message routing | Medium |
| 6 | Per-agent rendering | Low |
| 7 | Refactor View | High |
| 8 | Viewport initialization | Low |
| 9 | Agent order setup | Low |
| 10 | Fix tests | Low |
| 11 | Manual testing | Medium |
| 12 | Linting | Low |

**Total: 12 tasks**
