# AgentPipe v2: Group Chat UX Design

## Overview

AgentPipe v2 reimagines multi-agent conversations as a modern group chat experience, combining the familiarity of Slack/Discord with the power of AI orchestration. Users interact naturally through @mentions, /commands, and threaded conversations while multiple AI agents collaborate in real-time.

## Design Philosophy

**Core Principles:**
1. **Familiar**: Feels like Slack, Discord, or iMessage - no learning curve
2. **Powerful**: Advanced orchestration hidden behind simple interactions
3. **Interactive**: User can interrupt, redirect, or clarify at any moment
4. **Transparent**: Agent status, thinking, and reasoning visible
5. **Accessible**: Keyboard-first with screen reader support

## TUI Layout

### Three-Panel Layout (Default)

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│ AgentPipe v2.0.0 │ brainstorm-session │ 3 agents • 12 messages • 2.4k tokens   │
├─────────────────┬───────────────────────────────────────────────┬───────────────┤
│                 │                                               │               │
│  AGENTS (15%)   │          CONVERSATION (60%)                   │  PANEL (25%)  │
│                 │                                               │               │
│ ◉ claude        │ ┌─────────────────────────────────────────┐   │ Thread View:  │
│   claude-3.5    │ │ 10:23 AM                           You │   │               │
│   Active        │ │ Hey team, let's brainstorm features   │   │ Main: 8 msgs  │
│                 │ │ for the new dashboard @claude         │   │ Side: 4 msgs  │
│ ◉ gemini        │ └─────────────────────────────────────────┘   │               │
│   gemini-2.0    │                                               │ ─────────────  │
│   Thinking...   │ ┌─────────────────────────────────────────┐   │ Agent Details │
│                 │ │ 10:23 AM                         claude │   │               │
│ ○ qwen          │ │ Great idea! I'd suggest focusing on:  │   │ claude        │
│   qwen-max      │ │ 1. Real-time analytics                │   │ Model: 3.5    │
│   Idle          │ │ 2. Customizable widgets               │   │ Tokens: 1.2k  │
│                 │ │ 3. Dark mode support                  │   │ Cost: $0.02   │
│ ─────────────── │ │                                       │   │ Uptime: 5m    │
│                 │ │ @gemini what do you think?            │   │               │
│ u - User Input  │ └─────────────────────────────────────────┘   │ ─────────────  │
│ ? - Help        │                                               │               │
│ / - Commands    │ ┌─────────────────────────────────────────┐   │ Quick Actions │
│ @ - Mention     │ │ 10:24 AM                         gemini │   │               │
│ Ctrl+Q - Quit   │ │ ⚡ Typing...                          │   │ [E] Export    │
│                 │ └─────────────────────────────────────────┘   │ [S] Save      │
│                 │                                               │ [F] Find      │
│                 │                                               │ [T] Thread    │
│                 │                                               │               │
├─────────────────┴───────────────────────────────────────────────┴───────────────┤
│ > @gemini Let's also add user preferences                    [Ctrl+Enter: Send] │
│                                                                     45 / 4000 ↩  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### Compact Layout (Optional)

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│ AgentPipe v2 │ claude ◉ gemini ◉ qwen ○ │ 12 msgs • 2.4k tokens         [Help] │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│  10:23 AM  You                                                                  │
│  Hey team, let's brainstorm features for the new dashboard @claude             │
│                                                                                 │
│  10:23 AM  claude [claude-3.5]                                                  │
│  Great idea! I'd suggest focusing on:                                           │
│  1. Real-time analytics                                                         │
│  2. Customizable widgets                                                        │
│  3. Dark mode support                                                           │
│                                                                                 │
│  @gemini what do you think?                                                     │
│                                                                                 │
│  10:24 AM  gemini [gemini-2.0] ⚡ Typing...                                     │
│                                                                                 │
│                                                                                 │
├─────────────────────────────────────────────────────────────────────────────────┤
│ > @gemini Let's also add user preferences                    [Ctrl+Enter: Send] │
│                                                                     45 / 4000 ↩  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

## Component Breakdown

### 1. Agent Roster Panel (Left, 15%)

**Purpose:** Display all participating agents with real-time status

**Components:**

