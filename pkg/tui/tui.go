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

	"github.com/ASRagab/agentpipe/internal/branding"
	"github.com/ASRagab/agentpipe/internal/version"
	"github.com/ASRagab/agentpipe/pkg/core"
	"github.com/ASRagab/agentpipe/pkg/events"
	"github.com/ASRagab/agentpipe/pkg/manager"
	"github.com/ASRagab/agentpipe/pkg/tui/components"
	"github.com/ASRagab/agentpipe/pkg/tui/styles"
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
	manager  *manager.ConversationManager
	eventBus *events.Bus
	ctx      context.Context
	cancelFn context.CancelFunc

	// Components
	statusBar    components.StatusBarModel
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

	// Error details modal
	showErrorDetails bool

	// Event handling (pointers so they're shared across value copies)
	eventMu    *sync.Mutex
	eventQueue *[]core.Event
	lastRender time.Time

	// Error handling
	lastError string
}

// New creates a new TUI model.
func New(mgr *manager.ConversationManager, eventBus *events.Bus) Model {
	ctx, cancel := context.WithCancel(context.Background())

	agents := mgr.GetAgents()

	eventQueue := make([]core.Event, 0)
	m := Model{
		manager:      mgr,
		eventBus:     eventBus,
		ctx:          ctx,
		cancelFn:     cancel,
		statusBar:    components.NewStatusBarModel(),
		agentList:    components.NewAgentListModel(agents),
		conversation: components.NewConversationModel(),
		input:        components.NewInputModel(),
		focusedPanel: FocusInput, // Start with input focused
		eventMu:      &sync.Mutex{},
		eventQueue:   &eventQueue,
		lastRender:   time.Now(),
	}

	// Initialize status bar with conversation data
	m.statusBar.SetAgentCounts(len(agents), len(agents))

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
		m.startTypingAnimTicker(),
	)
}

// startCursorBlink starts the cursor blinking animation for streaming messages.
func (m Model) startCursorBlink() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return cursorBlinkMsg{}
	})
}

// startTypingAnimTicker starts the typing animation ticker (every 200ms for dots cycling).
func (m Model) startTypingAnimTicker() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return typingAnimTickMsg{}
	})
}

