package pool

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ASRagab/agentpipe/pkg/adapters"
	"github.com/ASRagab/agentpipe/pkg/adapters/mock"
	"github.com/ASRagab/agentpipe/pkg/core"
	"github.com/ASRagab/agentpipe/pkg/events"
)

func TestNewPool(t *testing.T) {
	bus := events.NewBus()
	defer bus.Close()

	pool := NewPool(bus, 30*time.Second)

	if pool == nil {
		t.Fatal("NewPool returned nil")
	}
	if pool.eventBus != bus {
		t.Error("eventBus not set correctly")
	}
	if pool.timeout != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", pool.timeout)
	}
	if pool.AgentCount() != 0 {
		t.Errorf("expected 0 agents, got %d", pool.AgentCount())
	}
}

func TestNewPool_DefaultTimeout(t *testing.T) {
	pool := NewPool(nil, 0)

	if pool.timeout != 60*time.Second {
		t.Errorf("expected default timeout 60s, got %v", pool.timeout)
	}
}

func TestAddAgent(t *testing.T) {
	pool := NewPool(nil, 0)

	agent := core.NewAgent("agent-1", "mock", "Agent 1", "mock-model", "mock")
	adapter := mock.NewMockAdapter()

	pool.AddAgent(agent, adapter)

	if pool.AgentCount() != 1 {
		t.Errorf("expected 1 agent, got %d", pool.AgentCount())
	}

	agents := pool.GetAgents()
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}
	if agents[0].ID != "agent-1" {
		t.Errorf("expected agent ID 'agent-1', got %q", agents[0].ID)
	}
}

func TestGetAgents(t *testing.T) {
	pool := NewPool(nil, 0)

	agent1 := core.NewAgent("agent-1", "mock", "Agent 1", "model-1", "mock")
	agent2 := core.NewAgent("agent-2", "mock", "Agent 2", "model-2", "mock")

	pool.AddAgent(agent1, mock.NewMockAdapter())
	pool.AddAgent(agent2, mock.NewMockAdapter())

	agents := pool.GetAgents()

	if len(agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(agents))
	}
}

func TestExecuteParallel(t *testing.T) {
	bus := events.NewBus()
	defer bus.Close()

	pool := NewPool(bus, 10*time.Second)

	// Create two mock agents
	agent1 := core.NewAgent("agent-1", "mock", "Agent 1", "model-1", "mock")
	adapter1 := mock.NewMockAdapter()
	adapter1.Response = "Response from Agent 1"
	adapter1.Delay = 50 * time.Millisecond // Add delay for Windows timer resolution

	agent2 := core.NewAgent("agent-2", "mock", "Agent 2", "model-2", "mock")
	adapter2 := mock.NewMockAdapter()
	adapter2.Response = "Response from Agent 2"
	adapter2.Delay = 50 * time.Millisecond

	pool.AddAgent(agent1, adapter1)
	pool.AddAgent(agent2, adapter2)

	// Execute
	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}
	responses := pool.ExecuteParallel(ctx, messages, nil)

	if len(responses) != 2 {
		t.Errorf("expected 2 responses, got %d", len(responses))
	}

	// Verify both agents responded
	responseMap := make(map[string]string)
	for _, resp := range responses {
		if resp.Error != nil {
			t.Errorf("agent %s returned error: %v", resp.AgentID, resp.Error)
			continue
		}
		if resp.Message != nil {
			responseMap[resp.AgentID] = resp.Message.Content
		}
	}

	if responseMap["agent-1"] != "Response from Agent 1" {
		t.Errorf("unexpected response from agent-1: %q", responseMap["agent-1"])
	}
	if responseMap["agent-2"] != "Response from Agent 2" {
		t.Errorf("unexpected response from agent-2: %q", responseMap["agent-2"])
	}
}

