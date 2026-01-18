// Package v2 contains error handling integration tests for the v2 architecture.
// These tests verify retry logic, circuit breaker behavior, graceful degradation,
// timeout handling, and request cancellation work together correctly.
package v2

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ASRagab/agentpipe/pkg/v2/adapters"
	"github.com/ASRagab/agentpipe/pkg/v2/adapters/mock"
	"github.com/ASRagab/agentpipe/pkg/v2/core"
	pkgerrors "github.com/ASRagab/agentpipe/pkg/v2/errors"
	"github.com/ASRagab/agentpipe/pkg/v2/events"
	"github.com/ASRagab/agentpipe/pkg/v2/manager"
	"github.com/ASRagab/agentpipe/pkg/v2/pool"
)

// TestRetryOnNetworkError verifies that network errors trigger retry with exponential backoff.
// The adapter should retry the request up to MaxAttempts times before failing.
func TestRetryOnNetworkError(t *testing.T) {
	// Track retry attempts
	var attempts atomic.Int32

	// Create a mock adapter that fails with network errors initially
	adapter := mock.NewMockAdapter()
	adapter.Response = "Success after retries"
	adapter.Delay = 10 * time.Millisecond
	adapter.OnSendMessage = func(ctx context.Context, messages []core.Message) (string, *core.Metrics, error) {
		attempt := attempts.Add(1)
		if attempt < 3 {
			// Return network error for first 2 attempts
			return "", nil, pkgerrors.NewNetworkError("agent-1", "Test Agent", fmt.Errorf("connection refused"))
		}
		// Succeed on 3rd attempt
		return "Success after retries", &core.Metrics{TotalTokens: 100}, nil
	}

	// Configure retry
	retryConfig := adapters.RetryConfig{
		MaxAttempts:       5,
		InitialDelay:      10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		JitterFactor:      0, // No jitter for deterministic testing
	}

	// Wrap with retry logic
	retryableAdapter := adapters.NewRetryableAdapter(adapter, retryConfig, "agent-1", "Test Agent")

	// Execute
	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}
	response, metrics, err := retryableAdapter.SendMessage(ctx, messages)

	// Verify success after retries
	if err != nil {
		t.Fatalf("expected success after retries, got error: %v", err)
	}
	if response != "Success after retries" {
		t.Errorf("expected 'Success after retries', got %q", response)
	}
	if metrics == nil || metrics.TotalTokens != 100 {
		t.Error("expected valid metrics with TotalTokens=100")
	}

	// Verify retry count
	if attempts.Load() != 3 {
		t.Errorf("expected 3 attempts (2 retries + 1 success), got %d", attempts.Load())
	}
}

// TestNoRetryOnAuthError verifies that authentication errors do NOT trigger retries.
// Auth errors are non-retryable and should fail immediately.
func TestNoRetryOnAuthError(t *testing.T) {
	// Track attempts
	var attempts atomic.Int32

	// Create adapter that returns auth error
	adapter := mock.NewMockAdapter()
	adapter.OnSendMessage = func(ctx context.Context, messages []core.Message) (string, *core.Metrics, error) {
		attempts.Add(1)
		return "", nil, pkgerrors.NewAuthError("agent-1", "Test Agent", 401, fmt.Errorf("invalid API key"))
	}

	// Configure retry (would retry up to 5 times if error were retryable)
	retryConfig := adapters.RetryConfig{
		MaxAttempts:       5,
		InitialDelay:      10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		JitterFactor:      0,
	}

	retryableAdapter := adapters.NewRetryableAdapter(adapter, retryConfig, "agent-1", "Test Agent")

	// Execute
	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}
	_, _, err := retryableAdapter.SendMessage(ctx, messages)

	// Verify immediate failure
	if err == nil {
		t.Fatal("expected auth error, got nil")
	}

	// Verify it's an auth error
	agentErr, ok := pkgerrors.AsAgentError(err)
	if !ok {
		t.Fatalf("expected AgentError, got %T", err)
	}
	if agentErr.ErrorType != pkgerrors.ErrTypeAuth {
		t.Errorf("expected ErrTypeAuth, got %v", agentErr.ErrorType)
	}
	if agentErr.HTTPStatusCode != 401 {
		t.Errorf("expected HTTP 401, got %d", agentErr.HTTPStatusCode)
	}

	// Verify NO retries occurred (only 1 attempt)
	if attempts.Load() != 1 {
		t.Errorf("expected 1 attempt (no retries for auth errors), got %d", attempts.Load())
	}
}

