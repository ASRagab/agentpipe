package components

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kevinelliott/agentpipe/pkg/v2/core"
	"github.com/kevinelliott/agentpipe/pkg/v2/tui/styles"
)

// StreamingMessage represents a message currently being streamed.
type StreamingMessage struct {
	// MessageID uniquely identifies this streaming message.
	MessageID string
	// AgentID is the ID of the agent streaming this message.
	AgentID string
	// AgentName is the name of the agent streaming this message.
	AgentName string
	// Content is the accumulated content from all chunks.
	Content string
	// StartTime is when the streaming started.
	StartTime time.Time
	// LastChunkTime is when the last chunk was received.
	LastChunkTime time.Time
	// ChunkCount is the number of chunks received.
	ChunkCount int
}

// ErrorMessage represents an error to be displayed inline in the conversation.
type ErrorMessage struct {
	// ID is the unique identifier for this error message.
	ID string
	// Timestamp is when the error occurred.
	Timestamp time.Time
	// AgentID is the ID of the agent that encountered the error (if applicable).
	AgentID string
	// AgentName is the name of the agent that encountered the error (if applicable).
	AgentName string
	// ErrorType categorizes the error for styling and behavior.
	ErrorType core.ErrorType
	// Message is the human-readable error message.
	Message string
	// Recoverable indicates whether the error can be retried.
	Recoverable bool
	// RetryHint provides guidance on how to retry (if recoverable).
	RetryHint string
}

// ConversationModel manages the conversation view panel.
type ConversationModel struct {
	messages          []core.Message
	streamingMessages map[string]*StreamingMessage // keyed by MessageID
	errorMessages     []ErrorMessage               // Inline error messages
	viewport          viewport.Model
	width             int
	height            int
	focused           bool
	ready             bool
	agentIndex        map[string]int // Maps agent ID to color index
	cursorVisible     bool           // For blinking cursor animation
	autoScroll        bool           // Whether to auto-scroll on new messages
	userScrolledUp    bool           // Whether user has scrolled up from bottom
	hasNewMessages    bool           // Whether new messages arrived while scrolled up
	prevScrollOffset  int            // Previous scroll offset for detecting user scroll direction
}

// NewConversationModel creates a new conversation model.
func NewConversationModel() ConversationModel {
	return ConversationModel{
		messages:          make([]core.Message, 0),
		streamingMessages: make(map[string]*StreamingMessage),
		errorMessages:     make([]ErrorMessage, 0),
		agentIndex:        make(map[string]int),
		cursorVisible:     true,
		autoScroll:        true,
		userScrolledUp:    false,
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
				m.detectUserScroll()
			case "pgdown":
				m.viewport.ViewDown()
				m.detectUserScroll()
			case "up", "k":
				m.viewport.LineUp(1)
				m.detectUserScroll()
			case "down", "j":
				m.viewport.LineDown(1)
				m.detectUserScroll()
			case "home":
				m.viewport.GotoTop()
				m.userScrolledUp = true
			case "end":
				m.viewport.GotoBottom()
				m.userScrolledUp = false
				m.hasNewMessages = false
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
		// Track scroll position after viewport update
		m.detectUserScroll()
	}

	return m, cmd
}

// detectUserScroll detects if the user has scrolled away from the bottom.
func (m *ConversationModel) detectUserScroll() {
	if !m.ready {
		return
	}
	// If user is not at the bottom, mark as scrolled up
	if !m.viewport.AtBottom() {
		m.userScrolledUp = true
	} else {
		// User is at bottom, reset scroll state and new messages indicator
		m.userScrolledUp = false
		m.hasNewMessages = false
	}
	m.prevScrollOffset = m.viewport.YOffset
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

	// Build the main view
	viewContent := header + m.viewport.View()

	// Add "New messages below" indicator if user has scrolled up and new messages arrived
	if m.hasNewMessages && m.userScrolledUp {
		indicator := m.renderNewMessagesIndicator()
		// Overlay the indicator at the bottom of the viewport
		viewContent = m.overlayIndicator(viewContent, indicator)
	}

	return borderStyle.
		Width(m.width - 2).
		Height(m.height - 2).
		Render(viewContent)
}

