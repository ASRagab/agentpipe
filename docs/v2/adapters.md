---
type: reference
title: AgentPipe v2 Adapter Guide
created: 2026-01-18
tags:
  - v2
  - adapters
  - api
  - cli
related:
  - "[[configuration]]"
  - "[[quickstart]]"
  - "[[architecture]]"
---

# AgentPipe v2 Adapter Guide

Adapters connect AgentPipe to different AI providers. v2 supports both API-based adapters (direct HTTP) and CLI-based adapters (external command-line tools).

## Adapter Types

### API-Based Adapters

Direct HTTP integration with AI providers. Benefits:
- No external CLI dependencies
- Lower latency
- Accurate token counts from API responses
- Real-time streaming support

### CLI-Based Adapters

Execute external CLI tools. Benefits:
- Leverage existing CLI installations
- Support for providers without public APIs
- Easy to add new providers

## Available Adapters

| Adapter | Type | Provider | Models |
|---------|------|----------|--------|
| `openrouter` | API | OpenRouter | 400+ models from multiple providers |
| `claude-api` | API | Anthropic | Claude 3 family (Opus, Sonnet, Haiku) |
| `claude` | CLI | Anthropic | Claude via `claude` CLI |
| `gemini` | CLI | Google | Gemini via `gemini` CLI |

---

## OpenRouter Adapter

**Type:** API
**Adapter Name:** `openrouter`

Access 400+ models from OpenAI, Anthropic, Google, Meta, Mistral, and more through a single API.

### Configuration

```yaml
agents:
  - id: gpt4
    type: openrouter
    name: GPT-4
    model: openai/gpt-4-turbo
    config:
      api_key_env: OPENROUTER_API_KEY
      system_prompt: "You are a helpful assistant."
      temperature: 0.7
      max_tokens: 2000
```

### Required Configuration

| Field | Description |
|-------|-------------|
| `model` | Model identifier (e.g., `openai/gpt-4-turbo`, `anthropic/claude-3-opus`) |
| `config.api_key_env` | Environment variable for API key (default: `OPENROUTER_API_KEY`) |

### Environment Variable

```bash
export OPENROUTER_API_KEY="your-openrouter-api-key"
```

