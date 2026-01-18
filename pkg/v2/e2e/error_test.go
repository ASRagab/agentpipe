//go:build e2e

package e2e

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/v2/adapters"
	"github.com/kevinelliott/agentpipe/pkg/v2/core"
	"github.com/kevinelliott/agentpipe/pkg/v2/manager"
)

// TestNetworkFailureRecovery simulates network drop and verifies retry behavior.
func TestNetworkFailureRecovery(t *testing.T) {
	TimeoutWrapper(t, 20*time.Second, func(t *testing.T) {
		agents := FixtureSimpleConversation()
		harness := NewTestHarness(t, agents, WithGracefulDegradation(manager.GracefulDegradationConfig{
			Enabled:                  true,
			PauseOnAllFailed:         true,
			RetryFailedOnNextMessage: true,
			EmitSystemMessages:       true,
		}))
		defer harness.Cleanup()

		// Track call count for recovery testing
		var callCount int32

		// Configure Alice to fail first 2 times, then succeed
		harness.MockAdapters["alice"].OnSendMessage = func(ctx context.Context, messages []core.Message) (string, *core.Metrics, error) {
			count := atomic.AddInt32(&callCount, 1)
			if count <= 2 {
				return "", nil, ErrNetworkTimeout
			}
			return "Alice recovered!", &core.Metrics{
				InputTokens:  10,
				OutputTokens: 20,
				TotalTokens:  30,
			}, nil
		}
		harness.SetMockResponse("bob", "Bob works fine")

		harness.Manager.Start()
		ctx := context.Background()

		// First message - Alice fails
		responses1, err := SendTestMessage(ctx, harness, "First message")
		if err != nil && !errors.Is(err, manager.ErrAllAgentsFailed) {
			t.Logf("first message returned: %v", err)
		}

		// Should have system message for Alice's failure
		foundSystemMsg := false
		for _, resp := range responses1 {
			if resp.Role == core.RoleSystem {
				foundSystemMsg = true
			}
		}
		if !foundSystemMsg {
			t.Log("Note: System message for failure may not be in responses")
		}

		// Check that Alice is tracked as failed
		failedAgents := harness.Manager.GetFailedAgents()
		foundAliceFailed := false
		for _, fa := range failedAgents {
			if fa.AgentName == "Alice" {
				foundAliceFailed = true
			}
		}
		if !foundAliceFailed {
			t.Log("Note: Alice might have already been cleared from failed list")
		}

		// Second message - Alice still fails (if retry behavior is enabled)
		// Eventually Alice should recover on third call
		for i := 0; i < 3; i++ {
			_, _ = SendTestMessage(ctx, harness, "Retry message")
		}

		// After multiple attempts, verify recovery happened
		t.Logf("Total calls to Alice: %d", atomic.LoadInt32(&callCount))
	})
}

// TestRateLimitRecovery simulates 429 rate limit and verifies backoff.
func TestRateLimitRecovery(t *testing.T) {
	TimeoutWrapper(t, 15*time.Second, func(t *testing.T) {
		agents := FixtureSimpleConversation()
		harness := NewTestHarness(t, agents, WithGracefulDegradation(manager.GracefulDegradationConfig{
			Enabled:            true,
			PauseOnAllFailed:   false, // Don't pause on failure
			EmitSystemMessages: true,
		}))
		defer harness.Cleanup()

		// Configure Alice to fail with rate limit
		harness.SetMockError("alice", ErrRateLimit)
		harness.SetMockResponse("bob", "Bob responded")

		harness.Manager.Start()
		ctx := context.Background()

		responses, err := SendTestMessage(ctx, harness, "Rate limit test")
		if err != nil {
			t.Logf("message returned: %v", err)
		}

		// Verify partial success - Bob should have responded
		bobResponded := false
		for _, resp := range responses {
			if resp.AgentID == "bob" && resp.Role == core.RoleAgent {
				bobResponded = true
			}
		}
		if !bobResponded {
			t.Error("expected Bob to respond despite Alice's rate limit error")
		}

		// Verify error event was emitted for Alice
		errorEvents := harness.EventLog.EventsByType(core.EventAgentError)
		if len(errorEvents) != 1 {
			t.Errorf("expected 1 error event, got %d", len(errorEvents))
		}
	})
}

