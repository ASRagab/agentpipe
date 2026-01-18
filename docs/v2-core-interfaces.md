# AgentPipe v2 Core Interfaces Reference

## Overview

This document defines the essential interfaces and types for AgentPipe v2 MVP. These are the contracts that all implementations must satisfy.

---

## 1. Core Data Types

### 1.1 Message

```go
// Message represents any message in the conversation (user, agent, or system)
type Message struct {
    // ID is a unique identifier for this message
    ID string `json:"id"`

    // Timestamp is when the message was created
    Timestamp time.Time `json:"timestamp"`

    // Role indicates who sent the message: "user", "agent", or "system"
    Role string `json:"role"`

    // AgentID is the ID of the agent (empty for user/system messages)
    AgentID string `json:"agent_id,omitempty"`

    // AgentName is the display name of the agent (empty for user/system)
    AgentName string `json:"agent_name,omitempty"`

    // Content is the actual message text
    Content string `json:"content"`

    // Status tracks the message state: "pending", "streaming", "complete", "error"
    Status string `json:"status"`

    // Metrics contains performance data (only for agent messages)
    Metrics *Metrics `json:"metrics,omitempty"`
}

// Example: User message
userMsg := Message{
    ID:        uuid.New().String(),
    Timestamp: time.Now(),
    Role:      "user",
    Content:   "Explain async/await in JavaScript",
    Status:    "complete",
}

// Example: Agent message with metrics
agentMsg := Message{
    ID:        uuid.New().String(),
    Timestamp: time.Now(),
    Role:      "agent",
    AgentID:   "claude",
    AgentName: "Claude",
    Content:   "Async/await is a modern syntax...",
    Status:    "complete",
    Metrics: &Metrics{
        Duration:     145 * time.Millisecond,
        InputTokens:  234,
        OutputTokens: 567,
        TotalTokens:  801,
        Model:        "claude-sonnet-4.5",
        Cost:         0.0125,
    },
}
```

### 1.2 Metrics

```go
// Metrics captures performance and cost information for agent responses
type Metrics struct {
    // Duration is how long the agent took to generate the response
    Duration time.Duration `json:"duration"`

    // InputTokens is the number of tokens in the input (prompt + history)
    InputTokens int `json:"input_tokens"`

    // OutputTokens is the number of tokens in the agent's response
    OutputTokens int `json:"output_tokens"`

    // TotalTokens is InputTokens + OutputTokens
    TotalTokens int `json:"total_tokens"`

    // Model is the specific model used (e.g., "claude-sonnet-4.5", "gpt-4")
    Model string `json:"model"`

    // Cost is the estimated monetary cost in USD
    Cost float64 `json:"cost"`
}
```

### 1.3 Agent

```go
// Agent represents a configured AI agent
type Agent struct {
    // ID is a unique identifier for this agent
    ID string `yaml:"id" json:"id"`

    // Type is either "api" or "cli"
    Type string `yaml:"type" json:"type"`

    // Name is the display name shown in the UI
    Name string `yaml:"name" json:"name"`

    // Model is the specific model to use (e.g., "claude-sonnet-4.5")
    Model string `yaml:"model" json:"model"`

    // AdapterName identifies which adapter to use (e.g., "openrouter", "claude-api")
    AdapterName string `yaml:"adapter" json:"adapter"`

    // Config contains adapter-specific configuration
    Config map[string]interface{} `yaml:"config" json:"config"`

    // adapter is the underlying adapter implementation (set at runtime)
    adapter AgentAdapter `yaml:"-" json:"-"`
}

// Example: OpenRouter agent
agent := Agent{
    ID:          "gpt4",
    Type:        "api",
    Name:        "GPT-4",
    Model:       "openai/gpt-4",
    AdapterName: "openrouter",
    Config: map[string]interface{}{
        "api_key_env": "OPENROUTER_API_KEY",
        "temperature": 0.7,
        "max_tokens":  2000,
    },
}
```

