---
type: reference
title: AgentPipe v2 Architecture
created: 2026-01-18
tags:
  - v2
  - architecture
  - technical
  - contributing
related:
  - "[[adapters]]"
  - "[[configuration]]"
  - "[[migration]]"
---

# AgentPipe v2 Architecture

Technical overview of AgentPipe v2 for contributors and developers extending the platform.

## Design Philosophy

v2 is built around these principles:

1. **Parallel by default** - All agents execute concurrently
2. **Event-driven** - Loose coupling via event bus
3. **Streaming-first** - Real-time response display
4. **API-first** - Direct API integrations, CLI as fallback

## Package Structure

```
pkg/
├── core/           # Core data types and interfaces
│   ├── message.go  # Message, Metrics, Role types
│   ├── agent.go    # Agent struct and config
│   ├── events.go   # Event types and data
│   └── conversation.go  # Conversation state
│
├── events/         # Event bus implementation
│   ├── bus.go      # EventBus interface and implementation
│   └── bus_test.go
│
├── adapters/       # Agent adapters
│   ├── adapter.go  # AgentAdapter interface
│   ├── registry.go # Adapter registry
│   ├── retry.go    # Retry logic with backoff
│   ├── circuit_breaker.go  # Circuit breaker pattern
│   ├── api/        # API-based adapters
│   │   ├── openrouter.go
│   │   ├── claude.go
│   │   └── http_errors.go
│   └── cli/        # CLI-based adapters
│       ├── base.go
│       ├── claude.go
│       └── gemini.go
│
├── manager/        # Conversation manager
│   ├── manager.go  # ConversationManager
│   └── manager_test.go
│
├── pool/           # Agent pool for parallel execution
│   ├── pool.go     # AgentPool implementation
│   └── pool_test.go
│
├── config/         # Configuration loading
│   ├── config.go   # Config types and loading
│   ├── migrate.go  # v1 → v2 migration
│   └── testdata/   # Test configurations
│
├── persistence/    # Save/load conversations
│   ├── store.go    # ConversationStore interface
│   └── json.go     # JSON file storage
│
├── tui/            # Terminal UI
│   ├── tui.go      # Main TUI model
│   ├── layout.go   # Layout calculations
│   ├── components/ # UI components
│   │   ├── status_bar.go
│   │   ├── agent_list.go
│   │   ├── conversation.go
│   │   └── input.go
│   └── styles/     # Lipgloss styles
│
└── errors/         # Error types and handling
    └── errors.go   # AgentError, error classification
```

## Core Components

### Event Bus

The event bus decouples components through publish-subscribe messaging.

```go
type EventBus interface {
    // Publish sends an event to all subscribers
    Publish(event core.Event)

    // Subscribe registers a handler for a specific event type
    Subscribe(eventType string, handler func(core.Event)) func()

    // SubscribeAll registers a handler for all events
    SubscribeAll(handler func(core.Event)) func()

    // Close waits for all handlers to complete
    Close() error
}
```

**Event Types:**

| Event | Data Type | Description |
|-------|-----------|-------------|
| `message.created` | `core.Message` | New message added |
| `message.chunk` | `core.MessageChunk` | Streaming chunk received |
| `agent.typing` | `core.AgentTypingData` | Agent started responding |
| `agent.done` | `core.AgentDoneData` | Agent completed response |
| `agent.error` | `core.AgentErrorData` | Agent encountered error |

**Usage:**

```go
// Subscribe to specific events
unsubscribe := bus.Subscribe(core.EventMessageCreated, func(e core.Event) {
    msg := e.Data.(core.Message)
    fmt.Printf("New message: %s\n", msg.Content)
})
defer unsubscribe()

// Publish events
bus.Publish(core.Event{
    Type:      core.EventMessageCreated,
    Timestamp: time.Now(),
    Data:      message,
})
```

### Conversation Manager

Orchestrates the conversation flow, coordinates agents, and manages state.

```go
type ConversationManager struct {
    config       Config
    conversation *core.Conversation
    agents       []core.Agent
    eventBus     *events.Bus
    agentPool    *pool.AgentPool
    store        persistence.ConversationStore
    mu           sync.RWMutex
}

// Key methods
func (m *ConversationManager) SendUserMessage(ctx context.Context, content string) (*core.Message, error)
func (m *ConversationManager) GetMessages() []core.Message
func (m *ConversationManager) GetAgents() []core.Agent
func (m *ConversationManager) Save() error
func (m *ConversationManager) Shutdown() error
```