```
┌─────────────────┐
│  AGENTS         │
├─────────────────┤
│ ◉ claude        │  ← Status indicator (◉ active, ◐ thinking, ○ idle, ✗ error)
│   claude-3.5    │  ← Model name
│   Active        │  ← Status text
│   ─────────     │
│   Msgs: 4       │  ← Quick stats
│   Tokens: 1.2k  │
│   Cost: $0.02   │
│                 │
│ ◉ gemini        │
│   gemini-2.0    │
│   Thinking...   │
│   ─────────     │
│   Msgs: 3       │
│   Tokens: 890   │
│   Cost: $0.01   │
│                 │
│ ○ qwen          │
│   qwen-max      │
│   Idle          │
│   ─────────     │
│   Msgs: 0       │
│   Tokens: 0     │
│   Cost: $0.00   │
└─────────────────┘
```

**Status Indicators:**
- `◉` Active - Currently responding or recently active
- `◐` Thinking - Processing/generating response
- `○` Idle - Available but not engaged
- `⚡` Typing - Streaming response in progress
- `✗` Error - Encountered an error
- `⏸` Paused - Temporarily disabled
- `🔇` Muted - Receiving but not responding

**Interaction:**
- Click/select agent to view details in right panel
- Right-click for context menu (mute, pause, inspect)
- Double-click to direct message

### 2. Conversation Panel (Center, 60%)

**Purpose:** Main conversation area with all messages

**Message Types:**

```
┌─────────────────────────────────────────────────────────────┐
│ USER MESSAGE                                                │
├─────────────────────────────────────────────────────────────┤
│ 10:23 AM                                             You    │
│ Hey team, let's brainstorm features @claude                 │
│                                              [↩ Reply] [⋯]  │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ AGENT MESSAGE                                               │
├─────────────────────────────────────────────────────────────┤
│ 10:23 AM                                         claude     │
│ Great idea! I'd suggest:                                    │
│ 1. Real-time analytics                                      │
│ 2. Customizable widgets                                     │
│                                                             │
│ @gemini what do you think?                                  │
│                                                             │
│ ⓘ 234 tokens • 1.2s • $0.004       [↩ Reply] [⊕] [👍] [⋯] │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ THREADED REPLY (indented)                                   │
├─────────────────────────────────────────────────────────────┤
│   │ 10:24 AM                                      gemini    │
│   │ I agree with analytics. Also add:                      │
│   │ - Export capabilities                                  │
│   │ - Role-based permissions                               │
│   │                                                         │
│   │ ⓘ 156 tokens • 0.8s • $0.002   [↩ Reply] [⊕] [👍] [⋯] │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ SYSTEM MESSAGE                                              │
├─────────────────────────────────────────────────────────────┤
│ 10:25 AM                                          ⓘ System  │
│ Conversation exported to: brainstorm-2025-01-18.md          │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ CODE BLOCK (syntax highlighted)                             │
├─────────────────────────────────────────────────────────────┤
│ 10:26 AM                                           claude   │
│ Here's a sample component:                                  │
│                                                             │
│ ```typescript                                               │
│ interface DashboardWidget {                                 │
│   id: string;                                               │
│   type: 'chart' | 'table' | 'metric';                       │
│   config: WidgetConfig;                                     │
│ }                                                           │
│ ```                                                         │
│                                                             │
│ ⓘ 189 tokens • 1.5s • $0.003       [📋 Copy] [↩] [⋯]      │
└─────────────────────────────────────────────────────────────┘
```

**Message Features:**
- Timestamps (relative or absolute)
- Sender name with badge
- Content with markdown rendering
- Metadata bar (tokens, duration, cost)
- Action buttons (reply, react, copy, more)
- Thread indicator (if part of thread)
- @mentions highlighted
- Code blocks with syntax highlighting
- Inline images/artifacts (future)

**Visual Indicators:**
- User messages: Right-aligned, blue accent
- Agent messages: Left-aligned, agent-color accent
- System messages: Centered, gray
- Threads: Indented with vertical line
- Reactions: Emoji below message
- Edits: "Edited" tag with timestamp

### 3. Context Panel (Right, 25%)

**Purpose:** Contextual information and quick actions

**Modes:**

#### Thread View
```
┌───────────────────┐
│ Thread View       │
├───────────────────┤
│ Main Conversation │
│ ├─ 8 messages     │
│ └─ Last: 10:26 AM │
│                   │
│ Side Threads      │
│ ├─ Analytics      │
│ │  └─ 3 messages  │
│ └─ Dark Mode      │
│    └─ 2 messages  │
│                   │
│ [View All]        │
└───────────────────┘
```