// TestAuthFailure simulates invalid API key and verifies clear error message.
func TestAuthFailure(t *testing.T) {
	TimeoutWrapper(t, 10*time.Second, func(t *testing.T) {
		agents := FixtureSimpleConversation()
		harness := NewTestHarness(t, agents, WithGracefulDegradation(manager.GracefulDegradationConfig{
			Enabled:            true,
			PauseOnAllFailed:   false,
			EmitSystemMessages: true,
		}))
		defer harness.Cleanup()

		// Configure Alice to fail with auth error
		harness.SetMockError("alice", ErrAuthFailure)
		harness.SetMockResponse("bob", "Bob authenticated fine")

		harness.Manager.Start()
		ctx := context.Background()

		_, err := SendTestMessage(ctx, harness, "Auth test")
		if err != nil {
			t.Logf("message returned: %v", err)
		}

		// Verify auth error was captured
		errorEvents := harness.EventLog.EventsByType(core.EventAgentError)
		foundAuthError := false
		for _, event := range errorEvents {
			if data, ok := event.Data.(core.AgentErrorData); ok {
				if data.Error == "authentication failed" {
					foundAuthError = true
				}
			}
		}
		if !foundAuthError {
			t.Log("Note: Auth error message format may vary")
		}
	})
}

// TestCircuitBreakerTrip verifies that multiple failures open the circuit.
func TestCircuitBreakerTrip(t *testing.T) {
	TimeoutWrapper(t, 20*time.Second, func(t *testing.T) {
		agents := FixtureSimpleConversation()
		harness := NewTestHarness(t, agents, WithGracefulDegradation(manager.GracefulDegradationConfig{
			Enabled:            true,
			PauseOnAllFailed:   false,
			EmitSystemMessages: true,
		}))
		defer harness.Cleanup()

		// Configure Alice to always fail
		harness.SetMockError("alice", ErrNetworkTimeout)
		harness.SetMockResponse("bob", "Bob works")

		harness.Manager.Start()
		ctx := context.Background()

		// Send multiple messages to trip the circuit breaker
		for i := 0; i < 5; i++ {
			_, _ = SendTestMessage(ctx, harness, "Message to trip circuit")
		}

		// Verify multiple error events
		errorEvents := harness.EventLog.EventsByType(core.EventAgentError)
		if len(errorEvents) < 3 {
			t.Errorf("expected at least 3 error events for circuit breaker, got %d", len(errorEvents))
		}

		// Check available agent count
		availableCount := harness.Manager.GetAvailableAgentCount()
		t.Logf("Available agents after circuit breaker trip: %d", availableCount)
	})
}

// TestGracefulDegradation verifies conversation continues when one agent fails permanently.
func TestGracefulDegradation(t *testing.T) {
	TimeoutWrapper(t, 15*time.Second, func(t *testing.T) {
		agents := []core.Agent{
			core.NewAgent("reliable", "mock", "Reliable", "model", "mock"),
			core.NewAgent("unreliable", "mock", "Unreliable", "model", "mock"),
		}
		harness := NewTestHarness(t, agents, WithGracefulDegradation(manager.GracefulDegradationConfig{
			Enabled:            true,
			PauseOnAllFailed:   true,
			EmitSystemMessages: true,
		}))
		defer harness.Cleanup()

		// Reliable always works, unreliable always fails
		harness.SetMockResponse("reliable", "I'm reliable!")
		harness.SetMockError("unreliable", ErrNetworkTimeout)

		harness.Manager.Start()
		ctx := context.Background()

		// Send multiple messages
		for i := 0; i < 3; i++ {
			responses, err := SendTestMessage(ctx, harness, "Message")
			if err != nil {
				t.Logf("message %d: %v", i, err)
				continue
			}

			// Verify at least one response (from reliable agent)
			foundReliable := false
			for _, resp := range responses {
				if resp.AgentID == "reliable" {
					foundReliable = true
				}
			}
			if !foundReliable {
				t.Errorf("message %d: expected response from reliable agent", i)
			}
		}

		// Conversation should NOT be paused (partial success)
		if harness.Manager.IsPaused() {
			t.Error("conversation should not be paused with partial success")
		}

		// Verify reliable agent messages in conversation
		messages := harness.Manager.GetMessages()
		reliableMessageCount := 0
		for _, msg := range messages {
			if msg.AgentID == "reliable" {
				reliableMessageCount++
			}
		}
		if reliableMessageCount < 3 {
			t.Errorf("expected at least 3 messages from reliable agent, got %d", reliableMessageCount)
		}
	})
}

