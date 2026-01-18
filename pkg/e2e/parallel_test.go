//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/ASRagab/agentpipe/pkg/core"
	"github.com/ASRagab/agentpipe/pkg/manager"
)

// TestParallelTiming verifies that N agents complete in O(1) not O(N).
func TestParallelTiming(t *testing.T) {
	TimeoutWrapper(t, 15*time.Second, func(t *testing.T) {
		// Create 5 agents, each with 50ms delay
		agents := FixtureLargeGroup()
		harness := NewTestHarness(t, agents)
		defer harness.Cleanup()

		const agentDelay = 50 * time.Millisecond
		for _, agent := range agents {
			harness.SetMockDelay(agent.ID, agentDelay)
			harness.SetMockResponse(agent.ID, "Response from "+agent.Name)
		}

		harness.Manager.Start()
		ctx := context.Background()

		start := time.Now()
		responses, err := SendTestMessage(ctx, harness, "Hello")
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("SendTestMessage failed: %v", err)
		}

		// All 5 agents should respond
		if len(responses) != 5 {
			t.Errorf("expected 5 responses, got %d", len(responses))
		}

		// If serial: 5 × 50ms = 250ms
		// If parallel: ~50-100ms (with overhead)
		// We allow up to 150ms for parallel execution
		maxParallelTime := 150 * time.Millisecond
		if elapsed > maxParallelTime {
			t.Errorf("execution took %v, expected < %v (parallel execution)", elapsed, maxParallelTime)
		}

		// Should be at least the delay of one agent
		minTime := agentDelay - 10*time.Millisecond
		if elapsed < minTime {
			t.Errorf("execution too fast: %v < %v", elapsed, minTime)
		}
	})
}

// TestMixedResponseTimes verifies that fast agents appear before slow agents.
func TestMixedResponseTimes(t *testing.T) {
	TimeoutWrapper(t, 15*time.Second, func(t *testing.T) {
		agents := []core.Agent{
			core.NewAgent("very-fast", "mock", "Very Fast", "model", "mock"),
			core.NewAgent("fast", "mock", "Fast", "model", "mock"),
			core.NewAgent("slow", "mock", "Slow", "model", "mock"),
		}
		harness := NewTestHarness(t, agents)
		defer harness.Cleanup()

		harness.SetMockDelay("very-fast", 25*time.Millisecond)
		harness.SetMockDelay("fast", 50*time.Millisecond)
		harness.SetMockDelay("slow", 150*time.Millisecond)

		harness.SetMockResponse("very-fast", "I'm very fast!")
		harness.SetMockResponse("fast", "I'm fast!")
		harness.SetMockResponse("slow", "I'm slow...")

		harness.Manager.Start()
		ctx := context.Background()

		start := time.Now()
		responses, err := SendTestMessage(ctx, harness, "Race!")
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("SendTestMessage failed: %v", err)
		}

		if len(responses) != 3 {
			t.Errorf("expected 3 responses, got %d", len(responses))
		}

		// Total time should be ~150ms (slowest), not 225ms (sum)
		if elapsed > 250*time.Millisecond {
			t.Errorf("took too long: %v (should be ~150ms)", elapsed)
		}

		// Verify done events have correct order
		doneEvents := harness.EventLog.EventsByType(core.EventAgentDone)
		if len(doneEvents) != 3 {
			t.Fatalf("expected 3 done events, got %d", len(doneEvents))
		}

		// Events should be in completion order (very-fast, fast, slow)
		expectedOrder := []string{"Very Fast", "Fast", "Slow"}
		for i, event := range doneEvents {
			if data, ok := event.Data.(core.AgentDoneData); ok {
				if data.AgentName != expectedOrder[i] {
					t.Errorf("event %d: expected %s, got %s", i, expectedOrder[i], data.AgentName)
				}
			}
		}
	})
}

// TestAllAgentsFail verifies graceful handling when all agents fail.
func TestAllAgentsFail(t *testing.T) {
	TimeoutWrapper(t, 10*time.Second, func(t *testing.T) {
		agents := FixtureSimpleConversation()
		harness := NewTestHarness(t, agents, WithGracefulDegradation(manager.GracefulDegradationConfig{
			Enabled:            true,
			PauseOnAllFailed:   true,
			EmitSystemMessages: true,
		}))
		defer harness.Cleanup()

		// Set all agents to fail
		harness.SetMockError("alice", ErrNetworkTimeout)
		harness.SetMockError("bob", ErrNetworkTimeout)

		harness.Manager.Start()
		ctx := context.Background()

		_, err := SendTestMessage(ctx, harness, "Hello")

		// Should return ErrAllAgentsFailed when all agents fail
		if err != manager.ErrAllAgentsFailed {
			t.Errorf("expected ErrAllAgentsFailed, got %v", err)
		}

		// Verify conversation is paused
		if !harness.Manager.IsPaused() {
			t.Error("expected conversation to be paused")
		}

		// Verify error events were emitted
		errorEvents := harness.EventLog.EventsByType(core.EventAgentError)
		if len(errorEvents) != 2 {
			t.Errorf("expected 2 error events, got %d", len(errorEvents))
		}
	})
}

