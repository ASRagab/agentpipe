# AgentPipe v1 → v2 Migration Guide

## Executive Summary

AgentPipe v2 is a **ground-up rewrite** focused on **real-time parallel multi-agent conversation**. While v1 emphasized turn-taking orchestration with various modes, v2 prioritizes **streaming responses** and **parallel execution** for a more natural conversational experience.

**Key Philosophy Shift**: v1 = "Agents take turns speaking" → v2 = "All agents respond simultaneously to user"

---

## 1. What Changed (High-Level)

### Architecture Changes

| Aspect | v1 | v2 |
|--------|----|----|
| **Execution Model** | Sequential or round-robin | Parallel by default |
| **Response Streaming** | Wait for full response | Real-time streaming |
| **Event System** | Tightly coupled orchestrator | Event bus pub/sub |
| **TUI Updates** | Polling-based | Event-driven |
| **Agent Types** | CLI-focused (API via adapters) | API-first (CLI for legacy) |
| **Config Complexity** | 5+ orchestration modes | 1 mode (parallel) in MVP |
| **Primary Use Case** | Structured turn-taking | Free-form collaboration |

### Package Structure Changes

```
v1:
pkg/
├── orchestrator/    # Monolithic orchestrator (1300 lines)
├── agent/           # Agent interface
├── adapters/        # CLI and API adapters
├── tui/             # TUI implementation
├── config/          # Config loading
└── logger/          # Chat logging

v2:
pkg/v2/
├── core/            # Core data types (Message, Agent, etc.)
├── events/          # Event bus (new!)
├── manager/         # Conversation manager (replaces orchestrator)
├── pool/            # Agent pool for parallel execution (new!)
├── adapters/        # Refactored adapters (api/ and cli/ subdirs)
├── tui/             # Event-driven TUI (rebuilt)
├── config/          # Config loading + v1 migration
└── persistence/     # Save/load (extracted from orchestrator)
```

---

## 2. Breaking Changes

### 2.1 Config Format (Backwards Compatible with Migration)

**v1 Config**:
```yaml
orchestrator:
  mode: round-robin  # or reactive, free-form
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

**v2 Config**:
```yaml
conversation:
  mode: parallel  # Only mode in MVP
  timeout: 30s    # Per-agent timeout
  save_dir: ~/.agentpipe/conversations

agents:
  - id: claude
    type: api                  # NEW: "api" or "cli"
    adapter: claude-api        # NEW: Explicit adapter
    name: Claude
    model: claude-sonnet-4.5
    config:                    # NEW: Nested config
      api_key_env: ANTHROPIC_API_KEY
      temperature: 0.7
      max_tokens: 2000
```

**Migration**: v2 auto-detects v1 config and upgrades it with a backup.

### 2.2 Orchestration Modes

**v1 Modes**:
- `round-robin`: Agents take turns in order
- `reactive`: Random agent responds (not the last speaker)
- `free-form`: All agents can respond if they want

**v2 Modes (MVP)**:
- `parallel`: All agents respond to every user message

**Migration**:
- `round-robin` → `parallel` (all agents respond, but UI can show in order)
- `reactive` → `parallel` (all agents respond)
- `free-form` → `parallel` (default behavior)

**Post-MVP** (v2.1+): Add selective participation, agent-to-agent messaging

### 2.3 Agent Execution

**v1**:
```go
// Sequential execution in orchestrator
for _, agent := range agents {
    response, err := agent.SendMessage(ctx, messages)
    // Display response
    // Wait for delay
}
```

**v2**:
```go
// Parallel execution in agent pool
responses := pool.ExecuteParallel(ctx, messages)
// All agents execute concurrently
// Responses stream in real-time
```

**Impact**: v2 is significantly faster for multi-agent conversations (N agents in time of slowest agent vs N × average agent time).

### 2.4 Event System

**v1**: No event system (direct method calls)

**v2**: Event bus for all communication

```go
// v1: Direct display in orchestrator
fmt.Fprintf(writer, "[%s] %s\n", agent.Name, response)