// subscribeToEvents sets up event bus subscriptions.
func (m Model) subscribeToEvents() tea.Cmd {
	// Capture pointers to shared state
	eventMu := m.eventMu
	eventQueue := m.eventQueue

	return func() tea.Msg {
		// Subscribe to all event types
		m.eventBus.SubscribeAll(func(event core.Event) {
			eventMu.Lock()
			*eventQueue = append(*eventQueue, event)
			eventMu.Unlock()
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

// typingAnimTickMsg is sent to advance the typing animation.
type typingAnimTickMsg struct{}

// eventMsg wraps an event for the Update loop.
type eventMsg struct {
	event core.Event
}

type userMessageResultMsg struct {
	err error
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
		m.statusBar.SetWidth(m.layout.StatusBarWidth)

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

		// Handle error details modal
		if m.showErrorDetails {
			switch msg.String() {
			case "esc", "q":
				m.showErrorDetails = false
				return m, nil
			case "r":
				// Retry the failed agent
				cmds = append(cmds, m.handleRetrySelectedAgent())
				m.showErrorDetails = false
				return m, tea.Batch(cmds...)
			}
			return m, nil
		}

		// Delegate to focused component
		switch m.focusedPanel {
		case FocusAgentList:
			// Handle special agent list keys
			switch msg.String() {
			case "enter":
				// Show error details for selected agent if it has an error
				if m.agentList.HasSelectedAgentError() {
					m.showErrorDetails = true
					return m, nil
				}
			case "r":
				// Retry the selected agent if it has a recoverable error
				cmds = append(cmds, m.handleRetrySelectedAgent())
			}
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

	case userMessageResultMsg:
		if msg.err != nil {
			m.lastError = msg.err.Error()
		} else if m.lastError != "" {
			m.lastError = ""
		}

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

	case typingAnimTickMsg:
		// Advance typing animation frame for agent list (cycles ., .., ...)
		m.agentList.AdvanceAnimationFrame()
		// Continue ticking
		cmds = append(cmds, tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
			return typingAnimTickMsg{}
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
	case "ctrl+s":
		return func() tea.Msg {
			if _, err := m.manager.Save(); err != nil {
				return userMessageResultMsg{err: err}
			}
			return nil
		}
	case "ctrl+e":
		return func() tea.Msg {
			outputPath := fmt.Sprintf("conversation_%s.md", time.Now().Format("2006-01-02_15-04-05"))
			if err := m.manager.ExportToMarkdown(outputPath); err != nil {
				return userMessageResultMsg{err: err}
			}
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
		return userMessageResultMsg{err: err}
	}
}

// handleRetrySelectedAgent triggers a retry for the currently selected agent if it has a recoverable error.
func (m *Model) handleRetrySelectedAgent() tea.Cmd {
	agent := m.agentList.GetSelectedAgent()
	if agent == nil {
		return nil
	}

	errInfo, ok := m.agentList.GetError(agent.ID)
	if !ok {
		return nil
	}

	if !errInfo.Recoverable {
		m.lastError = "Error is not recoverable"
		return nil
	}

	// Check if we're still in countdown
	if m.agentList.GetRetryCountdown(agent.ID) > 0 {
		m.lastError = "Please wait for retry countdown to complete"
		return nil
	}

	// Clear the error and reset status
	m.agentList.ClearError(agent.ID)
	m.agentList.UpdateStatus(agent.ID, components.AgentStatusReady)

	// Reset status bar if it was showing error
	if m.statusBar.GetStatus() == core.ConversationStatusError {
		m.statusBar.SetStatus(core.ConversationStatusActive)
	}

	// Clear last error display
	m.lastError = ""

	return nil
}

// processEventQueue processes pending events.
func (m *Model) processEventQueue() tea.Cmd {
	m.eventMu.Lock()
	events := *m.eventQueue
	*m.eventQueue = make([]core.Event, 0)
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
			// Only add user and system messages via this event
			// Agent messages are added via EventAgentDone -> CompleteStreaming
			// to avoid duplicate messages in the conversation panel
			if msg.Role != core.RoleAgent {
				m.conversation.AddMessage(msg)
				// Update status bar with message count
				m.statusBar.IncrementMessageCount()
			}
			// Count user messages as turns
			if msg.Role == core.RoleUser {
				m.statusBar.SetTurnCount(m.statusBar.GetTurnCount() + 1)
			}
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
			// Increment message count for agent messages (skipped in EventMessageCreated to avoid duplicates)
			m.statusBar.IncrementMessageCount()
			if data.Message.Metrics != nil {
				m.agentList.UpdateMetrics(data.AgentID, components.AgentMetrics{
					Duration: data.Message.Metrics.Duration,
					Tokens:   data.Message.Metrics.TotalTokens,
					Cost:     data.Message.Metrics.Cost,
				})
				// Update status bar totals
				m.statusBar.UpdateFromMetrics(data.Message.Metrics)
			}
		}

	case core.EventAgentError:
		if data, ok := event.Data.(core.AgentErrorData); ok {
			// Update agent list status
			m.agentList.UpdateStatus(data.AgentID, components.AgentStatusError)

			// Build detailed error info
			errInfo := components.AgentErrorInfo{
				Error:       data.Error,
				Timestamp:   event.Timestamp,
				ErrorType:   data.ErrorType,
				Recoverable: data.Recoverable,
				RetryAfter:  data.RetryAfter,
				RetryHint:   data.RetryHint,
			}

			// If error type is empty, try to classify from error message
			if errInfo.ErrorType == "" {
				errType := core.ClassifyError(data.Error)
				errInfo.ErrorType = string(errType)
				// Set recoverability based on error type
				switch errType {
				case core.ErrorTypeTimeout, core.ErrorTypeRateLimit, core.ErrorTypeNetwork:
					errInfo.Recoverable = true
				}
			}

			// Set retry hint if not provided
			if errInfo.RetryHint == "" && errInfo.Recoverable {
				switch core.ErrorType(errInfo.ErrorType) {
				case core.ErrorTypeTimeout:
					errInfo.RetryHint = "Press 'r' to retry or wait for automatic retry"
				case core.ErrorTypeRateLimit:
					errInfo.RetryHint = "Rate limit exceeded. Wait a moment and try again"
				case core.ErrorTypeNetwork:
					errInfo.RetryHint = "Check your connection and press 'r' to retry"
				}
			}

			m.agentList.UpdateErrorWithDetails(data.AgentID, errInfo)

			// Add error message to conversation for inline display
			m.conversation.AddAgentError(data.AgentID, data.AgentName, data.Error)

			// Update last error for bottom bar display
			m.lastError = fmt.Sprintf("%s: %s", data.AgentName, data.Error)

			// Update status bar to show error status
			m.statusBar.SetStatus(core.ConversationStatusError)

			// Cancel any streaming messages from this agent
			// (in case the error occurred during streaming)
			for messageID := range m.conversation.GetStreamingMessagesForAgent(data.AgentID) {
				m.conversation.CancelStreaming(messageID)
			}
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

	if m.showErrorDetails {
		return m.renderErrorDetailsOverlay()
	}

	mainView := m.renderMainView()

	if m.showHelp {
		return m.renderHelpOverlayOnTop(mainView)
	}

	return mainView
}

func (m Model) renderMainView() string {
	headerHeight := LogoHeight + 2
	footerHeight := InputHeight + 2
	mainHeight := m.height - headerHeight - footerHeight
	if mainHeight < 10 {
		mainHeight = 10
	}

	var b strings.Builder

	logoLines := strings.Split(branding.ASCIILogo, "\n")
	for i, line := range logoLines {
		if line != "" {
			centeredLine := lipgloss.NewStyle().
				Width(m.width).
				Align(lipgloss.Center).
				Render(line)
			b.WriteString(centeredLine)
		}
		if i < len(logoLines)-1 {
			b.WriteString("\n")
		}
	}

	versionStr := version.GetShortVersion()
	if versionStr != "dev" && !strings.HasPrefix(versionStr, "v") {
		versionStr = "v" + versionStr
	}
	versionStyle := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Foreground(lipgloss.Color("246"))
	b.WriteString(versionStyle.Render(versionStr))
	b.WriteString("\n")

	b.WriteString(m.statusBar.View())
	b.WriteString("\n")

	agentListView := m.agentList.View()
	conversationView := m.conversation.View()

	mainPanels := lipgloss.JoinHorizontal(
		lipgloss.Top,
		agentListView,
		conversationView,
	)

	centeredPanels := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		MaxHeight(mainHeight).
		Render(mainPanels)

	b.WriteString(centeredPanels)
	b.WriteString("\n")

	inputView := m.input.View()
	centeredInput := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Render(inputView)
	b.WriteString(centeredInput)

	if m.lastError != "" {
		b.WriteString("\n")
		errorView := lipgloss.NewStyle().
			Width(m.width).
			Align(lipgloss.Center).
			Render(styles.ErrorStyle().Render("Error: " + m.lastError))
		b.WriteString(errorView)
	}

	return b.String()
}

// renderErrorDetailsOverlay renders the error details modal.
func (m Model) renderErrorDetailsOverlay() string {
	modalWidth := m.width * 2 / 3
	if modalWidth < 40 {
		modalWidth = 40
	}
	if modalWidth > 80 {
		modalWidth = 80
	}

	return m.agentList.RenderErrorDetailsModal(modalWidth)
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
		{"Ctrl+S", "Save conversation"},
		{"Ctrl+E", "Export conversation"},
		{"", ""},
		{"Agent List (when focused):", ""},
		{"↑/↓, k/j", "Navigate agent list"},
		{"Home/End", "Go to first/last agent"},
		{"Enter", "Show error details (if agent has error)"},
		{"r", "Retry failed agent"},
		{"", ""},
		{"Conversation (when focused):", ""},
		{"↑/↓, k/j", "Scroll up/down"},
		{"PgUp/PgDown", "Page up/down"},
		{"Home/End", "Go to top/bottom"},
		{"", ""},
		{"Input (when focused):", ""},
		{"Ctrl+Enter", "Send message"},
		{"Ctrl+S", "Save conversation"},
		{"Ctrl+E", "Export conversation"},
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

func (m Model) renderHelpOverlayOnTop(background string) string {
	helpModal := m.renderHelpOverlay()

	helpLines := strings.Split(helpModal, "\n")
	modalHeight := len(helpLines)
	modalWidth := lipgloss.Width(helpModal)

	bgLines := strings.Split(background, "\n")

	for len(bgLines) < m.height {
		bgLines = append(bgLines, strings.Repeat(" ", m.width))
	}

	startRow := (m.height - modalHeight) / 2
	if startRow < 0 {
		startRow = 0
	}
	startCol := (m.width - modalWidth) / 2
	if startCol < 0 {
		startCol = 0
	}

	result := make([]string, len(bgLines))
	copy(result, bgLines)

	for i, helpLine := range helpLines {
		targetRow := startRow + i
		if targetRow >= len(result) {
			break
		}

		bgLine := result[targetRow]

		bgLineWidth := lipgloss.Width(bgLine)
		if bgLineWidth < m.width {
			bgLine = bgLine + strings.Repeat(" ", m.width-bgLineWidth)
		}

		prefix := ""
		if startCol > 0 {
			prefixRunes := []rune(bgLine)
			if len(prefixRunes) >= startCol {
				prefix = string(prefixRunes[:startCol])
			} else {
				prefix = bgLine + strings.Repeat(" ", startCol-len(prefixRunes))
			}
		}

		helpLineWidth := lipgloss.Width(helpLine)
		suffixStart := startCol + helpLineWidth
		suffix := ""
		bgRunes := []rune(bgLine)
		if suffixStart < len(bgRunes) {
			suffix = string(bgRunes[suffixStart:])
		}

		result[targetRow] = prefix + helpLine + suffix
	}

	return strings.Join(result, "\n")
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