// TestPartialFailure verifies that 2/3 agents succeed and partial results are shown.
func TestPartialFailure(t *testing.T) {
	TimeoutWrapper(t, 10*time.Second, func(t *testing.T) {
		agents := []core.Agent{
			core.NewAgent("success-1", "mock", "Success One", "model", "mock"),
			core.NewAgent("fail", "mock", "Failure", "model", "mock"),
			core.NewAgent("success-2", "mock", "Success Two", "model", "mock"),
		}
		harness := NewTestHarness(t, agents, WithGracefulDegradation(manager.GracefulDegradationConfig{
			Enabled:            true,
			PauseOnAllFailed:   true,
			EmitSystemMessages: true,
		}))
		defer harness.Cleanup()

		harness.SetMockResponse("success-1", "I worked!")
		harness.SetMockError("fail", ErrRateLimit)
		harness.SetMockResponse("success-2", "Me too!")

		harness.Manager.Start()
		ctx := context.Background()

		responses, err := SendTestMessage(ctx, harness, "Hello")

		// Should not error since some agents succeeded
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have 3 responses (2 success + 1 system message for failure)
		if len(responses) != 3 {
			t.Errorf("expected 3 responses, got %d", len(responses))
		}

		// Conversation should NOT be paused (partial success)
		if harness.Manager.IsPaused() {
			t.Error("conversation should not be paused with partial success")
		}

		// Verify one error event
		AssertEventCount(t, harness.EventLog, core.EventAgentError, 1)

		// Verify two success events
		AssertEventCount(t, harness.EventLog, core.EventAgentDone, 2)
	})
}

// TestAgentTimeout verifies that one agent times out while others succeed.
func TestAgentTimeout(t *testing.T) {
	TimeoutWrapper(t, 15*time.Second, func(t *testing.T) {
		agents := []core.Agent{
			core.NewAgent("fast-1", "mock", "Fast One", "model", "mock"),
			core.NewAgent("timeout", "mock", "Timeout Agent", "model", "mock"),
			core.NewAgent("fast-2", "mock", "Fast Two", "model", "mock"),
		}
		// Short timeout to test timeout behavior
		harness := NewTestHarness(t, agents, WithTimeout(200*time.Millisecond))
		defer harness.Cleanup()

		harness.SetMockDelay("fast-1", 30*time.Millisecond)
		harness.SetMockDelay("timeout", 500*time.Millisecond) // Will timeout
		harness.SetMockDelay("fast-2", 30*time.Millisecond)

		harness.SetMockResponse("fast-1", "Fast!")
		harness.SetMockResponse("timeout", "Too slow...")
		harness.SetMockResponse("fast-2", "Also fast!")

		harness.Manager.Start()
		ctx := context.Background()

		start := time.Now()
		responses, err := SendTestMessage(ctx, harness, "Hello")
		elapsed := time.Since(start)

		// No error because graceful degradation handles partial failures
		if err != nil {
			t.Logf("got error (may be expected): %v", err)
		}

		// Should get responses (or errors) for all agents
		if len(responses) < 2 {
			t.Errorf("expected at least 2 responses, got %d", len(responses))
		}

		// Should complete around the timeout (200ms)
		if elapsed > 400*time.Millisecond {
			t.Errorf("took too long: %v", elapsed)
		}

		t.Logf("completed in %v with %d responses", elapsed, len(responses))
	})
}

// TestNoAgentTimeout verifies normal operation within timeout.
func TestNoAgentTimeout(t *testing.T) {
	TimeoutWrapper(t, 10*time.Second, func(t *testing.T) {
		agents := FixtureSimpleConversation()
		harness := NewTestHarness(t, agents, WithTimeout(5*time.Second))
		defer harness.Cleanup()

		// Quick responses
		harness.SetMockDelay("alice", 30*time.Millisecond)
		harness.SetMockDelay("bob", 30*time.Millisecond)
		harness.SetMockResponse("alice", "Hello from Alice")
		harness.SetMockResponse("bob", "Hello from Bob")

		harness.Manager.Start()
		ctx := context.Background()

		responses, err := SendTestMessage(ctx, harness, "Hi")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(responses) != 2 {
			t.Errorf("expected 2 responses, got %d", len(responses))
		}

		// No timeout should have occurred
		AssertNoErrors(t, harness.EventLog)
	})
}