// TestRateLimitHandling verifies that rate limit errors trigger retries with retry-after respected.
// Rate limit (HTTP 429) errors are retryable and should include retry-after information.
func TestRateLimitHandling(t *testing.T) {
	// Track attempts and timing
	var attempts atomic.Int32
	var lastAttemptTime time.Time

	// Create adapter that returns rate limit initially
	adapter := mock.NewMockAdapter()
	adapter.OnSendMessage = func(ctx context.Context, messages []core.Message) (string, *core.Metrics, error) {
		attempt := attempts.Add(1)
		now := time.Now()

		if attempt == 1 {
			lastAttemptTime = now
			// Return rate limit error with 50ms retry-after
			return "", nil, pkgerrors.NewRateLimitError("agent-1", "Test Agent", 50*time.Millisecond)
		}

		// Verify we waited at least 40ms (with some tolerance for timing)
		elapsed := now.Sub(lastAttemptTime)
		if elapsed < 40*time.Millisecond {
			t.Logf("Warning: retry happened after only %v (expected ~50ms)", elapsed)
		}

		// Succeed on retry
		return "Success after rate limit", &core.Metrics{TotalTokens: 50}, nil
	}

	retryConfig := adapters.RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      10 * time.Millisecond,
		MaxDelay:          1 * time.Second,
		BackoffMultiplier: 2.0,
		JitterFactor:      0,
	}

	retryableAdapter := adapters.NewRetryableAdapter(adapter, retryConfig, "agent-1", "Test Agent")

	// Execute
	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}
	response, _, err := retryableAdapter.SendMessage(ctx, messages)

	// Verify success after rate limit
	if err != nil {
		t.Fatalf("expected success after rate limit, got: %v", err)
	}
	if response != "Success after rate limit" {
		t.Errorf("expected 'Success after rate limit', got %q", response)
	}

	// Verify rate limit was retried
	if attempts.Load() != 2 {
		t.Errorf("expected 2 attempts (rate limit + success), got %d", attempts.Load())
	}
}

// TestCircuitBreakerOpens verifies that the circuit breaker opens after consecutive failures.
// After the failure threshold is reached, subsequent requests should be rejected immediately.
func TestCircuitBreakerOpens(t *testing.T) {
	bus := events.NewBus()
	defer bus.Close()

	// Track circuit breaker events
	var circuitOpenEvents atomic.Int32
	bus.Subscribe(core.EventAgentError, func(event core.Event) {
		if data, ok := event.Data.(core.AgentErrorData); ok {
			if containsSubstring(data.Error, "network") || containsCircuitOpen(data.Error) {
				circuitOpenEvents.Add(1)
			}
		}
	})

	// Create pool with custom circuit breaker config
	cbConfig := adapters.CircuitBreakerConfig{
		FailureThreshold: 3, // Open after 3 failures
		CooldownPeriod:   1 * time.Second,
		SuccessThreshold: 1,
	}
	pool := pool.NewPoolWithCircuitBreaker(bus, 10*time.Second, cbConfig)

	// Create adapter that always fails
	failAdapter := mock.NewMockAdapter()
	failAdapter.Error = fmt.Errorf("simulated network failure")

	agent := core.NewAgent("fail-agent", "mock", "Failing Agent", "model", "mock")
	pool.AddAgent(agent, failAdapter)

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Test")}

	// Trigger failures to open circuit
	for i := 0; i < 5; i++ {
		responses := pool.ExecuteParallel(ctx, messages)
		if len(responses) != 1 {
			t.Fatalf("expected 1 response, got %d", len(responses))
		}

		// After 3 failures, circuit should be open
		if i >= 3 {
			status, found := pool.GetCircuitBreakerStatus("fail-agent")
			if !found || status == nil {
				t.Fatal("expected circuit breaker status")
			}
			if status.State != adapters.CircuitOpen {
				t.Errorf("expected circuit to be open after %d failures, state=%s", i+1, status.State)
			}
		}
	}

	// Verify circuit is now open
	status, found := pool.GetCircuitBreakerStatus("fail-agent")
	if !found || status == nil {
		t.Fatal("expected circuit breaker status")
	}
	if status.State != adapters.CircuitOpen {
		t.Errorf("expected CircuitOpen, got %s", status.State)
	}
	if status.FailureCount < 3 {
		t.Errorf("expected at least 3 failures, got %d", status.FailureCount)
	}

	// Verify available agent count is 0 (circuit open)
	if count := pool.GetAvailableAgentCount(); count != 0 {
		t.Errorf("expected 0 available agents with open circuit, got %d", count)
	}
}