### Agent Pool

Executes agents in parallel with controlled concurrency.

```go
type AgentPool struct {
    agents       map[string]*AgentRunner
    eventBus     *events.Bus
    maxConcurrent int
    timeout      time.Duration
}

// Execute all agents in parallel
func (p *AgentPool) ExecuteParallel(ctx context.Context, messages []core.Message) []AgentResponse
```

**Execution Flow:**

```
User Message
     │
     ▼
┌─────────────────────────────────────┐
│         Agent Pool                   │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐│
│  │ Agent 1 │ │ Agent 2 │ │ Agent 3 ││
│  │ (async) │ │ (async) │ │ (async) ││
│  └────┬────┘ └────┬────┘ └────┬────┘│
└───────┼──────────┼──────────┼───────┘
        │          │          │
        ▼          ▼          ▼
    ┌────────────────────────────┐
    │        Event Bus           │
    │  (message.chunk events)    │
    └────────────────────────────┘
              │
              ▼
         ┌────────┐
         │  TUI   │
         └────────┘
```

### Agent Adapter Interface

All adapters implement this interface:

```go
type AgentAdapter interface {
    // Initialize with agent configuration
    Initialize(agent core.Agent) error

    // Send messages and get response
    SendMessage(ctx context.Context, messages []core.Message) (string, *core.Metrics, error)

    // Stream response to writer
    StreamMessage(ctx context.Context, messages []core.Message, writer io.Writer) (*core.Metrics, error)

    // Check if adapter is configured
    IsAvailable() bool

    // Get configured model
    GetModel() string

    // Test API connectivity
    HealthCheck(ctx context.Context) error
}
```

### Error Handling

Standardized error types with classification:

```go
type AgentError struct {
    AgentID     string
    AgentName   string
    Type        ErrorType  // timeout, rate_limit, auth, network, etc.
    Message     string
    Recoverable bool
    RetryAfter  time.Duration
    StatusCode  int
    Cause       error
}

// Error classification
func ClassifyError(err error) ErrorType
func IsRetryable(err error) bool
func GetRetryAfter(err error) time.Duration
```

## Data Flow

### Message Lifecycle

```
1. User Input
   └─> ConversationManager.SendUserMessage()
       └─> Create user Message
       └─> Emit EventMessageCreated
       └─> Add to conversation history
       └─> AgentPool.ExecuteParallel()

2. Parallel Execution
   └─> For each agent (concurrent):
       └─> Emit EventAgentTyping
       └─> Call adapter.StreamMessage()
       └─> Stream chunks to event bus (EventMessageChunk)
       └─> On complete: Emit EventAgentDone
       └─> On error: Emit EventAgentError

3. TUI Updates
   └─> Subscribe to all events
   └─> Update components based on event type
   └─> Render at 60fps
```

### Streaming Flow

```
API Response (SSE)
     │
     ▼
Adapter.StreamMessage()
     │ (writes to io.Writer)
     ▼
ChunkWriter (wraps writer)
     │ (emits events)
     ▼
EventBus.Publish(EventMessageChunk)
     │
     ▼
TUI Handler
     │
     ▼
ConversationModel.AppendChunk()
     │
     ▼
Viewport.Render()
```

## Concurrency Model

### Thread Safety

- **Event Bus**: Uses sync.RWMutex for subscriber management
- **Conversation Manager**: Uses sync.RWMutex for message history
- **Agent Pool**: Uses WaitGroup for parallel execution
- **TUI**: Single-threaded update loop with event queue

### Context and Cancellation

```go
// All operations accept context for cancellation
func (m *ConversationManager) SendUserMessage(ctx context.Context, content string) (*core.Message, error) {
    // Context propagated to all agents
    ctx, cancel := context.WithTimeout(ctx, m.config.Timeout)
    defer cancel()

    // Agents check ctx.Done() and abort early
    responses := m.agentPool.ExecuteParallel(ctx, messages)
    // ...
}
```

### Graceful Shutdown

```go
func (m *ConversationManager) Shutdown() error {
    // Cancel ongoing operations
    m.cancel()

    // Wait for agent pool with timeout
    done := make(chan struct{})
    go func() {
        m.agentPool.Wait()
        close(done)
    }()

    select {
    case <-done:
        return nil
    case <-time.After(5 * time.Second):
        return errors.New("shutdown timeout")
    }
}
```

