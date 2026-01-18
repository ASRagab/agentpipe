---
type: reference
title: AgentPipe v2 Quick Start Guide
created: 2026-01-18
tags:
  - v2
  - quickstart
  - getting-started
related:
  - "[[configuration]]"
  - "[[adapters]]"
  - "[[tui]]"
---

# AgentPipe v2 Quick Start Guide

Get started with AgentPipe v2 in 5 minutes. This guide walks you through installation, configuration, and running your first multi-agent conversation.

## Prerequisites

- **Go 1.24+** (if building from source)
- **API Key** for at least one AI provider:
  - OpenRouter: `OPENROUTER_API_KEY`
  - Anthropic Claude: `ANTHROPIC_API_KEY`
  - Google Gemini CLI installed

## Installation

### Option 1: Homebrew (macOS)

```bash
brew install kevinelliott/tap/agentpipe
```

### Option 2: Go Install

```bash
go install github.com/ASRagab/agentpipe@latest
```

### Option 3: Build from Source

```bash
git clone https://github.com/ASRagab/agentpipe.git
cd agentpipe
go build -o agentpipe .
```

## Quick Setup

### 1. Set Your API Key

Choose at least one provider:

```bash
# OpenRouter (access 400+ models)
export OPENROUTER_API_KEY="your-openrouter-key"

# Anthropic Claude API
export ANTHROPIC_API_KEY="your-anthropic-key"
```

### 2. Create a Configuration File

Create `~/.agentpipe/config.yaml`:

```yaml
# Minimal v2 Configuration
conversation:
  mode: parallel          # All agents respond simultaneously
  timeout: 30s           # Per-agent response timeout

agents:
  - id: claude
    type: claude-api
    name: Claude
    model: claude-sonnet-4-20250514
    config:
      api_key_env: ANTHROPIC_API_KEY
      system_prompt: "You are a helpful AI assistant."

  - id: gpt4
    type: openrouter
    name: GPT-4
    model: openai/gpt-4-turbo
    config:
      api_key_env: OPENROUTER_API_KEY
      system_prompt: "You are a thoughtful AI that provides detailed explanations."

tui:
  enabled: true
  show_metrics: true
```

### 3. Run AgentPipe

```bash
# Run with v2 engine (recommended)
agentpipe run --v2 -c ~/.agentpipe/config.yaml

# Or set as default
export AGENTPIPE_V2=1
agentpipe run -c ~/.agentpipe/config.yaml
```

## Your First Conversation

When AgentPipe starts, you'll see the TUI with three panels:

```
┌─────────────────────────────────────────────────────────────┐
│ Status: Active | Agents: 2/2 | Messages: 0 | Cost: $0.00   │
├───────────────┬─────────────────────────────────────────────┤
│ Agents        │ Conversation                                │
│               │                                             │
│ 🟢 Claude     │                                             │
│ 🟢 GPT-4      │                                             │
│               │                                             │
├───────────────┴─────────────────────────────────────────────┤
│ Type your message and press Ctrl+Enter to send             │
└─────────────────────────────────────────────────────────────┘
```

1. **Type a message** in the input panel at the bottom
2. **Press `Ctrl+Enter`** to send
3. **Watch responses stream in** from all agents simultaneously

### Example Prompts

Try these to see agents collaborate:

```
"What are the key differences between REST and GraphQL APIs?"

"Explain async/await in JavaScript with examples."

"Help me design a simple user authentication system."
```

## Key Features

### Parallel Execution

All agents respond simultaneously, making conversations 2-4x faster than sequential execution.

### Real-Time Streaming

Responses stream character-by-character as they're generated.

### Live Metrics

See token usage, cost, and response time for each agent in real-time.

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Tab` | Cycle focus between panels |
| `Ctrl+Enter` | Send message |
| `Ctrl+C` | Quit |
| `?` or `h` | Show help |
| `↑/↓` | Scroll conversation |

## Next Steps

- **[[configuration]]** - Full configuration reference
- **[[adapters]]** - Detailed adapter documentation
- **[[tui]]** - Complete TUI guide
- **[[migration]]** - Migrate from v1

## Troubleshooting

### "API key not found"

Ensure your API key environment variable is set:

```bash
echo $OPENROUTER_API_KEY   # Should show your key
echo $ANTHROPIC_API_KEY    # Should show your key
```

### "Connection refused" or timeout

Check your internet connection and verify the API endpoint is reachable.

### Terminal too small

AgentPipe requires a minimum terminal size of 80x24. Resize your terminal window.

## Getting Help

- **GitHub Issues**: [github.com/ASRagab/agentpipe/issues](https://github.com/ASRagab/agentpipe/issues)
- **Discussions**: [github.com/ASRagab/agentpipe/discussions](https://github.com/ASRagab/agentpipe/discussions)