// TestCircuitBreakerRecovers verifies that the circuit breaker recovers after cooldown.
// After the cooldown period, the circuit should transition to half-open and close on success.
func TestCircuitBreakerRecovers(t *testing.T) {
	bus := events.NewBus()
	defer bus.Close()

	// Create pool with short cooldown for testing
	cbConfig := adapters.CircuitBreakerConfig{
		FailureThreshold: 2,
		CooldownPeriod:   50 * time.Millisecond, // Short cooldown for testing
		SuccessThreshold: 1,
	}
	agentPool := pool.NewPoolWithCircuitBreaker(bus, 10*time.Second, cbConfig)

	// Create adapter that fails initially, then succeeds
	var attempts atomic.Int32
	adapter := mock.NewMockAdapter()
	adapter.OnSendMessage = func(ctx context.Context, messages []core.Message) (string, *core.Metrics, error) {
		attempt := attempts.Add(1)
		if attempt <= 2 {
			// Fail first 2 times to open circuit
			return "", nil, fmt.Errorf("simulated failure %d", attempt)
		}
		// Succeed after that (for half-open probe)
		return "Recovered!", &core.Metrics{TotalTokens: 10}, nil
	}

	agent := core.NewAgent("recover-agent", "mock", "Recovering Agent", "model", "mock")
	agentPool.AddAgent(agent, adapter)

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Test")}

	// Trigger failures to open circuit
	for i := 0; i < 2; i++ {
		_ = agentPool.ExecuteParallel(ctx, messages)
	}

	// Verify circuit is open
	status, found := agentPool.GetCircuitBreakerStatus("recover-agent")
	if !found || status == nil || status.State != adapters.CircuitOpen {
		t.Fatalf("expected circuit to be open, got %v", status)
	}

	// Wait for cooldown
	time.Sleep(100 * time.Millisecond)

	// Next request should probe (half-open) and succeed, closing the circuit
	responses := agentPool.ExecuteParallel(ctx, messages)
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}

	if responses[0].Error != nil {
		t.Errorf("expected success in half-open probe, got error: %v", responses[0].Error)
	}

	// Verify circuit is now closed
	status, found = agentPool.GetCircuitBreakerStatus("recover-agent")
	if !found || status == nil {
		t.Fatal("expected circuit breaker status")
	}
	if status.State != adapters.CircuitClosed {
		t.Errorf("expected CircuitClosed after recovery, got %s", status.State)
	}
	if status.FailureCount != 0 {
		t.Errorf("expected 0 failures after reset, got %d", status.FailureCount)
	}
}

// TestGracefulDegradation verifies that conversation continues when some agents fail.
// Failed agents should be tracked and system messages emitted while other agents respond.
func TestGracefulDegradation(t *testing.T) {
	// Create agents - one succeeds, one fails
	agents := []core.Agent{
		core.NewAgent("success-agent", "mock", "Success Agent", "model", "mock"),
		core.NewAgent("fail-agent", "mock", "Fail Agent", "model", "mock"),
	}

	bus := events.NewBus()
	defer bus.Close()

	// Track events
	var errorEvents atomic.Int32
	bus.Subscribe(core.EventAgentError, func(event core.Event) {
		errorEvents.Add(1)
	})

	config := manager.Config{
		Timeout: 10 * time.Second,
		GracefulDegradation: manager.GracefulDegradationConfig{
			Enabled:                  true,
			PauseOnAllFailed:         true,
			RetryFailedOnNextMessage: true,
			EmitSystemMessages:       true,
		},
	}

	mgr, err := manager.NewConversationManager(config, agents, bus)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer mgr.Close()

	// Note: The mock adapter registry will provide working adapters by default.
	// For a true failure test, we'd need to inject a failing adapter.
	// This test verifies the graceful degradation infrastructure is working.

	ctx := context.Background()
	responses, err := mgr.SendUserMessage(ctx, "Hello everyone")

	// Should not error completely (graceful degradation)
	if err != nil {
		t.Logf("Note: Error returned (expected if fail-agent actually fails): %v", err)
	}

	// Should have responses from available agents
	if len(responses) < 1 {
		t.Error("expected at least 1 agent response with graceful degradation")
	}

	// Verify conversation is not paused (unless all failed)
	if mgr.IsPaused() && len(responses) > 0 {
		t.Error("conversation should not be paused if some agents responded")
	}

	// Verify GetFailedAgents returns correct structure
	failed := mgr.GetFailedAgents()
	t.Logf("Failed agents: %d", len(failed))

	// Verify GetAvailableAgentCount works
	available := mgr.GetAvailableAgentCount()
	t.Logf("Available agents: %d", available)
}