#### Agent Details
```
┌───────────────────┐
│ Agent: claude     │
├───────────────────┤
│ Model             │
│ claude-3.5-sonnet │
│                   │
│ Status            │
│ ◉ Active          │
│                   │
│ Statistics        │
│ Messages: 4       │
│ Tokens: 1,234     │
│ Cost: $0.024      │
│ Avg Time: 1.3s    │
│                   │
│ Last Active       │
│ 2 minutes ago     │
│                   │
│ Actions           │
│ [Mute]            │
│ [Pause]           │
│ [Inspect]         │
│ [Direct Message]  │
└───────────────────┘
```

#### Quick Actions
```
┌───────────────────┐
│ Quick Actions     │
├───────────────────┤
│ [E] Export Chat   │
│ [S] Save Session  │
│ [F] Find/Search   │
│ [T] New Thread    │
│ [C] Clear Screen  │
│ [P] Pause All     │
│ [R] Resume All    │
│ [L] Toggle Layout │
│                   │
│ Shortcuts         │
│ Ctrl+E - Export   │
│ Ctrl+F - Find     │
│ Ctrl+T - Thread   │
│ Ctrl+Q - Quit     │
└───────────────────┘
```

#### Search Results
```
┌───────────────────┐
│ Search: "widget"  │
├───────────────────┤
│ 3 results found   │
│                   │
│ 1. claude         │
│ "Customizable     │
│  widgets..."      │
│ 10:23 AM          │
│                   │
│ 2. gemini         │
│ "Widget config    │
│  should..."       │
│ 10:25 AM          │
│                   │
│ 3. You            │
│ "Widget API?"     │
│ 10:27 AM          │
│                   │
│ [Next] [Prev]     │
└───────────────────┘
```

### 4. Input Area (Bottom, 2 lines + metadata)

**Features:**

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ > @claude Can you refine the analytics feature?          [Ctrl+Enter: Send] │
│ @-mention | /-commands | Ctrl+M: Multiline | Esc: Cancel      67 / 4000 ↩  │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Multiline Mode (Ctrl+M):**
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ > @team I have a few thoughts:                                              │
│ 1. Analytics should be real-time                                            │
│ 2. Widgets need drag-and-drop                                               │
│ 3. Dark mode is essential                                                   │
│                                                                              │
│ What do you all think?                                                      │
│                                                                              │
│ [Ctrl+Enter: Send] [Esc: Cancel]                              124 / 4000 ↩  │
└─────────────────────────────────────────────────────────────────────────────┘
```

**@mention Autocomplete:**
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ > @cl█                                                                       │
│   ┌──────────────────┐                                                      │
│   │ @claude          │ ← Active agents                                      │
│   │ @claudehaiku     │                                                      │
│   ├──────────────────┤                                                      │
│   │ @team            │ ← Special mentions                                   │
│   │ @all             │                                                      │
│   │ @here            │                                                      │
│   └──────────────────┘                                                      │
│                                                              0 / 4000 ↩      │
└─────────────────────────────────────────────────────────────────────────────┘
```

**/command Autocomplete:**
```
┌─────────────────────────────────────────────────────────────────────────────┐
│ > /ex█                                                                       │
│   ┌───────────────────────────────────────────┐                             │
│   │ /export [format] - Export conversation    │                             │
│   │ /exit - Quit AgentPipe                    │                             │
│   └───────────────────────────────────────────┘                             │
│                                                              0 / 4000 ↩      │
└─────────────────────────────────────────────────────────────────────────────┘
```

## User Interaction Flows

### 1. Starting a Conversation

```
User Action                 System Response
───────────                 ───────────────
1. Launch: agentpipe        → Show TUI with agent roster
   run -t config.yaml         and empty conversation

2. Type message             → Show in input area with
                              character count

3. @mention agent           → Show autocomplete dropdown
   (optional)                 with available agents

4. Press Ctrl+Enter         → Send message
                            → Show "Typing..." indicator
                            → Stream agent response(s)
                            → Update token/cost metrics
```

### 2. Threading a Conversation