// TestConcurrentUserMessages verifies handling of rapid sequential messages.
func TestConcurrentUserMessages(t *testing.T) {
	TimeoutWrapper(t, 15*time.Second, func(t *testing.T) {
		agents := []core.Agent{
			core.NewAgent("agent", "mock", "Agent", "model", "mock"),
		}
		harness := NewTestHarness(t, agents)
		defer harness.Cleanup()

		harness.SetMockDelay("agent", 30*time.Millisecond)

		harness.Manager.Start()
		ctx := context.Background()

		// Send multiple messages rapidly
		for i := 0; i < 5; i++ {
			harness.SetMockResponse("agent", "Response to "+string(rune('A'+i)))
			_, err := SendTestMessage(ctx, harness, "Message "+string(rune('A'+i)))
			if err != nil {
				t.Errorf("message %d failed: %v", i, err)
			}
		}

		// Verify all messages processed
		messages := harness.Manager.GetMessages()
		// 5 user + 5 agent = 10
		if len(messages) != 10 {
			t.Errorf("expected 10 messages, got %d", len(messages))
		}

		// Verify message order is correct
		for i := 0; i < 5; i++ {
			userMsg := messages[i*2]
			if userMsg.Role != core.RoleUser {
				t.Errorf("message %d should be user", i*2)
			}
			agentMsg := messages[i*2+1]
			if agentMsg.Role != core.RoleAgent {
				t.Errorf("message %d should be agent", i*2+1)
			}
		}
	})
}

// TestParallelWithVaryingAgentCounts tests parallel execution with different agent counts.
func TestParallelWithVaryingAgentCounts(t *testing.T) {
	testCases := []int{1, 2, 3, 5, 10}

	for _, count := range testCases {
		t.Run(string(rune('0'+count))+"_agents", func(t *testing.T) {
			TimeoutWrapper(t, 10*time.Second, func(t *testing.T) {
				agents := make([]core.Agent, count)
				for i := 0; i < count; i++ {
					agents[i] = core.NewAgent(
						AgentID(i),
						"mock",
						AgentName(i),
						"model",
						"mock",
					)
				}
				harness := NewTestHarness(t, agents)
				defer harness.Cleanup()

				const agentDelay = 25 * time.Millisecond
				for _, agent := range agents {
					harness.SetMockDelay(agent.ID, agentDelay)
					harness.SetMockResponse(agent.ID, "Response")
				}

				harness.Manager.Start()
				ctx := context.Background()

				start := time.Now()
				responses, err := SendTestMessage(ctx, harness, "Hello")
				elapsed := time.Since(start)

				if err != nil {
					t.Fatalf("SendTestMessage failed: %v", err)
				}

				if len(responses) != count {
					t.Errorf("expected %d responses, got %d", count, len(responses))
				}

				// Parallel execution should complete in roughly the same time
				// regardless of agent count (within reasonable overhead)
				maxTime := agentDelay + time.Duration(count)*10*time.Millisecond + 50*time.Millisecond
				if elapsed > maxTime {
					t.Errorf("took %v, expected < %v", elapsed, maxTime)
				}
			})
		})
	}
}

// TestParallelExecutionConsistency verifies consistent results across multiple runs.
func TestParallelExecutionConsistency(t *testing.T) {
	TimeoutWrapper(t, 30*time.Second, func(t *testing.T) {
		for run := 0; run < 5; run++ {
			agents := FixtureSimpleConversation()
			harness := NewTestHarness(t, agents)

			harness.SetMockDelay("alice", 30*time.Millisecond)
			harness.SetMockDelay("bob", 30*time.Millisecond)
			harness.SetMockResponse("alice", "Alice says hi")
			harness.SetMockResponse("bob", "Bob says hello")

			harness.Manager.Start()
			ctx := context.Background()

			responses, err := SendTestMessage(ctx, harness, "Hello")
			if err != nil {
				t.Errorf("run %d: unexpected error: %v", run, err)
			}

			if len(responses) != 2 {
				t.Errorf("run %d: expected 2 responses, got %d", run, len(responses))
			}

			// Verify both agents responded
			foundAlice, foundBob := false, false
			for _, resp := range responses {
				if resp.AgentID == "alice" {
					foundAlice = true
				}
				if resp.AgentID == "bob" {
					foundBob = true
				}
			}

			if !foundAlice || !foundBob {
				t.Errorf("run %d: missing agent response (alice=%v, bob=%v)", run, foundAlice, foundBob)
			}

			harness.Cleanup()
		}
	})
}
