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

// ConversationModel manages the conversation view panel.
type ConversationModel struct {
	messages          []core.Message
	streamingMessages map[string]*StreamingMessage // keyed by MessageID
	viewport          viewport.Model
	width             int
	height            int
	focused           bool
	ready             bool
	agentIndex        map[string]int // Maps agent ID to color index
	cursorVisible     bool           // For blinking cursor animation
	autoScroll        bool           // Whether to auto-scroll on new messages
	userScrolledUp    bool           // Whether user has scrolled up from bottom
}

// NewConversationModel creates a new conversation model.
func NewConversationModel() ConversationModel {
	return ConversationModel{
		messages:          make([]core.Message, 0),
		streamingMessages: make(map[string]*StreamingMessage),
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