// v2: Emit event, TUI subscribes
eventBus.Publish(Event{
    Type: EventMessageCreated,
    Data: message,
})
```

**Migration**: TUI must be updated to subscribe to events instead of reading from writer.

### 2.5 TUI Updates

**v1**: TUI receives formatted strings via `io.Writer`

```go
orchestrator.SetWriter(tuiWriter)
```

**v2**: TUI subscribes to event bus

```go
manager.Subscribe(EventMessageCreated, func(e Event) {
    msg := e.Data.(Message)
    // Update UI
})
```

**Impact**: TUI is decoupled from conversation manager, can update independently.

---

## 3. What Stayed the Same

### 3.1 Agent Interface (Mostly)

**v1**:
```go
type Agent interface {
    GetID() string
    GetName() string
    GetType() string
    GetModel() string
    SendMessage(ctx context.Context, messages []Message) (string, error)
    StreamMessage(ctx context.Context, messages []Message, writer io.Writer) error
    IsAvailable() bool
    HealthCheck(ctx context.Context) error
    GetCLIVersion() string
}
```

**v2**:
```go
type AgentAdapter interface {
    Initialize(config map[string]interface{}) error
    SendMessage(ctx context.Context, messages []Message) (string, error)
    StreamMessage(ctx context.Context, messages []Message, writer io.Writer) error
    IsAvailable() bool
    GetModel() string
    HealthCheck(ctx context.Context) error
}
```

**Changes**:
- Renamed to `AgentAdapter` (clearer purpose)
- Added `Initialize()` for config
- Removed `GetID()`, `GetName()`, `GetType()` (moved to `Agent` struct)
- Removed `GetCLIVersion()` (not needed for API adapters)

**Migration**: v1 adapters can be ported with minimal changes.

### 3.2 Message Structure

**v1**:
```go
type Message struct {
    AgentID   string
    AgentName string
    AgentType string
    Content   string
    Timestamp int64
    Role      string
    Metrics   *ResponseMetrics
}
```

**v2**:
```go
type Message struct {
    ID        string           // NEW: Unique message ID
    Timestamp time.Time        // CHANGED: time.Time instead of int64
    Role      string
    AgentID   string
    AgentName string
    Content   string
    Status    string           // NEW: "pending", "streaming", "complete", "error"
    Metrics   *Metrics
}
```

**Changes**:
- Added `ID` field (unique identifier)
- Changed `Timestamp` to `time.Time` (more type-safe)
- Added `Status` field (for streaming updates)
- Removed `AgentType` (moved to `Agent` struct)

**Migration**: Easy to convert v1 messages to v2 format.

### 3.3 Chat Logging

**v1**: `pkg/logger/logger.go`
- Logs messages to JSON file
- Stores in `~/.agentpipe/chats/`

**v2**: Same functionality, moved to `pkg/v2/persistence/`
- Same JSON format (with version field)
- Same directory structure
- Auto-migration on load

**Migration**: Existing chat logs are auto-upgraded when loaded.

### 3.4 Token Estimation

**v1**: `pkg/utils/tokens.go`
- Estimates tokens from text
- Calculates cost based on model

**v2**: Same utilities, no changes

**Migration**: No changes needed.

---

## 4. Feature Comparison

### MVP Features (v2.0.0)

| Feature | v1 | v2 MVP |
|---------|----|----|
| User sends message → Agents respond | ✅ | ✅ |
| Real-time streaming responses | ❌ | ✅ |
| Parallel agent execution | ❌ | ✅ |
| TUI with agent list + conversation | ✅ | ✅ |
| Metrics display (tokens, cost, duration) | ✅ | ✅ |
| Save/resume conversations | ✅ | ✅ |
| Export to JSON/Markdown | ✅ | ✅ |
| YAML config | ✅ | ✅ |
| OpenRouter support | ✅ | ✅ |
| Claude API support | ✅ | ✅ |
| CLI agent support (Gemini, etc.) | ✅ | ✅ |

### Advanced Features (Not in MVP)

| Feature | v1 | v2.1+ |
|---------|----|----|
| Round-robin mode | ✅ | 🔜 |
| Reactive mode | ✅ | 🔜 |
| Free-form mode | ✅ | 🔜 |
| Agent-to-agent messaging | ❌ | 🔜 |
| @Mentions | ❌ | 🔜 |
| Conversation threading | ❌ | 🔜 |
| Artifact extraction | ✅ | 🔜 (v2.3) |
| Summary generation | ✅ | 🔜 (v2.4) |
| Middleware system | ✅ | ❌ (Simplified) |
| Rate limiting | ✅ | 🔜 |
| Retry logic | ✅ | ✅ |

---

## 5. Migration Steps (User)

### For End Users

**Option 1: Fresh Start (Recommended)**
1. Install v2: `brew install agentpipe@2` or `go install`
2. Run migration: `agentpipe migrate --from-v1`
3. Review migrated config: `~/.agentpipe/v2/config.yaml`
4. Test with a sample conversation

**Option 2: Side-by-Side**
1. Keep v1 installed
2. Install v2 in different location
3. Use different config directories
4. Manually port configs as needed

**Config Migration**:
```bash
# Auto-migrate v1 config to v2
agentpipe migrate --from-v1 --config ~/.agentpipe/config.yaml

