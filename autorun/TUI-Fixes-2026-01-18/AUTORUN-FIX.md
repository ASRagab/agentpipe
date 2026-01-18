# TUI Fixes Autorun - Implementation Guide

## Issue 1: Duplicate Messages in Conversation Panel

### Root Cause Analysis

The duplicate messages occur because:

1. `EventMessageCreated` is emitted in `SendUserMessage()` at line 407 of manager.go
2. The TUI's `handleEvent()` adds the message via `m.conversation.AddMessage(msg)` at line 393 of tui.go
3. For agent messages, `EventAgentDone` is also emitted and `CompleteStreaming()` at line 417 also adds the message to `m.messages`

The flow is:

- User message: `SendUserMessage()` -> `eventBus.Publish(NewMessageCreatedEvent)` -> TUI adds message
- Agent message: `pool.ExecuteParallel()` -> `EventAgentDone` -> `CompleteStreaming()` adds to messages
- BUT also: Manager calls `eventBus.Publish(NewMessageCreatedEvent(*resp.Message))` at line 478 -> TUI adds message AGAIN

### Fix Location

**File**: `pkg/tui/tui.go`
**Lines**: 391-400

### Fix Code

```go
// In handleEvent(), for EventMessageCreated, skip agent messages since they're handled by EventAgentDone
case core.EventMessageCreated:
    if msg, ok := event.Data.(core.Message); ok {
        // Only add user and system messages via this event
        // Agent messages are added via EventAgentDone -> CompleteStreaming
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
```

---

## Issue 2: Version Shows "vdev" Instead of Actual Version

### Root Cause Analysis

In `tui.go` line 561:

```go
versionStr := fmt.Sprintf("v%s", version.GetShortVersion())
```

`GetShortVersion()` returns `"dev"` by default (from version.go line 14).
Adding "v" prefix creates "vdev".

### Fix Location

**File**: `pkg/tui/tui.go`
**Line**: 561

### Fix Code

```go
// Don't add 'v' prefix if version already starts with 'v' or is 'dev'
versionStr := version.GetShortVersion()
if versionStr != "dev" && !strings.HasPrefix(versionStr, "v") {
    versionStr = "v" + versionStr
}
```

---

## Issue 3: Logo Cut Off at Top

### Root Cause Analysis

The logo has 6 lines but the ASCIILogo constant ends with `\n` on each line including the last.
When counting lines with `strings.Split(branding.ASCIILogo, "\n")`, we get 7 elements (6 lines + 1 empty).

The `LogoHeight = 6` constant in layout.go is correct, but the version line (1) is also added.
Total header: logo (6) + version (1) + status bar (1) = 8 lines minimum.

The issue is the `mainHeight` calculation at line 535:

```go
mainHeight := m.height - LogoHeight - 1 - StatusBarHeight - InputHeight - BorderPadding
```

This subtracts too much, causing panels to overflow and push the logo up.

### Fix Location

**File**: `pkg/tui/tui.go`
**Lines**: 533-535

### Fix Code

```go
// Calculate available height for main panels more accurately
// Total height minus: logo (6) + version line (1) + status bar (1) + input (3) + borders (4)
// Note: InputHeight=3 already, and we need 2 for panel borders top/bottom
headerHeight := LogoHeight + 2  // logo + version line + status bar
footerHeight := InputHeight + 2  // input + some padding
mainHeight := m.height - headerHeight - footerHeight
if mainHeight < 10 {
    mainHeight = 10  // Minimum height for panels
}
```

---

## Issue 4: Agent Panel Shows Duplicate Entries

### Root Cause Analysis

Looking at the screenshot, there appear to be TWO entries for "Assistant":

1. One with the model name below it
2. One without

This could be caused by:

1. The agent list being rendered twice
2. Multiple agents with the same name
3. The status bar also showing agent info

After reviewing `agent_list.go`, the View() function renders each agent once. The "duplication" in the screenshot appears to be:

- Line 1: Status indicator + Agent name
- Line 2: Model name (indented)

This is NOT a bug - it's the intended design. Each agent shows:

1. `● Assistant` (status + name)
2. `anthropic/claude-3.5-haiku` (model, indented)
3. Metrics or status (if applicable)

However, looking more closely at the screenshot, I see there's also an "Active" label at the top. This might be from the status bar bleeding into the agent panel area.

### Investigation Needed

Check if the issue is actually the status bars (there appear to be TWO status bars in the screenshot).

Looking at the screenshot again:

- Line 1: `● Active` with metrics at top (overall status bar)
- Line 2: Another `● Active` line (conversation-specific status)

This is the REAL duplication - two status bars!

### Fix Location

**File**: `pkg/tui/tui.go` - Check if status bar is rendered twice

### Fix Code

Review the View() function - the status bar should only be rendered once at line 570.

Actually, looking at the screenshot more carefully:

- Top status bar: `● Active | Turn: 1 | Msgs: 1 | Agents: 1/1 | 20s`
- Second status bar: `● Active | Turn: 1 | Msgs: 2 | Agents: 1/1 | 5062t | $0.0005 | 51s`

These are TWO different status bars with different values. One appears to be a "session" status and one is a "conversation" status.

The fix is to ensure only ONE status bar is rendered.

---

## Issue 5: Log Output Bleeding Through in TUI Mode

### Root Cause Analysis

The TUI runs with `tea.WithAltScreen()` (line 682), which should prevent log output.
However, zerolog is configured to write to os.Stderr, and some log messages are still appearing:

- `INF agent execution completed...`
- `INF conversation saved...`

These logs come from:

- `pkg/manager/manager.go` - uses `log.Info()` for agent completion and save events

### Fix Location

**File**: `cmd/run.go` - Need to suppress logging in TUI mode

### Fix Code

Add log suppression before TUI starts:

```go
// In executeConversation(), before running TUI:
if useTUI {
    // Suppress console logging in TUI mode to prevent output bleeding
    // The TUI handles its own display
    log.SetOutput(io.Discard)  // or redirect to a file
    return v2tui.RunWithContext(ctx, mgr, eventBus)
}
```

Alternatively, configure zerolog with a minimum level that suppresses INFO in TUI mode.

---

## Implementation Order

1. **Issue 1 (Duplicate messages)** - Highest priority, most visible bug
2. **Issue 5 (Log bleeding)** - High priority, breaks TUI visually
3. **Issue 2 (Version display)** - Medium priority, cosmetic
4. **Issue 3 (Logo cutoff)** - Medium priority, cosmetic
5. **Issue 4 (Status bar duplication)** - Needs more investigation

## Validation Commands

```bash
# Build
go build -o agentpipe .

# Run TUI to test
./agentpipe run -c examples/minimal.yaml

# Run tests
go test -v -race ./pkg/tui/...
go test -v -race ./pkg/core/...

# Lint
golangci-lint run --timeout=5m
```

## Success Criteria

- [x] Send "hello" - appears exactly once in conversation
  - **Fixed**: Modified `handleEvent()` in `pkg/tui/tui.go` to skip adding agent messages via `EventMessageCreated` since they are already added via `EventAgentDone -> CompleteStreaming()`. This prevents duplicate message display.
- [x] Agent responds - appears exactly once
  - **Fixed**: Same fix as above. Agent messages are now only added once via the `EventAgentDone` handler which calls `CompleteStreaming()`.
- [x] Version shows "dev" not "vdev"
  - **Fixed**: Modified line 568-572 in `pkg/tui/tui.go` to check if version is "dev" or already starts with "v" before prepending the "v" prefix. Now displays "dev" correctly instead of "vdev".
- [ ] Full logo visible (all 6 lines)
- [ ] No INF log messages visible in TUI
- [ ] Single status bar (not two)
- [ ] All tests pass
- [ ] No lint errors
