---
type: reference
title: AgentPipe v2 Configuration Reference
created: 2026-01-18
tags:
  - v2
  - configuration
  - reference
related:
  - "[[quickstart]]"
  - "[[adapters]]"
  - "[[migration]]"
---

# AgentPipe v2 Configuration Reference

Complete reference for all configuration options in AgentPipe v2.

## Configuration File Location

Default locations (in order of priority):

1. Path specified with `-c` or `--config` flag
2. `./agentpipe.yaml` (current directory)
3. `~/.agentpipe/config.yaml`

## Configuration Sections

### Conversation

Controls conversation-level behavior.

```yaml
conversation:
  mode: parallel              # Conversation mode
  timeout: 30s               # Default per-agent timeout
  global_timeout: 2m         # Maximum time for all agents combined
  max_turns: 50              # Maximum conversation turns (0 = unlimited)
  preserve_partial_response: true  # Keep partial responses on timeout
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `mode` | string | `parallel` | Conversation mode. Currently only `parallel` is supported in v2 MVP. |
| `timeout` | duration | `30s` | Default timeout for individual agent responses. |
| `global_timeout` | duration | `0` | Maximum time for all agents to respond (0 = disabled). |
| `max_turns` | int | `0` | Maximum number of conversation turns (0 = unlimited). |
| `preserve_partial_response` | bool | `true` | Whether to keep partial responses when a timeout occurs. |

#### Duration Format

Durations use Go duration format:
- `30s` - 30 seconds
- `2m` - 2 minutes
- `1h30m` - 1 hour 30 minutes

### Agents

List of agents participating in the conversation.

```yaml
agents:
  - id: claude                    # Unique identifier (required)
    type: claude-api              # Adapter type (required)
    adapter: claude-api           # Explicit adapter name (optional, defaults to type)
    name: Claude                  # Display name (required)
    model: claude-sonnet-4-20250514     # Model identifier (required)
    timeout: 45s                  # Per-agent timeout override
    config:                       # Adapter-specific configuration
      api_key_env: ANTHROPIC_API_KEY
      system_prompt: "You are a helpful assistant."
      temperature: 0.7
      max_tokens: 2000
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `id` | string | Yes | - | Unique identifier for this agent. |
| `type` | string | Yes | - | Agent type: `openrouter`, `claude-api`, `claude`, `gemini`. |
| `adapter` | string | No | Same as `type` | Explicit adapter name. |
| `name` | string | Yes | - | Display name shown in TUI. |
| `model` | string | Yes | - | Model identifier for the AI provider. |
| `timeout` | duration | No | Inherits from conversation | Per-agent timeout override. |
| `config` | object | No | `{}` | Adapter-specific configuration. |

### Agent Config (Adapter-Specific)

Configuration passed to the adapter.

```yaml
config:
  api_key_env: OPENROUTER_API_KEY   # Environment variable for API key
  system_prompt: "You are helpful." # System prompt for the agent
  temperature: 0.7                   # Response randomness (0-1)
  max_tokens: 2000                   # Maximum response length
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `api_key_env` | string | Adapter-specific | Environment variable containing the API key. |
| `system_prompt` | string | `""` | System prompt prepended to conversations. |
| `temperature` | float | `0` (model default) | Controls randomness (0 = deterministic, 1 = creative). |
| `max_tokens` | int | Model default | Maximum tokens in the response. |

### TUI (Terminal UI)

Controls terminal interface settings.

```yaml
tui:
  enabled: true              # Enable TUI (vs CLI mode)
  theme: dark                # Color theme
  show_metrics: true         # Display token/cost metrics
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `true` | Enable the terminal UI. Set `false` for headless/script mode. |
| `theme` | string | `"dark"` | Color theme (currently only `dark` supported). |
| `show_metrics` | bool | `true` | Display token usage, cost, and timing metrics. |

### Logging

Controls logging behavior.

```yaml
logging:
  level: info               # Log level: debug, info, warn, error
  format: text              # Log format: text, json
  file: ""                  # Log file path (empty = stdout)
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `level` | string | `info` | Log verbosity: `debug`, `info`, `warn`, `error`. |
| `format` | string | `text` | Log format: `text` (human-readable) or `json` (structured). |
| `file` | string | `""` | Path to log file. Empty means log to stdout. |

### Persistence

Controls conversation save/load behavior.

```yaml
persistence:
  save_dir: ~/.agentpipe/v2/chats  # Directory for saved conversations
  auto_save: true                   # Auto-save on exit
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `save_dir` | string | `~/.agentpipe/v2/chats` | Directory for saved conversation files. |
| `auto_save` | bool | `false` | Automatically save conversation on exit. |