# Backup is created at:
# ~/.agentpipe/config.yaml.v1.backup

# Migrated config:
# ~/.agentpipe/v2/config.yaml
```

**Chat History Migration**:
```bash
# Auto-migrate v1 chat logs to v2
agentpipe migrate --chats ~/.agentpipe/chats

# Migrated chats:
# ~/.agentpipe/v2/chats/
```

### For Developers (Extending AgentPipe)

**Porting v1 Adapters to v2**:

```go
// v1 Adapter
type MyAdapter struct {
    agent.BaseAgent
    // ... fields
}

func (a *MyAdapter) Initialize(config agent.AgentConfig) error {
    // ... setup
}

// v2 Adapter
type MyAdapter struct {
    // No embedding
    model  string
    apiKey string
}

func (a *MyAdapter) Initialize(config map[string]interface{}) error {
    a.model, _ = config["model"].(string)
    apiKeyEnv, _ := config["api_key_env"].(string)
    a.apiKey = os.Getenv(apiKeyEnv)
    return nil
}

// Register in v2
func init() {
    adapters.DefaultRegistry.Register("my-adapter", func() adapters.AgentAdapter {
        return &MyAdapter{}
    })
}
```

**Porting Custom Middleware** (v1 → v2):

v1 had a middleware system for message processing. v2 MVP does not include middleware (for simplicity), but can be added in v2.1+.

If you have custom middleware, it needs to be ported to event handlers:

```go
// v1: Middleware
type LoggingMiddleware struct{}

func (m *LoggingMiddleware) Process(ctx *middleware.MessageContext, msg *agent.Message) (*agent.Message, error) {
    log.Printf("Message: %s", msg.Content)
    return msg, nil
}

// v2: Event handler
manager.Subscribe(EventMessageCreated, func(e Event) {
    msg := e.Data.(Message)
    log.Printf("Message: %s", msg.Content)
})
```

---

## 6. Performance Comparison

### Response Latency (3 agents: Claude, GPT-4, Gemini)

**v1 (Sequential)**:
- Claude: 2s
- GPT-4: 3s
- Gemini: 1.5s
- **Total**: 6.5s (sum of all agents)

**v2 (Parallel)**:
- Claude: 2s
- GPT-4: 3s (slowest)
- Gemini: 1.5s
- **Total**: 3s (slowest agent)

**Improvement**: 2.2× faster

### Streaming Perception

**v1**: User waits 6.5s, sees all responses at once

**v2**: User sees first response in ~1.5s (Gemini), full conversation in 3s

**Perceived Improvement**: ~4× faster (first response)

### Memory Usage

**v1**: Single-threaded, ~50MB for 100-message conversation

**v2**: Multi-threaded, ~75MB for 100-message conversation (due to goroutines)

**Trade-off**: 50% more memory for 2-4× faster responses

---

## 7. What to Expect (User Experience)

### User Workflow: v1 vs v2

**v1 Experience**:
```
User: "Explain async/await"
[Wait 2s...]
Claude: "Async/await is..."
[Wait 1s delay...]
[Wait 3s...]
GPT-4: "It's a syntax..."
[Wait 1s delay...]
[Wait 1.5s...]
Gemini: "Think of it as..."

