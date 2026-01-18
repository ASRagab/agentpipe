// Package tui provides the terminal user interface for AgentPipe v2.
package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kevinelliott/agentpipe/pkg/v2/core"
	"github.com/kevinelliott/agentpipe/pkg/v2/events"
	"github.com/kevinelliott/agentpipe/pkg/v2/manager"
	"github.com/kevinelliott/agentpipe/pkg/v2/tui/components"
	"github.com/kevinelliott/agentpipe/pkg/v2/tui/styles"
)

// FocusedPanel represents which panel currently has focus.
type FocusedPanel int

const (
	FocusAgentList FocusedPanel = iota
	FocusConversation
	FocusInput
)

// Model is the main TUI model containing all components.
type Model struct {
	manager   *manager.ConversationManager
	eventBus  *events.Bus
	ctx       context.Context
	cancelFn  context.CancelFunc

	// Components
	agentList    components.AgentListModel
	conversation components.ConversationModel
	input        components.InputModel

	// Layout
	layout Layout
	width  int
	height int
	ready  bool

	// Focus management
	focusedPanel FocusedPanel

	// Help overlay
	showHelp bool

	// Event handling
	eventMu      sync.Mutex
	eventQueue   []core.Event
	lastRender   time.Time
	renderTicker *time.Ticker

	// Error handling
	lastError string
}

// New creates a new TUI model.
func New(mgr *manager.ConversationManager, eventBus *events.Bus) Model {
	ctx, cancel := context.WithCancel(context.Background())

	agents := mgr.GetAgents()

	m := Model{
		manager:      mgr,
		eventBus:     eventBus,
		ctx:          ctx,
		cancelFn:     cancel,
		agentList:    components.NewAgentListModel(agents),
		conversation: components.NewConversationModel(),
		input:        components.NewInputModel(),
		focusedPanel: FocusInput, // Start with input focused
		eventQueue:   make([]core.Event, 0),
		lastRender:   time.Now(),
	}

	// Set initial agent index for consistent colors
	agentIndex := make(map[string]int)
	for i, agent := range agents {
		agentIndex[agent.ID] = i
	}
	m.conversation.SetAgentIndex(agentIndex)

	return m
}

// Init initializes the TUI.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.input.Init(),
		m.subscribeToEvents(),
		m.startRenderTicker(),
		m.startCursorBlink(),
	)
}

// startCursorBlink starts the cursor blinking animation for streaming messages.
func (m Model) startCursorBlink() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return cursorBlinkMsg{}
	})
}

// subscribeToEvents sets up event bus subscriptions.
func (m Model) subscribeToEvents() tea.Cmd {
	return func() tea.Msg {
		// Subscribe to all event types
		m.eventBus.SubscribeAll(func(event core.Event) {
			m.eventMu.Lock()
			m.eventQueue = append(m.eventQueue, event)
			m.eventMu.Unlock()
		})
		return nil
	}
}

// startRenderTicker starts a ticker for rate-limited renders.
func (m Model) startRenderTicker() tea.Cmd {
	return tea.Tick(time.Second/60, func(t time.Time) tea.Msg {
		return tickMsg{time: t}
	})
}

// tickMsg is sent on each render tick.
type tickMsg struct {
	time time.Time
}

// cursorBlinkMsg is sent to toggle cursor visibility.
type cursorBlinkMsg struct{}

// eventMsg wraps an event for the Update loop.
type eventMsg struct {
	event core.Event
}