### 1.4 Conversation

```go
// Conversation represents the current conversation state
type Conversation struct {
    // ID is a unique identifier for this conversation
    ID string `json:"id"`

    // Messages contains all messages in chronological order
    Messages []Message `json:"messages"`

    // Agents contains all participating agents
    Agents []Agent `json:"agents"`

    // Started is when the conversation began
    Started time.Time `json:"started"`

    // Updated is when the conversation was last modified
    Updated time.Time `json:"updated"`

    // Status is the conversation state: "active", "paused", "completed"
    Status string `json:"status"`

    // Metadata contains optional conversation metadata
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}
```

### 1.5 Event

```go
// Event represents an event on the event bus
type Event struct {
    // Type identifies the event (e.g., "message.created", "agent.typing")
    Type string `json:"type"`

    // Timestamp is when the event occurred
    Timestamp time.Time `json:"timestamp"`

    // Data is the event-specific payload (type varies by event)
    Data interface{} `json:"data"`
}

// Event types (constants)
const (
    EventMessageCreated     = "message.created"      // User/agent/system message added
    EventMessageChunk       = "message.chunk"        // Streaming response chunk
    EventAgentTyping        = "agent.typing"         // Agent started generating
    EventAgentDone          = "agent.done"           // Agent finished generating
    EventAgentError         = "agent.error"          // Agent failed to respond
    EventConversationStarted = "conversation.started" // Conversation began
    EventConversationSaved  = "conversation.saved"   // Conversation persisted
)

// Example: Message created event
event := Event{
    Type:      EventMessageCreated,
    Timestamp: time.Now(),
    Data: Message{
        ID:      "msg-123",
        Role:    "user",
        Content: "Hello",
        Status:  "complete",
    },
}

// Example: Agent typing event
event := Event{
    Type:      EventAgentTyping,
    Timestamp: time.Now(),
    Data: map[string]string{
        "agent_id":   "claude",
        "agent_name": "Claude",
    },
}

// Example: Message chunk event (streaming)
event := Event{
    Type:      EventMessageChunk,
    Timestamp: time.Now(),
    Data: MessageChunk{
        MessageID: "msg-456",
        AgentID:   "claude",
        Chunk:     "Async/await is...",
    },
}
```

---

## 2. Core Interfaces

### 2.1 AgentAdapter

```go
// AgentAdapter handles communication with a specific agent implementation
// Implementations: OpenRouterAdapter, ClaudeAPIAdapter, GeminiCLIAdapter
type AgentAdapter interface {
    // Initialize configures the adapter with the provided config
    // Config keys are adapter-specific (e.g., "api_key_env", "temperature")
    // Returns error if configuration is invalid
    Initialize(config map[string]interface{}) error

    // SendMessage sends messages to the agent and returns the full response
    // This is a blocking call that waits for the complete response
    // Used for non-streaming interactions
    SendMessage(ctx context.Context, messages []Message) (string, error)

    // StreamMessage sends messages to the agent and streams the response
    // Chunks are written to the provided writer as they arrive
    // Used for real-time streaming interactions
    StreamMessage(ctx context.Context, messages []Message, writer io.Writer) error

    // IsAvailable checks if the agent is ready to accept requests
    // For API adapters: checks if API key is configured
    // For CLI adapters: checks if binary is installed and accessible
    IsAvailable() bool

    // GetModel returns the model identifier (e.g., "claude-sonnet-4.5")
    GetModel() string

    // HealthCheck performs a comprehensive health check
    // For API adapters: makes a test API call
    // For CLI adapters: executes a test command
    HealthCheck(ctx context.Context) error
}

// Example implementation signature (OpenRouter):
type OpenRouterAdapter struct {
    apiKey      string
    model       string
    temperature float64
    maxTokens   int
    httpClient  *http.Client
}

func (a *OpenRouterAdapter) Initialize(config map[string]interface{}) error {
    // Extract config values
    apiKeyEnv, ok := config["api_key_env"].(string)
    if !ok {
        return fmt.Errorf("missing api_key_env in config")
    }
    a.apiKey = os.Getenv(apiKeyEnv)
    if a.apiKey == "" {
        return fmt.Errorf("API key not found in environment: %s", apiKeyEnv)
    }

    a.model, _ = config["model"].(string)
    a.temperature, _ = config["temperature"].(float64)
    a.maxTokens, _ = config["max_tokens"].(int)

    return nil
}

func (a *OpenRouterAdapter) SendMessage(ctx context.Context, messages []Message) (string, error) {
    // Convert messages to API format
    // Make HTTP request
    // Parse response
    // Return content
}

func (a *OpenRouterAdapter) StreamMessage(ctx context.Context, messages []Message, writer io.Writer) error {
    // Convert messages to API format
    // Make streaming HTTP request (SSE)
    // Write chunks to writer as they arrive
    // Return error on failure
}
```

