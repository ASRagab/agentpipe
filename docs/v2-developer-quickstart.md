# AgentPipe v2 Developer Quickstart

## Overview

This guide helps you **start building AgentPipe v2** following the MVP implementation plan. Whether you're implementing a specific component or contributing to the project, this provides clear entry points.

**Target Audience**: Go developers contributing to AgentPipe v2 MVP

**Prerequisites**:
- Go 1.24+
- Basic understanding of concurrency (goroutines, channels)
- Familiarity with event-driven architecture
- Read: `docs/v2-mvp-implementation-plan.md` and `docs/v2-architecture-diagram.md`

---

## Quick Start: 5-Minute Setup

### 1. Clone and Setup

```bash
# Clone repo
git clone https://github.com/ASRagab/agentpipe.git
cd agentpipe

# Checkout v2 branch
git checkout -b feature/v2-mvp

# Install dependencies
go mod download

# Run tests (should pass for existing v1 code)
go test ./...
```

### 2. Create v2 Package Structure

```bash
# Create v2 package directories
mkdir -p pkg/v2/{core,events,manager,pool,adapters/api,adapters/cli,tui,config,persistence}

# Create initial files
touch pkg/v2/core/{message.go,agent.go,conversation.go,events.go}
touch pkg/v2/events/{bus.go,bus_test.go}
touch pkg/v2/manager/{manager.go,manager_test.go}
touch pkg/v2/pool/{pool.go,pool_test.go}
touch pkg/v2/adapters/{adapter.go,registry.go}
```

### 3. Run Example (Week 1 Goal)

```bash
# After implementing core types and event bus (Week 1)
go run examples/v2/simple_conversation.go
```

---

## Development Workflow

### Daily Workflow

```bash
# 1. Pull latest changes
git pull origin feature/v2-mvp

# 2. Create feature branch
git checkout -b feature/week1-event-bus

# 3. Write tests first (TDD)
cd pkg/v2/events
vim bus_test.go  # Write test
go test          # Watch it fail

# 4. Implement feature
vim bus.go       # Implement
go test          # Watch it pass

# 5. Lint and format
golangci-lint run --timeout=5m
gofmt -w .
goimports -local github.com/ASRagab/agentpipe -w .

# 6. Commit with clear message
git add .
git commit -m "feat(events): implement event bus with pub/sub"

# 7. Push and create PR
git push origin feature/week1-event-bus
# Open PR on GitHub
```

### Testing Strategy

```bash
# Unit tests (run frequently)
go test ./pkg/v2/...

# Integration tests (run before commit)
go test ./pkg/v2/... -tags=integration

# Race detection (run before PR)
go test -race ./pkg/v2/...

# Coverage report
go test -cover ./pkg/v2/...
go test -coverprofile=coverage.out ./pkg/v2/...
go tool cover -html=coverage.out
```

---

## Week 1: Core Infrastructure

### Task 1: Core Data Structures (Days 1-2)

**File**: `pkg/v2/core/message.go`

```go
package core

import (
    "time"
)

// Message represents any message in the conversation
type Message struct {
    ID        string    `json:"id"`
    Timestamp time.Time `json:"timestamp"`
    Role      string    `json:"role"` // "user", "agent", "system"
    AgentID   string    `json:"agent_id,omitempty"`
    AgentName string    `json:"agent_name,omitempty"`
    Content   string    `json:"content"`
    Status    string    `json:"status"` // "pending", "streaming", "complete", "error"
    Metrics   *Metrics  `json:"metrics,omitempty"`
}

// Metrics captures agent response performance
type Metrics struct {
    Duration     time.Duration `json:"duration"`
    InputTokens  int           `json:"input_tokens"`
    OutputTokens int           `json:"output_tokens"`
    TotalTokens  int           `json:"total_tokens"`
    Model        string        `json:"model"`
    Cost         float64       `json:"cost"`
}

// Example constructor
func NewUserMessage(content string) Message {
    return Message{
        ID:        generateID(),
        Timestamp: time.Now(),
        Role:      "user",
        Content:   content,
        Status:    "complete",
    }
}

func NewAgentMessage(agentID, agentName, content string, metrics *Metrics) Message {
    return Message{
        ID:        generateID(),
        Timestamp: time.Now(),
        Role:      "agent",
        AgentID:   agentID,
        AgentName: agentName,
        Content:   content,
        Status:    "complete",
        Metrics:   metrics,
    }
}

func generateID() string {
    // Use UUID or similar
    return uuid.New().String()
}
```

