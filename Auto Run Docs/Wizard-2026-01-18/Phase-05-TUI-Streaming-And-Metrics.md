# Phase 05: TUI Streaming and Metrics Display

This phase enhances the TUI with real-time streaming response display and comprehensive metrics visualization. Users see agent responses appear character-by-character as they are generated, creating an engaging and responsive experience. Metrics provide transparency into token usage, cost, and performance.

## Tasks

- [x] Implement streaming message display in conversation view:
  - Track in-progress messages with partial content
  - On EventMessageChunk: Append chunk to in-progress message, re-render
  - Show cursor/blinking indicator at end of streaming message
  - On EventAgentDone: Mark message as complete, remove cursor
  - Handle multiple agents streaming simultaneously
  - Maintain message order based on start time, not completion time
  - **COMPLETED**: Added StreamingMessage struct, streaming message tracking, chunk appending, blinking cursor animation (500ms toggle), auto-scrolling with user scroll detection, and comprehensive tests (17 test cases covering all streaming scenarios)

- [x] Implement typing indicators in agent list:
  - Show animated typing indicator (... cycling) while agent is typing
  - Update indicator on each frame tick (every 200ms)
  - Show elapsed time since typing started
  - Clear indicator and show metrics when done
  - Show error icon with tooltip on agent error
  - **COMPLETED**: Added AgentErrorInfo struct, typingStartMap and errorMap for tracking, AdvanceAnimationFrame() for 200ms animation cycles (., .., ...), renderTypingIndicator() with elapsed time display (ms/s/m formats), error display with truncation, TypingIndicatorStyle() in styles, typingAnimTickMsg handling in main TUI loop, and comprehensive tests (12 test cases covering animation, tracking, status transitions, and error handling)

- [x] Implement metrics display in conversation view:
  - Format metrics as subtle inline badge: `[Claude | 145ms | 234t | $0.012]`
  - Show input tokens, output tokens, total tokens on expand
  - Calculate and show cost using model-specific pricing
  - Show duration in human-readable format (ms, s, or m)
  - Right-align metrics badge in message box
  - **COMPLETED**: Enhanced `formatMetrics()` with shorter 't' suffix for tokens, added `formatMetricsWithAgent()` for badge format `[Claude | 145ms | 234t | $0.012]`, added `formatMetricsExpanded()` for detailed view `[Claude | 145ms | 100in/134out (234t) | $0.012]`, implemented `formatDuration()` for human-readable format (ms <1s, s <60s, m >=60s), implemented `formatCost()` with adaptive decimal places, added `MetricsBadgeStyle()`, `MetricsBadgeExpandedStyle()`, `CostHighStyle()`, and `CostLowStyle()` to styles package, comprehensive tests (10 new test cases covering formatting, edge cases, nil handling, and partial data)

- [x] Create status bar component in `pkg/v2/tui/components/status_bar.go`:
  - Show at top of TUI
  - Left side: Conversation status (Active, Paused, Completed)
  - Center: Current turn number and total messages
  - Right side: Running totals (Total tokens, Total cost, Elapsed time)
  - Update in real-time as conversation progresses
  - Show agent count: "Connected: 3/3 agents"
  - **COMPLETED**: Created comprehensive StatusBarModel with:
    - Status section with color-coded indicators (● Active green, ◐ Paused yellow, ✓ Completed blue, ✗ Error red)
    - Center section showing Turn count, Message count, and Agent counts (e.g., "Turn: 5 │ Msgs: 12 │ Agents: 3/3")
    - Totals section with running token count, cost, and elapsed time (e.g., "500t │ $0.050 │ 1m30s")
    - Added 8 new styles: StatusBarContainerStyle, StatusBarCenterStyle, StatusBarTotalsStyle, StatusActiveStyle, StatusPausedStyle, StatusCompletedStyle, StatusErrorStyle
    - Integrated with main TUI model for real-time updates from conversation events
    - UpdateFromConversation() to sync from conversation state
    - UpdateFromMetrics() to accumulate metrics as messages complete
    - 13 comprehensive test cases covering all functionality