### 2.2 ConversationManager

```go
// ConversationManager orchestrates the conversation flow
type ConversationManager interface {
    // Start begins a new conversation with the configured agents
    // Blocks until conversation completes or context is canceled
    Start(ctx context.Context) error

    // SendUserMessage adds a user message to the conversation and triggers agent responses
    // This is the main entry point for user input
    SendUserMessage(ctx context.Context, content string) error

    // GetMessages returns all messages in the conversation
    // Returns a copy of the messages slice (thread-safe)
    GetMessages() []Message

    // GetConversation returns the full conversation state
    GetConversation() *Conversation

    // Subscribe registers a handler for a specific event type
    // Returns an unsubscribe function
    Subscribe(eventType string, handler func(Event)) func()

    // Pause pauses the conversation (stops agent responses)
    Pause()

    // Resume resumes a paused conversation
    Resume()

    // Stop stops the conversation completely
    Stop()
}

// Example usage:
manager := NewConversationManager(config, agents, eventBus)

// Subscribe to message events
unsubscribe := manager.Subscribe(EventMessageCreated, func(e Event) {
    msg := e.Data.(Message)
    fmt.Printf("[%s] %s\n", msg.Role, msg.Content)
})
defer unsubscribe()

// Start conversation in background
go manager.Start(ctx)

// Send user message
err := manager.SendUserMessage(ctx, "Explain async/await")
```

### 2.3 EventBus

```go
// EventBus provides publish/subscribe event distribution
type EventBus interface {
    // Publish sends an event to all subscribers
    // Non-blocking: handlers are called in goroutines
    Publish(event Event)

    // Subscribe registers a handler for a specific event type
    // Returns an unsubscribe function to remove the handler
    Subscribe(eventType string, handler func(Event)) func()

    // SubscribeAll registers a handler for all event types
    // Useful for logging or debugging
    SubscribeAll(handler func(Event)) func()

    // Close shuts down the event bus and waits for all handlers to complete
    Close() error
}

// Example implementation:
type eventBus struct {
    mu          sync.RWMutex
    subscribers map[string][]func(Event) // eventType -> handlers
    wg          sync.WaitGroup            // tracks active handlers
}

func (b *eventBus) Publish(event Event) {
    b.mu.RLock()
    handlers := b.subscribers[event.Type]
    b.mu.RUnlock()

    // Call handlers in goroutines (non-blocking)
    for _, handler := range handlers {
        b.wg.Add(1)
        go func(h func(Event)) {
            defer b.wg.Done()
            defer func() {
                if r := recover(); r != nil {
                    log.Printf("event handler panic: %v", r)
                }
            }()
            h(event)
        }(handler)
    }
}

func (b *eventBus) Subscribe(eventType string, handler func(Event)) func() {
    b.mu.Lock()
    defer b.mu.Unlock()

    b.subscribers[eventType] = append(b.subscribers[eventType], handler)

    // Return unsubscribe function
    return func() {
        b.mu.Lock()
        defer b.mu.Unlock()
        handlers := b.subscribers[eventType]
        for i, h := range handlers {
            if &h == &handler {
                b.subscribers[eventType] = append(handlers[:i], handlers[i+1:]...)
                break
            }
        }
    }
}

func (b *eventBus) Close() error {
    b.wg.Wait() // Wait for all handlers to complete
    return nil
}
```

