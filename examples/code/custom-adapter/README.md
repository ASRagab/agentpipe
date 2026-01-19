# Custom Adapter Example

This example demonstrates how to implement a custom `AgentAdapter` for the AgentPipe v2 engine. Custom adapters allow you to connect AgentPipe to any AI provider or service.

## Overview

The `AgentAdapter` interface defines 6 methods that must be implemented:

```go
type AgentAdapter interface {
    Initialize(agent core.Agent) error
    SendMessage(ctx context.Context, messages []core.Message) (string, *core.Metrics, error)
    StreamMessage(ctx context.Context, messages []core.Message, writer io.Writer) (*core.Metrics, error)
    IsAvailable() bool
    GetModel() string
    HealthCheck(ctx context.Context) error
}
```

## Files

- `adapter.go` - The custom adapter implementation
- `main.go` - Example usage of the custom adapter
- `go.mod` - Module definition (for standalone compilation)

## Quick Start

```bash
# Navigate to the example directory
cd examples/v2/code/custom-adapter

# Initialize the module (if running standalone)
go mod init github.com/ASRagab/agentpipe/examples/v2/code/custom-adapter
go mod tidy

# Run the example
go run .
```

## Implementation Guide

### 1. Create Your Adapter Struct

Define a struct that holds the configuration and state for your adapter:

```go
type MyCustomAdapter struct {
    apiKey      string
    model       string
    temperature float64
    maxTokens   int
    // ... any other fields you need
}
```

### 2. Implement the Interface Methods

See `adapter.go` for a complete implementation example. Key points:

- **Initialize**: Set up your adapter with configuration from `core.Agent`
- **IsAvailable**: Check if the adapter is properly configured (e.g., API key present)
- **HealthCheck**: Perform a minimal test request to verify connectivity
- **SendMessage**: Send messages and return the response synchronously
- **StreamMessage**: Send messages and stream the response to a writer
- **GetModel**: Return the model name

### 3. Register Your Adapter

Register your adapter with the global registry so it can be used in configurations:

```go
func init() {
    adapters.Register("my-custom", NewMyCustomAdapter)
}
```

### 4. Use in Configuration

Once registered, use your adapter in YAML configurations:

```yaml
agents:
  - id: my-agent
    type: my-custom
    adapter: my-custom  # Optional if same as type
    name: My Custom Agent
    model: my-model-name
    config:
      system_prompt: "You are a helpful assistant."
      temperature: 0.7
```

## Error Handling

Use the v2 errors package for consistent error classification:

```go
import "github.com/ASRagab/agentpipe/pkg/errors"

// Create typed errors
errors.NewRateLimitError(agentID, agentName, retryAfter, nil)
errors.NewAuthenticationError(agentID, agentName, nil)
errors.NewTimeoutError(agentID, agentName, nil)
errors.NewNetworkError(agentID, agentName, nil)
```

## Best Practices

1. **Always validate configuration** in `Initialize()`
2. **Implement retry logic** with exponential backoff for transient failures
3. **Use proper context handling** to support timeouts and cancellation
4. **Return accurate metrics** when available (tokens, duration, cost)
5. **Log appropriately** using the `pkg/log` package
6. **Handle streaming gracefully** even if your API doesn't support it (fall back to buffered response)

## Related Documentation

- [Adapters Guide](../../../docs/v2/adapters.md)
- [Configuration Reference](../../../docs/v2/configuration.md)
- [Architecture Overview](../../../docs/v2/architecture.md)
