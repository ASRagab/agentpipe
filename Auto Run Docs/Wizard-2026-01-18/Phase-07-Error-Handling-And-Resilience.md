# Phase 07: Error Handling and Resilience

This phase implements comprehensive error handling, retry logic, and resilience patterns across all components. The system gracefully handles API failures, network issues, rate limits, and timeouts while providing clear feedback to users and maintaining conversation integrity.

## Tasks

- [x] Define error types in `pkg/v2/errors/errors.go`:
  - AgentError wrapping errors with AgentID, AgentName, ErrorType fields
  - Error types enum: ErrTypeNetwork, ErrTypeRateLimit, ErrTypeTimeout, ErrTypeAuth, ErrTypeInvalidResponse, ErrTypeUnknown
  - IsRetryable() method returning true for network/ratelimit/timeout
  - UserFriendlyMessage() returning actionable error text
  - Unwrap() for error chain inspection

  **Completed**: Created comprehensive error package with:
  - `AgentError` struct with full field set (AgentID, AgentName, ErrorType, Message, Err, Timestamp, RetryCount, RetryAfter, HTTPStatusCode)
  - `ErrorType` enum as `int` with 6 types: ErrTypeUnknown, ErrTypeNetwork, ErrTypeRateLimit, ErrTypeTimeout, ErrTypeAuth, ErrTypeInvalidResponse
  - `IsRetryable()` returns true for network/ratelimit/timeout, false for auth/invalid_response/unknown
  - `UserFriendlyMessage()` returns context-aware actionable text for each error type
  - `Unwrap()` for standard library error chain inspection (works with `errors.Is`/`errors.As`)
  - Helper constructors: `NewNetworkError()`, `NewRateLimitError()`, `NewTimeoutError()`, `NewAuthError()`, `NewInvalidResponseError()`
  - `ClassifyError()` for automatic error classification from error messages
  - `WrapError()` for wrapping generic errors with automatic classification
  - 21 comprehensive tests covering all functionality (100% pass)

- [x] Implement retry logic in `pkg/v2/adapters/retry.go`:
  - RetryConfig struct with MaxAttempts, InitialDelay, MaxDelay, BackoffMultiplier
  - DefaultRetryConfig: 3 attempts, 1s initial, 30s max, 2x multiplier
  - WithRetry() wrapper function for adapter methods
  - Exponential backoff with jitter
  - Only retry on IsRetryable() errors
  - Log each retry attempt with attempt number and delay
  - Return last error after all attempts exhausted

  **Completed**: Created comprehensive retry package with:
  - `RetryConfig` struct with MaxAttempts, InitialDelay, MaxDelay, BackoffMultiplier, JitterFactor
  - `DefaultRetryConfig()` returns 3 attempts, 1s initial, 30s max, 2x multiplier, 10% jitter
  - `RetryableAdapter` struct wrapping any AgentAdapter with retry logic
  - `NewRetryableAdapter()` factory with agentID and agentName for logging
  - `calculateDelay()` with exponential backoff: delay = InitialDelay * BackoffMultiplier^(attempt-1)
  - Jitter applied as ±JitterFactor% of calculated delay
  - `WithRetry[T]()` generic wrapper function for any operation returning (T, error)
  - `RetryOnError()` for simple void operations with retry
  - Integration with `errors.IsRetryable()` - only retries network/ratelimit/timeout errors
  - Logs each retry with structured fields: agent_id, agent_name, attempt, max, delay, error
  - Context cancellation properly handled during backoff wait
  - Returns `AgentError` with RetryCount set after all attempts exhausted
  - 18 comprehensive tests covering all functionality:
    - TestDefaultRetryConfig, TestRetryConfig_CalculateDelay, TestRetryConfig_CalculateDelayWithJitter
    - TestRetryableAdapter_SendMessageSuccess, TestRetryableAdapter_SendMessageRetryOnNetworkError
    - TestRetryableAdapter_SendMessageNoRetryOnAuthError, TestRetryableAdapter_SendMessageExhaustedRetries
    - TestRetryableAdapter_SendMessageContextCancellation, TestRetryableAdapter_StreamMessageRetry
    - TestRetryableAdapter_RateLimitHandling, TestWithRetry_Success, TestWithRetry_RetryableError
    - TestWithRetry_NonRetryableError, TestRetryOnError_Success, TestRetryableAdapter_DelegatesMethods
    - TestRetryConfig_ZeroMaxAttempts

