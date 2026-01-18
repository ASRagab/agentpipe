# Webhook Integration Example

This example demonstrates how to integrate AgentPipe v2 with external systems using webhooks. Events from conversations can be sent to any HTTP endpoint for logging, analytics, or triggering external workflows.

## Overview

AgentPipe emits events for all major conversation activities:

- `conversation.started` - When a conversation begins
- `message.created` - When a message is added (user or agent)
- `agent.typing` - When an agent starts generating
- `agent.done` - When an agent finishes
- `agent.error` - When an agent encounters an error
- `conversation.completed` - When the conversation ends

These events can be forwarded to webhooks for:

- **Analytics** - Track usage, costs, and performance
- **Logging** - Centralized conversation logging
- **Notifications** - Slack, Discord, email alerts
- **Automation** - Trigger workflows in Zapier, n8n, etc.
- **Monitoring** - Send to Datadog, Prometheus, etc.

## Files

- `main.go` - Main example with webhook integration
- `webhook_handler.go` - Custom webhook handler implementation
- `server.go` - Example webhook receiver server (for testing)

## Quick Start

```bash
# Terminal 1: Start the webhook receiver
cd examples/v2/code/webhook
go run server.go

# Terminal 2: Run the webhook example
export OPENROUTER_API_KEY="your-key-here"
go run main.go webhook_handler.go
```

## Implementation

### 1. Subscribe to Events

```go
import (
    "github.com/ASRagab/agentpipe/pkg/v2/core"
    "github.com/ASRagab/agentpipe/pkg/v2/events"
)

eventBus := events.NewBus()

// Forward all events to webhook
eventBus.SubscribeAll(func(event core.Event) {
    sendToWebhook(webhookURL, event)
})

// Or subscribe to specific events
eventBus.Subscribe(core.EventMessageCreated, func(event core.Event) {
    sendToWebhook(webhookURL, event)
})
```

### 2. Send Events via HTTP

```go
func sendToWebhook(url string, event core.Event) error {
    payload, _ := json.Marshal(event)

    resp, err := http.Post(url, "application/json", bytes.NewReader(payload))
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("webhook returned %d", resp.StatusCode)
    }

    return nil
}
```

### 3. Handle Events in Your Server

```go
func handleWebhook(w http.ResponseWriter, r *http.Request) {
    var event struct {
        ID        string    `json:"id"`
        Type      string    `json:"type"`
        Timestamp time.Time `json:"timestamp"`
        Data      json.RawMessage `json:"data"`
    }

    if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    switch event.Type {
    case "message.created":
        // Process new message
    case "conversation.completed":
        // Process completion
    }

    w.WriteHeader(http.StatusOK)
}
```

## Event Payloads

### conversation.started

```json
{
  "id": "evt-123",
  "type": "conversation.started",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "conversation_id": "conv-abc",
    "agents": [
      {"id": "agent-1", "name": "Claude", "model": "claude-3-haiku"}
    ]
  }
}
```

### message.created

```json
{
  "id": "evt-124",
  "type": "message.created",
  "timestamp": "2024-01-15T10:30:05Z",
  "data": {
    "id": "msg-456",
    "role": "agent",
    "agent_id": "agent-1",
    "agent_name": "Claude",
    "content": "Hello! How can I help?",
    "metrics": {
      "duration_ns": 1500000000,
      "input_tokens": 10,
      "output_tokens": 8,
      "total_tokens": 18,
      "model": "claude-3-haiku",
      "cost": 0.00005
    }
  }
}
```

### conversation.completed

```json
{
  "id": "evt-125",
  "type": "conversation.completed",
  "timestamp": "2024-01-15T10:35:00Z",
  "data": {
    "conversation_id": "conv-abc",
    "summary": {
      "status": "completed",
      "message_count": 10,
      "agent_count": 2,
      "total_tokens": 500,
      "total_cost": 0.0012,
      "duration": 300000000000
    }
  }
}
```

## Advanced Patterns

### Async Webhook Delivery

For high-throughput scenarios, use a queue:

```go
type WebhookQueue struct {
    events chan core.Event
}

func (q *WebhookQueue) Start(webhookURL string) {
    go func() {
        for event := range q.events {
            sendToWebhook(webhookURL, event)
        }
    }()
}

func (q *WebhookQueue) Enqueue(event core.Event) {
    q.events <- event
}
```

### Retry with Backoff

```go
func sendWithRetry(url string, event core.Event, maxRetries int) error {
    for i := 0; i <= maxRetries; i++ {
        err := sendToWebhook(url, event)
        if err == nil {
            return nil
        }

        if i < maxRetries {
            backoff := time.Duration(1<<uint(i)) * time.Second
            time.Sleep(backoff)
        }
    }
    return fmt.Errorf("webhook delivery failed after %d attempts", maxRetries)
}
```

### Multiple Webhooks

```go
func fanOut(event core.Event, webhooks []string) {
    var wg sync.WaitGroup
    for _, url := range webhooks {
        wg.Add(1)
        go func(url string) {
            defer wg.Done()
            sendToWebhook(url, event)
        }(url)
    }
    wg.Wait()
}
```

## Using the Built-in Bridge

AgentPipe also includes a built-in bridge for streaming to AgentPipe Web:

```bash
# Enable the bridge
agentpipe bridge setup

# Test connectivity
agentpipe bridge test

# Events are automatically streamed during conversations
agentpipe run --v2 -c config.yaml
```

See [internal/bridge](../../../internal/bridge) for the full implementation.

## Related Documentation

- [Architecture Overview](../../../docs/v2/architecture.md)
- [Event Types](../../../pkg/v2/core/events.go)
- [Bridge Implementation](../../../internal/bridge/)