**Test**: `pkg/v2/core/message_test.go`

```go
package core

import (
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
)

func TestNewUserMessage(t *testing.T) {
    msg := NewUserMessage("Hello")

    assert.NotEmpty(t, msg.ID)
    assert.Equal(t, "user", msg.Role)
    assert.Equal(t, "Hello", msg.Content)
    assert.Equal(t, "complete", msg.Status)
    assert.WithinDuration(t, time.Now(), msg.Timestamp, time.Second)
}

func TestNewAgentMessage(t *testing.T) {
    metrics := &Metrics{
        Duration:     100 * time.Millisecond,
        InputTokens:  50,
        OutputTokens: 100,
        TotalTokens:  150,
        Model:        "test-model",
        Cost:         0.001,
    }

    msg := NewAgentMessage("agent1", "TestAgent", "Response", metrics)

    assert.NotEmpty(t, msg.ID)
    assert.Equal(t, "agent", msg.Role)
    assert.Equal(t, "agent1", msg.AgentID)
    assert.Equal(t, "TestAgent", msg.AgentName)
    assert.Equal(t, "Response", msg.Content)
    assert.Equal(t, metrics, msg.Metrics)
}
```

**Run**:
```bash
cd pkg/v2/core
go test -v
```

### Task 2: Event Bus (Days 3-4)

**File**: `pkg/v2/events/bus.go`

```go
package events

import (
    "log"
    "sync"

    "github.com/ASRagab/agentpipe/pkg/v2/core"
)

// Event types
const (
    EventMessageCreated     = "message.created"
    EventMessageChunk       = "message.chunk"
    EventAgentTyping        = "agent.typing"
    EventAgentDone          = "agent.done"
    EventAgentError         = "agent.error"
    EventConversationStarted = "conversation.started"
    EventConversationSaved  = "conversation.saved"
)

// EventBus handles event distribution
type EventBus interface {
    Publish(event core.Event)
    Subscribe(eventType string, handler func(core.Event)) func()
    SubscribeAll(handler func(core.Event)) func()
    Close() error
}

// eventBus implementation
type eventBus struct {
    mu          sync.RWMutex
    subscribers map[string][]func(core.Event)
    wg          sync.WaitGroup
}

// NewEventBus creates a new event bus
func NewEventBus() EventBus {
    return &eventBus{
        subscribers: make(map[string][]func(core.Event)),
    }
}

func (b *eventBus) Publish(event core.Event) {
    b.mu.RLock()
    handlers := append([]func(core.Event){}, b.subscribers[event.Type]...)
    allHandlers := append([]func(core.Event){}, b.subscribers["*"]...)
    b.mu.RUnlock()

    // Call specific event handlers
    for _, handler := range handlers {
        b.wg.Add(1)
        go b.safeCall(handler, event)
    }

    // Call "all events" handlers
    for _, handler := range allHandlers {
        b.wg.Add(1)
        go b.safeCall(handler, event)
    }
}

func (b *eventBus) safeCall(handler func(core.Event), event core.Event) {
    defer b.wg.Done()
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Event handler panic: %v", r)
        }
    }()
    handler(event)
}

func (b *eventBus) Subscribe(eventType string, handler func(core.Event)) func() {
    b.mu.Lock()
    defer b.mu.Unlock()

    b.subscribers[eventType] = append(b.subscribers[eventType], handler)

    // Return unsubscribe function
    return func() {
        b.mu.Lock()
        defer b.mu.Unlock()
        handlers := b.subscribers[eventType]
        for i, h := range handlers {
            // Compare function pointers
            if &h == &handler {
                b.subscribers[eventType] = append(handlers[:i], handlers[i+1:]...)
                break
            }
        }
    }
}

func (b *eventBus) SubscribeAll(handler func(core.Event)) func() {
    return b.Subscribe("*", handler)
}

func (b *eventBus) Close() error {
    b.wg.Wait() // Wait for all handlers to complete
    return nil
}
```