func TestParallelTiming(t *testing.T) {
	pool := NewPool(nil, 10*time.Second)

	// Fast agent: 100ms
	agent1 := core.NewAgent("fast", "mock", "Fast Agent", "model", "mock")
	adapter1 := mock.NewMockAdapter()
	adapter1.Delay = 100 * time.Millisecond

	// Slow agent: 300ms
	agent2 := core.NewAgent("slow", "mock", "Slow Agent", "model", "mock")
	adapter2 := mock.NewMockAdapter()
	adapter2.Delay = 300 * time.Millisecond

	pool.AddAgent(agent1, adapter1)
	pool.AddAgent(agent2, adapter2)

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}

	start := time.Now()
	responses := pool.ExecuteParallel(ctx, messages, nil)
	elapsed := time.Since(start)

	if len(responses) != 2 {
		t.Errorf("expected 2 responses, got %d", len(responses))
	}

	// Should complete in ~300ms (parallel), not 400ms (sequential)
	// Allow some margin for test overhead
	if elapsed > 500*time.Millisecond {
		t.Errorf("parallel execution took too long: %v (should be ~300ms)", elapsed)
	}
	if elapsed < 300*time.Millisecond {
		t.Errorf("execution was too fast: %v (expected at least 300ms)", elapsed)
	}
}

func TestEventEmission(t *testing.T) {
	bus := events.NewBus()
	defer bus.Close()

	pool := NewPool(bus, 10*time.Second)

	agent := core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock")
	adapter := mock.NewMockAdapter()
	adapter.Delay = 30 * time.Millisecond // Add delay for Windows timer resolution
	pool.AddAgent(agent, adapter)

	var typingEventReceived atomic.Bool
	var doneEventReceived atomic.Bool

	bus.Subscribe(core.EventAgentTyping, func(event core.Event) {
		typingEventReceived.Store(true)
	})

	bus.Subscribe(core.EventAgentDone, func(event core.Event) {
		doneEventReceived.Store(true)
	})

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}
	_ = pool.ExecuteParallel(ctx, messages, nil)

	// Wait for async events
	time.Sleep(100 * time.Millisecond)

	if !typingEventReceived.Load() {
		t.Error("EventAgentTyping should have been emitted")
	}
	if !doneEventReceived.Load() {
		t.Error("EventAgentDone should have been emitted")
	}
}

func TestAgentError(t *testing.T) {
	bus := events.NewBus()
	defer bus.Close()

	pool := NewPool(bus, 10*time.Second)

	// Successful agent
	agent1 := core.NewAgent("success", "mock", "Success Agent", "model", "mock")
	adapter1 := mock.NewMockAdapter()
	adapter1.Response = "I worked!"
	adapter1.Delay = 20 * time.Millisecond

	// Failing agent
	agent2 := core.NewAgent("fail", "mock", "Fail Agent", "model", "mock")
	adapter2 := mock.NewMockAdapter()
	adapter2.Error = errors.New("simulated failure")

	pool.AddAgent(agent1, adapter1)
	pool.AddAgent(agent2, adapter2)

	var errorEventReceived atomic.Bool
	bus.Subscribe(core.EventAgentError, func(event core.Event) {
		errorEventReceived.Store(true)
	})

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}
	responses := pool.ExecuteParallel(ctx, messages, nil)

	if len(responses) != 2 {
		t.Errorf("expected 2 responses, got %d", len(responses))
	}

	// Check that one succeeded and one failed
	var successCount, failCount int
	for _, resp := range responses {
		if resp.Error != nil {
			failCount++
			if resp.Error.Error() != "simulated failure" {
				t.Errorf("unexpected error: %v", resp.Error)
			}
		} else {
			successCount++
			if resp.Message.Content != "I worked!" {
				t.Errorf("unexpected content: %q", resp.Message.Content)
			}
		}
	}

	if successCount != 1 {
		t.Errorf("expected 1 success, got %d", successCount)
	}
	if failCount != 1 {
		t.Errorf("expected 1 failure, got %d", failCount)
	}

	// Wait for async events
	time.Sleep(100 * time.Millisecond)

	if !errorEventReceived.Load() {
		t.Error("EventAgentError should have been emitted")
	}
}

func TestAllAgentsFail(t *testing.T) {
	pool := NewPool(nil, 10*time.Second)

	agent1 := core.NewAgent("fail-1", "mock", "Fail Agent 1", "model", "mock")
	adapter1 := mock.NewMockAdapter()
	adapter1.Error = errors.New("failure 1")

	agent2 := core.NewAgent("fail-2", "mock", "Fail Agent 2", "model", "mock")
	adapter2 := mock.NewMockAdapter()
	adapter2.Error = errors.New("failure 2")

	pool.AddAgent(agent1, adapter1)
	pool.AddAgent(agent2, adapter2)

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}
	responses := pool.ExecuteParallel(ctx, messages, nil)

	if len(responses) != 2 {
		t.Errorf("expected 2 responses, got %d", len(responses))
	}

	for _, resp := range responses {
		if resp.Error == nil {
			t.Errorf("expected error for agent %s", resp.AgentID)
		}
	}
}