## Environment Variables

Environment variables override config file values:

| Variable | Description |
|----------|-------------|
| `AGENTPIPE_V2` | Set to `1` to use v2 engine by default |
| `OPENROUTER_API_KEY` | OpenRouter API key |
| `ANTHROPIC_API_KEY` | Anthropic Claude API key |
| `GOOGLE_API_KEY` | Google AI API key (for Gemini CLI) |

## Complete Example Configuration

```yaml
# AgentPipe v2 Configuration
# Full example with all options

conversation:
  mode: parallel
  timeout: 30s
  global_timeout: 2m
  max_turns: 100
  preserve_partial_response: true

agents:
  # Claude via Anthropic API
  - id: claude
    type: claude-api
    name: Claude
    model: claude-sonnet-4-20250514
    timeout: 45s
    config:
      api_key_env: ANTHROPIC_API_KEY
      system_prompt: |
        You are Claude, a helpful AI assistant created by Anthropic.
        Be concise but thorough in your responses.
      temperature: 0.7
      max_tokens: 2000

  # GPT-4 via OpenRouter
  - id: gpt4
    type: openrouter
    name: GPT-4
    model: openai/gpt-4-turbo
    config:
      api_key_env: OPENROUTER_API_KEY
      system_prompt: "You are a thoughtful assistant. Provide detailed explanations."
      temperature: 0.8
      max_tokens: 2000

  # Claude 3 Haiku via OpenRouter (faster, cheaper)
  - id: haiku
    type: openrouter
    name: Haiku
    model: anthropic/claude-3-haiku
    timeout: 15s
    config:
      api_key_env: OPENROUTER_API_KEY
      system_prompt: "Be quick and concise."
      max_tokens: 500

tui:
  enabled: true
  theme: dark
  show_metrics: true

logging:
  level: info
  format: text
  file: ""

persistence:
  save_dir: ~/.agentpipe/v2/chats
  auto_save: true
```

## v1 Configuration Compatibility

v2 automatically detects and migrates v1 configuration files. Key differences:

| v1 Field | v2 Field | Notes |
|----------|----------|-------|
| `orchestrator.mode` | `conversation.mode` | Only `parallel` supported in v2 MVP |
| `orchestrator.max_turns` | `conversation.max_turns` | Same behavior |
| `orchestrator.turn_timeout` | `conversation.timeout` | Renamed |
| Agent top-level fields | Nested under `config` | API keys, temperature, etc. |

See [[migration]] for detailed migration guide.

## CLI Flags

Flags override configuration file values:

```bash
agentpipe run --v2 \
  -c config.yaml \           # Config file path
  --timeout 45s \            # Override conversation timeout
  --no-tui \                 # Disable TUI (headless mode)
  --export output.json       # Export conversation after exit
```

| Flag | Description |
|------|-------------|
| `--v2` | Use v2 engine |
| `-c, --config` | Config file path |
| `--timeout` | Per-agent timeout |
| `--no-tui` | Run without TUI |
| `--export` | Export conversation on exit |
| `--resume` | Resume a saved conversation |
| `--migrate-config` | Migrate v1 config to v2 |

## Validation

AgentPipe validates configuration on startup:

```bash
# Validate configuration without running
agentpipe validate -c config.yaml
```

Common validation errors:

- **"at least one agent must be configured"** - Add at least one agent to the `agents` list
- **"agent X: id is required"** - Every agent needs a unique `id`
- **"agent X: unknown adapter"** - Use a valid adapter type: `openrouter`, `claude-api`, `claude`, `gemini`
- **"API key not found"** - Set the required environment variable

## Tips

### Multiple Configurations

Keep separate configs for different use cases:

```bash
# Development/debugging
agentpipe run --v2 -c ~/.agentpipe/dev.yaml

# Code review with experts
agentpipe run --v2 -c ~/.agentpipe/code-review.yaml

# Brainstorming with creative settings
agentpipe run --v2 -c ~/.agentpipe/brainstorm.yaml
```

### Per-Agent Timeouts

Use longer timeouts for larger models:

```yaml
agents:
  - id: fast
    type: openrouter
    model: anthropic/claude-3-haiku
    timeout: 15s    # Fast model, short timeout

  - id: thorough
    type: openrouter
    model: openai/gpt-4
    timeout: 60s    # Slower model, longer timeout
```

### Cost Management

Use `max_tokens` to control costs:

```yaml
config:
  max_tokens: 500   # Limit responses to ~500 tokens
```