**Test**: `pkg/v2/events/bus_test.go`

```go
package events

import (
    "sync"
    "testing"
    "time"

    "github.com/ASRagab/agentpipe/pkg/v2/core"
    "github.com/stretchr/testify/assert"
)

func TestEventBusPublishSubscribe(t *testing.T) {
    bus := NewEventBus()
    defer bus.Close()

    received := make(chan core.Event, 1)

    // Subscribe
    unsubscribe := bus.Subscribe(EventMessageCreated, func(e core.Event) {
        received <- e
    })
    defer unsubscribe()

    // Publish
    event := core.Event{
        Type:      EventMessageCreated,
        Timestamp: time.Now(),
        Data:      "test",
    }
    bus.Publish(event)

    // Verify
    select {
    case e := <-received:
        assert.Equal(t, EventMessageCreated, e.Type)
        assert.Equal(t, "test", e.Data)
    case <-time.After(time.Second):
        t.Fatal("Event not received")
    }
}

func TestEventBusMultipleSubscribers(t *testing.T) {
    bus := NewEventBus()
    defer bus.Close()

    var wg sync.WaitGroup
    count := 0
    mu := sync.Mutex{}

    // Multiple subscribers
    for i := 0; i < 3; i++ {
        wg.Add(1)
        bus.Subscribe(EventMessageCreated, func(e core.Event) {
            defer wg.Done()
            mu.Lock()
            count++
            mu.Unlock()
        })
    }

    // Publish
    bus.Publish(core.Event{Type: EventMessageCreated})

    // Wait for all handlers
    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        assert.Equal(t, 3, count)
    case <-time.After(time.Second):
        t.Fatal("Handlers did not complete")
    }
}

func TestEventBusUnsubscribe(t *testing.T) {
    bus := NewEventBus()
    defer bus.Close()

    received := make(chan core.Event, 1)

    // Subscribe and unsubscribe
    unsubscribe := bus.Subscribe(EventMessageCreated, func(e core.Event) {
        received <- e
    })
    unsubscribe()

    // Publish (should not be received)
    bus.Publish(core.Event{Type: EventMessageCreated})

    // Verify no event received
    select {
    case <-received:
        t.Fatal("Event received after unsubscribe")
    case <-time.After(100 * time.Millisecond):
        // Expected
    }
}

func TestEventBusPanicRecovery(t *testing.T) {
    bus := NewEventBus()
    defer bus.Close()

    // Subscribe handler that panics
    bus.Subscribe(EventMessageCreated, func(e core.Event) {
        panic("test panic")
    })

    // Should not crash
    assert.NotPanics(t, func() {
        bus.Publish(core.Event{Type: EventMessageCreated})
        time.Sleep(100 * time.Millisecond) // Let handler panic
    })
}
```

**Run**:
```bash
cd pkg/v2/events
go test -v -race
```

### Task 3: Agent Adapters (Days 5-7)

**File**: `pkg/v2/adapters/adapter.go`

```go
package adapters

import (
    "context"
    "io"

    "github.com/ASRagab/agentpipe/pkg/v2/core"
)

// AgentAdapter handles communication with a specific agent
type AgentAdapter interface {
    Initialize(config map[string]interface{}) error
    SendMessage(ctx context.Context, messages []core.Message) (string, error)
    StreamMessage(ctx context.Context, messages []core.Message, writer io.Writer) error
    IsAvailable() bool
    GetModel() string
    HealthCheck(ctx context.Context) error
}
```

