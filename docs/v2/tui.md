---
type: reference
title: AgentPipe v2 TUI Guide
created: 2026-01-18
tags:
  - v2
  - tui
  - terminal
  - interface
related:
  - "[[quickstart]]"
  - "[[configuration]]"
---

# AgentPipe v2 TUI Guide

The Terminal User Interface (TUI) provides a rich, interactive experience for multi-agent conversations with real-time streaming, agent status indicators, and live metrics.

## Layout Overview

The TUI has four main areas:

```
┌────────────────────────────────────────────────────────────────────┐
│ [1] STATUS BAR                                                      │
├──────────────────┬─────────────────────────────────────────────────┤
│ [2] AGENT LIST   │ [3] CONVERSATION PANEL                          │
│                  │                                                  │
│ 🟢 Claude        │ User: What is async/await?                      │
│    2.3s | $0.02  │                                                  │
│                  │ Claude: Async/await is a syntax for...          │
│ 🟡 GPT-4...      │                                                  │
│    typing...     │ GPT-4: It's a pattern that...                   │
│                  │                                                  │
│ 🟢 Gemini        │                                                  │
│    1.1s | $0.01  │                                                  │
├──────────────────┴─────────────────────────────────────────────────┤
│ [4] INPUT PANEL                                                     │
│ > Type your message here...                                        │
└────────────────────────────────────────────────────────────────────┘
```

### 1. Status Bar

Displays conversation-level information:

| Element | Description |
|---------|-------------|
| Status | Current state: `Active`, `Waiting`, `Completed`, `Error` |
| Agents | Active/total agent count (e.g., `3/3`) |
| Messages | Total message count |
| Turns | Number of conversation turns |
| Tokens | Total tokens used |
| Cost | Estimated total cost |

### 2. Agent List (Left Panel)

Shows each agent's status and metrics:

**Status Indicators:**

- 🟢 **Ready** - Agent is idle, ready to respond
- 🟡 **Typing** - Agent is generating a response (animated dots)
- 🔴 **Error** - Agent encountered an error

**Metrics Display:**

- Response duration (e.g., `2.3s`)
- Cost for last response (e.g., `$0.02`)
- Token count when available

### 3. Conversation Panel (Right Panel)

Displays the conversation history:

- **User messages** - Your input
- **Agent messages** - AI responses with agent name badges
- **Streaming content** - Real-time character-by-character display with blinking cursor
- **Error messages** - Inline error display when agents fail

**Color Coding:**
Each agent gets a unique color for easy identification.

### 4. Input Panel (Bottom)

Where you type messages:

- Multi-line input supported
- `Ctrl+Enter` to send
- `Esc` to clear

## Keyboard Shortcuts

### Global Shortcuts

| Key | Action |
|-----|--------|
| `Ctrl+C` | Quit application |
| `q` | Quit (when not in input panel) |
| `Tab` | Cycle focus to next panel |
| `Shift+Tab` | Cycle focus to previous panel |
| `?` or `h` | Toggle help overlay |

### Agent List (when focused)

| Key | Action |
|-----|--------|
| `↑` / `k` | Select previous agent |
| `↓` / `j` | Select next agent |
| `Home` | Select first agent |
| `End` | Select last agent |
| `Enter` | Show error details (if agent has error) |
| `r` | Retry failed agent |

### Conversation Panel (when focused)

| Key | Action |
|-----|--------|
| `↑` / `k` | Scroll up |
| `↓` / `j` | Scroll down |
| `Page Up` | Page up |
| `Page Down` | Page down |
| `Home` | Go to top |
| `End` | Go to bottom |

### Input Panel (when focused)

| Key | Action |
|-----|--------|
| `Ctrl+Enter` | Send message |
| `Esc` | Clear input |
| Standard text editing | Arrow keys, backspace, etc. |

## Features

### Real-Time Streaming

Responses appear character-by-character as they're generated:

```
GPT-4: The key differences between REST and GraphQL are█
       (blinking cursor indicates streaming)
```

When streaming completes, the cursor disappears and metrics appear.

### Parallel Agent Responses

All agents respond simultaneously. You'll see:

1. Status indicators change to "typing" (🟡)
2. Responses stream in concurrently
3. Status returns to "ready" (🟢) as each agent finishes

### Error Handling

