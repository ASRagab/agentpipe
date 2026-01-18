# AgentPipe v2.0.0-mvp Release Notes

**Release Date:** January 18, 2026

AgentPipe v2.0.0-mvp is a **complete architecture rewrite** that introduces parallel agent execution, real-time streaming, and a modern event-driven architecture. This release represents a fundamental upgrade to how multi-agent conversations work.

---

## 🚀 What's New

### Parallel Agent Execution
All agents can now respond **simultaneously** instead of taking turns:
- **2-3x faster conversations** - Time equals the slowest agent, not the sum of all
- Configurable via `conversation.mode: parallel`
- Reduces wait time dramatically for multi-agent setups

### Real-Time Token Streaming
Watch responses as they're generated:
- Word-level streaming with 60fps TUI updates
- Server-Sent Events (SSE) support for API adapters
- Streaming chunks emitted via event bus

### Event-Driven Architecture
New `pkg/v2/events/` package with publish-subscribe pattern:
- Event types: `message.created`, `message.chunk`, `agent.typing`, `agent.done`, `agent.error`
- Loose coupling for extensibility
- Subscribe to specific events with unsubscribe functions

### API-First Adapters
Direct API integration without CLI dependencies:
- New `claude-api` adapter for Anthropic API
- OpenRouter integration for 400+ models
- Lower latency than CLI-based adapters
- CLI adapters remain as fallbacks

### Auto-Save & Resume
Never lose conversation progress:
- Conversations auto-saved on exit
- `--resume latest` continues last conversation
- `--resume <id>` for specific conversations

### Circuit Breaker Pattern
Intelligent failure handling:
- Automatic circuit opening after repeated failures
- Half-open state for recovery testing
- Prevents cascade failures across agents

---

## ⚡ Quick Start

```bash
# Enable v2 with the --v2 flag
agentpipe run --v2 -a claude:Alice -a gemini:Bob -p "Discuss AI ethics"

# Resume a saved conversation
agentpipe run --v2 --resume latest

# Export conversation on exit
agentpipe run --v2 --export conversation.md -a claude:Assistant

# Migrate v1 config to v2 format
agentpipe run --v2 --migrate-config -c my-config.yaml
```

---

## 📊 Performance Improvements

| Metric | v1 | v2 | Improvement |
|--------|----|----|-------------|
| 2-agent conversation (10 turns) | ~45s | ~20s | **2.25x faster** |
| 4-agent parallel responses | N/A | ~8s | **New capability** |
| TUI render latency | ~50ms | ~16ms | **3x faster** |
| Memory usage (10 messages) | ~25MB | ~15MB | **40% less** |
| First response visible | After complete | Immediately | **Streaming** |

### Benchmark Results (Apple M3 Max)
- **Single Message Latency**: ~54μs (target: <100ms) ✅
- **Event Bus Throughput**: 1,501,172 events/s ✅
- **Pool Execution**: 63ns/op ✅
- **Rate Limiter**: 49ns/op ✅

---

## ⚠️ Breaking Changes

| Feature | v1 | v2 | Migration |
|---------|----|----|-----------|
| Config section | `orchestrator:` | `conversation:` | Rename in YAML |
| Timeout field | `turn_timeout` | `timeout` | Rename in YAML |
| System prompt | `prompt:` | `config.system_prompt:` | Move to nested config |
| Agent types | `claude`, `gemini` | `claude-api`, `claude`, `gemini` | Update type field |
| Execution | Sequential | Parallel only (MVP) | Expected behavior change |
| `--headless` flag | Supported | Removed | Use `--no-tui` |

---

## 🔄 Upgrade Instructions

### 1. Backup Your v1 Config
```bash
cp my-config.yaml my-config.yaml.backup
```

### 2. Use Auto-Migration
```bash
agentpipe run --v2 --migrate-config -c my-config.yaml
```
This creates a backup (`*.v1.backup`) and converts to v2 format.

### 3. Manual Migration (Optional)

**Before (v1):**
```yaml
orchestrator:
  mode: round-robin
  turn_timeout: 30s

agents:
  - id: claude-agent
    type: claude
    prompt: "You are a helpful assistant."
```

**After (v2):**
```yaml
conversation:
  mode: parallel
  timeout: 30s

agents:
  - id: claude-agent
    type: claude-api
    config:
      system_prompt: "You are a helpful assistant."
```

### 4. Test Before Committing
```bash
# Test with v2 flag before setting as default
agentpipe run --v2 -c my-config.yaml

# Set v2 as default via environment variable
export AGENTPIPE_V2=true
```

---

## 📦 New v2 CLI Flags

| Flag | Description |
|------|-------------|
| `--v2` | Use v2 parallel execution engine |
| `--parallel` | Enable parallel agent execution (default: true) |
| `--v2-timeout` | Agent timeout in seconds (default: 60) |
| `--save-dir` | Directory for conversation saves |
| `--resume` | Resume a saved conversation (`latest` or ID) |
| `--export` | Export conversation to Markdown on exit |
| `--auto-save` | Enable auto-save (default: true) |
| `--migrate-config` | Migrate v1 config to v2 format with backup |

---

## ⚠️ Known Issues

1. **Parallel mode only**: v2 MVP only supports `parallel` mode. Round-robin and reactive modes planned for v2.1.

2. **CLI adapter latency**: CLI-based adapters have higher latency than API adapters due to process spawning.

3. **Large conversation memory**: Very long conversations may use significant memory. Bounded history is planned.

4. **E2E test flakiness**: Some E2E tests have pre-existing race conditions with timestamp-based file names.

---

## 🙏 Contributors

Thank you to everyone who made this release possible:

- **Kevin Elliott** - Lead developer and maintainer
- **Ahmad Ragab** - V2 architecture, testing, and release automation
- **GitHub Copilot SWE Agent** - AI-assisted development
- **Dependabot** - Dependency updates

---

## 📅 Deprecation Timeline

| Date | Status |
|------|--------|
| 2026-01-18 | v2.0.0 released with `--v2` flag |
| +1 month | v2 becomes default engine |
| +3 months | v1 enters maintenance mode (security fixes only) |
| +6 months | v1 end of life |

**Recommendation**: Migrate to v2 within 3 months for best support and new features.

---

## 📚 Documentation

- [v2 Quickstart Guide](docs/v2/quickstart.md)
- [Configuration Reference](docs/v2/configuration.md)
- [Adapter Guide](docs/v2/adapters.md)
- [Architecture Overview](docs/v2/architecture.md)
- [Migration Guide](docs/v2/migration.md)
- [Troubleshooting](docs/v2/troubleshooting.md)

---

## 🔗 Links

- **GitHub Repository**: https://github.com/ASRagab/agentpipe
- **Full Changelog**: https://github.com/ASRagab/agentpipe/blob/main/CHANGELOG.md
- **Issue Tracker**: https://github.com/ASRagab/agentpipe/issues

---

**Note**: This is an MVP release. Pre-release versions (like v2.0.0-mvp) skip automatic Homebrew formula updates. Homebrew formula will be auto-updated when a stable release (e.g., v2.0.0 without suffix) is published.