// renderNewMessagesIndicator renders the "New messages below" indicator.
func (m ConversationModel) renderNewMessagesIndicator() string {
	return styles.NewMessagesIndicatorStyle().Render("↓ New messages below (End to jump)")
}

// overlayIndicator overlays the indicator at the bottom of the content.
func (m ConversationModel) overlayIndicator(content, indicator string) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return content
	}

	// Calculate position for the indicator (near the bottom of visible area)
	indicatorWidth := lipgloss.Width(indicator)
	contentWidth := m.width - 4 // Account for borders

	// Center the indicator
	padding := (contentWidth - indicatorWidth) / 2
	if padding < 0 {
		padding = 0
	}

	paddedIndicator := strings.Repeat(" ", padding) + indicator

	// Replace the last visible line with the indicator (overlay style)
	// For now, append below the viewport content
	return content + "\n" + paddedIndicator
}

// renderMessages renders all messages to a string including streaming messages.
func (m ConversationModel) renderMessages() string {
	// Count total messages including streaming
	if len(m.messages) == 0 && len(m.streamingMessages) == 0 {
		return styles.PlaceholderStyle().Render("No messages yet. Start typing to begin the conversation.")
	}

	var b strings.Builder

	// Render completed messages
	for _, msg := range m.messages {
		m.renderMessage(&b, msg)
		b.WriteString("\n")
	}

	// Render streaming messages (sorted by start time for consistent ordering)
	streamingSlice := m.getSortedStreamingMessages()
	for _, sm := range streamingSlice {
		m.renderStreamingMessage(&b, sm)
		b.WriteString("\n")
	}

	return b.String()
}

// getSortedStreamingMessages returns streaming messages sorted by start time.
func (m ConversationModel) getSortedStreamingMessages() []*StreamingMessage {
	result := make([]*StreamingMessage, 0, len(m.streamingMessages))
	for _, sm := range m.streamingMessages {
		result = append(result, sm)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].StartTime.Before(result[j].StartTime)
	})
	return result
}

// ProgressBarConfig contains configuration for the progress bar rendering.
type ProgressBarConfig struct {
	// Width is the total width of the progress bar in characters.
	Width int
	// EstimatedDuration is the estimated total duration for completion.
	// Used to calculate fill percentage. If zero, uses a default of 5 seconds.
	EstimatedDuration time.Duration
}

// DefaultProgressBarConfig returns the default progress bar configuration.
func DefaultProgressBarConfig() ProgressBarConfig {
	return ProgressBarConfig{
		Width:             20,
		EstimatedDuration: 5 * time.Second,
	}
}

// renderProgressBar renders a thin progress bar for streaming messages.
// The bar fills based on elapsed time vs estimated completion time.
// Color coding: Green (<1s), Yellow (1-3s), Red (>3s).
func (m ConversationModel) renderProgressBar(elapsed time.Duration, config ProgressBarConfig) string {
	// Use default if config values are zero
	if config.Width <= 0 {
		config.Width = 20
	}
	if config.EstimatedDuration <= 0 {
		config.EstimatedDuration = 5 * time.Second
	}

	// Calculate fill percentage (cap at 100%)
	fillPercent := float64(elapsed) / float64(config.EstimatedDuration)
	if fillPercent > 1.0 {
		fillPercent = 1.0
	}

	// Calculate fill and empty widths
	fillWidth := int(float64(config.Width) * fillPercent)
	emptyWidth := config.Width - fillWidth

	// Get elapsed seconds for color coding
	elapsedSecs := elapsed.Seconds()

	// Build the progress bar
	var b strings.Builder

	// Filled portion (using thin block character)
	if fillWidth > 0 {
		fillChars := strings.Repeat("━", fillWidth)
		b.WriteString(styles.ProgressBarFillStyle(elapsedSecs).Render(fillChars))
	}

	// Empty portion
	if emptyWidth > 0 {
		emptyChars := strings.Repeat("─", emptyWidth)
		b.WriteString(styles.ProgressBarEmptyStyle().Render(emptyChars))
	}

	// Append duration text
	durationStr := formatDuration(elapsed)
	b.WriteString(" ")
	b.WriteString(styles.ProgressBarDurationStyle(elapsedSecs).Render(durationStr))

	return styles.ProgressBarContainerStyle().Render(b.String())
}