When an agent encounters an error:

1. Status changes to error (🔴)
2. Error message appears inline in conversation
3. Error details available via agent list

**Viewing Error Details:**

1. Focus the agent list (`Tab` to cycle)
2. Select the errored agent (`↑`/`↓`)
3. Press `Enter` to view details

**Retrying Failed Agents:**

1. Select the errored agent
2. Press `r` to retry

Recoverable errors (timeouts, rate limits, network issues) can be retried.

### Agent Metrics

Each agent displays its last response metrics:

```
🟢 Claude
   2.3s | 150 tok | $0.02
```

- **Duration**: Time to generate response
- **Tokens**: Total tokens used
- **Cost**: Estimated cost based on model pricing

### Help Overlay

Press `?` or `h` to show keyboard shortcuts:

```
┌─────────────────────────────────────┐
│ 📖 Keyboard Shortcuts              │
│                                     │
│ q, Ctrl+C    Quit application      │
│ Tab          Cycle panel focus     │
│ ?, h         Toggle help           │
│                                     │
│ Agent List:                        │
│ ↑/↓, k/j     Navigate agents       │
│ Enter        Show error details    │
│ r            Retry failed agent    │
│                                     │
│ Conversation:                      │
│ ↑/↓          Scroll                │
│ PgUp/PgDown  Page scroll           │
│                                     │
│ Input:                             │
│ Ctrl+Enter   Send message          │
│ Esc          Clear input           │
│                                     │
│ Press ?, h, or Esc to close        │
└─────────────────────────────────────┘
```

## Configuration

### Enabling/Disabling TUI

In your config file:

```yaml
tui:
  enabled: true      # Set to false for headless mode
  show_metrics: true # Show token/cost metrics
```

Or via CLI flags:

```bash
# Enable TUI (default)
agentpipe run --v2 -c config.yaml

# Disable TUI (headless mode)
agentpipe run --v2 -c config.yaml --no-tui
```

### Headless Mode

When TUI is disabled, AgentPipe runs in headless mode:

- Input via stdin
- Output via stdout
- Suitable for scripts and automation

```bash
echo "Explain async/await" | agentpipe run --v2 --no-tui -c config.yaml
```

## Requirements

### Terminal Size

Minimum terminal size: **80x24** characters

If your terminal is too small, you'll see:

```
Terminal too small. Please resize to at least 80x24.
Current size: 60x20
```

### Terminal Features

For best experience:

- **True color support** - For proper color coding
- **Unicode support** - For status indicators and borders
- **256-color minimum** - Falls back gracefully if true color unavailable

Recommended terminals:

- iTerm2 (macOS)
- Windows Terminal
- Kitty
- Alacritty
- GNOME Terminal
- Konsole

## Tips

### Efficient Navigation

1. **Start with input focused** - Ready to type immediately
2. **Use Tab** to quickly access agent list or conversation
3. **Use shortcuts** - `j`/`k` faster than arrow keys

### Reading Long Responses

1. Focus the conversation panel (`Tab`)
2. Use `Page Down` for fast scrolling
3. Use `End` to jump to latest

### Managing Errors

1. Check agent list for 🔴 indicators
2. Focus agent list and select errored agent
3. Press `Enter` for details, `r` to retry

### Multi-Agent Comparisons

When comparing agent responses:

1. Ask the same question to all agents
2. Use agent list to track which have responded
3. Scroll conversation to compare answers
4. Agent colors help distinguish responses

## Troubleshooting

### Display Issues

**Garbled output:**

- Ensure your terminal supports Unicode
- Try a different terminal emulator
- Check `TERM` environment variable

**Missing colors:**

- Verify terminal supports 256 colors or true color
- Try `export TERM=xterm-256color`

**Wrong layout:**

- Resize terminal to at least 80x24
- Restart AgentPipe after resizing

### Input Issues

**Can't type:**

- Make sure input panel is focused (use `Tab`)
- Check if help overlay is open (press `Esc`)

**Message not sending:**

- Use `Ctrl+Enter`, not just `Enter`
- Verify input panel is focused

### Performance Issues

**Slow updates:**

- TUI renders at 60fps, should be smooth
- Check if terminal is in compatibility mode
- Try a faster terminal emulator

**High CPU:**

- Normal during streaming (processing chunks)
- Should idle between messages