func TestTimeout(t *testing.T) {
	pool := NewPool(nil, 100*time.Millisecond) // Very short timeout

	agent := core.NewAgent("slow", "mock", "Slow Agent", "model", "mock")
	adapter := mock.NewMockAdapter()
	adapter.Delay = 500 * time.Millisecond // Longer than timeout

	pool.AddAgent(agent, adapter)

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}

	start := time.Now()
	responses := pool.ExecuteParallel(ctx, messages, nil)
	elapsed := time.Since(start)

	if len(responses) != 1 {
		t.Errorf("expected 1 response, got %d", len(responses))
	}

	resp := responses[0]
	if resp.Error == nil {
		t.Error("expected timeout error")
	}
	if !errors.Is(resp.Error, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded error, got %v", resp.Error)
	}

	// Should complete around the timeout, not the full delay
	if elapsed > 300*time.Millisecond {
		t.Errorf("should have timed out sooner: %v", elapsed)
	}
}

func TestContextCancellation(t *testing.T) {
	pool := NewPool(nil, 10*time.Second)

	agent := core.NewAgent("slow", "mock", "Slow Agent", "model", "mock")
	adapter := mock.NewMockAdapter()
	adapter.Delay = 500 * time.Millisecond

	pool.AddAgent(agent, adapter)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after 50ms
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	messages := []core.Message{core.NewUserMessage("Hello")}

	start := time.Now()
	responses := pool.ExecuteParallel(ctx, messages, nil)
	elapsed := time.Since(start)

	if len(responses) != 1 {
		t.Errorf("expected 1 response, got %d", len(responses))
	}

	resp := responses[0]
	if resp.Error == nil {
		t.Error("expected cancellation error")
	}

	// Should complete quickly due to cancellation
	if elapsed > 200*time.Millisecond {
		t.Errorf("should have cancelled sooner: %v", elapsed)
	}
}

func TestGetStatus(t *testing.T) {
	pool := NewPool(nil, 10*time.Second)

	agent1 := core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock")
	adapter1 := mock.NewMockAdapter()
	adapter1.Delay = 50 * time.Millisecond

	pool.AddAgent(agent1, adapter1)

	// Initial status should be idle
	status := pool.GetStatus()
	if status["agent-1"] != core.AgentStatusIdle {
		t.Errorf("expected initial status Idle, got %q", status["agent-1"])
	}
}

func TestGetAgentState(t *testing.T) {
	pool := NewPool(nil, 10*time.Second)

	agent := core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock")
	adapter := mock.NewMockAdapter()
	adapter.Delay = 50 * time.Millisecond

	pool.AddAgent(agent, adapter)

	state, found := pool.GetAgentState("agent-1")
	if !found {
		t.Fatal("agent state should be found")
	}
	if state.Status != core.AgentStatusIdle {
		t.Errorf("expected status Idle, got %q", state.Status)
	}

	// Unknown agent
	_, found = pool.GetAgentState("unknown")
	if found {
		t.Error("should not find unknown agent")
	}
}

func TestEmptyPool(t *testing.T) {
	pool := NewPool(nil, 10*time.Second)

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}
	responses := pool.ExecuteParallel(ctx, messages, nil)

	if len(responses) != 0 {
		t.Errorf("expected 0 responses from empty pool, got %d", len(responses))
	}
}

func TestNilEventBus(t *testing.T) {
	// Pool should work without event bus
	pool := NewPool(nil, 10*time.Second)

	agent := core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock")
	adapter := mock.NewMockAdapter()
	adapter.Response = "Hello"
	adapter.Delay = 20 * time.Millisecond

	pool.AddAgent(agent, adapter)

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hi")}
	responses := pool.ExecuteParallel(ctx, messages, nil)

	if len(responses) != 1 {
		t.Errorf("expected 1 response, got %d", len(responses))
	}
	if responses[0].Error != nil {
		t.Errorf("unexpected error: %v", responses[0].Error)
	}
}