// renderStreamingMessage renders a message that is currently being streamed.
func (m ConversationModel) renderStreamingMessage(b *strings.Builder, sm *StreamingMessage) {
	timestamp := sm.StartTime.Format("15:04:05")

	// Agent messages with color-coded name
	colorIndex := m.getAgentColorIndex(sm.AgentID)
	color := styles.AgentColorForIndex(colorIndex)

	header := fmt.Sprintf("[%s] %s:", timestamp, sm.AgentName)
	headerStyle := styles.AgentNameStyle(color)
	b.WriteString(headerStyle.Render(header))

	// Show streaming indicator with elapsed time
	elapsed := time.Since(sm.StartTime)
	var elapsedStr string
	if elapsed < time.Second {
		elapsedStr = fmt.Sprintf("%dms", elapsed.Milliseconds())
	} else {
		elapsedStr = fmt.Sprintf("%.1fs", elapsed.Seconds())
	}
	streamingIndicator := fmt.Sprintf(" [streaming... %s]", elapsedStr)
	b.WriteString(styles.MetricsStyle().Render(streamingIndicator))

	b.WriteString("\n")

	// Render the content with blinking cursor
	content := sm.Content
	if m.cursorVisible {
		content += "▌"
	}
	b.WriteString(styles.AgentMessageStyle(color).Render(content))

	b.WriteString("\n")

	// Render progress bar below the streaming message
	progressBar := m.renderProgressBar(elapsed, DefaultProgressBarConfig())
	b.WriteString(progressBar)
	b.WriteString("\n")
}