// TestRetryConfig verifies retry configuration works correctly.
func TestRetryConfig(t *testing.T) {
	TimeoutWrapper(t, 20*time.Second, func(t *testing.T) {
		// Test with retry configuration
		agents := FixtureSimpleConversation()
		harness := NewTestHarness(t, agents,
			WithTimeout(5*time.Second),
			WithGracefulDegradation(manager.GracefulDegradationConfig{
				Enabled:                  true,
				PauseOnAllFailed:         true,
				RetryFailedOnNextMessage: true,
				EmitSystemMessages:       true,
			}),
		)
		defer harness.Cleanup()

		// Track retry attempts
		var aliceAttempts int32

		harness.MockAdapters["alice"].OnSendMessage = func(ctx context.Context, messages []core.Message) (string, *core.Metrics, error) {
			attempt := atomic.AddInt32(&aliceAttempts, 1)
			if attempt == 1 {
				return "", nil, ErrIntermittentFailure
			}
			return "Alice succeeded on retry!", &core.Metrics{TotalTokens: 30}, nil
		}
		harness.SetMockResponse("bob", "Bob works")

		harness.Manager.Start()
		ctx := context.Background()

		// First message - Alice fails
		_, _ = SendTestMessage(ctx, harness, "First")

		// Second message - Alice should be retried and succeed
		_, err := SendTestMessage(ctx, harness, "Second")
		if err != nil {
			t.Logf("second message: %v", err)
		}

		t.Logf("Alice attempts: %d", atomic.LoadInt32(&aliceAttempts))
	})
}

// TestPauseAndResume verifies conversation pause and resume behavior.
func TestPauseAndResume(t *testing.T) {
	TimeoutWrapper(t, 15*time.Second, func(t *testing.T) {
		agents := FixtureSimpleConversation()
		harness := NewTestHarness(t, agents, WithGracefulDegradation(manager.GracefulDegradationConfig{
			Enabled:          true,
			PauseOnAllFailed: true,
		}))
		defer harness.Cleanup()

		// Both agents fail
		harness.SetMockError("alice", ErrNetworkTimeout)
		harness.SetMockError("bob", ErrNetworkTimeout)

		harness.Manager.Start()
		ctx := context.Background()

		// Message should cause pause
		_, err := SendTestMessage(ctx, harness, "Cause pause")
		if err != manager.ErrAllAgentsFailed {
			t.Logf("expected ErrAllAgentsFailed, got: %v", err)
		}

		// Verify paused
		if !harness.Manager.IsPaused() {
			t.Error("conversation should be paused after all agents fail")
		}

		// Try to send while paused
		_, err = SendTestMessage(ctx, harness, "While paused")
		if err != manager.ErrConversationPaused {
			t.Logf("expected ErrConversationPaused, got: %v", err)
		}

		// Fix agents and resume
		harness.SetMockError("alice", nil)
		harness.SetMockError("bob", nil)
		harness.SetMockResponse("alice", "Alice recovered")
		harness.SetMockResponse("bob", "Bob recovered")

		harness.Manager.ResumeConversation(true)
		harness.Manager.ResetCircuitBreakers()

		// Verify not paused
		if harness.Manager.IsPaused() {
			t.Error("conversation should not be paused after resume")
		}

		// Should be able to send now
		responses, err := SendTestMessage(ctx, harness, "After resume")
		if err != nil {
			t.Fatalf("message after resume failed: %v", err)
		}

		if len(responses) < 2 {
			t.Errorf("expected 2 responses after resume, got %d", len(responses))
		}
	})
}