func TestConcurrentExecutions(t *testing.T) {
	// Test concurrent creation of pools and execution - each pool has its own state
	// This tests thread-safety of pool construction and execution
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Create a new pool for each goroutine to avoid shared AgentState race
			pool := NewPool(nil, 10*time.Second)
			agent := core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock")
			adapter := mock.NewMockAdapter()
			adapter.Response = "Response"
			adapter.Delay = 20 * time.Millisecond
			pool.AddAgent(agent, adapter)

			ctx := context.Background()
			messages := []core.Message{core.NewUserMessage("Hello")}
			responses := pool.ExecuteParallel(ctx, messages, nil)
			if len(responses) != 1 {
				t.Errorf("expected 1 response, got %d", len(responses))
			}
		}()
	}

	wg.Wait()
	// Test passes if no race conditions detected
}

func TestAgentStateAfterExecution(t *testing.T) {
	pool := NewPool(nil, 10*time.Second)

	agent := core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock")
	adapter := mock.NewMockAdapter()
	adapter.Response = "Response"
	adapter.Delay = 20 * time.Millisecond
	adapter.Metrics = &core.Metrics{
		TotalTokens: 100,
		Cost:        0.01,
	}

	pool.AddAgent(agent, adapter)

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}
	_ = pool.ExecuteParallel(ctx, messages, nil)

	state, _ := pool.GetAgentState("agent-1")
	if state.MessageCount != 1 {
		t.Errorf("expected MessageCount=1, got %d", state.MessageCount)
	}
	if state.TotalTokens != 100 {
		t.Errorf("expected TotalTokens=100, got %d", state.TotalTokens)
	}
	if state.TotalCost != 0.01 {
		t.Errorf("expected TotalCost=0.01, got %f", state.TotalCost)
	}
	if state.Status != core.AgentStatusIdle {
		t.Errorf("expected status Idle after execution, got %q", state.Status)
	}
}

func TestMetricsPopulatedInResponse(t *testing.T) {
	pool := NewPool(nil, 10*time.Second)

	agent := core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock")
	adapter := mock.NewMockAdapter()
	adapter.Response = "Response"
	adapter.Delay = 50 * time.Millisecond

	pool.AddAgent(agent, adapter)

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}
	responses := pool.ExecuteParallel(ctx, messages, nil)

	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}

	resp := responses[0]
	if resp.Message == nil {
		t.Fatal("expected non-nil message")
	}
	if resp.Message.Metrics == nil {
		t.Fatal("expected non-nil metrics")
	}
	// Duration should be populated
	if resp.Message.Metrics.Duration < 50*time.Millisecond {
		t.Errorf("expected Duration >= 50ms, got %v", resp.Message.Metrics.Duration)
	}
}

// Circuit Breaker Integration Tests

func TestCircuitBreakerInitialization(t *testing.T) {
	pool := NewPool(nil, 10*time.Second)

	agent := core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock")
	adapter := mock.NewMockAdapter()

	pool.AddAgent(agent, adapter)

	// Circuit breaker should be initialized for the agent
	cbInfo, found := pool.GetCircuitBreakerStatus("agent-1")
	if !found {
		t.Fatal("circuit breaker should be found for agent-1")
	}

	if cbInfo.State != adapters.CircuitClosed {
		t.Errorf("expected initial state Closed, got %s", cbInfo.State)
	}
	if cbInfo.FailureCount != 0 {
		t.Errorf("expected initial failure count 0, got %d", cbInfo.FailureCount)
	}
}

func TestCircuitBreakerCustomConfig(t *testing.T) {
	customConfig := adapters.CircuitBreakerConfig{
		FailureThreshold: 3,
		CooldownPeriod:   10 * time.Second,
		SuccessThreshold: 2,
	}

	pool := NewPoolWithCircuitBreaker(nil, 10*time.Second, customConfig)

	agent := core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock")
	adapter := mock.NewMockAdapter()
	adapter.Error = errors.New("simulated failure")

	pool.AddAgent(agent, adapter)

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}

	// Execute 3 times to trip circuit (custom threshold)
	for i := 0; i < 3; i++ {
		_ = pool.ExecuteParallel(ctx, messages, nil)
	}

	cbInfo, _ := pool.GetCircuitBreakerStatus("agent-1")
	if cbInfo.State != adapters.CircuitOpen {
		t.Errorf("expected circuit to be open after 3 failures, got %s", cbInfo.State)
	}
}