- [x] Add retry support to API adapters:
  - Wrap SendMessage() and StreamMessage() with retry logic
  - Detect rate limit responses (HTTP 429) and parse retry-after header
  - Detect network errors (connection refused, timeout, DNS)
  - Detect auth errors (HTTP 401, 403) - do not retry
  - Emit EventAgentError with retry count on each failure

  **Completed**: Implemented comprehensive error classification and retry support for API adapters:
  - Created `pkg/v2/adapters/api/http_errors.go` with `HTTPErrorClassifier`:
    - `ClassifyHTTPError()` detects HTTP 429 (rate limit), 401/403 (auth), 5xx (retryable network), 4xx (non-retryable)
    - `ClassifyConnectionError()` handles connection-level errors (timeout, DNS, connection refused)
    - `parseRetryAfter()` parses both seconds and HTTP-date format from Retry-After header
    - Helper functions: `IsRateLimitError()`, `IsAuthError()`, `IsNetworkError()`, `IsTimeoutError()`
    - `GetRetryAfter()` and `GetHTTPStatusCode()` for extracting error metadata
  - Updated `OpenRouterAdapter` with proper error classification:
    - Added `agentID`, `agentName`, `errorClassifier` fields
    - `doRequestWithRetry()` now uses `errors.IsRetryable()` and respects retry-after durations
    - `doRequest()` classifies connection-level errors via `ClassifyConnectionError()`
    - `handleErrorResponse()` uses `ClassifyHTTPError()` for proper error types
    - Comprehensive logging with attempt count, delay, and retryability status
    - Returns `AgentError` with `RetryCount` set after exhausting all attempts
  - Updated `ClaudeAPIAdapter` with identical error classification:
    - Same pattern as OpenRouter with agent info and error classifier
    - Full integration with v2 errors package
    - Proper logging and retry-after handling
  - Removed deprecated `shouldRetry()` function (replaced by proper classification)
  - Created `pkg/v2/adapters/api/http_errors_test.go` with 15 test cases:
    - Rate limit detection with Retry-After parsing
    - Auth error detection (401, 403)
    - Server error detection (500, 502, 503, 504) - retryable
    - Client error detection (400, 404, 422) - non-retryable
    - Connection error classification (timeout, DNS, refused)
    - Helper function tests (IsRateLimitError, GetRetryAfter, etc.)
    - Integration test with httptest server
  - All tests pass (100%)

- [x] Implement circuit breaker in `pkg/v2/adapters/circuit_breaker.go`:
  - CircuitBreaker struct with state (closed, open, half-open), failureCount, lastFailure
  - Open circuit after N consecutive failures (default: 5)
  - Half-open after cooldown period (default: 30s)
  - Close circuit on successful request in half-open state
  - AllowRequest() checking if request should proceed
  - RecordSuccess() and RecordFailure() updating state
  - Per-adapter circuit breakers

  **Completed**: Created comprehensive circuit breaker implementation with:
  - `CircuitState` enum with `CircuitClosed`, `CircuitOpen`, `CircuitHalfOpen` states with String() method
  - `CircuitBreakerConfig` struct with FailureThreshold (default: 5), CooldownPeriod (default: 30s), SuccessThreshold (default: 1)
  - `CircuitBreaker` struct with thread-safe state management (sync.RWMutex)
  - `AllowRequest()` checks state and transitions open→half-open after cooldown
  - `RecordSuccess()` resets failure count (closed) or closes circuit after SuccessThreshold (half-open)
  - `RecordFailure()` opens circuit after FailureThreshold, re-opens immediately in half-open
  - `Reset()` for manual circuit reset to closed state
  - `TimeUntilRetry()` returns remaining cooldown time for open circuit
  - `OnStateChange()` callback for state transition notifications
  - `CircuitBreakerAdapter` wrapper implementing full AgentAdapter interface
  - `IsCircuitOpenError()` helper to detect circuit open errors
  - Comprehensive logging with structured fields for all state transitions
  - 21 tests covering all functionality:
    - TestCircuitBreakerState, TestDefaultCircuitBreakerConfig, TestCircuitBreakerInitialState
    - TestCircuitBreakerOpensAfterFailures, TestCircuitBreakerSuccessResets
    - TestCircuitBreakerHalfOpenAfterCooldown, TestCircuitBreakerClosesAfterSuccessInHalfOpen
    - TestCircuitBreakerReopensOnFailureInHalfOpen, TestCircuitBreakerReset
    - TestCircuitBreakerTimeUntilRetry, TestCircuitBreakerStateChangeCallback
    - TestCircuitBreakerConcurrency (thread safety), TestCircuitBreakerAdapterSendMessage
    - TestCircuitBreakerAdapterOpens, TestCircuitBreakerAdapterRecovery
    - TestCircuitBreakerAdapterStreamMessage, TestCircuitBreakerAdapterIsAvailable
    - TestCircuitBreakerAdapterGetModel, TestCircuitBreakerAdapterHealthCheck
    - TestCircuitBreakerAdapterInitialize, TestIsCircuitOpenError
  - All tests pass (100%) with race detection enabled

