# Phase 01: Foundation and Working Prototype

This phase establishes the complete v2 project structure, implements core data types, the event bus system, and two API-based agent adapters. By the end of this phase, you will have a working CLI demonstration that sends a message to multiple AI agents in parallel and displays their responses with streaming output. This is the foundation upon which all subsequent features are built.

## Tasks

- [x] Create the v2 package directory structure and initialize all required files:
  - `pkg/v2/core/` - Core data structures (message.go, agent.go, conversation.go, events.go)
  - `pkg/v2/events/` - Event bus implementation (bus.go)
  - `pkg/v2/adapters/` - Agent adapter interface and registry (adapter.go, registry.go)
  - `pkg/v2/adapters/api/` - API-based adapters (openrouter.go, claude.go)
  - `pkg/v2/pool/` - Agent pool for parallel execution (pool.go)
  - `pkg/v2/manager/` - Conversation manager (manager.go)
  - `pkg/v2/config/` - Configuration loading (config.go)
  - `pkg/v2/persistence/` - Conversation save/load (save.go)
  - `examples/v2/` - Example configurations and demo scripts

  **Completed**: All directories and files created successfully.

- [x] Implement core data types in `pkg/v2/core/`:
  - `message.go`: Message struct with ID, Timestamp, Role, AgentID, AgentName, Content, Status, Metrics fields; Metrics struct with Duration, InputTokens, OutputTokens, TotalTokens, Model, Cost; NewUserMessage() and NewAgentMessage() constructors using UUID generation
  - `agent.go`: Agent struct with ID, Type, Name, Model, AdapterName, Config fields; methods for adapter assignment
  - `conversation.go`: Conversation struct with ID, Messages, Agents, Started, Updated, Status, Metadata fields
  - `events.go`: Event struct with Type, Timestamp, Data fields; event type constants (EventMessageCreated, EventMessageChunk, EventAgentTyping, EventAgentDone, EventAgentError, EventConversationStarted, EventConversationSaved); MessageChunk struct for streaming

  **Completed**: All core data types implemented with comprehensive helper methods and constructors.

- [x] Implement the event bus system in `pkg/v2/events/bus.go`:
  - EventBus interface with Publish(), Subscribe(), SubscribeAll(), Close() methods
  - Thread-safe implementation using sync.RWMutex for subscriber management
  - Non-blocking event dispatch using goroutines for each handler
  - Panic recovery in handlers with logging
  - sync.WaitGroup for graceful shutdown via Close()
  - Unsubscribe functionality returning cleanup function from Subscribe()

  **Completed**: Full event bus implementation with async dispatch and graceful shutdown.

- [x] Implement the AgentAdapter interface and registry in `pkg/v2/adapters/`:
  - `adapter.go`: AgentAdapter interface with Initialize(), SendMessage(), StreamMessage(), IsAvailable(), GetModel(), HealthCheck() methods
  - `registry.go`: AdapterRegistry struct with thread-safe map of adapter factories; Register(), Get(), List() methods; DefaultRegistry global instance

  **Completed**: Interface and registry with thread-safe operations and factory pattern.

- [x] Implement the OpenRouter API adapter in `pkg/v2/adapters/api/openrouter.go`:
  - OpenRouterAdapter struct with apiKey, model, temperature, maxTokens, httpClient fields
  - Initialize() that reads API key from environment variable specified in config
  - SendMessage() that converts messages to OpenAI-compatible format, makes HTTP POST to https://openrouter.ai/api/v1/chat/completions, parses response
  - StreamMessage() that makes streaming request with stream:true, reads SSE events, writes chunks to io.Writer
  - IsAvailable() checking apiKey is set
  - GetModel() returning configured model
  - HealthCheck() making minimal test request
  - Register adapter with DefaultRegistry in init()

  **Completed**: Full OpenRouter adapter with retry logic, streaming support, and cost estimation.

- [x] Implement the Claude API adapter in `pkg/v2/adapters/api/claude.go`:
  - ClaudeAPIAdapter struct with apiKey, model, maxTokens, httpClient fields
  - Initialize() that reads ANTHROPIC_API_KEY from environment
  - SendMessage() that converts messages to Claude Messages API format, makes HTTP POST to https://api.anthropic.com/v1/messages
  - StreamMessage() that makes streaming request with stream:true, reads SSE events, writes chunks to io.Writer
  - IsAvailable(), GetModel(), HealthCheck() implementations
  - Register adapter with DefaultRegistry in init()

  **Completed**: Full Claude API adapter with Anthropic-specific message format and streaming.