func TestCircuitBreakerOpensAfterConsecutiveFailures(t *testing.T) {
	pool := NewPool(nil, 10*time.Second)

	agent := core.NewAgent("fail-agent", "mock", "Fail Agent", "model", "mock")
	adapter := mock.NewMockAdapter()
	adapter.Error = errors.New("simulated failure")

	pool.AddAgent(agent, adapter)

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}

	// Default threshold is 5, so we need 5 failures
	for i := 0; i < 5; i++ {
		responses := pool.ExecuteParallel(ctx, messages, nil)
		if len(responses) != 1 {
			t.Fatalf("expected 1 response, got %d", len(responses))
		}
		if responses[0].Error == nil {
			t.Fatal("expected error on each attempt")
		}
	}

	// Circuit should now be open
	cbInfo, _ := pool.GetCircuitBreakerStatus("fail-agent")
	if cbInfo.State != adapters.CircuitOpen {
		t.Errorf("expected circuit to be open after 5 failures, got %s", cbInfo.State)
	}
	if cbInfo.FailureCount != 5 {
		t.Errorf("expected failure count 5, got %d", cbInfo.FailureCount)
	}
}

func TestCircuitBreakerSkipsOpenCircuit(t *testing.T) {
	bus := events.NewBus()
	defer bus.Close()

	pool := NewPool(bus, 10*time.Second)

	agent := core.NewAgent("fail-agent", "mock", "Fail Agent", "model", "mock")
	adapter := mock.NewMockAdapter()
	adapter.Error = errors.New("simulated failure")

	pool.AddAgent(agent, adapter)

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}

	// Trip the circuit breaker (5 failures)
	for i := 0; i < 5; i++ {
		_ = pool.ExecuteParallel(ctx, messages, nil)
	}

	// Circuit is now open - next call should be skipped with circuit open error
	responses := pool.ExecuteParallel(ctx, messages, nil)
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}

	resp := responses[0]
	if resp.Error == nil {
		t.Fatal("expected error for open circuit")
	}

	errStr := resp.Error.Error()
	if !contains(errStr, "circuit breaker open") {
		t.Errorf("expected 'circuit breaker open' error, got: %s", errStr)
	}
}

func TestCircuitBreakerEmitsErrorEventOnOpen(t *testing.T) {
	bus := events.NewBus()
	defer bus.Close()

	pool := NewPool(bus, 10*time.Second)

	agent := core.NewAgent("fail-agent", "mock", "Fail Agent", "model", "mock")
	adapter := mock.NewMockAdapter()
	adapter.Error = errors.New("simulated failure")

	pool.AddAgent(agent, adapter)

	var circuitOpenEventReceived atomic.Bool
	bus.Subscribe(core.EventAgentError, func(event core.Event) {
		data, ok := event.Data.(core.AgentErrorData)
		if ok && contains(data.Error, "circuit breaker opened") {
			circuitOpenEventReceived.Store(true)
		}
	})

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}

	// Trip the circuit breaker (5 failures)
	for i := 0; i < 5; i++ {
		_ = pool.ExecuteParallel(ctx, messages, nil)
	}

	// Wait for async events
	time.Sleep(100 * time.Millisecond)

	if !circuitOpenEventReceived.Load() {
		t.Error("expected EventAgentError for circuit breaker opening")
	}
}

func TestCircuitBreakerSuccessResetsFailureCount(t *testing.T) {
	pool := NewPool(nil, 10*time.Second)

	agent := core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock")
	adapter := mock.NewMockAdapter()
	adapter.Delay = 20 * time.Millisecond

	pool.AddAgent(agent, adapter)

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}

	// Simulate 3 failures
	adapter.Error = errors.New("temporary failure")
	for i := 0; i < 3; i++ {
		_ = pool.ExecuteParallel(ctx, messages, nil)
	}

	cbInfo, _ := pool.GetCircuitBreakerStatus("agent-1")
	if cbInfo.FailureCount != 3 {
		t.Errorf("expected failure count 3, got %d", cbInfo.FailureCount)
	}

	// Now a success should reset the count
	adapter.Error = nil
	adapter.Response = "Success!"
	_ = pool.ExecuteParallel(ctx, messages, nil)

	cbInfo, _ = pool.GetCircuitBreakerStatus("agent-1")
	if cbInfo.FailureCount != 0 {
		t.Errorf("expected failure count 0 after success, got %d", cbInfo.FailureCount)
	}
}