- [x] Add circuit breaker to AgentPool:
  - Track circuit breaker per agent
  - Skip agents with open circuits in ExecuteParallel()
  - Emit EventAgentError with "circuit open" message
  - Periodically attempt half-open requests
  - Log circuit state transitions

  **Completed**: Integrated circuit breaker pattern into AgentPool with:
  - `AgentEntry` now includes a `CircuitBreaker` field for per-agent tracking
  - `Pool` struct has `circuitBreakerConfig` field with default configuration
  - `NewPool()` initializes with `DefaultCircuitBreakerConfig()` (5 failures, 30s cooldown, 1 success to close)
  - `NewPoolWithCircuitBreaker()` constructor for custom configuration
  - `AddAgent()` creates circuit breaker with state change callback for logging
  - `ExecuteParallel()` checks `AllowRequest()` before executing - skips agents with open circuits
  - Emits `EventAgentError` with "circuit breaker open" message when:
    - Circuit transitions to open state (via OnStateChange callback)
    - Agent is skipped due to open circuit (includes retry-in duration)
  - `executeAgent()` calls `RecordFailure()` on error and `RecordSuccess()` on success
  - Circuit transitions open→half-open after cooldown, then closed on successful probe
  - New helper methods:
    - `CircuitBreakerInfo` struct with AgentID, State, FailureCount, TimeUntilRetry, LastFailure
    - `GetCircuitBreakerStatus(agentID)` returns status for specific agent
    - `GetAllCircuitBreakerStatus()` returns status for all agents
    - `ResetCircuitBreaker(agentID)` manually resets a specific circuit
    - `ResetAllCircuitBreakers()` resets all circuits
    - `GetAvailableAgentCount()` returns count of agents with circuits allowing requests
  - Comprehensive logging with structured fields for all state transitions
  - 17 new tests covering all circuit breaker integration scenarios:
    - TestCircuitBreakerInitialization, TestCircuitBreakerCustomConfig
    - TestCircuitBreakerOpensAfterConsecutiveFailures, TestCircuitBreakerSkipsOpenCircuit
    - TestCircuitBreakerEmitsErrorEventOnOpen, TestCircuitBreakerSuccessResetsFailureCount
    - TestGetAllCircuitBreakerStatus, TestResetCircuitBreaker, TestResetCircuitBreakerNotFound
    - TestResetAllCircuitBreakers, TestGetAvailableAgentCount
    - TestMixedAgentsContinueWithOpenCircuit, TestCircuitBreakerHalfOpenRecovery
  - All tests pass (100%) with race detection enabled