// Update handles all messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !IsMinimumSize(msg.Width, msg.Height) {
			// Terminal too small
			return m, nil
		}

		m.layout = CalculateLayout(msg.Width, msg.Height)
		m.updateComponentSizes()

		if !m.ready {
			m.conversation.InitViewport(m.layout.ConversationWidth, m.layout.ConversationHeight)
			m.ready = true
			m.updateFocus()
		}

	case tea.KeyMsg:
		// Handle global shortcuts first
		cmd := m.handleGlobalKeys(msg)
		if cmd != nil {
			return m, cmd
		}

		// Handle help overlay
		if m.showHelp {
			switch msg.String() {
			case "?", "esc", "q":
				m.showHelp = false
				return m, nil
			}
			return m, nil
		}

		// Delegate to focused component
		switch m.focusedPanel {
		case FocusAgentList:
			var cmd tea.Cmd
			m.agentList, cmd = m.agentList.Update(msg)
			cmds = append(cmds, cmd)
		case FocusConversation:
			var cmd tea.Cmd
			m.conversation, cmd = m.conversation.Update(msg)
			cmds = append(cmds, cmd)
		case FocusInput:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
		}

	case components.InputSubmittedMsg:
		// Handle user input submission
		cmds = append(cmds, m.handleInputSubmit(msg.Content))

	case tickMsg:
		// Process event queue
		cmds = append(cmds, m.processEventQueue())
		// Continue ticking
		cmds = append(cmds, tea.Tick(time.Second/60, func(t time.Time) tea.Msg {
			return tickMsg{time: t}
		}))

	case cursorBlinkMsg:
		// Toggle cursor visibility for streaming messages
		m.conversation.ToggleCursor()
		// Continue blinking
		cmds = append(cmds, tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
			return cursorBlinkMsg{}
		}))

	case eventMsg:
		m.handleEvent(msg.event)
	}

	// Update components
	if m.ready {
		var cmd tea.Cmd
		m.agentList, cmd = m.agentList.Update(msg)
		cmds = append(cmds, cmd)

		m.conversation, cmd = m.conversation.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleGlobalKeys handles global keyboard shortcuts.
func (m *Model) handleGlobalKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+c":
		m.cancelFn()
		return tea.Quit
	case "q":
		if m.focusedPanel != FocusInput && !m.showHelp {
			m.cancelFn()
			return tea.Quit
		}
	case "?", "h":
		if m.focusedPanel != FocusInput {
			m.showHelp = !m.showHelp
			return nil
		}
	case "tab":
		m.cycleFocus()
		return nil
	case "shift+tab":
		m.cycleFocusReverse()
		return nil
	}
	return nil
}

// handleInputSubmit processes user input submission.
func (m *Model) handleInputSubmit(content string) tea.Cmd {
	return func() tea.Msg {
		// Send user message through manager
		_, err := m.manager.SendUserMessage(m.ctx, content)
		if err != nil {
			m.lastError = err.Error()
		}
		return nil
	}
}

// processEventQueue processes pending events.
func (m *Model) processEventQueue() tea.Cmd {
	m.eventMu.Lock()
	events := m.eventQueue
	m.eventQueue = make([]core.Event, 0)
	m.eventMu.Unlock()

	for _, event := range events {
		m.handleEvent(event)
	}

	return nil
}

// handleEvent processes a single event.
func (m *Model) handleEvent(event core.Event) {
	switch event.Type {
	case core.EventMessageCreated:
		if msg, ok := event.Data.(core.Message); ok {
			m.conversation.AddMessage(msg)
		}

	case core.EventMessageChunk:
		if chunk, ok := event.Data.(core.MessageChunk); ok {
			// Append chunk to streaming message
			m.conversation.AppendChunk(chunk)
		}

	case core.EventAgentTyping:
		if data, ok := event.Data.(core.AgentTypingData); ok {
			m.agentList.UpdateStatus(data.AgentID, components.AgentStatusTyping)
		}

	case core.EventAgentDone:
		if data, ok := event.Data.(core.AgentDoneData); ok {
			m.agentList.UpdateStatus(data.AgentID, components.AgentStatusReady)
			// Complete streaming for this message if it was being streamed
			m.conversation.CompleteStreaming(data.Message.ID, data.Message)
			if data.Message.Metrics != nil {
				m.agentList.UpdateMetrics(data.AgentID, components.AgentMetrics{
					Duration: data.Message.Metrics.Duration,
					Tokens:   data.Message.Metrics.TotalTokens,
					Cost:     data.Message.Metrics.Cost,
				})
			}
		}

	case core.EventAgentError:
		if data, ok := event.Data.(core.AgentErrorData); ok {
			m.agentList.UpdateStatus(data.AgentID, components.AgentStatusError)
			m.lastError = fmt.Sprintf("%s: %s", data.AgentName, data.Error)
		}
	}
}

// cycleFocus moves focus to the next panel.
func (m *Model) cycleFocus() {
	m.focusedPanel = (m.focusedPanel + 1) % 3
	m.updateFocus()
}

// cycleFocusReverse moves focus to the previous panel.
func (m *Model) cycleFocusReverse() {
	m.focusedPanel = (m.focusedPanel + 2) % 3
	m.updateFocus()
}