### 2.4 AgentPool

```go
// AgentPool manages parallel execution of agent requests
type AgentPool interface {
    // ExecuteParallel sends messages to all agents concurrently
    // Returns when all agents complete (or timeout)
    // Failed agents don't block successful ones
    ExecuteParallel(ctx context.Context, messages []Message) []Response

    // GetStatus returns the current status of each agent
    // Keys: agent IDs, Values: "pending", "typing", "done", "error"
    GetStatus() map[string]string

    // CancelAgent cancels a specific agent's request (e.g., if taking too long)
    CancelAgent(agentID string)
}

// Response represents an agent's response (or error)
type Response struct {
    AgentID   string
    AgentName string
    Message   *Message // nil if error occurred
    Error     error    // nil if successful
}

// Example implementation:
func (p *pool) ExecuteParallel(ctx context.Context, messages []Message) []Response {
    var wg sync.WaitGroup
    responses := make([]Response, len(p.agents))

    for i, agent := range p.agents {
        wg.Add(1)
        go func(idx int, a Agent) {
            defer wg.Done()

            // Emit typing event
            p.eventBus.Publish(Event{
                Type: EventAgentTyping,
                Data: map[string]string{
                    "agent_id": a.ID,
                    "agent_name": a.Name,
                },
            })

            // Execute agent
            startTime := time.Now()
            content, err := a.adapter.SendMessage(ctx, messages)
            duration := time.Since(startTime)

            if err != nil {
                responses[idx] = Response{
                    AgentID:   a.ID,
                    AgentName: a.Name,
                    Error:     err,
                }

                // Emit error event
                p.eventBus.Publish(Event{
                    Type: EventAgentError,
                    Data: map[string]string{
                        "agent_id": a.ID,
                        "error":    err.Error(),
                    },
                })
                return
            }

            // Create message with metrics
            msg := Message{
                ID:        uuid.New().String(),
                Timestamp: time.Now(),
                Role:      "agent",
                AgentID:   a.ID,
                AgentName: a.Name,
                Content:   content,
                Status:    "complete",
                Metrics: &Metrics{
                    Duration:    duration,
                    InputTokens: estimateTokens(messages),
                    OutputTokens: estimateTokens(content),
                    Model:       a.Model,
                    Cost:        calculateCost(a.Model, inputTokens, outputTokens),
                },
            }

            responses[idx] = Response{
                AgentID:   a.ID,
                AgentName: a.Name,
                Message:   &msg,
            }

            // Emit done event
            p.eventBus.Publish(Event{
                Type: EventAgentDone,
                Data: msg,
            })
        }(i, agent)
    }

    wg.Wait()
    return responses
}
```

---

## 3. Adapter Registry