func TestGetAllCircuitBreakerStatus(t *testing.T) {
	pool := NewPool(nil, 10*time.Second)

	agent1 := core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock")
	agent2 := core.NewAgent("agent-2", "mock", "Agent 2", "model", "mock")

	pool.AddAgent(agent1, mock.NewMockAdapter())
	pool.AddAgent(agent2, mock.NewMockAdapter())

	statuses := pool.GetAllCircuitBreakerStatus()
	if len(statuses) != 2 {
		t.Fatalf("expected 2 circuit breaker statuses, got %d", len(statuses))
	}

	// Check that both agents are represented
	agentIDs := make(map[string]bool)
	for _, s := range statuses {
		agentIDs[s.AgentID] = true
	}

	if !agentIDs["agent-1"] || !agentIDs["agent-2"] {
		t.Error("expected both agents in status list")
	}
}

func TestResetCircuitBreaker(t *testing.T) {
	pool := NewPool(nil, 10*time.Second)

	agent := core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock")
	adapter := mock.NewMockAdapter()
	adapter.Error = errors.New("failure")

	pool.AddAgent(agent, adapter)

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}

	// Trip the circuit
	for i := 0; i < 5; i++ {
		_ = pool.ExecuteParallel(ctx, messages, nil)
	}

	cbInfo, _ := pool.GetCircuitBreakerStatus("agent-1")
	if cbInfo.State != adapters.CircuitOpen {
		t.Fatalf("expected circuit to be open, got %s", cbInfo.State)
	}

	// Reset the circuit
	reset := pool.ResetCircuitBreaker("agent-1")
	if !reset {
		t.Error("expected reset to return true")
	}

	cbInfo, _ = pool.GetCircuitBreakerStatus("agent-1")
	if cbInfo.State != adapters.CircuitClosed {
		t.Errorf("expected circuit to be closed after reset, got %s", cbInfo.State)
	}
	if cbInfo.FailureCount != 0 {
		t.Errorf("expected failure count 0 after reset, got %d", cbInfo.FailureCount)
	}
}

func TestResetCircuitBreakerNotFound(t *testing.T) {
	pool := NewPool(nil, 10*time.Second)

	reset := pool.ResetCircuitBreaker("nonexistent")
	if reset {
		t.Error("expected reset to return false for nonexistent agent")
	}
}

func TestResetAllCircuitBreakers(t *testing.T) {
	pool := NewPool(nil, 10*time.Second)

	agent1 := core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock")
	agent2 := core.NewAgent("agent-2", "mock", "Agent 2", "model", "mock")

	adapter1 := mock.NewMockAdapter()
	adapter1.Error = errors.New("failure")
	adapter2 := mock.NewMockAdapter()
	adapter2.Error = errors.New("failure")

	pool.AddAgent(agent1, adapter1)
	pool.AddAgent(agent2, adapter2)

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}

	// Trip both circuits
	for i := 0; i < 5; i++ {
		_ = pool.ExecuteParallel(ctx, messages, nil)
	}

	// Verify both are open
	for _, agentID := range []string{"agent-1", "agent-2"} {
		cbInfo, _ := pool.GetCircuitBreakerStatus(agentID)
		if cbInfo.State != adapters.CircuitOpen {
			t.Errorf("expected circuit %s to be open, got %s", agentID, cbInfo.State)
		}
	}

	// Reset all
	pool.ResetAllCircuitBreakers()

	// Verify both are closed
	for _, agentID := range []string{"agent-1", "agent-2"} {
		cbInfo, _ := pool.GetCircuitBreakerStatus(agentID)
		if cbInfo.State != adapters.CircuitClosed {
			t.Errorf("expected circuit %s to be closed after reset, got %s", agentID, cbInfo.State)
		}
	}
}

