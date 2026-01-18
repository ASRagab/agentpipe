# Multi-Window TUI Design

**Date:** 2026-01-18
**Branch:** `feature/multi-window-tui`
**Status:** Approved

## Overview

Replace the current single conversation panel with a multi-window TUI where each agent gets its own dedicated window for real-time streaming output.

## Layout

```
┌────────────────────────────────────────────────────────────────────┐
│                           Logo / Header                            │
├─────────────────────────────────────────────┬──────────────────────┤
│                                             │   Preview Tiles      │
│                                             │  ┌──────────────┐   │
│           Main Agent Window                 │  │ Agent B      │   │
│     (Full output for selected agent)        │  │ last 2-3 lines│   │
│                                             │  └──────────────┘   │
│     - Scrollable viewport                   │  ┌──────────────┐   │
│     - Shows streaming output in real-time   │  │ Agent C      │   │
│     - Full history for this agent           │  │ ● responding │   │
│                                             │  └──────────────┘   │
│                                             │  ┌──────────────┐   │
│                                             │  │ Agent D      │   │
│                                             │  │ last 2-3 lines│   │
│                                             │  └──────────────┘   │
├─────────────────────────────────────────────┴──────────────────────┤
│  [◀ Agent A ▶]  Tab indicator + navigation                         │
├────────────────────────────────────────────────────────────────────┤
│  Status bar: Tab/←→ Switch | ↑↓ Scroll | f Toggle auto-follow     │
└────────────────────────────────────────────────────────────────────┘
```

**Key Elements:**
- **Main Window** (~70% width): Full viewport for currently selected agent's output
- **Preview Sidebar** (~30% width): Stacked tiles showing other agents' last 2-3 lines
- **Tab Indicator**: Shows current agent name with arrows indicating navigation
- **Auto-follow toggle**: `f` key to enable/disable following active agent

## State Management

### New State Structure

```go
type MultiWindowModel struct {
    // Per-agent message buffers (key = agent name)
    agentMessages map[string][]agent.Message

    // Navigation
    selectedAgentIndex int      // Which agent is in main view
    agentOrder         []string // Ordered list of agent names

    // Auto-follow
    autoFollow    bool   // Toggle for following active agent
    activeAgent   string // Currently responding agent

    // Viewports - one per agent
    agentViewports map[string]viewport.Model

    // Preview state
    previewLines int // How many lines to show in previews (default: 3)
}
```

### Message Routing

- When a message arrives, route it to `agentMessages[msg.AgentName]`
- Update the corresponding `agentViewports[agentName]`
- If `autoFollow` is enabled and agent starts responding, switch `selectedAgentIndex`
- Preview tiles re-render from tail of each agent's message buffer

### Key Difference from Current

- **Current:** Single `messages []agent.Message` array, single `conversation viewport.Model`
- **New:** Per-agent message storage and viewports, main view switches between them

## Preview Tiles

Each preview tile shows:

```
┌─────────────────────┐
│ Claude        ●     │  ← Active indicator (green when responding)
│ ...last line of     │
│ output continues    │
│ here with wrapping  │
└─────────────────────┘
```

- **Header**: Agent name + color badge + activity indicator (● green = responding, ○ grey = idle)
- **Content**: Last 2-3 lines of agent's output, word-wrapped to fit tile width
- **Border**: Highlighted if this agent is actively responding

## Navigation

| Key | Action |
|-----|--------|
| `←` / `h` | Previous agent tab |
| `→` / `l` | Next agent tab |
| `↑` / `k` | Scroll up in main window |
| `↓` / `j` | Scroll down in main window |
| `f` | Toggle auto-follow mode |
| `Tab` | Cycle focus (main view → input → main view) |
| `Enter` | Send message (when input focused) |

### Auto-Follow Logic

```
On message received:
  if msg.Role == "active" && autoFollow:
    selectedAgentIndex = indexOfAgent(msg.AgentName)
```

## Implementation Plan

### Files to Modify/Create

| File | Change |
|------|--------|
| `pkg/tui/enhanced.go` | Major refactor - replace `EnhancedModel` with `MultiWindowModel` |
| `pkg/tui/preview.go` | New file - preview tile rendering logic |
| `pkg/tui/enhanced_test.go` | Update tests for new model structure |

### Implementation Steps

1. **Create `MultiWindowModel` struct** with per-agent state
2. **Refactor `Init()`** to initialize viewports for each agent
3. **Refactor `Update()`** to:
   - Route messages to per-agent buffers
   - Handle left/right navigation for tab switching
   - Handle `f` key for auto-follow toggle
4. **Refactor `View()`** to:
   - Render main window from selected agent's viewport
   - Render preview tiles sidebar from other agents' buffers
   - Render tab indicator bar
5. **Update `messageWriter`** to emit agent-specific updates
6. **Remove unused code** (single conversation viewport, old layout logic)

### What Stays the Same

- Logo panel, status bar, user input panel
- Message channel architecture from orchestrator
- Agent initialization flow
- Styling system (colors, borders, etc.)

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Layout | Tabbed with previews | Balances focus on one agent with awareness of others |
| Preview content | Last 2-3 lines | Provides useful context without clutter |
| Navigation | Arrow keys + Enter | Terminal-native, simple |
| Auto-follow | Configurable toggle | Gives users control based on their workflow |
| Combined view | Removed entirely | Simplifies UI, reduces maintenance burden |