```go
// AdapterRegistry manages available agent adapters
type AdapterRegistry struct {
    adapters map[string]func() AgentAdapter // adapterName -> factory
}

// Global registry
var DefaultRegistry = &AdapterRegistry{
    adapters: make(map[string]func() AgentAdapter),
}

// Register adds an adapter factory to the registry
func (r *AdapterRegistry) Register(name string, factory func() AgentAdapter) {
    r.adapters[name] = factory
}

// Get retrieves and initializes an adapter by name
func (r *AdapterRegistry) Get(name string, config map[string]interface{}) (AgentAdapter, error) {
    factory, ok := r.adapters[name]
    if !ok {
        return nil, fmt.Errorf("unknown adapter: %s", name)
    }

    adapter := factory()
    if err := adapter.Initialize(config); err != nil {
        return nil, fmt.Errorf("adapter initialization failed: %w", err)
    }

    return adapter, nil
}

// List returns all registered adapter names
func (r *AdapterRegistry) List() []string {
    names := make([]string, 0, len(r.adapters))
    for name := range r.adapters {
        names = append(names, name)
    }
    sort.Strings(names)
    return names
}

// Example: Registering adapters (in init() functions)
func init() {
    DefaultRegistry.Register("openrouter", func() AgentAdapter {
        return &OpenRouterAdapter{}
    })

    DefaultRegistry.Register("claude-api", func() AgentAdapter {
        return &ClaudeAPIAdapter{}
    })

    DefaultRegistry.Register("gemini-cli", func() AgentAdapter {
        return &GeminiCLIAdapter{}
    })
}

// Example: Creating an agent with an adapter
agent := Agent{
    ID:          "gpt4",
    Type:        "api",
    Name:        "GPT-4",
    Model:       "openai/gpt-4",
    AdapterName: "openrouter",
    Config: map[string]interface{}{
        "api_key_env": "OPENROUTER_API_KEY",
    },
}

adapter, err := DefaultRegistry.Get(agent.AdapterName, agent.Config)
if err != nil {
    log.Fatal(err)
}
agent.adapter = adapter
```

---

## 4. Example: Complete Flow

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/ASRagab/agentpipe/pkg/v2/core"
    "github.com/ASRagab/agentpipe/pkg/v2/events"
    "github.com/ASRagab/agentpipe/pkg/v2/manager"
    "github.com/ASRagab/agentpipe/pkg/v2/adapters"
)

func main() {
    // 1. Create event bus
    eventBus := events.NewEventBus()
    defer eventBus.Close()

    // 2. Configure agents
    agents := []core.Agent{
        {
            ID:          "claude",
            Type:        "api",
            Name:        "Claude",
            Model:       "claude-sonnet-4.5",
            AdapterName: "claude-api",
            Config: map[string]interface{}{
                "api_key_env": "ANTHROPIC_API_KEY",
                "temperature": 0.7,
                "max_tokens":  2000,
            },
        },
        {
            ID:          "gpt4",
            Type:        "api",
            Name:        "GPT-4",
            Model:       "openai/gpt-4",
            AdapterName: "openrouter",
            Config: map[string]interface{}{
                "api_key_env": "OPENROUTER_API_KEY",
                "temperature": 0.7,
                "max_tokens":  2000,
            },
        },
    }

    // 3. Initialize adapters
    for i := range agents {
        adapter, err := adapters.DefaultRegistry.Get(
            agents[i].AdapterName,
            agents[i].Config,
        )
        if err != nil {
            log.Fatalf("Failed to initialize adapter: %v", err)
        }
        agents[i].adapter = adapter
    }

    // 4. Create conversation manager
    config := manager.Config{
        Timeout: 30 * time.Second,
        SaveDir: "~/.agentpipe/conversations",
    }
    mgr := manager.NewConversationManager(config, agents, eventBus)

    // 5. Subscribe to events
    mgr.Subscribe(core.EventMessageCreated, func(e core.Event) {
        msg := e.Data.(core.Message)
        fmt.Printf("[%s] %s: %s\n",
            msg.Timestamp.Format("15:04:05"),
            msg.Role,
            msg.Content,
        )
    })

    mgr.Subscribe(core.EventAgentTyping, func(e core.Event) {
        data := e.Data.(map[string]string)
        fmt.Printf("[%s] %s is typing...\n",
            e.Timestamp.Format("15:04:05"),
            data["agent_name"],
        )
    })

    mgr.Subscribe(core.EventAgentDone, func(e core.Event) {
        msg := e.Data.(core.Message)
        fmt.Printf("[%s] %s finished (tokens: %d, cost: $%.4f)\n",
            e.Timestamp.Format("15:04:05"),
            msg.AgentName,
            msg.Metrics.TotalTokens,
            msg.Metrics.Cost,
        )
    })

    // 6. Start conversation
    ctx := context.Background()
    go mgr.Start(ctx)

    // 7. Send user message
    err := mgr.SendUserMessage(ctx, "Explain async/await in JavaScript")
    if err != nil {
        log.Fatalf("Failed to send message: %v", err)
    }

    // 8. Wait for responses
    time.Sleep(10 * time.Second)

    // 9. Get full conversation
    conv := mgr.GetConversation()
    fmt.Printf("\nConversation ID: %s\n", conv.ID)
    fmt.Printf("Total messages: %d\n", len(conv.Messages))
}
```

---

## 5. Testing Interfaces

```go
// MockAdapter for testing (no real API calls)
type MockAdapter struct {
    model    string
    response string
    delay    time.Duration
    err      error
}