- [x] Implement graceful degradation in ConversationManager:
  - Continue conversation even if some agents fail
  - Emit system message when agent fails: "Claude is currently unavailable"
  - Track failed agents and retry on next user message
  - Option to pause conversation if all agents fail
  - Resume with available agents when possible

  **Completed**: Implemented comprehensive graceful degradation in ConversationManager with:
  - `GracefulDegradationConfig` struct with configurable options:
    - `Enabled` (default: true) - Enable/disable graceful degradation
    - `PauseOnAllFailed` (default: true) - Pause conversation when all agents fail
    - `RetryFailedOnNextMessage` (default: true) - Retry failed agents on next user message
    - `EmitSystemMessages` (default: true) - Emit user-friendly system messages for failures
  - `DefaultGracefulDegradationConfig()` factory with sensible defaults
  - `FailedAgentInfo` struct for tracking failed agents:
    - AgentID, AgentName, LastError, FailedAt timestamp, RetryCount
  - Sentinel errors for graceful error handling:
    - `ErrAllAgentsFailed` - Returned when all agents fail to respond
    - `ErrConversationPaused` - Returned when conversation is paused
  - `SendUserMessage()` updated with degradation logic:
    - Returns `ErrConversationPaused` if paused (user must call Resume first)
    - Clears failed agents before retrying (if RetryFailedOnNextMessage enabled)
    - Continues processing even when some agents fail
    - Pauses conversation when all agents fail (if PauseOnAllFailed enabled)
  - `processResponsesWithDegradation()` method:
    - Counts successful vs failed responses
    - Tracks failed agents with `trackFailedAgent()`
    - Clears successful agents from failed list with `clearFailedAgent()`
    - Emits system messages for failed agents via `createUnavailableSystemMessage()`
    - Uses `core.ErrorTypeFromError()` for user-friendly error classification
  - New public methods:
    - `GetFailedAgents()` - Returns map of currently failed agents
    - `IsPaused()` - Check if conversation is paused
    - `ResumeConversation(clearFailed bool)` - Resume paused conversation
    - `ResetCircuitBreakers()` - Reset all circuit breakers via pool
    - `GetAvailableAgentCount()` - Get count of available agents
  - 17 comprehensive tests covering all functionality:
    - TestDefaultGracefulDegradationConfig, TestGracefulDegradation_ContinuesWithSomeAgentsFailing
    - TestGracefulDegradation_TracksFailedAgents, TestGracefulDegradation_IsPausedAndResume
    - TestGracefulDegradation_SendMessageWhenPaused, TestGracefulDegradation_ResetCircuitBreakers
    - TestGracefulDegradation_SystemMessageFormat, TestGracefulDegradation_FailedAgentRetryTracking
    - TestGracefulDegradation_DisabledEmitsNoSystemMessages, TestGracefulDegradation_DisabledPauseOnAllFailed
    - TestGracefulDegradation_ConfigDefaults, TestGracefulDegradation_FailedAgentInfoFields
    - TestGracefulDegradation_MultipleAgentsPartialFailure
    - TestErrAllAgentsFailed_ErrorMessage, TestErrConversationPaused_ErrorMessage
  - All 32 manager package tests pass (100%) with race detection enabled

