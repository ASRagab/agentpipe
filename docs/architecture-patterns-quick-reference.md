# Architecture Patterns Quick Reference

**For**: AgentPipe Development Team
**Date**: 2026-01-18
**Source**: Research on Modern Go Application Architecture

---

## Pattern Catalog

### ✅ Currently Implemented (Well Done!)

| Pattern | Location | Use Case | Quality |
|---------|----------|----------|---------|
| **Interface-Driven Design** | `pkg/agent/agent.go` | 15+ agent adapters | A+ |
| **Middleware Chain** | `pkg/middleware/` | Message processing pipeline | A+ |
| **Factory Pattern** | `pkg/adapters/` | Agent creation | A |
| **Repository Pattern** | `internal/registry/`, `internal/providers/` | Data access | A |
| **Strategy Pattern** | `pkg/orchestrator/` | 3 conversation modes | A |
| **Observer Pattern** | `pkg/orchestrator/` | Artifact callbacks, TUI channels | A |
| **Builder Pattern** | `pkg/adapters/*/buildPrompt()` | Structured prompts | B+ |
| **Hexagonal Architecture** | `pkg/agent/` (port), `pkg/adapters/` (adapters) | Clean separation | A |

### 🔄 Should Add (High Impact)

| Pattern | Where to Add | Benefit | Priority |
|---------|--------------|---------|----------|
| **Worker Pool** | `pkg/orchestrator/pool.go` | Parallel agent execution | HIGH |
| **Errgroup** | `pkg/orchestrator/modes.go` | Structured concurrency | HIGH |
| **Circuit Breaker** | `pkg/resilience/breaker.go` | Failing agent isolation | MED |
| **Pipeline** | `pkg/pipeline/pipeline.go` | Clear message flow | MED |
| **State Machine** | `pkg/tui/state/` | TUI state management | MED |
| **Event Bus** | `pkg/events/bus.go` | Loose coupling | LOW |

---

## Code Examples

### Worker Pool Pattern

```go
// pkg/orchestrator/pool.go
type WorkerPool struct {
    maxWorkers int
    tasks      chan func()
    wg         sync.WaitGroup
}

func NewWorkerPool(maxWorkers int) *WorkerPool {
    p := &WorkerPool{
        maxWorkers: maxWorkers,
        tasks:      make(chan func(), 100),
    }

    for i := 0; i < maxWorkers; i++ {
        p.wg.Add(1)
        go p.worker()
    }

    return p
}

func (p *WorkerPool) worker() {
    defer p.wg.Done()
    for task := range p.tasks {
        task()
    }
}

func (p *WorkerPool) Submit(task func()) {
    p.tasks <- task
}

func (p *WorkerPool) Close() {
    close(p.tasks)
    p.wg.Wait()
}
```

**Usage in Orchestrator**:
```go
// In runFreeForm or runRoundRobin
pool := NewWorkerPool(5) // Max 5 concurrent agents

for _, agent := range o.agents {
    a := agent
    pool.Submit(func() {
        o.getAgentResponse(ctx, a)
    })
}

pool.Close() // Wait for all to complete
```

---

### Errgroup for Structured Concurrency

```go
import "golang.org/x/sync/errgroup"

// In runFreeForm
func (o *Orchestrator) runFreeForm(ctx context.Context) error {
    g, ctx := errgroup.WithContext(ctx)

    for _, agent := range o.agents {
        a := agent // Capture
        g.Go(func() error {
            return o.getAgentResponse(ctx, a)
        })
    }

    // Wait for all agents, return first error
    return g.Wait()
}
```

**Benefits**:
- Automatic cancellation on first error
- Clean error aggregation
- Context propagation

---

### Circuit Breaker Pattern

```go
// pkg/resilience/breaker.go
type State int

const (
    StateClosed State = iota // Normal operation
    StateOpen                 // Failing, reject requests
    StateHalfOpen            // Testing recovery
)

type CircuitBreaker struct {
    maxFailures int
    timeout     time.Duration
    failures    int
    lastFailure time.Time
    state       State
    mu          sync.Mutex
}

func (cb *CircuitBreaker) Call(fn func() error) error {
    cb.mu.Lock()
    state := cb.getState()
    cb.mu.Unlock()

    if state == StateOpen {
        return ErrCircuitOpen
    }

    err := fn()

    cb.mu.Lock()
    defer cb.mu.Unlock()

    if err != nil {
        cb.failures++
        cb.lastFailure = time.Now()
        if cb.failures >= cb.maxFailures {
            cb.state = StateOpen
        }
        return err
    }

    // Success
    cb.failures = 0
    cb.state = StateClosed
    return nil
}

func (cb *CircuitBreaker) getState() State {
    if cb.state == StateOpen {
        if time.Since(cb.lastFailure) > cb.timeout {
            cb.state = StateHalfOpen
        }
    }
    return cb.state
}
```