// renderMessage renders a single message.
func (m ConversationModel) renderMessage(b *strings.Builder, msg core.Message) {
	timestamp := msg.Timestamp.Format("15:04:05")

	// Check if this is an error message (has error status)
	if msg.Status == core.MessageStatusError {
		m.renderErrorStatusMessage(b, msg)
		return
	}

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

// renderErrorStatusMessage renders a message with error status inline with red styling.
func (m ConversationModel) renderErrorStatusMessage(b *strings.Builder, msg core.Message) {
	timestamp := msg.Timestamp.Format("15:04:05")

	// Error icon and header
	var header string
	if msg.AgentName != "" {
		header = fmt.Sprintf("[%s] %s %s failed to respond:", timestamp, styles.ErrorIconStyle().Render("✗"), msg.AgentName)
	} else {
		header = fmt.Sprintf("[%s] %s Error:", timestamp, styles.ErrorIconStyle().Render("✗"))
	}
	b.WriteString(styles.ErrorMessageHeaderStyle().Render(header))
	b.WriteString("\n")

	// Error content with red styling
	b.WriteString(styles.ErrorMessageStyle().Render(msg.Content))
	b.WriteString("\n")
}

// renderInlineError renders a detailed error message with type badge and retry hint.
func (m ConversationModel) renderInlineError(b *strings.Builder, errMsg ErrorMessage) {
	timestamp := errMsg.Timestamp.Format("15:04:05")

	// Error type badge
	typeBadge := m.formatErrorTypeBadge(errMsg.ErrorType)

	// Header with error icon, type badge, and agent name
	var header string
	if errMsg.AgentName != "" {
		header = fmt.Sprintf("[%s] %s %s %s:",
			timestamp,
			styles.ErrorIconStyle().Render("✗"),
			typeBadge,
			styles.ErrorAgentStyle().Render(errMsg.AgentName+" failed to respond"))
	} else {
		header = fmt.Sprintf("[%s] %s %s",
			timestamp,
			styles.ErrorIconStyle().Render("✗"),
			typeBadge)
	}
	b.WriteString(header)
	b.WriteString("\n")

	// Error message content
	b.WriteString(styles.ErrorMessageStyle().Render(errMsg.Message))
	b.WriteString("\n")

	// Retry hint for recoverable errors
	if errMsg.Recoverable && errMsg.RetryHint != "" {
		b.WriteString(styles.ErrorRetryHintStyle().Render("→ " + errMsg.RetryHint))
		b.WriteString("\n")
	}
}

// formatErrorTypeBadge formats an error type as a styled badge.
func (m ConversationModel) formatErrorTypeBadge(errType core.ErrorType) string {
	var label string
	switch errType {
	case core.ErrorTypeTimeout:
		label = "TIMEOUT"
	case core.ErrorTypeRateLimit:
		label = "RATE LIMIT"
	case core.ErrorTypeNetwork:
		label = "NETWORK"
	case core.ErrorTypeAuthentication:
		label = "AUTH"
	case core.ErrorTypeAPI:
		label = "API"
	case core.ErrorTypeInternal:
		label = "INTERNAL"
	default:
		label = "ERROR"
	}
	return styles.ErrorTypeStyle().Render(label)
}

// formatMetrics formats metrics for inline display.
// Format: [145ms | 234t | $0.012]
func (m ConversationModel) formatMetrics(metrics *core.Metrics) string {
	if metrics == nil {
		return ""
	}

	parts := make([]string, 0, 3)

	if metrics.Duration > 0 {
		parts = append(parts, formatDuration(metrics.Duration))
	}

	if metrics.TotalTokens > 0 {
		parts = append(parts, fmt.Sprintf("%dt", metrics.TotalTokens))
	}

	if metrics.Cost > 0 {
		parts = append(parts, formatCost(metrics.Cost))
	}

	if len(parts) == 0 {
		return ""
	}

	return "[" + strings.Join(parts, " | ") + "]"
}

// formatMetricsWithAgent formats metrics with agent name for inline display.
// Format: [Claude | 145ms | 234t | $0.012]
func (m ConversationModel) formatMetricsWithAgent(agentName string, metrics *core.Metrics) string {
	if metrics == nil {
		return ""
	}

	parts := make([]string, 0, 4)

	if agentName != "" {
		parts = append(parts, agentName)
	}

	if metrics.Duration > 0 {
		parts = append(parts, formatDuration(metrics.Duration))
	}

	if metrics.TotalTokens > 0 {
		parts = append(parts, fmt.Sprintf("%dt", metrics.TotalTokens))
	}

	if metrics.Cost > 0 {
		parts = append(parts, formatCost(metrics.Cost))
	}

	if len(parts) == 0 {
		return ""
	}

	return "[" + strings.Join(parts, " | ") + "]"
}

// formatMetricsExpanded formats metrics with full detail including input/output breakdown.
// Format: [Claude | 145ms | 100in/134out (234t) | $0.012]
func (m ConversationModel) formatMetricsExpanded(agentName string, metrics *core.Metrics) string {
	if metrics == nil {
		return ""
	}

	parts := make([]string, 0, 4)

	if agentName != "" {
		parts = append(parts, agentName)
	}

	if metrics.Duration > 0 {
		parts = append(parts, formatDuration(metrics.Duration))
	}

	// Show input/output breakdown if available
	if metrics.InputTokens > 0 || metrics.OutputTokens > 0 {
		tokenPart := fmt.Sprintf("%din/%dout", metrics.InputTokens, metrics.OutputTokens)
		if metrics.TotalTokens > 0 {
			tokenPart += fmt.Sprintf(" (%dt)", metrics.TotalTokens)
		}
		parts = append(parts, tokenPart)
	} else if metrics.TotalTokens > 0 {
		parts = append(parts, fmt.Sprintf("%dt", metrics.TotalTokens))
	}

	if metrics.Cost > 0 {
		parts = append(parts, formatCost(metrics.Cost))
	}

	if len(parts) == 0 {
		return ""
	}

	return "[" + strings.Join(parts, " | ") + "]"
}

// formatDuration formats a duration in human-readable format.
// Returns: "145ms" for < 1s, "2.3s" for < 60s, "1.5m" for >= 60s
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	} else if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}