**File**: `pkg/v2/adapters/registry.go`

```go
package adapters

import (
    "fmt"
    "sort"
    "sync"
)

// AdapterRegistry manages available adapters
type AdapterRegistry struct {
    mu       sync.RWMutex
    adapters map[string]func() AgentAdapter
}

var DefaultRegistry = &AdapterRegistry{
    adapters: make(map[string]func() AgentAdapter),
}

func (r *AdapterRegistry) Register(name string, factory func() AgentAdapter) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.adapters[name] = factory
}

func (r *AdapterRegistry) Get(name string, config map[string]interface{}) (AgentAdapter, error) {
    r.mu.RLock()
    factory, ok := r.adapters[name]
    r.mu.RUnlock()

    if !ok {
        return nil, fmt.Errorf("unknown adapter: %s", name)
    }

    adapter := factory()
    if err := adapter.Initialize(config); err != nil {
        return nil, fmt.Errorf("adapter initialization failed: %w", err)
    }

    return adapter, nil
}

func (r *AdapterRegistry) List() []string {
    r.mu.RLock()
    defer r.mu.RUnlock()

    names := make([]string, 0, len(r.adapters))
    for name := range r.adapters {
        names = append(names, name)
    }
    sort.Strings(names)
    return names
}
```

**File**: `pkg/v2/adapters/api/openrouter.go` (Example)

```go
package api

import (
    "context"
    "fmt"
    "io"
    "os"

    "github.com/ASRagab/agentpipe/pkg/v2/adapters"
    "github.com/ASRagab/agentpipe/pkg/v2/core"
    "github.com/ASRagab/agentpipe/pkg/client" // Reuse v1 HTTP client
)

type OpenRouterAdapter struct {
    apiKey      string
    model       string
    temperature float64
    maxTokens   int
    client      *client.OpenAICompatClient
}

func init() {
    adapters.DefaultRegistry.Register("openrouter", func() adapters.AgentAdapter {
        return &OpenRouterAdapter{}
    })
}

func (a *OpenRouterAdapter) Initialize(config map[string]interface{}) error {
    // Extract config
    apiKeyEnv, ok := config["api_key_env"].(string)
    if !ok {
        return fmt.Errorf("missing api_key_env")
    }

    a.apiKey = os.Getenv(apiKeyEnv)
    if a.apiKey == "" {
        return fmt.Errorf("API key not found: %s", apiKeyEnv)
    }

    a.model, _ = config["model"].(string)
    a.temperature, _ = config["temperature"].(float64)
    a.maxTokens, _ = config["max_tokens"].(int)

    // Create HTTP client (reuse v1 client)
    a.client = client.NewOpenAICompatClient(
        "https://openrouter.ai/api/v1",
        a.apiKey,
    )

    return nil
}

func (a *OpenRouterAdapter) SendMessage(ctx context.Context, messages []core.Message) (string, error) {
    // Convert to API format and call
    // (Implementation reuses v1 logic)
    return a.client.SendMessage(ctx, a.model, messages)
}

func (a *OpenRouterAdapter) StreamMessage(ctx context.Context, messages []core.Message, writer io.Writer) error {
    // Stream response using SSE
    return a.client.StreamMessage(ctx, a.model, messages, writer)
}

func (a *OpenRouterAdapter) IsAvailable() bool {
    return a.apiKey != ""
}

func (a *OpenRouterAdapter) GetModel() string {
    return a.model
}

func (a *OpenRouterAdapter) HealthCheck(ctx context.Context) error {
    // Test API call
    _, err := a.SendMessage(ctx, []core.Message{
        {Role: "user", Content: "test"},
    })
    return err
}
```