**Usage**:
```go
// Wrap agent calls
breaker := NewCircuitBreaker(3, 5*time.Minute)
err := breaker.Call(func() error {
    _, err := agent.SendMessage(ctx, messages)
    return err
})

if errors.Is(err, ErrCircuitOpen) {
    log.Warn("Circuit breaker open, skipping agent")
}
```

---

### Pipeline Pattern

```go
// pkg/pipeline/pipeline.go
type Stage func(context.Context, *agent.Message) (*agent.Message, error)

type Pipeline struct {
    stages []Stage
}

func NewPipeline(stages ...Stage) *Pipeline {
    return &Pipeline{stages: stages}
}

func (p *Pipeline) Process(ctx context.Context, msg *agent.Message) (*agent.Message, error) {
    var err error
    for _, stage := range p.stages {
        msg, err = stage(ctx, msg)
        if err != nil {
            return nil, err
        }
    }
    return msg, nil
}

// Stages
func ValidateStage(ctx context.Context, msg *agent.Message) (*agent.Message, error) {
    if len(msg.Content) == 0 {
        return nil, errors.New("empty message")
    }
    return msg, nil
}

func TransformStage(ctx context.Context, msg *agent.Message) (*agent.Message, error) {
    msg.Content = strings.TrimSpace(msg.Content)
    return msg, nil
}

func EnrichStage(ctx context.Context, msg *agent.Message) (*agent.Message, error) {
    if msg.Metrics == nil {
        msg.Metrics = &agent.ResponseMetrics{}
    }
    return msg, nil
}
```

**Usage**:
```go
pipeline := NewPipeline(
    ValidateStage,
    TransformStage,
    EnrichStage,
)

processedMsg, err := pipeline.Process(ctx, msg)
```

---

### State Machine for TUI

```go
// pkg/tui/state/machine.go
type State int

const (
    StateInitializing State = iota
    StateWaitingForAgent
    StateUserTurn
    StateProcessing
    StateCompleted
    StateError
)

type StateMachine struct {
    current State
    mu      sync.RWMutex
}

func (sm *StateMachine) Transition(to State) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    // Validate transition
    if !sm.isValidTransition(sm.current, to) {
        return fmt.Errorf("invalid transition: %v -> %v", sm.current, to)
    }

    log.WithFields(map[string]interface{}{
        "from": sm.current,
        "to":   to,
    }).Debug("state transition")

    sm.current = to
    return nil
}

func (sm *StateMachine) isValidTransition(from, to State) bool {
    validTransitions := map[State][]State{
        StateInitializing:    {StateWaitingForAgent, StateError},
        StateWaitingForAgent: {StateProcessing, StateUserTurn, StateCompleted},
        StateUserTurn:        {StateProcessing},
        StateProcessing:      {StateWaitingForAgent, StateUserTurn, StateCompleted, StateError},
        StateCompleted:       {},
        StateError:           {StateInitializing},
    }

    allowed := validTransitions[from]
    for _, s := range allowed {
        if s == to {
            return true
        }
    }
    return false
}

func (sm *StateMachine) Current() State {
    sm.mu.RLock()
    defer sm.mu.RUnlock()
    return sm.current
}
```

---

## Refactoring Guide

### Orchestrator Splitting

**Current**: `orchestrator.go` (1297 lines)

**Split into**:

```
pkg/orchestrator/
├── orchestrator.go      # Core struct, initialization (200 lines)
│   - NewOrchestrator()
│   - AddAgent()
│   - SetLogger(), SetMetrics(), SetBridgeEmitter()
│
├── modes.go             # Conversation modes (300 lines)
│   - runRoundRobin()
│   - runReactive()
│   - runFreeForm()
│
├── retry.go             # Retry logic (100 lines)
│   - calculateBackoffDelay()
│   - retryWithBackoff()
│
├── artifacts.go         # Artifact handling (150 lines)
│   - processArtifacts()
│   - isFirstTurnForAgent()
│
├── summary.go           # Summary generation (200 lines)
│   - generateSummary()
│   - parseDualSummary()
│
├── agent_response.go    # Agent communication (200 lines)
│   - getAgentResponse()
│   - sendWithRetry()
│
└── helpers.go           # Utility functions (147 lines)
    - getMessages()
    - selectNextAgent()
    - shouldRespond()
```