```
User Action                 System Response
───────────                 ───────────────
1. Hover over message       → Show action buttons
                              [↩ Reply] [⊕] [👍] [⋯]

2. Click [↩ Reply]          → Switch input to reply mode
   or press 'r'               > In reply to @claude...
                            → Show thread indicator

3. Type reply               → Context shows thread structure
                            → @mention auto-filled (optional)

4. Send                     → Message appears indented
                            → Thread count updates
                            → Other agents can join thread
```

### 3. Interrupting Agents

```
User Action                 System Response
───────────                 ───────────────
1. Agent is responding      → "Thinking..." or "Typing..."
                              indicator shows

2. Press Ctrl+I (interrupt) → Pause agent generation
   or start typing            → Show "Interrupted" badge
                            → Input area becomes active

3. Type clarification       → Send new message
   or correction            → Agent sees interruption context
                            → Resume with new direction

4. Agent responds           → References your interruption
                            → Continues with clarification
```

### 4. Searching Conversation

```
User Action                 System Response
───────────                 ───────────────
1. Press Ctrl+F or /find    → Switch right panel to search
                            → Focus search input

2. Type query               → Real-time results
                            → Highlight matches
                            → Show context snippets

3. Navigate results         → Jump to message in conversation
   (Up/Down arrows)         → Highlight search term
                            → Scroll to position

4. Press Esc                → Exit search
                            → Return to previous panel view
```

### 5. Managing Agents

```
User Action                 System Response
───────────────────────     ─────────────────────────
1. Click agent in roster    → Show agent details in right panel
   or press number key      → Display stats, status, actions

2. Click [Mute]             → Agent receives messages
                            → Agent doesn't respond
                            → Badge shows 🔇 Muted

3. Click [Pause]            → Agent stops all activity
                            → Badge shows ⏸ Paused
                            → Can resume later

4. Click [Direct Message]   → Create 1:1 conversation
                            → Other agents don't see
                            → Can return to group chat
```

### 6. Exporting Conversation

```
User Action                 System Response
───────────────────────     ─────────────────────────
1. Press Ctrl+E             → Show export modal
   or /export [format]      → List format options

2. Select format:           → Generate file:
   - Markdown               → .md with headers, metadata
   - JSON                   → .json with full structure
   - Text                   → .txt plain text
   - HTML                   → .html with styling

3. Choose scope:            → Filter messages:
   - Full conversation      → All messages
   - Current thread         → Thread + context
   - Date range             → Time-bounded
   - By agent               → Agent-specific

4. Confirm                  → Save to ~/.agentpipe/exports/
                            → Show success message with path
                            → Option to open in editor
```

## Message Rendering

### 1. Text Formatting (Markdown)

```
Input                       Rendered
─────                       ────────
**bold**                    bold (bright/bold terminal attribute)
*italic*                    italic (underline or color in terminal)
`code`                      code (monospace, gray background)
~~strikethrough~~           strikethrough (dim + strikethrough)

# Heading                   Heading (larger, bold)
## Subheading               Subheading (bold)

- List item                 • List item
1. Numbered                 1. Numbered

[Link](url)                 Link (blue, underlined)
@mention                    @mention (highlighted, bold, color)

> Quote                     │ Quote (indented, gray bar)
```

### 2. Code Blocks

```
Input:
```python
def hello():
    print("world")
```

Rendered:
┌───────────────────────────────────┐
│ python                            │
├───────────────────────────────────┤
│ def hello():                      │ ← Syntax highlighting
│     print("world")                │ ← Proper indentation
└───────────────────────────────────┘
   [📋 Copy Code]
```

### 3. Tables

```
Input:
| Feature  | Status |
|----------|--------|
| Analytics| Done   |
| Widgets  | TODO   |

Rendered:
┌──────────────┬────────────┐
│ Feature      │ Status     │
├──────────────┼────────────┤
│ Analytics    │ Done       │
│ Widgets      │ TODO       │
└──────────────┴────────────┘
```

### 4. Special Elements

**Reactions:**
```
┌─────────────────────────────────────────────┐
│ 10:23 AM                         claude     │
│ Great idea!                                 │
│                                             │
│ 👍 3   ❤️ 2   🎉 1                          │
└─────────────────────────────────────────────┘
```

**Edits:**
```
┌─────────────────────────────────────────────┐
│ 10:23 AM (Edited 10:24 AM)          You    │
│ Let's brainstorm dashboard features        │
│                                             │
│ [View Edit History]                         │
└─────────────────────────────────────────────┘
```