## TUI Architecture

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea):

```go
type Model struct {
    manager      *manager.ConversationManager
    eventBus     *events.Bus

    // Components (each is a tea.Model)
    statusBar    components.StatusBarModel
    agentList    components.AgentListModel
    conversation components.ConversationModel
    input        components.InputModel

    // State
    focusedPanel FocusedPanel
    eventQueue   []core.Event
}

func (m Model) Init() tea.Cmd {
    return tea.Batch(
        m.subscribeToEvents(),
        m.startRenderTicker(),
    )
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Handle key presses, events, ticks
}

func (m Model) View() string {
    // Compose layout from components
    return lipgloss.JoinVertical(
        statusBar.View(),
        lipgloss.JoinHorizontal(agentList.View(), conversation.View()),
        input.View(),
    )
}
```

## Testing Strategy

### Unit Tests

Test individual components in isolation:

```go
func TestEventBusPublishSubscribe(t *testing.T) {
    bus := events.NewEventBus()
    defer bus.Close()

    received := make(chan core.Event, 1)
    bus.Subscribe(core.EventMessageCreated, func(e core.Event) {
        received <- e
    })

    bus.Publish(core.Event{Type: core.EventMessageCreated})

    select {
    case <-received:
        // Success
    case <-time.After(time.Second):
        t.Fatal("Event not received")
    }
}
```

### Integration Tests

Test component interactions:

```go
func TestConversationFlow(t *testing.T) {
    // Setup with mock adapters
    bus := events.NewEventBus()
    pool := pool.NewAgentPool(mockAgents, bus)
    mgr := manager.New(config, pool, bus)

    // Execute conversation
    _, err := mgr.SendUserMessage(ctx, "Hello")
    require.NoError(t, err)

    // Verify messages
    msgs := mgr.GetMessages()
    assert.Len(t, msgs, 3) // 1 user + 2 agents
}
```

### E2E Tests

Test full system with real or mock APIs:

```bash
go test ./pkg/e2e/... -tags=e2e
```

## Contributing Guidelines

### Adding a New Adapter

1. Create adapter file in `pkg/adapters/api/` or `pkg/adapters/cli/`
2. Implement `AgentAdapter` interface
3. Register in `init()` function:

```go
func init() {
    adapters.Register("my-adapter", NewMyAdapter)
}
```

1. Add tests with mocked HTTP/CLI responses
2. Update documentation in `docs/v2/adapters.md`

### Adding New Event Types

1. Define event type constant in `pkg/core/events.go`
2. Create data type for event payload
3. Add handling in TUI and other subscribers
4. Document in this architecture guide

### Code Standards

- **Go Version**: 1.24+
- **Test Coverage**: >80% for new code
- **Race Detection**: All tests run with `-race`
- **Documentation**: Godoc for all exported types
- **Error Handling**: Use `pkg/errors` types

### Development Workflow

```bash
# Run tests
go test -race ./pkg/...

# Run linter
golangci-lint run --timeout=5m

# Build
go build -o agentpipe .

# Run with debug logging
AGENTPIPE_LOG_LEVEL=debug ./agentpipe run --v2 -c config.yaml
```

## Performance Considerations

### Memory Management

- Message history is bounded (configurable)
- Streaming chunks are processed immediately, not buffered
- Event handlers run in goroutines (avoid blocking)

### Latency Optimization

- Parallel execution: N agents in time of slowest
- Streaming: First chunk visible immediately
- TUI: 60fps render loop, event-driven updates

### CPU Usage

- Idle when waiting for user input
- Spikes during streaming (processing SSE chunks)
- Event processing in separate goroutines

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/charmbracelet/bubbletea` | TUI framework |
| `github.com/charmbracelet/lipgloss` | TUI styling |
| `github.com/charmbracelet/bubbles` | TUI components |
| `gopkg.in/yaml.v3` | Configuration parsing |
| `github.com/stretchr/testify` | Testing utilities |

## Future Architecture Considerations

### v2.1+ Features

- **Multiple Conversation Modes**: Add round-robin, reactive modes
- **Agent-to-Agent Messaging**: Direct communication between agents
- **Plugin System**: External adapter loading
- **Middleware Pipeline**: Pre/post processing hooks

### Scalability

- Event bus could be replaced with message queue for distributed deployment
- Agent pool could support dynamic scaling
- Persistence could support multiple backends (SQLite, PostgreSQL)