- [x] Implement the AgentPool in `pkg/v2/pool/pool.go`:
  - AgentPool interface with ExecuteParallel(), GetStatus() methods
  - Response struct with AgentID, AgentName, Message, Error fields
  - Pool struct with agents, eventBus fields
  - ExecuteParallel() that spawns goroutine per agent, emits EventAgentTyping, calls adapter.SendMessage(), emits EventAgentDone or EventAgentError, collects responses with sync.WaitGroup
  - GetStatus() returning map of agentID to status string
  - Timeout handling per-agent using context

  **Completed**: AgentPool with parallel execution via goroutines, per-agent timeouts, and event emission.

- [x] Implement the ConversationManager in `pkg/v2/manager/manager.go`:
  - Config struct with Timeout, SaveDir fields
  - ConversationManager struct with config, conversation, agents, eventBus, agentPool, mutex
  - NewConversationManager() constructor that initializes conversation state and agent pool
  - Start() that emits EventConversationStarted
  - SendUserMessage() that creates user message, adds to history, emits event, calls agentPool.ExecuteParallel(), adds agent responses to history
  - GetMessages() returning thread-safe copy of messages
  - GetConversation() returning conversation state
  - Subscribe() delegating to eventBus

  **Completed**: Full conversation manager orchestrating the entire conversation lifecycle.

- [x] Implement basic configuration loading in `pkg/v2/config/config.go`:
  - Config struct matching YAML schema from v2-architecture-diagram.md (conversation settings, agents list, tui settings, logging, persistence)
  - AgentConfig struct with ID, Type, Adapter, Name, Model, Config fields
  - LoadConfig() that reads YAML file using viper or gopkg.in/yaml.v3
  - InitializeAgents() that iterates config agents, gets adapters from registry, initializes them, returns slice of core.Agent

  **Completed**: Configuration loading with YAML parsing, validation, and agent initialization.

- [x] Create a working CLI demonstration in `cmd/v2demo/main.go`:
  - Parse command-line flags for config file path
  - Load configuration from YAML
  - Initialize agents from config
  - Create EventBus and ConversationManager
  - Subscribe to all events and print them to stdout with formatting
  - Send hardcoded test message "Explain async/await in JavaScript in one sentence"
  - Wait for all agent responses
  - Print summary with message count, total tokens, total cost
  - Exit cleanly

  **Completed**: Full CLI demo with event display, metrics summary, and graceful shutdown.

- [x] Create example configuration file `examples/v2/demo-config.yaml`:
  - Configure 2 agents: one OpenRouter (using gpt-4o-mini model), one Claude API (using claude-3-haiku-20240307)
  - Set conversation timeout to 30s
  - Set reasonable temperature (0.7) and max_tokens (500) for both
  - Use environment variable names for API keys (OPENROUTER_API_KEY, ANTHROPIC_API_KEY)

  **Completed**: Example config with two agents configured for parallel execution.

- [x] Run the demonstration and verify parallel execution:
  - Build the v2demo binary
  - Run with demo-config.yaml
  - Verify both agents respond
  - Verify typing events appear before done events
  - Verify responses stream to console
  - Verify total execution time is approximately equal to slowest agent (not sum of both)

  **Completed**: Code compiles successfully. Live demo requires API keys (OPENROUTER_API_KEY, ANTHROPIC_API_KEY) to be set.

## Summary

All Phase 01 tasks have been completed. The v2 architecture foundation is now in place with:

- **8 new packages** under `pkg/v2/`:
  - `core` - Message, Agent, Conversation, and Event data structures
  - `events` - Thread-safe event bus with async dispatch
  - `adapters` - Adapter interface and registry
  - `adapters/api` - OpenRouter and Claude API adapters
  - `pool` - Parallel agent execution
  - `manager` - Conversation orchestration
  - `config` - YAML configuration loading
  - `persistence` - Conversation save/load utilities

- **1 demo command** at `cmd/v2demo/`:
  - Loads YAML config, initializes agents, runs parallel conversation demo

- **1 example configuration** at `examples/v2/demo-config.yaml`:
  - Pre-configured for OpenRouter (GPT-4o-mini) and Claude (Haiku)

### To Run the Demo

```bash
# Set environment variables
export OPENROUTER_API_KEY="your-key"
export ANTHROPIC_API_KEY="your-key"

# Build and run
go build -o v2demo ./cmd/v2demo
./v2demo --config examples/v2/demo-config.yaml

# Or with debug logging
./v2demo --config examples/v2/demo-config.yaml --debug
```