**Attachments/Artifacts:**
```
┌─────────────────────────────────────────────┐
│ 10:23 AM                         claude     │
│ I created a prototype:                      │
│                                             │
│ 📎 dashboard-wireframe.png (45 KB)         │
│ [View] [Download] [Open]                    │
└─────────────────────────────────────────────┘
```

## Keyboard Shortcuts

### Global Shortcuts

| Key              | Action                          |
|------------------|---------------------------------|
| `Ctrl+C`         | Copy selected text              |
| `Ctrl+Q`         | Quit AgentPipe                  |
| `Ctrl+L`         | Toggle layout (compact/default) |
| `Ctrl+H`         | Show help modal                 |
| `Esc`            | Cancel current action           |
| `?`              | Quick help overlay              |

### Navigation

| Key              | Action                          |
|------------------|---------------------------------|
| `↑` / `↓`        | Scroll conversation             |
| `PgUp` / `PgDn`  | Page up/down                    |
| `Home` / `End`   | Jump to top/bottom              |
| `Tab`            | Cycle focus (roster → conv → panel → input) |
| `Shift+Tab`      | Reverse cycle focus             |
| `1-9`            | Select agent by number          |

### Input

| Key              | Action                          |
|------------------|---------------------------------|
| `Ctrl+Enter`     | Send message                    |
| `Ctrl+M`         | Toggle multiline mode           |
| `@`              | Trigger mention autocomplete    |
| `/`              | Trigger command autocomplete    |
| `Esc`            | Cancel input/clear              |
| `Ctrl+U`         | Clear input line                |
| `Ctrl+K`         | Delete from cursor to end       |

### Messages

| Key              | Action                          |
|------------------|---------------------------------|
| `r`              | Reply to selected message       |
| `e`              | Edit your message (if recent)   |
| `d`              | Delete your message (if recent) |
| `c`              | Copy message content            |
| `t`              | Create thread from message      |
| `+`              | React to message                |
| `Enter`          | Select message                  |

### Actions

| Key              | Action                          |
|------------------|---------------------------------|
| `Ctrl+E`         | Export conversation             |
| `Ctrl+S`         | Save session                    |
| `Ctrl+F`         | Find/search                     |
| `Ctrl+T`         | New thread                      |
| `Ctrl+I`         | Interrupt agents                |
| `Ctrl+P`         | Pause all agents                |
| `Ctrl+R`         | Resume all agents               |

### Agent Control

| Key              | Action                          |
|------------------|---------------------------------|
| `m`              | Mute/unmute selected agent      |
| `p`              | Pause/resume selected agent     |
| `i`              | Inspect agent details           |
| `@agent+d`       | Direct message to agent         |

## Command Reference

### Chat Commands (type in input)

| Command          | Description                     | Example                    |
|------------------|---------------------------------|----------------------------|
| `/help`          | Show help modal                 | `/help`                    |
| `/clear`         | Clear conversation screen       | `/clear`                   |
| `/export`        | Export conversation             | `/export markdown`         |
| `/save`          | Save session                    | `/save brainstorm-v2`      |
| `/load`          | Load saved session              | `/load brainstorm-v2`      |
| `/agents`        | List all agents                 | `/agents`                  |
| `/mute`          | Mute agent(s)                   | `/mute @claude`            |
| `/unmute`        | Unmute agent(s)                 | `/unmute @claude`          |
| `/pause`         | Pause agent(s)                  | `/pause @gemini`           |
| `/resume`        | Resume agent(s)                 | `/resume @gemini`          |
| `/status`        | Show agent status               | `/status`                  |
| `/thread`        | Create new thread               | `/thread Analytics feature`|
| `/join`          | Join thread                     | `/join #analytics`         |
| `/dm`            | Direct message to agent         | `/dm @claude Let's chat`   |
| `/search`        | Search conversation             | `/search widgets`          |
| `/stats`         | Show conversation stats         | `/stats`                   |
| `/mode`          | Change orchestrator mode        | `/mode reactive`           |
| `/layout`        | Toggle layout                   | `/layout compact`          |
| `/quit`          | Quit AgentPipe                  | `/quit`                    |

### Special Mentions