- [x] Implement timeout handling:
  - Per-agent timeout in config (default: 30s)
  - Global conversation timeout for all agents combined
  - Context cancellation propagated to adapters
  - Partial response preservation on timeout
  - Emit EventAgentError with timeout details
  - Show timeout message in TUI

  **Completed**: Implemented comprehensive timeout handling with:
  - Created `pkg/v2/pool/timeout.go` with `TimeoutConfig` and `TimeoutHandler`:
    - `TimeoutConfig` with DefaultAgentTimeout (30s), GlobalConversationTimeout (120s), GracePeriod (2s), PreservePartialResponse (true)
    - `DefaultTimeoutConfig()` factory with sensible defaults
    - `TimeoutHandler` for managing per-agent timeouts with thread-safe operations
    - `SetAgentTimeout(agentID, timeout)` for per-agent configuration
    - `GetAgentTimeout(agentID)` with fallback to default
    - `SetAgentTimeouts([]AgentTimeoutConfig)` for bulk configuration
    - `CreateAgentContext()` creates context with agent-specific timeout
    - `CreateGlobalContext()` creates context for global conversation timeout
  - `TimeoutError` struct with detailed timeout context:
    - AgentID, AgentName, TimeoutDuration, ElapsedTime, PartialContent, IsGlobalTimeout
    - `ToAgentError()` converts to v2 errors.AgentError
    - `HasPartialContent()` helper method
  - `TimeoutResult` struct for operation results:
    - Success, Content, Metrics, Error, TimedOut, ElapsedTime, PartialContent
  - `ExecuteWithTimeout()` method with partial response preservation:
    - Runs operation in goroutine with timeout context
    - Collects partial content via channel during execution
    - Emits EventAgentError with timeout details on timeout
    - Logs timeout with structured fields
  - `TimeoutStats` for tracking timeout statistics:
    - TotalTimeouts, TimeoutsByAgent, AverageTimeoutDuration
    - PartialResponsesPreserved, GlobalTimeouts counters
    - Thread-safe with RWMutex
  - Updated `pkg/v2/pool/pool.go`:
    - Added `timeoutHandler` and `timeoutStats` fields to Pool
    - `NewPool()` initializes TimeoutHandler with default config
    - `NewPoolWithTimeoutConfig()` for custom timeout configuration
    - `NewPoolWithFullConfig()` for both circuit breaker and timeout config
    - `executeAgent()` uses per-agent timeout from handler
    - Records timeout stats when timeouts occur
    - Enhanced error logging with timeout flag
    - New methods: `SetAgentTimeout()`, `GetAgentTimeout()`, `SetAgentTimeouts()`
    - `GetTimeoutStats()`, `GetTimeoutHandler()` for monitoring
  - Updated `pkg/v2/config/config.go`:
    - Added `GlobalTimeout` and `PreservePartialResponse` to ConversationConfig
    - Added `Timeout` field to AgentConfig for per-agent override
    - `TimeoutConfigInfo` struct for extracting timeout configuration
    - `GetTimeoutConfig()` extracts all timeout settings
    - `GetAgentTimeout(agentID)` returns agent-specific or default timeout
  - Created `pkg/v2/pool/timeout_test.go` with 20 comprehensive tests:
    - TestDefaultTimeoutConfig, TestTimeoutHandler_SetAndGetAgentTimeout
    - TestTimeoutHandler_SetAgentTimeouts, TestTimeoutHandler_CreateAgentContext
    - TestTimeoutHandler_CreateGlobalContext, TestTimeoutHandler_CreateGlobalContext_Disabled
    - TestTimeoutError, TestTimeoutError_GlobalTimeout
    - TestTimeoutHandler_ExecuteWithTimeout_Success, TestTimeoutHandler_ExecuteWithTimeout_Timeout
    - TestTimeoutHandler_ExecuteWithTimeout_NonRetryableError
    - TestTimeoutStats, TestTimeoutStats_Concurrent, TestIsContextTimeout
    - TestPool_SetAgentTimeout, TestPool_GetTimeoutStats, TestNewPoolWithTimeoutConfig
    - TestPool_SetAgentTimeouts, TestTimeoutResult_Fields
  - All 12 v2 test packages pass (100%) with race detection enabled

- [ ] Add request cancellation:
  - Cancel() method on AgentPool to stop all pending requests
  - Cancel individual agent via CancelAgent(agentID)
  - Clean cancellation without goroutine leaks
  - Update agent status to "cancelled" in TUI
  - User can press Ctrl+C during generation to cancel

- [ ] Write error handling tests:
  - TestRetryOnNetworkError: Verify retry with backoff
  - TestNoRetryOnAuthError: Verify immediate failure
  - TestRateLimitHandling: Verify retry-after respected
  - TestCircuitBreakerOpens: Verify circuit opens after failures
  - TestCircuitBreakerRecovers: Verify half-open and close
  - TestGracefulDegradation: Some agents fail, conversation continues
  - TestTimeout: Verify context cancellation works
  - TestCancellation: Verify clean cancellation

- [ ] Update TUI for error display:
  - Show error icon in agent list for failed agents
  - Show error details on agent select
  - Display inline error message in conversation
  - Show retry countdown for rate-limited agents
  - "Retrying in X seconds..." indicator
  - Option to manually retry failed agent

- [ ] Add health monitoring:
  - Periodic health checks on idle agents (every 60s)
  - Pre-flight health check before starting conversation
  - Warn user if agent unhealthy before sending message
  - Health status in agent list tooltip