Total wait: ~9s
```

**v2 Experience**:
```
User: "Explain async/await"
[All agents start typing immediately]
Claude: "Async..." [streaming]
Gemini: "Think..." [streaming]
GPT-4: "It's..." [streaming]

Total wait: ~3s (for all agents to complete)
User sees first response in ~1.5s
```

### TUI Experience

**v1 TUI**:
- Shows one agent response at a time
- "Waiting for Claude..."
- Response appears all at once
- Metrics shown after response completes

**v2 TUI**:
- Shows all agents simultaneously
- Typing indicators for each agent
- Responses stream in character-by-character
- Metrics update in real-time
- Color-coded agent badges
- Status indicators (🟢 done, 🟡 typing, 🔴 error)

---

## 8. Deprecation Timeline

**v1 Support**:
- ✅ **v1.x**: Supported until v2.0 stable release
- ⚠️ **After v2.0**: Security fixes only (no new features)
- ❌ **6 months after v2.0**: End of life

**Migration Timeline**:
- **Now - Week 4**: v2 MVP development
- **Week 5-6**: v2 beta testing
- **Week 7**: v2.0 stable release
- **Week 7 - Month 6**: v1 security fixes only
- **Month 6+**: v1 end of life

**Recommendation**: Migrate to v2 within 3 months of v2.0 release.

---

## 9. FAQ

### Q: Can I run v1 and v2 side-by-side?
**A**: Yes, they use different config directories. v1 uses `~/.agentpipe/`, v2 uses `~/.agentpipe/v2/`.

### Q: Will my v1 configs work in v2?
**A**: Yes, with auto-migration. Run `agentpipe migrate --from-v1` to upgrade.

### Q: What happens to my chat history?
**A**: Chat logs are auto-upgraded when loaded. A backup is created.

### Q: Can I go back to v1 after migrating?
**A**: Yes, v1 backups are preserved. You can revert anytime.

### Q: Will v2 be slower due to parallel execution?
**A**: No, v2 is 2-4× faster overall. Parallel execution is more efficient.

### Q: What if I rely on round-robin mode?
**A**: It's not in MVP, but will be added in v2.1. Use v1 until then, or adapt to parallel mode.

### Q: Are all v1 adapters compatible with v2?
**A**: Most are (OpenRouter, Claude API, Gemini CLI). Some may need minor updates.

### Q: What about custom middleware I wrote for v1?
**A**: Port it to event handlers. See "Porting Custom Middleware" above.

### Q: Will v2 use more memory?
**A**: Yes, about 50% more due to goroutines. Trade-off for faster responses.

### Q: Can I contribute to v2 development?
**A**: Yes! Check `docs/v2-mvp-implementation-plan.md` for tasks and timeline.

---

## 10. Summary

**Key Takeaways**:

1. **v2 is faster**: Parallel execution = 2-4× faster than v1
2. **v2 is simpler**: 1 orchestration mode (parallel) vs 5 modes in v1
3. **v2 is event-driven**: Cleaner architecture, easier to extend
4. **Migration is easy**: Auto-migration for configs and chat logs
5. **Backwards compatibility**: v1 configs work with auto-upgrade
6. **Timeline**: v1 supported for 6 months after v2.0 release

**Action Items**:

- **Users**: Try v2 MVP when released, provide feedback
- **Developers**: Review `docs/v2-mvp-implementation-plan.md` for contribution opportunities
- **Early Adopters**: Join v2 beta testing in Week 5

**Resources**:
- MVP Plan: `docs/v2-mvp-implementation-plan.md`
- Architecture: `docs/v2-architecture-diagram.md`
- Core Interfaces: `docs/v2-core-interfaces.md`
- Migration Guide: This document

---

**Questions?** Open an issue on GitHub or join the discussion.