**Deliverable (Week 1)**:
- ✅ Core types defined and tested
- ✅ Event bus working with tests
- ✅ 2+ adapters implemented (OpenRouter, Claude API)
- ✅ Adapter registry working

---

## Week 2: Conversation Engine

### Task 1: Conversation Manager (Days 8-10)

**File**: `pkg/v2/manager/manager.go`

```go
package manager

import (
    "context"
    "sync"

    "github.com/ASRagab/agentpipe/pkg/v2/core"
    "github.com/ASRagab/agentpipe/pkg/v2/events"
    "github.com/ASRagab/agentpipe/pkg/v2/pool"
)

type ConversationManager struct {
    config       Config
    conversation *core.Conversation
    agents       []core.Agent
    eventBus     events.EventBus
    agentPool    pool.AgentPool
    mu           sync.RWMutex
}

type Config struct {
    Timeout time.Duration
    SaveDir string
}

func NewConversationManager(cfg Config, agents []core.Agent, bus events.EventBus) *ConversationManager {
    mgr := &ConversationManager{
        config: cfg,
        conversation: &core.Conversation{
            ID:       generateID(),
            Messages: make([]core.Message, 0),
            Agents:   agents,
            Started:  time.Now(),
            Status:   "active",
        },
        agents:   agents,
        eventBus: bus,
    }

    mgr.agentPool = pool.NewAgentPool(agents, bus)

    return mgr
}

func (m *ConversationManager) SendUserMessage(ctx context.Context, content string) error {
    // Create user message
    msg := core.NewUserMessage(content)

    // Add to conversation
    m.mu.Lock()
    m.conversation.Messages = append(m.conversation.Messages, msg)
    m.mu.Unlock()

    // Emit event
    m.eventBus.Publish(core.Event{
        Type:      events.EventMessageCreated,
        Timestamp: time.Now(),
        Data:      msg,
    })

    // Execute agents in parallel
    responses := m.agentPool.ExecuteParallel(ctx, m.GetMessages())

    // Process responses
    for _, resp := range responses {
        if resp.Error != nil {
            // Handle error
            continue
        }

        // Add agent message to conversation
        m.mu.Lock()
        m.conversation.Messages = append(m.conversation.Messages, *resp.Message)
        m.mu.Unlock()

        // Event already emitted by agent pool
    }

    return nil
}

func (m *ConversationManager) GetMessages() []core.Message {
    m.mu.RLock()
    defer m.mu.RUnlock()

    messages := make([]core.Message, len(m.conversation.Messages))
    copy(messages, m.conversation.Messages)
    return messages
}

// ... more methods
```

**Test**: Integration test with mock agents

```go
func TestConversationManagerParallelExecution(t *testing.T) {
    bus := events.NewEventBus()
    defer bus.Close()

    // Create mock agents
    agents := []core.Agent{
        {ID: "agent1", adapter: &MockAdapter{response: "Response 1"}},
        {ID: "agent2", adapter: &MockAdapter{response: "Response 2"}},
    }

    mgr := NewConversationManager(Config{}, agents, bus)

    // Send user message
    err := mgr.SendUserMessage(context.Background(), "Hello")
    assert.NoError(t, err)

    // Verify responses
    messages := mgr.GetMessages()
    assert.Len(t, messages, 3) // 1 user + 2 agents
}
```

---

## Week 3: TUI Interface

### Task 1: Basic TUI (Days 15-17)

**File**: `pkg/v2/tui/tui.go`