// updateFocus updates component focus states.
func (m *Model) updateFocus() {
	m.agentList.SetFocused(m.focusedPanel == FocusAgentList)
	m.conversation.SetFocused(m.focusedPanel == FocusConversation)
	m.input.SetFocused(m.focusedPanel == FocusInput)
}

// updateComponentSizes updates all component sizes based on layout.
func (m *Model) updateComponentSizes() {
	m.agentList.SetSize(m.layout.AgentListWidth, m.layout.AgentListHeight)
	m.conversation.SetSize(m.layout.ConversationWidth, m.layout.ConversationHeight)
	m.input.SetSize(m.layout.InputWidth, m.layout.InputHeight)
}

// View renders the entire TUI.
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	if !IsMinimumSize(m.width, m.height) {
		return MinimumSizeMessage()
	}

	// Show help overlay if active
	if m.showHelp {
		return m.renderHelpOverlay()
	}

	var b strings.Builder

	// Status bar at top
	b.WriteString(m.renderStatusBar())
	b.WriteString("\n")

	// Main panels (agent list + conversation) side by side
	agentListView := m.agentList.View()
	conversationView := m.conversation.View()

	mainPanels := lipgloss.JoinHorizontal(
		lipgloss.Top,
		agentListView,
		conversationView,
	)
	b.WriteString(mainPanels)
	b.WriteString("\n")

	// Input panel at bottom
	b.WriteString(m.input.View())

	// Error display
	if m.lastError != "" {
		b.WriteString("\n")
		b.WriteString(styles.ErrorStyle().Render("Error: " + m.lastError))
	}

	return b.String()
}

// renderStatusBar renders the status bar at the top.
func (m Model) renderStatusBar() string {
	// Left side: title
	title := styles.TitleStyle().Render("🚀 AgentPipe v2")

	// Right side: stats
	agentCount := m.agentList.AgentCount()
	msgCount := m.conversation.MessageCount()
	stats := fmt.Sprintf("Agents: %d | Messages: %d", agentCount, msgCount)
	statsStyled := styles.StatusBarStyle().Render(stats)

	// Combine with spacing
	spacing := m.width - lipgloss.Width(title) - lipgloss.Width(statsStyled) - 2
	if spacing < 0 {
		spacing = 0
	}

	return title + strings.Repeat(" ", spacing) + statsStyled
}

// renderHelpOverlay renders the help overlay.
func (m Model) renderHelpOverlay() string {
	var b strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Render("📖 Keyboard Shortcuts")

	b.WriteString(title)
	b.WriteString("\n\n")

	shortcuts := []struct {
		key  string
		desc string
	}{
		{"q, Ctrl+C", "Quit application"},
		{"Tab", "Cycle focus between panels"},
		{"Shift+Tab", "Cycle focus backwards"},
		{"?, h", "Toggle this help screen"},
		{"", ""},
		{"Agent List (when focused):", ""},
		{"↑/↓, k/j", "Navigate agent list"},
		{"Home/End", "Go to first/last agent"},
		{"", ""},
		{"Conversation (when focused):", ""},
		{"↑/↓, k/j", "Scroll up/down"},
		{"PgUp/PgDown", "Page up/down"},
		{"Home/End", "Go to top/bottom"},
		{"", ""},
		{"Input (when focused):", ""},
		{"Ctrl+Enter", "Send message"},
		{"Esc", "Clear input"},
	}

	for _, s := range shortcuts {
		if s.key == "" && s.desc == "" {
			b.WriteString("\n")
			continue
		}
		if strings.HasSuffix(s.key, ":") {
			// Section header
			b.WriteString(lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("86")).
				Render(s.key))
			b.WriteString("\n")
			continue
		}
		keyStyled := styles.HelpKeyStyle().Width(16).Render(s.key)
		descStyled := styles.HelpDescStyle().Render(s.desc)
		b.WriteString(keyStyled + "  " + descStyled + "\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpStyle().Render("Press ?, h, or Esc to close"))

	return styles.HelpOverlayStyle().Render(b.String())
}

// Run starts the TUI program.
func Run(mgr *manager.ConversationManager, eventBus *events.Bus) error {
	m := New(mgr, eventBus)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// RunWithContext starts the TUI program with a context.
func RunWithContext(ctx context.Context, mgr *manager.ConversationManager, eventBus *events.Bus) error {
	m := New(mgr, eventBus)
	m.ctx = ctx
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Handle context cancellation
	go func() {
		<-ctx.Done()
		p.Quit()
	}()

	_, err := p.Run()
	return err
}
