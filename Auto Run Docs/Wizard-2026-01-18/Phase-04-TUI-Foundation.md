# Phase 04: TUI Foundation

This phase implements the foundational TUI (Terminal User Interface) using the bubbletea framework. The TUI provides a three-panel layout displaying the agent list, conversation messages, and user input area. This creates the visual foundation for the real-time multi-agent conversation experience.

## Tasks

- [x] Set up the bubbletea TUI framework in `pkg/v2/tui/tui.go`:
  - Model struct containing manager, eventBus, messages, agents, input, viewport fields
  - New() constructor that accepts ConversationManager and EventBus
  - Init() tea.Cmd returning nil for initial setup
  - Update() tea.Msg handler for key events and window resize
  - View() rendering three-panel layout as string
  - Run() method that creates and runs bubbletea.Program

- [x] Implement the agent list panel in `pkg/v2/tui/components/agent_list.go`:
  - AgentListModel struct with agents slice, statusMap, selectedIndex, width, height
  - Render agent name with status indicator emoji (🟢 ready, 🟡 typing, 🔴 error)
  - Show agent model name in smaller text below name
  - Show response metrics when available (duration, tokens)
  - Highlight selected agent
  - Handle up/down navigation to select agents
  - Method to update status for specific agent

- [x] Implement the conversation view panel in `pkg/v2/tui/components/conversation.go`:
  - ConversationModel struct with messages, viewport, width, height
  - Use bubbletea viewport component for scrolling
  - Render messages with role-based styling:
    - User messages: Right-aligned or prefixed with "You:"
    - Agent messages: Left-aligned with agent name and color
    - System messages: Centered, gray text
  - Show metrics inline for agent messages: [145ms | 234 tokens | $0.01]
  - Auto-scroll to bottom when new messages arrive
  - Handle Page Up/Down for manual scrolling

- [x] Implement the user input panel in `pkg/v2/tui/components/input.go`:
  - InputModel struct with textarea component, focused flag, width, height
  - Use bubbletea textarea for multi-line input
  - Handle Ctrl+Enter to submit message
  - Handle Escape to clear input
  - Show placeholder text when empty: "Type your message..."
  - Show character count in corner
  - Handle focus/blur for keyboard navigation

- [x] Implement color coding for agents in `pkg/v2/tui/styles.go`:
  - Define distinct lipgloss styles for each agent (up to 8 colors)
  - Colors: Blue, Green, Yellow, Magenta, Cyan, Orange, Pink, Purple
  - AgentColorForIndex(i int) returning consistent color
  - Style functions for message rendering:
    - UserMessageStyle
    - AgentMessageStyle(color)
    - SystemMessageStyle
    - MetricsStyle
    - ErrorStyle

- [x] Implement the layout manager in `pkg/v2/tui/layout.go`:
  - CalculateLayout(width, height) returning panel dimensions
  - Agent list: 20% of width (min 20 chars)
  - Conversation: 80% of width
  - Input area: 3 lines at bottom
  - Status bar: 1 line at top
  - Handle terminal resize by recalculating and re-rendering

- [x] Connect TUI to event bus for real-time updates:
  - In TUI Model Init(), subscribe to all event types
  - Create tea.Cmd functions that send custom messages to Update()
  - On EventMessageCreated: Add message to conversation view
  - On EventAgentTyping: Update agent status to "typing"
  - On EventAgentDone: Update agent status to "done", update metrics
  - On EventAgentError: Update agent status to "error"
  - Rate-limit renders to 60fps max using time.Ticker

- [x] Implement keyboard shortcuts in TUI:
  - q or Ctrl+C: Quit application
  - Ctrl+Enter: Send message (when input focused)
  - Tab: Cycle focus between panels
  - Up/Down: Navigate agent list (when agent list focused)
  - Page Up/Down: Scroll conversation (when conversation focused)
  - h or ?: Show help overlay
  - Create help overlay component showing all shortcuts

- [x] Create TUI demo command in `cmd/v2tui/main.go`:
  - Parse flags for config file
  - Load configuration and initialize agents
  - Create EventBus and ConversationManager
  - Create and run TUI
  - Handle graceful shutdown on quit

- [x] Test TUI renders correctly:
  - Manually run `go run cmd/v2tui/main.go --config examples/v2/demo-config.yaml`
  - Verify three-panel layout displays
  - Verify typing in input area works
  - Verify Enter submits message
  - Verify agent responses appear in conversation
  - Verify agent status indicators update
  - Verify scrolling works for long conversations