func (m *MockAdapter) Initialize(config map[string]interface{}) error {
    m.model, _ = config["model"].(string)
    m.response, _ = config["mock_response"].(string)
    m.delay, _ = config["mock_delay"].(time.Duration)
    return m.err
}

func (m *MockAdapter) SendMessage(ctx context.Context, messages []Message) (string, error) {
    if m.delay > 0 {
        time.Sleep(m.delay)
    }
    if m.err != nil {
        return "", m.err
    }
    return m.response, nil
}

func (m *MockAdapter) StreamMessage(ctx context.Context, messages []Message, writer io.Writer) error {
    if m.err != nil {
        return m.err
    }

    // Simulate streaming by writing chunks
    chunks := strings.Split(m.response, " ")
    for _, chunk := range chunks {
        if m.delay > 0 {
            time.Sleep(m.delay / time.Duration(len(chunks)))
        }
        _, err := writer.Write([]byte(chunk + " "))
        if err != nil {
            return err
        }
    }
    return nil
}

func (m *MockAdapter) IsAvailable() bool {
    return true
}

func (m *MockAdapter) GetModel() string {
    return m.model
}

func (m *MockAdapter) HealthCheck(ctx context.Context) error {
    return m.err
}

// Example test:
func TestParallelExecution(t *testing.T) {
    eventBus := events.NewEventBus()
    defer eventBus.Close()

    agents := []core.Agent{
        {
            ID:   "fast",
            Name: "Fast Agent",
            adapter: &MockAdapter{
                model:    "fast-model",
                response: "Fast response",
                delay:    100 * time.Millisecond,
            },
        },
        {
            ID:   "slow",
            Name: "Slow Agent",
            adapter: &MockAdapter{
                model:    "slow-model",
                response: "Slow response",
                delay:    500 * time.Millisecond,
            },
        },
    }

    pool := NewAgentPool(agents, eventBus)

    start := time.Now()
    responses := pool.ExecuteParallel(context.Background(), []core.Message{
        {Role: "user", Content: "Test"},
    })
    duration := time.Since(start)

    // Verify parallel execution (should be ~500ms, not 600ms)
    assert.Less(t, duration, 550*time.Millisecond)
    assert.Equal(t, 2, len(responses))
    assert.NoError(t, responses[0].Error)
    assert.NoError(t, responses[1].Error)
}
```

---

## Summary

**Core Interfaces**:
1. **AgentAdapter** - Communication with AI agents (API or CLI)
2. **ConversationManager** - Orchestrates conversation flow
3. **EventBus** - Pub/sub event distribution
4. **AgentPool** - Parallel agent execution

**Core Types**:
1. **Message** - User/agent/system messages with metrics
2. **Agent** - Configured AI agent with adapter
3. **Conversation** - Full conversation state
4. **Event** - Event bus events
5. **Metrics** - Performance and cost tracking

**Key Design Principles**:
- **Interface-first**: Define contracts before implementations
- **Thread-safe**: All operations safe for concurrent use
- **Testable**: Mock adapters for testing without API calls
- **Extensible**: Easy to add new adapters and event handlers
- **Type-safe**: Strong typing with clear semantics

This forms the foundation for the 4-week MVP implementation.