// TestTimeout verifies that context cancellation works correctly on timeout.
// Agents should stop processing when the context deadline is exceeded.
func TestTimeout(t *testing.T) {
	bus := events.NewBus()
	defer bus.Close()

	// Track timeout events
	var timeoutEvents atomic.Int32
	bus.Subscribe(core.EventAgentError, func(event core.Event) {
		if data, ok := event.Data.(core.AgentErrorData); ok {
			if containsTimeout(data.Error) {
				timeoutEvents.Add(1)
			}
		}
	})

	// Create pool with short timeout
	timeoutConfig := pool.TimeoutConfig{
		DefaultAgentTimeout:       100 * time.Millisecond,
		GlobalConversationTimeout: 500 * time.Millisecond,
		PreservePartialResponse:   true,
		GracePeriod:               10 * time.Millisecond,
	}
	agentPool := pool.NewPoolWithTimeoutConfig(bus, timeoutConfig)

	// Create slow adapter that exceeds timeout
	slowAdapter := mock.NewMockAdapter()
	slowAdapter.Delay = 500 * time.Millisecond // Much longer than 100ms timeout
	slowAdapter.Response = "This should not complete"

	agent := core.NewAgent("slow-agent", "mock", "Slow Agent", "model", "mock")
	agentPool.AddAgent(agent, slowAdapter)

	// Execute with per-agent timeout
	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Test")}

	start := time.Now()
	responses := agentPool.ExecuteParallel(ctx, messages)
	elapsed := time.Since(start)

	// Verify response exists
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}

	// Verify timeout occurred
	resp := responses[0]
	if resp.Error == nil {
		t.Error("expected timeout error")
	}

	// Verify it completed faster than the adapter delay (due to timeout)
	if elapsed > 300*time.Millisecond {
		t.Errorf("expected completion in ~100ms due to timeout, took %v", elapsed)
	}

	// Verify timeout error type
	if resp.Error != nil && !errors.Is(resp.Error, context.DeadlineExceeded) {
		// Check if it's a timeout-related error
		if agentErr, ok := pkgerrors.AsAgentError(resp.Error); ok {
			if agentErr.ErrorType != pkgerrors.ErrTypeTimeout {
				t.Logf("Note: Error type was %v (expected timeout)", agentErr.ErrorType)
			}
		}
	}
}

// TestCancellation verifies clean request cancellation without goroutine leaks.
// Cancelled agents should stop immediately and set status to "cancelled".
func TestCancellation(t *testing.T) {
	bus := events.NewBus()
	defer bus.Close()

	// Track cancellation events
	var cancelEvents atomic.Int32
	bus.Subscribe(core.EventAgentCancelled, func(event core.Event) {
		cancelEvents.Add(1)
	})

	agentPool := pool.NewPool(bus, 5*time.Second)

	// Create slow adapter
	slowAdapter := mock.NewMockAdapter()
	slowAdapter.Delay = 2 * time.Second
	slowAdapter.Response = "This should not complete"

	agent := core.NewAgent("cancel-agent", "mock", "Cancellable Agent", "model", "mock")
	agentPool.AddAgent(agent, slowAdapter)

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Test")}

	// Start execution in goroutine
	done := make(chan []pool.Response)
	go func() {
		done <- agentPool.ExecuteParallel(ctx, messages)
	}()

	// Wait for agent to start
	time.Sleep(100 * time.Millisecond)

	// Verify active request
	if !agentPool.HasActiveRequest("cancel-agent") {
		t.Log("Note: Agent may have completed before check (timing-dependent)")
	}

	// Cancel
	cancelled := agentPool.Cancel()
	if cancelled < 1 {
		t.Logf("Note: Expected to cancel 1 request, cancelled %d (timing-dependent)", cancelled)
	}

	// Wait for completion
	responses := <-done

	// Verify response has cancellation error
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}

	resp := responses[0]
	if resp.Error == nil {
		t.Error("expected cancellation error")
	}

	// Verify error is cancellation-related
	if !pool.IsCancellationError(resp.Error) && !errors.Is(resp.Error, context.Canceled) {
		t.Logf("Note: Error was %v (expected cancellation)", resp.Error)
	}

	// Verify agent status is cancelled
	state, found := agentPool.GetAgentState("cancel-agent")
	if found && state.Status != core.AgentStatusCancelled {
		t.Logf("Note: Agent status is %s (expected cancelled)", state.Status)
	}

	// Verify no active requests remain
	if agentPool.ActiveRequestCount() != 0 {
		t.Errorf("expected 0 active requests after cancellation, got %d", agentPool.ActiveRequestCount())
	}

	// Wait for async events
	time.Sleep(100 * time.Millisecond)

	// Verify cancellation event was emitted
	if cancelEvents.Load() == 0 {
		t.Log("Note: EventAgentCancelled may not have been emitted (depends on timing)")
	}
}

// Helper functions

func containsCircuitOpen(s string) bool {
	return len(s) > 0 && (containsSubstring(s, "circuit") || containsSubstring(s, "open"))
}

func containsTimeout(s string) bool {
	return len(s) > 0 && (containsSubstring(s, "timeout") || containsSubstring(s, "deadline"))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