// formatCost formats a cost value for display.
// Uses fewer decimal places for larger costs.
func formatCost(cost float64) string {
	if cost >= 1.0 {
		return fmt.Sprintf("$%.2f", cost)
	} else if cost >= 0.01 {
		return fmt.Sprintf("$%.3f", cost)
	}
	return fmt.Sprintf("$%.4f", cost)
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

// StartStreaming begins tracking a new streaming message.
func (m *ConversationModel) StartStreaming(messageID, agentID, agentName string) {
	now := time.Now()
	m.streamingMessages[messageID] = &StreamingMessage{
		MessageID:     messageID,
		AgentID:       agentID,
		AgentName:     agentName,
		Content:       "",
		StartTime:     now,
		LastChunkTime: now,
		ChunkCount:    0,
	}
	m.refreshContent()
}

// AppendChunk appends a chunk to a streaming message.
func (m *ConversationModel) AppendChunk(chunk core.MessageChunk) {
	sm, ok := m.streamingMessages[chunk.MessageID]
	if !ok {
		// Auto-start streaming if not already started
		m.StartStreaming(chunk.MessageID, chunk.AgentID, chunk.AgentName)
		sm = m.streamingMessages[chunk.MessageID]
	}

	sm.Content += chunk.Content
	sm.LastChunkTime = time.Now()
	sm.ChunkCount++

	m.refreshContent()
}

// CompleteStreaming finishes streaming and converts to a completed message.
func (m *ConversationModel) CompleteStreaming(messageID string, finalMsg core.Message) {
	delete(m.streamingMessages, messageID)
	m.messages = append(m.messages, finalMsg)
	m.refreshContent()
}

// CancelStreaming cancels a streaming message without completing it.
func (m *ConversationModel) CancelStreaming(messageID string) {
	delete(m.streamingMessages, messageID)
	m.refreshContent()
}

// ToggleCursor toggles the cursor visibility for animation.
func (m *ConversationModel) ToggleCursor() {
	m.cursorVisible = !m.cursorVisible
	if len(m.streamingMessages) > 0 {
		m.refreshContent()
	}
}

// HasStreamingMessages returns true if there are active streaming messages.
func (m *ConversationModel) HasStreamingMessages() bool {
	return len(m.streamingMessages) > 0
}

// GetStreamingMessageCount returns the number of active streaming messages.
func (m *ConversationModel) GetStreamingMessageCount() int {
	return len(m.streamingMessages)
}

// refreshContent updates the viewport content and handles auto-scrolling.
func (m *ConversationModel) refreshContent() {
	if !m.ready {
		return
	}

	// Check if we're at the bottom before updating
	atBottom := m.isAtBottom()

	content := m.renderMessages()
	m.viewport.SetContent(content)

	// Auto-scroll only if we were at the bottom and user hasn't scrolled up
	if atBottom && !m.userScrolledUp {
		m.viewport.GotoBottom()
	} else if m.userScrolledUp {
		// User is scrolled up, mark that new messages arrived
		m.hasNewMessages = true
	}
}

// isAtBottom checks if the viewport is scrolled to the bottom.
func (m *ConversationModel) isAtBottom() bool {
	if !m.ready {
		return true
	}
	// Consider "at bottom" if within 1 line of the actual bottom
	return m.viewport.AtBottom()
}

// SetUserScrolledUp marks that the user has scrolled up.
func (m *ConversationModel) SetUserScrolledUp(scrolled bool) {
	m.userScrolledUp = scrolled
}

// IsUserScrolledUp returns whether the user has scrolled up from bottom.
func (m *ConversationModel) IsUserScrolledUp() bool {
	return m.userScrolledUp
}

// TotalMessageCount returns the total count of completed and streaming messages.
func (m *ConversationModel) TotalMessageCount() int {
	return len(m.messages) + len(m.streamingMessages)
}

// HasNewMessages returns whether there are new messages while user is scrolled up.
func (m *ConversationModel) HasNewMessages() bool {
	return m.hasNewMessages
}

// ClearNewMessagesIndicator clears the new messages indicator.
func (m *ConversationModel) ClearNewMessagesIndicator() {
	m.hasNewMessages = false
}

// JumpToBottom scrolls to the bottom and clears indicators.
func (m *ConversationModel) JumpToBottom() {
	if m.ready {
		m.viewport.GotoBottom()
		m.userScrolledUp = false
		m.hasNewMessages = false
	}
}

// AddErrorMessage adds an inline error message to the conversation.
// This creates a system message with error status for display.
func (m *ConversationModel) AddErrorMessage(errInfo core.ErrorInfo) {
	// Create an error message using the core helper
	msg := core.NewErrorMessage(errInfo)
	m.messages = append(m.messages, msg)

	// Also add to errorMessages for detailed tracking
	errMsg := ErrorMessage{
		ID:          msg.ID,
		Timestamp:   errInfo.Timestamp,
		AgentID:     errInfo.AgentID,
		AgentName:   errInfo.AgentName,
		ErrorType:   errInfo.Type,
		Message:     errInfo.Message,
		Recoverable: errInfo.Recoverable,
		RetryHint:   errInfo.RetryHint,
	}
	m.errorMessages = append(m.errorMessages, errMsg)

	// Refresh content
	m.refreshContent()
}

// AddAgentError adds an error message for a specific agent failure.
// This is a convenience method for the common case of an agent failing to respond.
func (m *ConversationModel) AddAgentError(agentID, agentName, errorMsg string) {
	errType := core.ClassifyError(errorMsg)
	errInfo := core.NewAgentErrorInfo(errType, errorMsg, agentID, agentName)
	m.AddErrorMessage(errInfo)
}

// GetErrorMessages returns all error messages in the conversation.
func (m *ConversationModel) GetErrorMessages() []ErrorMessage {
	return m.errorMessages
}

// GetErrorMessageCount returns the number of error messages.
func (m *ConversationModel) GetErrorMessageCount() int {
	return len(m.errorMessages)
}

// HasErrors returns true if there are any error messages.
func (m *ConversationModel) HasErrors() bool {
	return len(m.errorMessages) > 0
}

// ClearErrors removes all error messages from the conversation.
func (m *ConversationModel) ClearErrors() {
	m.errorMessages = make([]ErrorMessage, 0)
	// Don't clear from messages array - those are part of history
}

// GetLastError returns the most recent error message, if any.
func (m *ConversationModel) GetLastError() *ErrorMessage {
	if len(m.errorMessages) == 0 {
		return nil
	}
	return &m.errorMessages[len(m.errorMessages)-1]
}

// GetAgentErrors returns all error messages for a specific agent.
func (m *ConversationModel) GetAgentErrors(agentID string) []ErrorMessage {
	var errors []ErrorMessage
	for _, err := range m.errorMessages {
		if err.AgentID == agentID {
			errors = append(errors, err)
		}
	}
	return errors
}

// GetStreamingMessagesForAgent returns a map of message IDs to streaming messages
// for a specific agent. Useful for cancelling streaming on error.
func (m *ConversationModel) GetStreamingMessagesForAgent(agentID string) map[string]*StreamingMessage {
	result := make(map[string]*StreamingMessage)
	for id, sm := range m.streamingMessages {
		if sm.AgentID == agentID {
			result[id] = sm
		}
	}
	return result
}