**Migration Steps**:
1. Create new files with package declarations
2. Move functions to appropriate files
3. Keep all exported (public) in orchestrator.go
4. Make mode functions methods (keep orchestrator.go small)
5. Run tests after each file split
6. Update imports

---

### TUI Component Extraction

**Current**: `enhanced.go` (large file)

**Split into**:

```
pkg/tui/
├── enhanced.go          # Main coordinator
│   - NewEnhancedModel()
│   - Init(), Update(), View()
│   - Delegation to components
│
├── components/
│   ├── agent_list.go    # Agent list panel
│   │   - AgentListComponent
│   │   - Update(), View()
│   │
│   ├── conversation.go  # Conversation viewport
│   │   - ConversationComponent
│   │   - Update(), View()
│   │
│   ├── input.go         # User input panel
│   │   - InputComponent
│   │   - Update(), View()
│   │
│   └── modal.go         # Modal dialogs
│       - ModalComponent
│       - Update(), View()
│
├── state/
│   ├── model.go         # Core model
│   │   - EnhancedModel struct
│   │
│   └── messages.go      # Message types
│       - ArtifactSavedMsg
│       - ConversationMsg
│
└── styles.go            # Lipgloss styles
    - All lipgloss.Style definitions
```

**Migration Steps**:
1. Create component interface
2. Extract agent list component first (smallest)
3. Extract conversation component
4. Extract input component
5. Extract modal component
6. Update enhanced.go to delegate
7. Run TUI integration tests

---

## Testing Patterns

### Golden File Testing (for TUI)

```go
// pkg/tui/tui_test.go
import "github.com/sebdah/goldie/v2"

func TestTUIRender(t *testing.T) {
    tests := []struct {
        name  string
        state TUIState
    }{
        {"agent_list", TUIState{activePanel: agentsPanel}},
        {"conversation", TUIState{activePanel: conversationPanel}},
        {"modal", TUIState{showModal: true, modalContent: "Test"}},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            model := initModelWithState(tt.state)
            output := model.View()

            g := goldie.New(t)
            g.Assert(t, tt.name, []byte(output))
        })
    }
}

// Update golden files: go test -update
```

**Golden file**: `testdata/TestTUIRender/agent_list.golden`

---

### Contract Testing (for API Adapters)

```go
// pkg/adapters/openrouter_test.go
func TestOpenRouterContract(t *testing.T) {
    contract := OpenRouterContract{
        Endpoint: "https://openrouter.ai/api/v1/chat/completions",
        Method:   "POST",
        Headers: map[string]string{
            "Authorization": "Bearer sk-...",
            "Content-Type":  "application/json",
        },
        RequestSchema: loadSchema("openrouter_request.json"),
        ResponseSchema: loadSchema("openrouter_response.json"),
    }

    adapter := NewOpenRouterAgent()
    verifyContract(t, adapter, contract)
}

func verifyContract(t *testing.T, adapter agent.Agent, contract Contract) {
    // Intercept HTTP request
    client := &http.Client{
        Transport: &RecordingTransport{},
    }

    // Send test message
    adapter.SendMessage(context.Background(), testMessages)

    // Verify request matches contract
    assert.Equal(t, contract.Endpoint, recordedRequest.URL.String())
    assert.Equal(t, contract.Method, recordedRequest.Method)

    // Validate JSON schema
    validateJSON(t, recordedRequest.Body, contract.RequestSchema)
}
```

---

## Performance Patterns

### Connection Pooling

```go
// pkg/client/http_pool.go
var defaultHTTPClient = &http.Client{
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
        TLSHandshakeTimeout: 10 * time.Second,
    },
    Timeout: 30 * time.Second,
}

// Use in adapters
func (o *OpenRouterAdapter) SendMessage(ctx context.Context, messages []agent.Message) (string, error) {
    req, _ := http.NewRequestWithContext(ctx, "POST", apiURL, body)
    resp, err := defaultHTTPClient.Do(req) // Reuse pool
    // ...
}
```

---

### Response Caching