Get your API key at: [openrouter.ai](https://openrouter.ai/)

### Popular Models

| Model ID | Description | Cost (per 1M tokens) |
|----------|-------------|----------------------|
| `openai/gpt-4-turbo` | GPT-4 Turbo | $10 input / $30 output |
| `openai/gpt-4o` | GPT-4o | $5 input / $15 output |
| `openai/gpt-3.5-turbo` | GPT-3.5 Turbo | $0.50 input / $1.50 output |
| `anthropic/claude-3-opus` | Claude 3 Opus | $15 input / $75 output |
| `anthropic/claude-3-sonnet` | Claude 3 Sonnet | $3 input / $15 output |
| `anthropic/claude-3-haiku` | Claude 3 Haiku | $0.25 input / $1.25 output |
| `google/gemini-pro` | Gemini Pro | $0.50 input / $1.50 output |
| `meta-llama/llama-3-70b-instruct` | Llama 3 70B | $0.80 input / $0.80 output |

Full model list: [openrouter.ai/models](https://openrouter.ai/models)

### Performance Characteristics

- **Latency:** Low (direct API)
- **Streaming:** Supported (SSE)
- **Retry Logic:** Exponential backoff (1s, 2s, 4s)
- **Rate Limits:** Varies by model

---

## Claude API Adapter

**Type:** API
**Adapter Name:** `claude-api`

Direct integration with Anthropic's Claude API.

### Configuration

```yaml
agents:
  - id: claude
    type: claude-api
    name: Claude
    model: claude-sonnet-4-20250514
    config:
      api_key_env: ANTHROPIC_API_KEY
      system_prompt: "You are Claude, a helpful AI assistant."
      temperature: 0.7
      max_tokens: 2000
```

### Required Configuration

| Field | Description |
|-------|-------------|
| `model` | Model identifier (e.g., `claude-sonnet-4-20250514`, `claude-3-opus-20240229`) |
| `config.api_key_env` | Environment variable for API key (default: `ANTHROPIC_API_KEY`) |

### Environment Variable

```bash
export ANTHROPIC_API_KEY="your-anthropic-api-key"
```

Get your API key at: [console.anthropic.com](https://console.anthropic.com/)

### Available Models

| Model ID | Description | Context | Cost (per 1M tokens) |
|----------|-------------|---------|----------------------|
| `claude-sonnet-4-20250514` | Claude Sonnet 4 (latest) | 200K | $3 input / $15 output |
| `claude-3-opus-20240229` | Claude 3 Opus (most capable) | 200K | $15 input / $75 output |
| `claude-3-sonnet-20240229` | Claude 3 Sonnet (balanced) | 200K | $3 input / $15 output |
| `claude-3-haiku-20240307` | Claude 3 Haiku (fastest) | 200K | $0.25 input / $1.25 output |

### Special Notes

- Claude API requires `max_tokens` to be set (defaults to 1024 if not specified)
- System prompts are passed via the `system` parameter, not as a message
- Claude 3 models support vision (image inputs)

### Performance Characteristics

- **Latency:** Low (direct API)
- **Streaming:** Supported (SSE with event types)
- **Retry Logic:** Exponential backoff (1s, 2s, 4s)
- **Rate Limits:** Tier-based (increases with usage)

---

## Claude CLI Adapter

**Type:** CLI
**Adapter Name:** `claude`

Uses the official `claude` CLI from Anthropic.

### Prerequisites

Install the Claude CLI:

```bash
# macOS
brew install anthropic/tap/claude

# Or via npm
npm install -g @anthropic-ai/cli
```

### Configuration

```yaml
agents:
  - id: claude-cli
    type: claude
    name: Claude
    model: claude-sonnet-4-20250514
    config:
      system_prompt: "You are a helpful assistant."
```

### Required Configuration

| Field | Description |
|-------|-------------|
| `model` | Model identifier (same as Claude API) |

### Notes

- Requires `claude` CLI to be installed and authenticated
- Uses the CLI's built-in authentication
- Slower than API adapter due to CLI overhead

---

## Gemini CLI Adapter

**Type:** CLI
**Adapter Name:** `gemini`

Uses Google's Gemini CLI.

### Prerequisites

Install the Gemini CLI:

```bash
# Follow instructions at:
# https://ai.google.dev/gemini-api/docs/get-started/tutorial?lang=cli
```

### Configuration

```yaml
agents:
  - id: gemini
    type: gemini
    name: Gemini
    model: gemini-pro
    config:
      system_prompt: "You are a helpful AI assistant."
```

### Required Configuration

| Field | Description |
|-------|-------------|
| `model` | Model identifier (e.g., `gemini-pro`, `gemini-ultra`) |

### Environment Variable

```bash
export GOOGLE_API_KEY="your-google-api-key"
```

### Notes

- Requires Gemini CLI to be installed
- Slower than API adapter due to CLI overhead
- Consider using OpenRouter's `google/gemini-pro` for API access

---

## Adapter Selection Guide

| Use Case | Recommended Adapter | Reason |
|----------|--------------------|----|
| Production use | `openrouter` or `claude-api` | Lowest latency, best reliability |
| Multiple model providers | `openrouter` | Single API key, 400+ models |
| Anthropic-only | `claude-api` | Direct connection, no middleman |
| Existing CLI setup | `claude` or `gemini` | Leverage existing auth |
| Local models | `openrouter` (via LocalAI) | Unified interface |

## Error Handling

All adapters implement standardized error handling:

### Error Types

| Type | Retryable | Description |
|------|-----------|-------------|
| `timeout` | Yes | Request timed out |
| `rate_limit` | Yes | Rate limit exceeded (respects Retry-After) |
| `network` | Yes | Network connectivity issues |
| `auth` | No | Authentication failed (invalid API key) |
| `invalid_request` | No | Malformed request |
| `server_error` | Yes | Server-side error (5xx) |

### Retry Behavior

- **Default retries:** 3 attempts
- **Backoff:** Exponential (1s, 2s, 4s)
- **Rate limits:** Respects `Retry-After` header when present

## Creating Custom Adapters

Adapters implement the `AgentAdapter` interface:

```go
type AgentAdapter interface {
    // Initialize configures the adapter
    Initialize(agent core.Agent) error

    // SendMessage sends messages and returns the response
    SendMessage(ctx context.Context, messages []core.Message) (string, *core.Metrics, error)

    // StreamMessage sends messages and streams the response
    StreamMessage(ctx context.Context, messages []core.Message, writer io.Writer) (*core.Metrics, error)

    // IsAvailable checks if the adapter is properly configured
    IsAvailable() bool

    // GetModel returns the configured model name
    GetModel() string

    // HealthCheck performs a minimal test request
    HealthCheck(ctx context.Context) error
}
```

Register custom adapters:

```go
import "github.com/kevinelliott/agentpipe/pkg/v2/adapters"

func init() {
    adapters.Register("my-adapter", func() adapters.AgentAdapter {
        return &MyCustomAdapter{}
    })
}
```

See [[architecture]] for implementation details.

## Troubleshooting

### "API key not found"

```bash
# Check if environment variable is set
echo $OPENROUTER_API_KEY
echo $ANTHROPIC_API_KEY

# Verify it's exported to child processes
export OPENROUTER_API_KEY="sk-..."
```

### "Unknown adapter: X"

Valid adapter names:
- `openrouter`
- `claude-api`
- `claude`
- `gemini`

Check for typos in your configuration.

### "Health check failed"

1. Verify API key is valid
2. Check internet connectivity
3. Verify model name is correct
4. Check provider status page

### Slow Responses

1. Use API adapters instead of CLI adapters
2. Use faster models (e.g., Haiku instead of Opus)
3. Reduce `max_tokens` to limit response length
4. Check network latency to provider