func TestGetAvailableAgentCount(t *testing.T) {
	pool := NewPool(nil, 10*time.Second)

	agent1 := core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock")
	agent2 := core.NewAgent("agent-2", "mock", "Agent 2", "model", "mock")

	adapter1 := mock.NewMockAdapter()
	adapter1.Response = "OK"
	adapter1.Delay = 20 * time.Millisecond

	adapter2 := mock.NewMockAdapter()
	adapter2.Error = errors.New("failure")

	pool.AddAgent(agent1, adapter1)
	pool.AddAgent(agent2, adapter2)

	// Initially both are available
	if count := pool.GetAvailableAgentCount(); count != 2 {
		t.Errorf("expected 2 available agents, got %d", count)
	}

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}

	// Trip agent-2's circuit
	for i := 0; i < 5; i++ {
		_ = pool.ExecuteParallel(ctx, messages, nil)
	}

	// Now only agent-1 should be available
	if count := pool.GetAvailableAgentCount(); count != 1 {
		t.Errorf("expected 1 available agent after tripping circuit, got %d", count)
	}
}

func TestMixedAgentsContinueWithOpenCircuit(t *testing.T) {
	pool := NewPool(nil, 10*time.Second)

	// Healthy agent
	agent1 := core.NewAgent("healthy", "mock", "Healthy Agent", "model", "mock")
	adapter1 := mock.NewMockAdapter()
	adapter1.Response = "I'm healthy!"
	adapter1.Delay = 20 * time.Millisecond

	// Failing agent
	agent2 := core.NewAgent("failing", "mock", "Failing Agent", "model", "mock")
	adapter2 := mock.NewMockAdapter()
	adapter2.Error = errors.New("always fails")

	pool.AddAgent(agent1, adapter1)
	pool.AddAgent(agent2, adapter2)

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}

	// Trip the failing agent's circuit (5 failures)
	for i := 0; i < 5; i++ {
		_ = pool.ExecuteParallel(ctx, messages, nil)
	}

	// Now execute again - healthy agent should still work
	responses := pool.ExecuteParallel(ctx, messages, nil)
	if len(responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(responses))
	}

	// Check responses
	var healthyResponse, failingResponse *Response
	for i := range responses {
		if responses[i].AgentID == "healthy" {
			healthyResponse = &responses[i]
		} else {
			failingResponse = &responses[i]
		}
	}

	if healthyResponse == nil || healthyResponse.Error != nil {
		t.Error("expected healthy agent to succeed")
	}
	if healthyResponse.Message == nil || healthyResponse.Message.Content != "I'm healthy!" {
		t.Error("expected healthy agent response content")
	}

	if failingResponse == nil || failingResponse.Error == nil {
		t.Error("expected failing agent to have circuit open error")
	}
	if !contains(failingResponse.Error.Error(), "circuit breaker open") {
		t.Errorf("expected circuit breaker open error, got: %s", failingResponse.Error)
	}
}

func TestCircuitBreakerHalfOpenRecovery(t *testing.T) {
	// Use a short cooldown for testing
	config := adapters.CircuitBreakerConfig{
		FailureThreshold: 2,
		CooldownPeriod:   100 * time.Millisecond, // Very short for testing
		SuccessThreshold: 1,
	}

	pool := NewPoolWithCircuitBreaker(nil, 10*time.Second, config)

	agent := core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock")
	adapter := mock.NewMockAdapter()
	adapter.Error = errors.New("temporary failure")

	pool.AddAgent(agent, adapter)

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}

	// Trip the circuit (2 failures with custom threshold)
	for i := 0; i < 2; i++ {
		_ = pool.ExecuteParallel(ctx, messages, nil)
	}

	cbInfo, _ := pool.GetCircuitBreakerStatus("agent-1")
	if cbInfo.State != adapters.CircuitOpen {
		t.Fatalf("expected circuit to be open, got %s", cbInfo.State)
	}

	// Wait for cooldown
	time.Sleep(150 * time.Millisecond)

	// Fix the adapter
	adapter.Error = nil
	adapter.Response = "Recovered!"
	adapter.Delay = 20 * time.Millisecond

	// Execute - should transition to half-open and then close on success
	responses := pool.ExecuteParallel(ctx, messages, nil)
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}

	resp := responses[0]
	if resp.Error != nil {
		t.Fatalf("expected success after recovery, got error: %v", resp.Error)
	}

	// Circuit should now be closed
	cbInfo, _ = pool.GetCircuitBreakerStatus("agent-1")
	if cbInfo.State != adapters.CircuitClosed {
		t.Errorf("expected circuit to be closed after successful probe, got %s", cbInfo.State)
	}
}

// Helper function for tests
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
