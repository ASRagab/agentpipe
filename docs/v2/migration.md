---
type: reference
title: AgentPipe v1 to v2 Migration Guide
created: 2026-01-18
tags:
  - v2
  - migration
  - upgrade
related:
  - "[[configuration]]"
  - "[[quickstart]]"
  - "[[adapters]]"
---

# AgentPipe v1 to v2 Migration Guide

This guide helps you migrate from AgentPipe v1 to v2. v2 is a ground-up rewrite focused on parallel execution and real-time streaming.

## Quick Migration

The fastest way to migrate:

```bash
# Migrate your v1 config automatically
agentpipe run --v2 --migrate-config -c ~/.agentpipe/config.yaml

# Or migrate without running
agentpipe migrate --from-v1 --config ~/.agentpipe/config.yaml
```

This creates a backup and migrates your configuration to v2 format.

## What Changed

### Philosophy

| v1 | v2 |
|----|----|
| Agents take turns speaking | All agents respond simultaneously |
| Wait for complete responses | Real-time streaming |
| CLI-focused adapters | API-first, CLI as fallback |
| 5+ orchestration modes | Parallel mode (MVP) |

### Performance

| Metric | v1 | v2 | Improvement |
|--------|----|----|-------------|
| Response time (3 agents) | Sum of all | Time of slowest | 2-4x faster |
| First response visible | After complete | Immediately (streaming) | Perceived 4x faster |
| TUI updates | Polling | Event-driven (60fps) | Smoother |

## Configuration Migration

### Basic Structure

**v1:**
```yaml
orchestrator:
  mode: round-robin
  max_turns: 10
  turn_timeout: 30s
  response_delay: 1s

agents:
  - id: claude
    type: claude
    name: Claude
    prompt: "You are helpful"
    model: claude-sonnet-4.5
```

**v2:**
```yaml
conversation:
  mode: parallel
  timeout: 30s
  max_turns: 10

agents:
  - id: claude
    type: claude-api
    name: Claude
    model: claude-sonnet-4-20250514
    config:
      system_prompt: "You are helpful"
      api_key_env: ANTHROPIC_API_KEY
```

### Field Mappings

| v1 Field | v2 Field | Notes |
|----------|----------|-------|
| `orchestrator.mode` | `conversation.mode` | Only `parallel` in MVP |
| `orchestrator.max_turns` | `conversation.max_turns` | Same behavior |
| `orchestrator.turn_timeout` | `conversation.timeout` | Renamed |
| `orchestrator.response_delay` | (removed) | Not needed with parallel |
| Agent `type: claude` | Agent `type: claude-api` | Explicit adapter type |
| Agent `prompt` | `config.system_prompt` | Moved to nested config |
| Agent `model` | Agent `model` | Same location |
| Agent API key fields | `config.api_key_env` | Moved to nested config |

### Mode Migration

| v1 Mode | v2 Equivalent | Notes |
|---------|---------------|-------|
| `round-robin` | `parallel` | All respond simultaneously |
| `reactive` | `parallel` | All respond simultaneously |
| `free-form` | `parallel` | Default behavior |

> **Note:** Additional modes (round-robin, reactive) planned for v2.1+

## Step-by-Step Migration

### Step 1: Backup Your Config

```bash
cp ~/.agentpipe/config.yaml ~/.agentpipe/config.yaml.v1.backup
```

### Step 2: Run Auto-Migration

```bash
# Migrate and preview
agentpipe migrate --from-v1 --config ~/.agentpipe/config.yaml --dry-run

# Migrate and save
agentpipe migrate --from-v1 --config ~/.agentpipe/config.yaml
```

### Step 3: Review Migrated Config

The migrated config is saved to `~/.agentpipe/v2/config.yaml`. Review for:

1. **Adapter types** - Ensure correct adapter selected
2. **API key environment variables** - Verify env var names
3. **System prompts** - Check they migrated correctly
4. **Timeouts** - Adjust if needed for parallel execution

### Step 4: Set Environment Variables

Ensure API keys are exported:

```bash
export ANTHROPIC_API_KEY="your-key"
export OPENROUTER_API_KEY="your-key"
```

### Step 5: Test

```bash
# Run with v2 engine
agentpipe run --v2 -c ~/.agentpipe/v2/config.yaml

# Or set v2 as default
export AGENTPIPE_V2=1
agentpipe run -c ~/.agentpipe/v2/config.yaml
```

## Chat History Migration

Your v1 chat logs are automatically upgraded when loaded in v2:

```bash
# Migrate chat history
agentpipe migrate --chats ~/.agentpipe/chats
```

Chat logs are:
- Read from original location
- Upgraded to v2 format (with version field)
- Saved to `~/.agentpipe/v2/chats/`
- Originals preserved

## Adapter Migration

### v1 Agent Types → v2 Adapters

| v1 Type | v2 Adapter | Notes |
|---------|------------|-------|
| `claude` | `claude-api` | Direct API (recommended) |
| `claude` | `claude` | CLI-based (if preferred) |
| `openrouter` | `openrouter` | Same name |
| `gemini` | `gemini` | CLI-based |