```go
// pkg/cache/response_cache.go
type ResponseCache struct {
    cache map[string]CacheEntry
    ttl   time.Duration
    mu    sync.RWMutex
}

type CacheEntry struct {
    Response  string
    Timestamp time.Time
    Metrics   *agent.ResponseMetrics
}

func (c *ResponseCache) Get(key string) (string, *agent.ResponseMetrics, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    entry, ok := c.cache[key]
    if !ok || time.Since(entry.Timestamp) > c.ttl {
        return "", nil, false
    }

    return entry.Response, entry.Metrics, true
}

func (c *ResponseCache) Set(key, response string, metrics *agent.ResponseMetrics) {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.cache[key] = CacheEntry{
        Response:  response,
        Timestamp: time.Now(),
        Metrics:   metrics,
    }
}

// Cache key generation
func generateCacheKey(agentID string, messages []agent.Message) string {
    h := sha256.New()
    h.Write([]byte(agentID))
    for _, msg := range messages {
        h.Write([]byte(msg.Content))
    }
    return hex.EncodeToString(h.Sum(nil))
}
```

---

## Security Patterns

### Secret Management (OS Keychain)

```go
// pkg/config/secrets.go
import "github.com/zalando/go-keyring"

const serviceName = "agentpipe"

func GetAPIKey(provider string) (string, error) {
    // Try keychain first
    key, err := keyring.Get(serviceName, provider)
    if err == nil {
        return key, nil
    }

    // Fallback to environment variable
    envKey := provider + "_API_KEY"
    key = os.Getenv(envKey)
    if key == "" {
        return "", fmt.Errorf("API key not found for %s", provider)
    }

    return key, nil
}

func SetAPIKey(provider, key string) error {
    return keyring.Set(serviceName, provider, key)
}

func DeleteAPIKey(provider string) error {
    return keyring.Delete(serviceName, provider)
}
```

**Usage**:
```go
// Instead of: os.Getenv("OPENROUTER_API_KEY")
apiKey, err := secrets.GetAPIKey("openrouter")
```

---

### Input Validation Middleware

```go
// pkg/middleware/validation.go
func SecurityValidationMiddleware() Middleware {
    return NewValidationMiddleware("security", func(ctx *MessageContext, msg *agent.Message) error {
        // Length check
        if len(msg.Content) > 100000 {
            return errors.New("message too long (max 100KB)")
        }

        // SQL injection patterns
        sqlPatterns := []string{
            `(?i)(\bDROP\s+TABLE\b)`,
            `(?i)(\bDELETE\s+FROM\b)`,
            `(?i)(\b--\s*$)`,
            `(?i)(\bUNION\s+SELECT\b)`,
        }

        for _, pattern := range sqlPatterns {
            if matched, _ := regexp.MatchString(pattern, msg.Content); matched {
                return errors.New("potential SQL injection detected")
            }
        }

        // Path traversal patterns
        if strings.Contains(msg.Content, "../") || strings.Contains(msg.Content, "..\\") {
            return errors.New("potential path traversal detected")
        }

        return nil
    })
}
```

---

## Quick Decision Tree

### When to Use Worker Pool?

```
Need parallel agent execution?
├─ Yes → Number of agents?
│  ├─ 1-3 → Use errgroup (simpler)
│  ├─ 4-10 → Use worker pool (controlled concurrency)
│  └─ 10+ → Use worker pool + bulkhead (isolation by type)
└─ No → Sequential execution is fine
```

### When to Use Circuit Breaker?

```
Agent failures affecting others?
├─ Yes → Frequency of failures?
│  ├─ Rare (< 1%) → Retry logic sufficient
│  ├─ Occasional (1-10%) → Add circuit breaker
│  └─ Frequent (> 10%) → Investigate root cause first
└─ No → Not needed
```

### When to Use Caching?

```
Repeated identical requests?
├─ Yes → Request frequency?
│  ├─ Low (< 10/min) → May not be worth complexity
│  ├─ Medium (10-100/min) → Cache with 5-min TTL
│  └─ High (> 100/min) → Cache with smart invalidation
└─ No → Skip caching
```

---

## Resources

- Full Research: `docs/architecture-research-modern-patterns.md`
- Summary: `docs/research-summary.txt`
- Go Patterns: https://github.com/tmrts/go-patterns
- Concurrency: https://blog.golang.org/pipelines
- Testing: https://github.com/golang/go/wiki/TableDrivenTests

---

**Last Updated**: 2026-01-18
**Maintained By**: AgentPipe Development Team
