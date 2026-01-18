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

- [ ] Implement retry logic in `pkg/v2/adapters/retry.go`:
  - RetryConfig struct with MaxAttempts, InitialDelay, MaxDelay, BackoffMultiplier
  - DefaultRetryConfig: 3 attempts, 1s initial, 30s max, 2x multiplier
  - WithRetry() wrapper function for adapter methods
  - Exponential backoff with jitter
  - Only retry on IsRetryable() errors
  - Log each retry attempt with attempt number and delay
  - Return last error after all attempts exhausted

- [ ] Add retry support to API adapters:
  - Wrap SendMessage() and StreamMessage() with retry logic
  - Detect rate limit responses (HTTP 429) and parse retry-after header
  - Detect network errors (connection refused, timeout, DNS)
  - Detect auth errors (HTTP 401, 403) - do not retry
  - Emit EventAgentError with retry count on each failure

- [ ] Implement circuit breaker in `pkg/v2/adapters/circuit_breaker.go`:
  - CircuitBreaker struct with state (closed, open, half-open), failureCount, lastFailure
  - Open circuit after N consecutive failures (default: 5)
  - Half-open after cooldown period (default: 30s)
  - Close circuit on successful request in half-open state
  - AllowRequest() checking if request should proceed
  - RecordSuccess() and RecordFailure() updating state
  - Per-adapter circuit breakers

- [ ] Add circuit breaker to AgentPool:
  - Track circuit breaker per agent
  - Skip agents with open circuits in ExecuteParallel()
  - Emit EventAgentError with "circuit open" message
  - Periodically attempt half-open requests
  - Log circuit state transitions

- [ ] Implement graceful degradation in ConversationManager:
  - Continue conversation even if some agents fail
  - Emit system message when agent fails: "Claude is currently unavailable"
  - Track failed agents and retry on next user message
  - Option to pause conversation if all agents fail
  - Resume with available agents when possible

- [ ] Implement timeout handling:
  - Per-agent timeout in config (default: 30s)
  - Global conversation timeout for all agents combined
  - Context cancellation propagated to adapters
  - Partial response preservation on timeout
  - Emit EventAgentError with timeout details
  - Show timeout message in TUI

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