### Example: Claude Agent

**v1:**
```yaml
agents:
  - id: claude
    type: claude
    name: Claude
    model: claude-sonnet-4.5
    prompt: "Be helpful"
    api_key: ${ANTHROPIC_API_KEY}
```

**v2:**
```yaml
agents:
  - id: claude
    type: claude-api
    name: Claude
    model: claude-sonnet-4-20250514
    config:
      api_key_env: ANTHROPIC_API_KEY
      system_prompt: "Be helpful"
```

### Example: OpenRouter Agent

**v1:**
```yaml
agents:
  - id: gpt4
    type: openrouter
    name: GPT-4
    model: openai/gpt-4-turbo
    api_key: ${OPENROUTER_API_KEY}
    temperature: 0.7
```

**v2:**
```yaml
agents:
  - id: gpt4
    type: openrouter
    name: GPT-4
    model: openai/gpt-4-turbo
    config:
      api_key_env: OPENROUTER_API_KEY
      temperature: 0.7
```

## Breaking Changes

### Removed Features (v2 MVP)

| Feature | Status | Alternative |
|---------|--------|-------------|
| Round-robin mode | Planned v2.1 | Use parallel mode |
| Reactive mode | Planned v2.1 | Use parallel mode |
| Response delay | Removed | Not needed with parallel |
| Middleware system | Simplified | Use event handlers |
| `--headless` flag | Renamed | Use `--no-tui` |

### Changed Behavior

| Behavior | v1 | v2 |
|----------|----|----|
| Agent execution | Sequential | Parallel |
| Response display | Complete then show | Stream in real-time |
| Error handling | Stops conversation | Continues with other agents |
| TUI updates | Polling | Event-driven |

### Removed Fields

These v1 config fields are not used in v2:

- `orchestrator.response_delay` - Not needed with parallel
- `orchestrator.initial_prompt` - Use agent system prompts
- `bridge.*` - v1 streaming bridge (replaced by event system)

## Running v1 and v2 Side-by-Side

You can run both versions simultaneously:

### Different Config Directories

- **v1**: `~/.agentpipe/`
- **v2**: `~/.agentpipe/v2/`

### Explicit Version Selection

```bash
# Run v1 explicitly
agentpipe run -c ~/.agentpipe/config.yaml

# Run v2 explicitly
agentpipe run --v2 -c ~/.agentpipe/v2/config.yaml
```

### Environment Variable

```bash
# Always use v2
export AGENTPIPE_V2=1

# Use v1 (default when env not set)
unset AGENTPIPE_V2
```

## Common Migration Issues

### "Unknown adapter: claude"

v2 renamed the Claude adapter:

```yaml
# Wrong
type: claude

# Correct for API
type: claude-api

# Correct for CLI
type: claude
```

### "API key not found"

Environment variable names changed:

```bash
# v1 might have used different names
# v2 expects standard names:
export ANTHROPIC_API_KEY="your-key"
export OPENROUTER_API_KEY="your-key"
```

Or specify in config:

```yaml
config:
  api_key_env: MY_CUSTOM_VAR_NAME
```

### "Mode 'round-robin' not supported"

v2 MVP only supports parallel mode:

```yaml
# v2 MVP
conversation:
  mode: parallel  # Only option currently
```

Round-robin and reactive modes planned for v2.1.

### "prompt" field not recognized

System prompts moved to nested config:

```yaml
# Wrong (v1 style)
prompt: "Be helpful"

# Correct (v2 style)
config:
  system_prompt: "Be helpful"
```

## Rollback

If you need to revert to v1:

```bash
# Restore v1 config
cp ~/.agentpipe/config.yaml.v1.backup ~/.agentpipe/config.yaml

# Run without --v2 flag
agentpipe run -c ~/.agentpipe/config.yaml

# Or unset environment variable
unset AGENTPIPE_V2
```

Your v1 installation remains functional.

## Deprecation Timeline

| Date | Status |
|------|--------|
| Now | v2 MVP available with `--v2` flag |
| v2.0 Stable | v2 becomes default, v1 enters maintenance |
| +3 months | v1 security fixes only |
| +6 months | v1 end of life |

**Recommendation:** Migrate within 3 months of v2.0 stable release.

## Getting Help

- **Migration issues**: [GitHub Issues](https://github.com/ASRagab/agentpipe/issues)
- **Questions**: [GitHub Discussions](https://github.com/ASRagab/agentpipe/discussions)
- **Documentation**: See [[configuration]] for full v2 config reference

## Quick Reference

### Commands

```bash
# Migrate config
agentpipe migrate --from-v1 --config path/to/config.yaml

# Migrate chat history
agentpipe migrate --chats ~/.agentpipe/chats

# Run with v2
agentpipe run --v2 -c config.yaml

# Set v2 as default
export AGENTPIPE_V2=1
```

### Key Config Changes

```yaml
# v1 → v2

orchestrator: → conversation:
  mode: round-robin → mode: parallel
  turn_timeout: 30s → timeout: 30s

agents:
  - prompt: "..." → config:
                      system_prompt: "..."
    type: claude → type: claude-api
```