| Mention          | Description                     |
|------------------|---------------------------------|
| `@agent`         | Direct message to specific agent|
| `@team`          | Message to all active agents    |
| `@all`           | Message to all agents (including paused) |
| `@here`          | Message to all active agents (alias for @team) |
| `@channel`       | Message to all in current thread|

## Real-Time Indicators

### Agent Status Badges

```
Agent Status        Badge       Description
────────────        ─────       ───────────
Active              ◉           Currently engaged, recently active
Thinking            ◐           Processing, generating response
Typing              ⚡          Streaming response in progress
Idle                ○           Available but not engaged
Error               ✗           Encountered an error
Paused              ⏸           Temporarily disabled by user
Muted               🔇          Receiving but not responding
Offline             ●           Not connected or unavailable
```

### Message Status

```
Status              Indicator   Description
──────              ─────────   ───────────
Sending             ↻           Message being sent
Sent                ✓           Message delivered
Read                ✓✓          Message read by agent(s)
Failed              ✗           Message failed to send
Edited              (Edited)    Message was edited
Deleted             [Deleted]   Message was deleted
```

### Typing Indicators

```
Single Agent:
┌─────────────────────────────────────────────┐
│ claude is typing...                     ⚡  │
└─────────────────────────────────────────────┘

Multiple Agents:
┌─────────────────────────────────────────────┐
│ claude, gemini are typing...            ⚡  │
└─────────────────────────────────────────────┘

Many Agents (3+):
┌─────────────────────────────────────────────┐
│ Several agents are typing...            ⚡  │
└─────────────────────────────────────────────┘
```

## Thread Visualization

### Thread Structure

```
Main Conversation
├─ Message 1 (You)
│  └─ Reply 1.1 (claude)
│     └─ Reply 1.1.1 (gemini)
├─ Message 2 (claude)
│  ├─ Reply 2.1 (You)
│  └─ Reply 2.2 (qwen)
└─ Message 3 (gemini)
```

### Visual Representation

```
┌───────────────────────────────────────────────────────────────┐
│ 10:23 AM                                               You    │
│ Let's brainstorm dashboard features @team                     │
│                                                               │
│ 3 replies                                    [View Thread]    │
└───────────────────────────────────────────────────────────────┘
  │
  ├─┬─────────────────────────────────────────────────────────┐
  │ │ 10:24 AM                                       claude   │
  │ │ I suggest real-time analytics                          │
  │ └─────────────────────────────────────────────────────────┘
  │
  ├─┬─────────────────────────────────────────────────────────┐
  │ │ 10:25 AM                                        gemini  │
  │ │ Also customizable widgets                              │
  │ └─────────────────────────────────────────────────────────┘
  │
  └─┬─────────────────────────────────────────────────────────┐
    │ 10:26 AM                                          qwen  │
    │ Dark mode is essential                                 │
    └─────────────────────────────────────────────────────────┘
```

### Thread Navigation

```
Current Thread: Analytics Feature (5 messages)
┌───────────────────────────────────────────────┐
│ ← Back to Main    Analytics Feature   [⋯]    │
├───────────────────────────────────────────────┤
│                                               │
│ Thread started by @You at 10:23 AM            │
│                                               │
│ [All 5 messages shown in conversation panel]  │
│                                               │
│ > Reply to thread...                          │
└───────────────────────────────────────────────┘
```

## Accessibility Considerations

### Screen Reader Support

1. **Semantic Structure**
   - Use ARIA labels for all interactive elements
   - Proper heading hierarchy (h1 → h2 → h3)
   - Landmark regions (navigation, main, complementary)

2. **Announcements**
   - New messages: "New message from [agent] at [time]"
   - Status changes: "Agent [name] is now [status]"
   - System events: "Conversation exported successfully"

3. **Navigation**
   - All features accessible via keyboard
   - Skip links to main content
   - Focus management (visible focus indicators)

### Visual Accessibility

1. **Color Contrast**
   - Minimum 4.5:1 for normal text
   - Minimum 3:1 for large text and UI components
   - Never rely on color alone (use icons + text)

2. **Text Sizing**
   - Respect terminal font size settings
   - Support for zooming (if terminal allows)
   - No minimum width requirements

3. **Motion**
   - Option to disable animations
   - Option to disable typing indicators
   - Option to disable auto-scrolling

### Cognitive Accessibility

1. **Clear Language**
   - Simple, direct command names
   - Consistent terminology
   - Help text for all commands