```go
package tui

import (
    "github.com/charmbracelet/bubbletea"
    "github.com/ASRagab/agentpipe/pkg/v2/core"
    "github.com/ASRagab/agentpipe/pkg/v2/events"
    "github.com/ASRagab/agentpipe/pkg/v2/manager"
)

type Model struct {
    manager  *manager.ConversationManager
    eventBus events.EventBus
    messages []core.Message
    // ... TUI state
}

func New(mgr *manager.ConversationManager, bus events.EventBus) Model {
    model := Model{
        manager:  mgr,
        eventBus: bus,
        messages: make([]core.Message, 0),
    }

    // Subscribe to events
    bus.Subscribe(events.EventMessageCreated, model.onMessageCreated)
    bus.Subscribe(events.EventAgentTyping, model.onAgentTyping)
    bus.Subscribe(events.EventAgentDone, model.onAgentDone)

    return model
}

func (m Model) Init() tea.Cmd {
    return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Handle bubbletea messages
    // Update UI based on events
    return m, nil
}

func (m Model) View() string {
    // Render three-panel layout
    return ""
}

func (m *Model) onMessageCreated(e core.Event) {
    msg := e.Data.(core.Message)
    m.messages = append(m.messages, msg)
    // Trigger TUI update
}

// ... more event handlers
```

---

## Common Patterns

### Pattern 1: Thread-Safe State Management

```go
type Manager struct {
    mu    sync.RWMutex
    state map[string]interface{}
}

func (m *Manager) Get(key string) interface{} {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.state[key]
}

func (m *Manager) Set(key string, value interface{}) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.state[key] = value
}
```

### Pattern 2: Context with Timeout

```go
func (a *Adapter) SendMessage(ctx context.Context, messages []Message) (string, error) {
    // Add timeout if not already set
    if _, hasDeadline := ctx.Deadline(); !hasDeadline {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
        defer cancel()
    }

    // Make API call with context
    return a.client.Call(ctx, messages)
}
```

### Pattern 3: Graceful Shutdown

```go
func (m *Manager) Shutdown() error {
    // Cancel all ongoing operations
    m.cancel()

    // Wait for goroutines with timeout
    done := make(chan struct{})
    go func() {
        m.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        return nil
    case <-time.After(5 * time.Second):
        return fmt.Errorf("shutdown timeout")
    }
}
```

---

## Debugging Tips

### Enable Verbose Logging

```go
// In main.go
log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
log.SetOutput(os.Stderr)

// Use structured logging
log.Printf("[%s] Message: %s", event.Type, event.Data)
```

### Debug Event Bus

```go
// Subscribe to all events for debugging
bus.SubscribeAll(func(e core.Event) {
    log.Printf("Event: %s, Data: %+v", e.Type, e.Data)
})
```

### Profile Performance

```go
import _ "net/http/pprof"

go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()

// Visit http://localhost:6060/debug/pprof/
```

### Detect Race Conditions

```bash
# Run with race detector
go test -race ./...

# Run with verbose output
go test -race -v ./pkg/v2/events
```

---

## Resources

- **Documentation**: `docs/v2-mvp-implementation-plan.md`
- **Architecture**: `docs/v2-architecture-diagram.md`
- **Interfaces**: `docs/v2-core-interfaces.md`
- **Migration**: `docs/v2-migration-guide.md`

- **Go Concurrency**: https://go.dev/tour/concurrency
- **Bubbletea**: https://github.com/charmbracelet/bubbletea
- **Testing**: https://go.dev/doc/tutorial/add-a-test

---

## Getting Help

- **GitHub Issues**: https://github.com/ASRagab/agentpipe/issues
- **Discussions**: https://github.com/ASRagab/agentpipe/discussions
- **Code Review**: Open a draft PR for early feedback

---

## Summary

**Start Here**:
1. Read the MVP plan and architecture docs
2. Set up development environment
3. Pick a task from the timeline (Week 1, 2, 3, or 4)
4. Write tests first (TDD)
5. Implement feature
6. Run tests, lint, format
7. Submit PR

**Key Principles**:
- **Test-driven**: Write tests before implementation
- **Event-driven**: Use event bus for all communication
- **Concurrent**: Use goroutines for parallel execution
- **Simple**: Start with the simplest solution that works

**Next Steps**:
- Join Week 1 kickoff
- Claim a task from the timeline
- Set up development environment
- Start coding!

Let's build AgentPipe v2!