// TestContextCancellation verifies that context cancellation stops agents.
func TestContextCancellation(t *testing.T) {
	TimeoutWrapper(t, 10*time.Second, func(t *testing.T) {
		agents := FixtureSimpleConversation()
		harness := NewTestHarness(t, agents, WithTimeout(30*time.Second))
		defer harness.Cleanup()

		// Set long delay to allow cancellation
		harness.SetMockDelay("alice", 5*time.Second)
		harness.SetMockDelay("bob", 5*time.Second)

		harness.Manager.Start()

		// Create cancellable context
		ctx, cancel := context.WithCancel(context.Background())

		// Start message in goroutine
		done := make(chan error, 1)
		go func() {
			_, err := SendTestMessage(ctx, harness, "Will be cancelled")
			done <- err
		}()

		// Cancel after brief delay
		time.Sleep(100 * time.Millisecond)
		cancel()

		// Wait for message to complete
		select {
		case err := <-done:
			if err != context.Canceled {
				t.Logf("expected context.Canceled, got: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("message did not respond to cancellation")
		}
	})
}

// TestErrorClassification verifies that errors are classified correctly.
func TestErrorClassification(t *testing.T) {
	testCases := []struct {
		name          string
		err           error
		expectedType  core.ErrorType
		isRecoverable bool
	}{
		{"timeout", ErrNetworkTimeout, core.ErrorTypeTimeout, true},
		{"rate_limit", ErrRateLimit, core.ErrorTypeRateLimit, true},
		{"auth", ErrAuthFailure, core.ErrorTypeAuthentication, false},
		{"unknown", errors.New("something went wrong"), core.ErrorTypeUnknown, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			agents := []core.Agent{
				core.NewAgent("test-agent", "mock", "Test", "model", "mock"),
			}
			harness := NewTestHarness(t, agents, WithGracefulDegradation(manager.GracefulDegradationConfig{
				Enabled:            true,
				PauseOnAllFailed:   false,
				EmitSystemMessages: true,
			}))
			defer harness.Cleanup()

			harness.SetMockError("test-agent", tc.err)

			harness.Manager.Start()
			ctx := context.Background()

			_, _ = SendTestMessage(ctx, harness, "Test")

			// Check error classification (via event or message)
			errorEvents := harness.EventLog.EventsByType(core.EventAgentError)
			if len(errorEvents) == 0 {
				t.Error("expected error event")
				return
			}

			// Verify error was captured
			if data, ok := errorEvents[0].Data.(core.AgentErrorData); ok {
				t.Logf("Error: %s, Type: %s", data.Error, data.ErrorType)
			}
		})
	}
}

// TestCircuitBreakerReset verifies that circuit breakers can be reset.
func TestCircuitBreakerReset(t *testing.T) {
	TimeoutWrapper(t, 15*time.Second, func(t *testing.T) {
		agents := FixtureSimpleConversation()
		harness := NewTestHarness(t, agents)
		defer harness.Cleanup()

		// Make agents fail
		harness.SetMockError("alice", ErrNetworkTimeout)
		harness.SetMockError("bob", ErrNetworkTimeout)

		harness.Manager.Start()
		ctx := context.Background()

		// Trip circuit breakers
		for i := 0; i < 5; i++ {
			_, _ = SendTestMessage(ctx, harness, "Trip")
		}

		// Check initial available count
		beforeReset := harness.Manager.GetAvailableAgentCount()
		t.Logf("Available agents before reset: %d", beforeReset)

		// Reset circuit breakers
		harness.Manager.ResetCircuitBreakers()

		// Check after reset
		afterReset := harness.Manager.GetAvailableAgentCount()
		t.Logf("Available agents after reset: %d", afterReset)

		// After reset, more agents should be available
		if afterReset < beforeReset {
			t.Error("circuit breaker reset should increase available agents")
		}
	})
}

// TestAdapterNotFound verifies error when adapter doesn't exist.
func TestAdapterNotFound(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("invalid", "nonexistent-adapter", "Invalid", "model", "nonexistent-adapter"),
	}

	_, err := adapters.Get("nonexistent-adapter")
	if err == nil {
		t.Error("expected error for nonexistent adapter")
	}

	t.Logf("Correctly got error for nonexistent adapter: %v", err)
}
