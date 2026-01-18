# Phase 02: Core Tests and Mock Adapter

This phase adds comprehensive test coverage for all Phase 1 components and creates a MockAdapter for testing without real API calls. Achieving >80% test coverage on core packages ensures reliability and provides confidence for future development. The mock adapter enables fast, deterministic testing of parallel execution and event handling.

## Tasks

- [x] Create the MockAdapter for testing in `pkg/v2/adapters/mock/mock.go`:
  - MockAdapter struct with configurable Response, Delay, Error, StreamChunks fields
  - Initialize() storing config values
  - SendMessage() that sleeps for Delay, returns Error if set, otherwise returns Response
  - StreamMessage() that writes StreamChunks to writer with delays between chunks
  - IsAvailable() returning true by default
  - GetModel() returning "mock-model"
  - HealthCheck() returning configured Error
  - Register with DefaultRegistry as "mock"

- [x] Write comprehensive tests for core data types in `pkg/v2/core/`:
  - `message_test.go`: Test NewUserMessage() creates valid message with UUID, timestamp, role; Test NewAgentMessage() includes metrics; Test message JSON serialization/deserialization
  - `agent_test.go`: Test Agent struct field assignments; Test adapter assignment
  - `conversation_test.go`: Test Conversation initialization; Test adding messages; Test status transitions
  - `events_test.go`: Test Event creation; Test all event type constants are unique

- [x] Write comprehensive tests for the event bus in `pkg/v2/events/bus_test.go`:
  - TestPublishSubscribe: Subscribe, publish, verify handler receives event
  - TestMultipleSubscribers: Multiple handlers for same event type all receive it
  - TestUnsubscribe: Unsubscribed handler no longer receives events
  - TestSubscribeAll: Handler receives all event types
  - TestPanicRecovery: Handler panics, bus continues functioning, other handlers still called
  - TestConcurrentPublish: Multiple goroutines publishing simultaneously without races
  - TestClose: Close waits for all handlers to complete
  - Run with -race flag to verify thread safety

- [x] Write tests for the adapter registry in `pkg/v2/adapters/registry_test.go`:
  - TestRegister: Register adapter, verify List() includes it
  - TestGet: Get registered adapter, verify Initialize() is called
  - TestGetUnknown: Get unknown adapter returns error
  - TestGetInitializeError: Get adapter that fails Initialize() returns wrapped error
  - TestList: Verify returned list is sorted alphabetically
  - TestConcurrentAccess: Multiple goroutines registering and getting without races

- [x] Write tests for the AgentPool in `pkg/v2/pool/pool_test.go`:
  - TestExecuteParallel: Two mock agents respond, verify both responses collected
  - TestParallelTiming: Fast (100ms) and slow (500ms) agents, verify total time ~500ms not 600ms
  - TestEventEmission: Verify EventAgentTyping emitted before response, EventAgentDone emitted after
  - TestAgentError: One agent fails, other succeeds, verify partial results returned
  - TestAllAgentsFail: All agents fail, verify errors returned correctly
  - TestTimeout: Agent exceeds timeout, verify context cancellation works
  - TestGetStatus: Verify status map reflects agent states during execution

- [x] Write tests for the ConversationManager in `pkg/v2/manager/manager_test.go`:
  - TestNewConversationManager: Verify initialization with valid config and agents
  - TestSendUserMessage: Send message, verify added to history, verify EventMessageCreated emitted
  - TestParallelAgentResponses: Send message, verify all agents respond, verify order preserved
  - TestGetMessages: Verify thread-safe copy returned, mutations don't affect original
  - TestGetConversation: Verify conversation state accessible
  - TestSubscribe: Verify event subscription works through manager

- [x] Write tests for configuration loading in `pkg/v2/config/config_test.go`:
  - TestLoadConfig: Load valid YAML file, verify all fields parsed correctly
  - TestLoadConfigMissingFile: Return clear error for missing file
  - TestLoadConfigInvalidYAML: Return clear error for malformed YAML
  - TestInitializeAgents: Verify agents created with correct adapters
  - TestInitializeAgentsUnknownAdapter: Return error for unknown adapter name
  - Create test fixtures in `pkg/v2/config/testdata/` directory

- [x] Write integration tests in `pkg/v2/integration_test.go`:
  - TestFullConversationFlow: Create manager with mock agents, send message, verify full flow works
  - TestStreamingFlow: Verify chunks emitted during streaming
  - TestMultipleMessages: Send multiple user messages, verify conversation grows correctly
  - Tag with `//go:build integration` for optional execution

- [x] Run all tests and verify coverage:
  - Execute `go test -race -coverprofile=coverage.out ./pkg/v2/...`
  - Generate HTML report with `go tool cover -html=coverage.out -o coverage.html`
  - Verify coverage is >80% for core, events, adapters, pool, manager packages
  - Fix any failing tests before proceeding

## Coverage Results

| Package | Coverage |
|---------|----------|
| adapters | 100.0% |
| adapters/mock | 90.9% |
| config | 100.0% |
| core | 100.0% |
| events | 100.0% |
| manager | 86.6% |
| pool | 98.6% |

All packages meet the >80% coverage requirement.
