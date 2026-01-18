# Programmatic Usage Example

This example demonstrates how to use the AgentPipe v2 engine programmatically in your Go applications, without relying on the CLI or configuration files.

## Overview

Using AgentPipe v2 as a library gives you full control over:

- Agent configuration and initialization
- Conversation flow and message handling
- Event subscriptions for real-time updates
- Streaming responses
- Persistence (save/load conversations)

## Files

- `main.go` - Basic programmatic usage
- `streaming.go` - Streaming example with real-time output
- `persistence.go` - Save and resume conversations

## Quick Start

```bash
# Navigate to the example directory
cd examples/v2/code/programmatic

# Set your API key
export OPENROUTER_API_KEY="your-key-here"

# Run the example
go run main.go
```

## Key Components

### 1. Core Types

```go
import "github.com/kevinelliott/agentpipe/pkg/v2/core"

// Create an agent
agent := core.Agent{
    ID:          "agent-1",
    Type:        "openrouter",
    Name:        "Claude",
    Model:       "anthropic/claude-3-haiku",
    AdapterName: "openrouter",
    Config: core.AgentAdapterConfig{
        SystemPrompt: "You are a helpful assistant.",
        Temperature:  0.7,
        MaxTokens:    4096,
    },
}

// Create a message
msg := core.NewUserMessage("Hello!")

// Create a conversation
conv := core.NewConversation(agents)
```

### 2. Conversation Manager

```go
import (
    "github.com/kevinelliott/agentpipe/pkg/v2/manager"
    "github.com/kevinelliott/agentpipe/pkg/v2/events"
)

// Create event bus
eventBus := events.NewBus()

// Configure the manager
config := manager.Config{
    Timeout: 60 * time.Second,
    Persistence: manager.PersistenceConfig{
        Enabled: true,
        SaveDir: "~/.agentpipe/v2/chats",
    },
}

// Create the manager
mgr, err := manager.NewConversationManager(config, agents, eventBus)
if err != nil {
    log.Fatal(err)
}
defer mgr.Close()

// Start and use
mgr.Start()
responses, err := mgr.SendUserMessage(ctx, "Hello!")
mgr.Complete()
```

### 3. Event Subscriptions

```go
// Subscribe to specific events
eventBus.Subscribe(core.EventMessageCreated, func(event core.Event) {
    if msg, ok := event.Data.(core.Message); ok {
        fmt.Printf("[%s] %s\n", msg.AgentName, msg.Content)
    }
})

// Subscribe to all events
eventBus.SubscribeAll(func(event core.Event) {
    fmt.Printf("Event: %s\n", event.Type)
})
```

### 4. Streaming

```go
import "github.com/kevinelliott/agentpipe/pkg/v2/adapters"

// Get the adapter
adapter, _ := adapters.Get("openrouter")
adapter.Initialize(agent)

// Stream to stdout
metrics, err := adapter.StreamMessage(ctx, messages, os.Stdout)
```

### 5. Loading Configuration

```go
import "github.com/kevinelliott/agentpipe/pkg/v2/config"

// Load from YAML file
cfg, err := config.LoadConfig("config.yaml")
if err != nil {
    log.Fatal(err)
}

// Initialize agents from config
agents, err := cfg.InitializeAgents()
```

## Advanced Usage

### Multiple Agent Orchestration

```go
// All agents respond in parallel
responses, err := mgr.SendUserMessage(ctx, message)

// Process all responses
for _, resp := range responses {
    if resp.Role == core.RoleAgent {
        fmt.Printf("%s said: %s\n", resp.AgentName, resp.Content)
    }
}
```

### Custom Event Handling

```go
// Track metrics
var totalTokens int
var totalCost float64

eventBus.Subscribe(core.EventAgentDone, func(event core.Event) {
    if data, ok := event.Data.(core.AgentDoneData); ok {
        if data.Message.Metrics != nil {
            totalTokens += data.Message.Metrics.TotalTokens
            totalCost += data.Message.Metrics.Cost
        }
    }
})
```

### Graceful Degradation

```go
config := manager.Config{
    GracefulDegradation: manager.GracefulDegradationConfig{
        Enabled:                  true,
        PauseOnAllFailed:         true,
        RetryFailedOnNextMessage: true,
        EmitSystemMessages:       true,
    },
}

// If all agents fail, conversation pauses instead of erroring
// Check and resume:
if mgr.IsPaused() {
    mgr.ResumeConversation(true) // true = clear failed agents
}
```

## Related Documentation

- [Configuration Reference](../../../docs/v2/configuration.md)
- [Architecture Overview](../../../docs/v2/architecture.md)
- [Adapters Guide](../../../docs/v2/adapters.md)