2. **Progressive Disclosure**
   - Don't overwhelm with all features at once
   - Contextual help (? key)
   - Guided tours for new users

3. **Error Prevention**
   - Confirm destructive actions
   - Auto-save drafts
   - Undo support where possible

## Configuration Options

### Layout Preferences

```yaml
# ~/.agentpipe/config.yaml
ui:
  layout:
    default: three-panel      # three-panel | compact | custom
    agent_panel_width: 15     # percentage
    context_panel_width: 25   # percentage
    conversation_width: 60    # percentage

  theme:
    color_scheme: auto        # auto | light | dark | custom
    agent_colors:             # per-agent color customization
      claude: blue
      gemini: green
      qwen: purple

  behavior:
    auto_scroll: true         # scroll to new messages
    typing_indicators: true   # show "typing..." indicators
    timestamps: relative      # relative | absolute | both
    animations: true          # enable/disable animations
```

### Keyboard Customization

```yaml
# ~/.agentpipe/config.yaml
keybindings:
  send_message: "Ctrl+Enter"
  interrupt: "Ctrl+I"
  search: "Ctrl+F"
  export: "Ctrl+E"
  quit: "Ctrl+Q"
  # ... all shortcuts customizable
```

### Message Display

```yaml
# ~/.agentpipe/config.yaml
messages:
  format:
    show_tokens: true         # show token counts
    show_cost: true           # show cost estimates
    show_duration: true       # show response time
    show_model: true          # show model name

  rendering:
    markdown: true            # render markdown
    syntax_highlight: true    # syntax highlighting in code blocks
    wrap_width: 80           # character wrap width (0 = no wrap)
    max_message_height: 20   # max lines before truncating
```

## Future Enhancements

### Phase 2 (v2.1)
- Reaction emojis (+1, heart, celebration)
- Message editing (with history)
- Direct messages (1:1 with agents)
- Rich presence (away, busy, do-not-disturb)
- Notification sounds (optional, configurable)

### Phase 3 (v2.2)
- Inline images/artifacts
- File attachments
- Screen sharing (ASCII art preview)
- Voice input (TTS → text → agents)
- Multi-language support

### Phase 4 (v2.3)
- Collaborative editing (agent + user co-authoring)
- Real-time collaboration (multiple users)
- Video/audio calls (agent TTS/voice)
- Integration with external tools (GitHub, Jira, etc.)
- Plugin system for custom agents

## Implementation Notes

### Technology Stack
- **TUI Framework**: Bubble Tea (Go)
- **Markdown Rendering**: Glamour (Go)
- **Syntax Highlighting**: Chroma (Go)
- **Keyboard Input**: Bubble Tea built-in
- **Layout**: Lipgloss (Go)

### Performance Considerations
1. **Lazy Loading**: Only render visible messages
2. **Virtual Scrolling**: Handle 1000+ messages efficiently
3. **Streaming**: Real-time message updates without flicker
4. **Caching**: Cache rendered messages for instant redraw
5. **Debouncing**: Debounce typing indicators (300ms)

### State Management
```go
type AppState struct {
    Conversation    *Conversation
    Agents          map[string]*Agent
    CurrentThread   *Thread
    SelectedAgent   *Agent
    InputMode       InputMode
    SearchQuery     string
    RightPanelMode  PanelMode
    // ...
}
```

## Summary

AgentPipe v2's group chat UX transforms multi-agent orchestration into a familiar, intuitive experience. By borrowing patterns from Slack, Discord, and iMessage, users can interact naturally with AI agents through @mentions, /commands, and threaded conversations—no learning curve required.

**Key Differentiators:**
- **Familiar**: Feels like existing chat tools
- **Powerful**: Advanced orchestration hidden behind simple interactions
- **Interactive**: User can interrupt and redirect at any moment
- **Transparent**: Agent status, thinking, and metrics visible
- **Accessible**: Keyboard-first with full screen reader support

**Design Principles:**
1. Progressive disclosure - complexity hidden until needed
2. Keyboard-first - every feature accessible without mouse
3. Real-time feedback - instant visual feedback for all actions
4. Context-aware - right panel adapts to current task
5. Flexible - customizable layout, colors, and shortcuts

This design creates a powerful yet approachable interface that makes multi-agent conversations feel as natural as chatting with colleagues in a group chat.