- [x] Implement smooth scrolling for new messages:
  - Track viewport scroll position
  - When new message arrives at bottom, auto-scroll
  - If user has scrolled up (not at bottom), don't auto-scroll
  - Show "New messages below" indicator when not at bottom
  - Press End key to jump to bottom
  - **COMPLETED**: Added `hasNewMessages` and `prevScrollOffset` fields to ConversationModel, implemented `detectUserScroll()` to track scroll position, added `renderNewMessagesIndicator()` and `overlayIndicator()` for visual feedback, enhanced keyboard handling to detect scroll actions and clear indicators on End key, added `JumpToBottom()`, `HasNewMessages()`, and `ClearNewMessagesIndicator()` helper methods, created `NewMessagesIndicatorStyle()` and `ScrollIndicatorStyle()` in styles package, comprehensive tests (12 new test cases covering scroll detection, indicator rendering, and state management)

- [x] Add response timing visualization:
  - Show thin progress bar below each streaming message
  - Bar fills based on estimated completion time
  - Color code: Green (<1s), Yellow (1-3s), Red (>3s)
  - Show exact duration when complete
  - **COMPLETED**: Added `ProgressBarConfig` struct with configurable width and estimated duration, implemented `renderProgressBar()` function with thin box-drawing characters (━ for filled, ─ for empty), color-coded styling via `ProgressBarFillStyle()` based on elapsed time (green <1s, yellow 1-3s, red >3s), integrated progress bar into `renderStreamingMessage()` to display below each streaming message, added 4 new styles (`ProgressBarContainerStyle`, `ProgressBarFillStyle`, `ProgressBarEmptyStyle`, `ProgressBarDurationStyle`), and comprehensive tests (13 new test cases covering zero elapsed, partial fill, full fill, color thresholds, custom config, zero/negative config handling, and integration with streaming messages)

- [x] Implement error display in TUI:
  - Show error messages inline in conversation with red styling
  - Include error type and actionable message
  - Show retry button/hint for recoverable errors
  - Agent list shows error status with details on select
  - System message for "Agent X failed to respond: [reason]"
  - **COMPLETED**: Created comprehensive error display system:
    - Added `pkg/v2/core/errors.go` with ErrorType enum (timeout, rate_limit, network, auth, api, internal, unknown), ErrorInfo struct, ClassifyError() for auto-detection, FormatErrorMessage(), NewErrorMessage()
    - Added 10 new styles to styles package: ErrorMessageStyle, ErrorMessageHeaderStyle, ErrorTypeStyle, ErrorRetryHintStyle, ErrorIconStyle, RecoverableErrorStyle, NonRecoverableErrorStyle, ErrorBorderStyle, ErrorAgentStyle
    - Added ErrorMessage struct to conversation model with ID, Timestamp, AgentID, AgentName, ErrorType, Message, Recoverable, RetryHint fields
    - Implemented renderErrorStatusMessage() for inline error display with ✗ icon and red styling
    - Implemented renderInlineError() with error type badge and retry hints for recoverable errors
    - Implemented formatErrorTypeBadge() for styled error type labels (TIMEOUT, RATE LIMIT, NETWORK, AUTH, API, INTERNAL, ERROR)
    - Added AddErrorMessage(), AddAgentError(), GetErrorMessages(), GetErrorMessageCount(), HasErrors(), ClearErrors(), GetLastError(), GetAgentErrors(), GetStreamingMessagesForAgent() methods
    - Updated TUI handleEvent to add inline error messages to conversation and cancel streaming on agent error
    - Added 17 comprehensive tests covering initialization, error adding, type classification, formatting, rendering, and streaming cancellation

- [ ] Add token/cost estimation for user input:
  - Estimate input tokens as user types (simple word count * 1.3)
  - Show estimated cost in input panel footer
  - Update estimate in real-time as user types
  - Clear when message sent

- [ ] Write tests for streaming display:
  - TestStreamingChunks: Mock chunks arrive, verify display updates
  - TestMultipleAgentsStreaming: Two agents stream simultaneously, verify both displayed
  - TestScrollBehavior: Auto-scroll when at bottom, no scroll when user scrolled up
  - TestMetricsDisplay: Verify metrics formatted correctly
  - TestStatusBarUpdate: Verify totals update after each message

- [ ] Manual testing checklist:
  - Send message to multiple agents
  - Verify responses stream in real-time
  - Verify typing indicators show during generation
  - Verify metrics appear after completion
  - Verify status bar totals are accurate
  - Test with fast (haiku) and slow (opus) models to see timing differences
