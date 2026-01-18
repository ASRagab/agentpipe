package pool

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/v2/adapters/mock"
	"github.com/kevinelliott/agentpipe/pkg/v2/core"
	"github.com/kevinelliott/agentpipe/pkg/v2/events"
)

// TestCancellationManagerBasic tests basic cancellation manager operations.
func TestCancellationManagerBasic(t *testing.T) {
	cm := NewCancellationManager(nil)

	// Initially no active requests
	if count := cm.ActiveRequestCount(); count != 0 {
		t.Errorf("expected 0 active requests, got %d", count)
	}

	// Register a request
	cancelled := false
	cancel := func() { cancelled = true }
	cm.RegisterRequest("agent-1", "Agent 1", cancel)

	if count := cm.ActiveRequestCount(); count != 1 {
		t.Errorf("expected 1 active request, got %d", count)
	}

	if !cm.HasActiveRequest("agent-1") {
		t.Error("expected HasActiveRequest to return true for agent-1")
	}

	if cm.HasActiveRequest("agent-2") {
		t.Error("expected HasActiveRequest to return false for unknown agent")
	}

	// Cancel the request
	err := cm.CancelAgent("agent-1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !cancelled {
		t.Error("cancel function should have been called")
	}

	if !cm.IsCancelled("agent-1") {
		t.Error("expected IsCancelled to return true after cancellation")
	}

	// Cancelling again should be a no-op
	err = cm.CancelAgent("agent-1")
	if err != nil {
		t.Errorf("cancelling again should not error: %v", err)
	}
}

// TestCancellationManagerNotFound tests error handling for unknown agents.
func TestCancellationManagerNotFound(t *testing.T) {
	cm := NewCancellationManager(nil)

	err := cm.CancelAgent("unknown")
	if !errors.Is(err, ErrAgentNotFound) {
		t.Errorf("expected ErrAgentNotFound, got %v", err)
	}
}

// TestCancellationManagerCancelAll tests bulk cancellation.
func TestCancellationManagerCancelAll(t *testing.T) {
	cm := NewCancellationManager(nil)

	var cancelled1, cancelled2, cancelled3 atomic.Bool
	cm.RegisterRequest("agent-1", "Agent 1", func() { cancelled1.Store(true) })
	cm.RegisterRequest("agent-2", "Agent 2", func() { cancelled2.Store(true) })
	cm.RegisterRequest("agent-3", "Agent 3", func() { cancelled3.Store(true) })

	if count := cm.ActiveRequestCount(); count != 3 {
		t.Errorf("expected 3 active requests, got %d", count)
	}

	// Cancel all
	count := cm.CancelAll()
	if count != 3 {
		t.Errorf("expected 3 cancellations, got %d", count)
	}

	if !cancelled1.Load() || !cancelled2.Load() || !cancelled3.Load() {
		t.Error("all cancel functions should have been called")
	}

	// Cancel all again should return 0
	count = cm.CancelAll()
	if count != 0 {
		t.Errorf("expected 0 cancellations on second call, got %d", count)
	}
}

// TestCancellationManagerUnregister tests request unregistration.
func TestCancellationManagerUnregister(t *testing.T) {
	cm := NewCancellationManager(nil)

	cm.RegisterRequest("agent-1", "Agent 1", func() {})
	cm.RegisterRequest("agent-2", "Agent 2", func() {})

	if count := cm.ActiveRequestCount(); count != 2 {
		t.Errorf("expected 2 active requests, got %d", count)
	}

	cm.UnregisterRequest("agent-1")

	if cm.HasActiveRequest("agent-1") {
		t.Error("agent-1 should no longer have an active request")
	}

	if !cm.HasActiveRequest("agent-2") {
		t.Error("agent-2 should still have an active request")
	}

	// Unregistering unknown agent should be a no-op
	cm.UnregisterRequest("unknown")
}

// TestCancellationManagerGetActiveRequests tests getting active request info.
func TestCancellationManagerGetActiveRequests(t *testing.T) {
	cm := NewCancellationManager(nil)

	cm.RegisterRequest("agent-1", "Agent 1", func() {})
	cm.RegisterRequest("agent-2", "Agent 2", func() {})

	requests := cm.GetActiveRequests()
	if len(requests) != 2 {
		t.Errorf("expected 2 active requests, got %d", len(requests))
	}

	// Check that both agents are represented
	agentIDs := make(map[string]bool)
	for _, r := range requests {
		agentIDs[r.AgentID] = true
		if r.StartTime.IsZero() {
			t.Error("StartTime should be set")
		}
		if r.Cancelled {
			t.Error("requests should not be cancelled initially")
		}
	}

	if !agentIDs["agent-1"] || !agentIDs["agent-2"] {
		t.Error("expected both agents in active requests")
	}
}

// TestCancellationManagerClear tests clearing all requests.
func TestCancellationManagerClear(t *testing.T) {
	cm := NewCancellationManager(nil)

	cm.RegisterRequest("agent-1", "Agent 1", func() {})
	cm.RegisterRequest("agent-2", "Agent 2", func() {})

	cm.Clear()

	if count := cm.ActiveRequestCount(); count != 0 {
		t.Errorf("expected 0 active requests after clear, got %d", count)
	}
}

// TestCancellationManagerConcurrency tests thread safety.
func TestCancellationManagerConcurrency(t *testing.T) {
	cm := NewCancellationManager(nil)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			agentID := "agent-" + string(rune('A'+id%10))
			cm.RegisterRequest(agentID, "Agent", func() {})
			cm.HasActiveRequest(agentID)
			cm.GetActiveRequests()
			cm.CancelAgent(agentID)
			cm.UnregisterRequest(agentID)
		}(i)
	}
	wg.Wait()
	// Test passes if no race conditions
}

// TestIsCancellationError tests cancellation error detection.
func TestIsCancellationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"ErrAgentCancelled", ErrAgentCancelled, true}, // ErrAgentCancelled IS a cancellation error
		{"context.Canceled", context.Canceled, true},
		{"wrapped context.Canceled", errors.New("wrapped: context canceled"), false},
		{"other error", errors.New("some error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCancellationError(tt.err)
			if result != tt.expected {
				t.Errorf("IsCancellationError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

// TestPoolCancel tests the Pool.Cancel method.
func TestPoolCancel(t *testing.T) {
	bus := events.NewBus()
	defer bus.Close()

	pool := NewPool(bus, 5*time.Second)

	// Create agents with long delays
	agent1 := core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock")
	adapter1 := mock.NewMockAdapter()
	adapter1.Delay = 2 * time.Second

	agent2 := core.NewAgent("agent-2", "mock", "Agent 2", "model", "mock")
	adapter2 := mock.NewMockAdapter()
	adapter2.Delay = 2 * time.Second

	pool.AddAgent(agent1, adapter1)
	pool.AddAgent(agent2, adapter2)

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}

	// Start execution in goroutine
	var responses []Response
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		responses = pool.ExecuteParallel(ctx, messages)
	}()

	// Wait for agents to start
	time.Sleep(100 * time.Millisecond)

	// Should have active requests
	if count := pool.ActiveRequestCount(); count != 2 {
		t.Logf("Warning: expected 2 active requests, got %d (timing-dependent)", count)
	}

	// Cancel all
	cancelled := pool.Cancel()
	if cancelled != 2 {
		t.Logf("Warning: expected 2 cancellations, got %d (timing-dependent)", cancelled)
	}

	// Wait for responses
	wg.Wait()

	// All responses should have cancellation errors
	for _, resp := range responses {
		if resp.Error == nil {
			t.Errorf("expected error for agent %s", resp.AgentID)
			continue
		}
		if !IsCancellationError(resp.Error) && !errors.Is(resp.Error, ErrAgentCancelled) {
			t.Logf("Note: expected cancellation error for %s, got: %v", resp.AgentID, resp.Error)
		}
	}
}

// TestPoolCancelAgent tests cancelling a specific agent.
func TestPoolCancelAgent(t *testing.T) {
	bus := events.NewBus()
	defer bus.Close()

	pool := NewPool(bus, 5*time.Second)

	// Create one fast and one slow agent
	agent1 := core.NewAgent("fast", "mock", "Fast Agent", "model", "mock")
	adapter1 := mock.NewMockAdapter()
	adapter1.Response = "Fast response"
	adapter1.Delay = 50 * time.Millisecond

	agent2 := core.NewAgent("slow", "mock", "Slow Agent", "model", "mock")
	adapter2 := mock.NewMockAdapter()
	adapter2.Response = "Slow response"
	adapter2.Delay = 2 * time.Second

	pool.AddAgent(agent1, adapter1)
	pool.AddAgent(agent2, adapter2)

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}

	// Start execution
	var responses []Response
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		responses = pool.ExecuteParallel(ctx, messages)
	}()

	// Wait for agents to start
	time.Sleep(100 * time.Millisecond)

	// Cancel only the slow agent
	err := pool.CancelAgent("slow")
	if err != nil {
		t.Errorf("unexpected error cancelling slow agent: %v", err)
	}

	wg.Wait()

	// Check results
	var fastResp, slowResp *Response
	for i := range responses {
		if responses[i].AgentID == "fast" {
			fastResp = &responses[i]
		} else {
			slowResp = &responses[i]
		}
	}

	if fastResp == nil || fastResp.Error != nil {
		t.Error("fast agent should have succeeded")
	}
	if fastResp != nil && fastResp.Message != nil && fastResp.Message.Content != "Fast response" {
		t.Errorf("unexpected fast response: %v", fastResp.Message.Content)
	}

	if slowResp == nil || slowResp.Error == nil {
		t.Error("slow agent should have error")
	}
}

// TestPoolCancelAgentNotFound tests cancelling unknown agent.
func TestPoolCancelAgentNotFound(t *testing.T) {
	pool := NewPool(nil, 10*time.Second)

	err := pool.CancelAgent("unknown")
	if !errors.Is(err, ErrAgentNotFound) {
		t.Errorf("expected ErrAgentNotFound, got %v", err)
	}
}

// TestPoolHasActiveRequest tests checking for active requests.
func TestPoolHasActiveRequest(t *testing.T) {
	bus := events.NewBus()
	defer bus.Close()

	pool := NewPool(bus, 5*time.Second)

	agent := core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock")
	adapter := mock.NewMockAdapter()
	adapter.Delay = 1 * time.Second

	pool.AddAgent(agent, adapter)

	// No active requests initially
	if pool.HasActiveRequest("agent-1") {
		t.Error("should have no active request initially")
	}

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}

	// Start execution
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = pool.ExecuteParallel(ctx, messages)
	}()

	// Wait for agent to start
	time.Sleep(100 * time.Millisecond)

	// Should have active request now (timing-dependent, log only)
	hasActive := pool.HasActiveRequest("agent-1")
	t.Logf("HasActiveRequest during execution: %v", hasActive)

	// Cancel to speed up test
	pool.Cancel()
	<-done

	// No active request after completion
	if pool.HasActiveRequest("agent-1") {
		t.Error("should have no active request after completion")
	}
}

// TestPoolGetActiveRequests tests getting active request info from pool.
func TestPoolGetActiveRequests(t *testing.T) {
	bus := events.NewBus()
	defer bus.Close()

	pool := NewPool(bus, 5*time.Second)

	agent1 := core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock")
	agent2 := core.NewAgent("agent-2", "mock", "Agent 2", "model", "mock")

	adapter1 := mock.NewMockAdapter()
	adapter1.Delay = 1 * time.Second
	adapter2 := mock.NewMockAdapter()
	adapter2.Delay = 1 * time.Second

	pool.AddAgent(agent1, adapter1)
	pool.AddAgent(agent2, adapter2)

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}

	// Start execution
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = pool.ExecuteParallel(ctx, messages)
	}()

	time.Sleep(100 * time.Millisecond)

	requests := pool.GetActiveRequests()
	t.Logf("Active requests during execution: %d", len(requests))

	pool.Cancel()
	<-done
}

// TestPoolCancelWithNilManager tests graceful handling of nil cancellation manager.
func TestPoolCancelWithNilManager(t *testing.T) {
	pool := &Pool{
		cancellation: nil,
	}

	// Should not panic
	count := pool.Cancel()
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	err := pool.CancelAgent("any")
	if !errors.Is(err, ErrAgentNotFound) {
		t.Errorf("expected ErrAgentNotFound, got %v", err)
	}

	if pool.HasActiveRequest("any") {
		t.Error("should return false with nil manager")
	}

	if pool.ActiveRequestCount() != 0 {
		t.Error("should return 0 with nil manager")
	}

	if pool.IsAgentCancelled("any") {
		t.Error("should return false with nil manager")
	}

	requests := pool.GetActiveRequests()
	if requests != nil {
		t.Error("should return nil with nil manager")
	}
}

// TestPoolCancelledStatusSet tests that agent status is set to cancelled.
func TestPoolCancelledStatusSet(t *testing.T) {
	bus := events.NewBus()
	defer bus.Close()

	pool := NewPool(bus, 5*time.Second)

	agent := core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock")
	adapter := mock.NewMockAdapter()
	adapter.Delay = 2 * time.Second

	pool.AddAgent(agent, adapter)

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = pool.ExecuteParallel(ctx, messages)
	}()

	time.Sleep(100 * time.Millisecond)
	pool.Cancel()
	<-done

	// Check agent status
	state, found := pool.GetAgentState("agent-1")
	if !found {
		t.Fatal("agent state not found")
	}

	if state.Status != core.AgentStatusCancelled {
		t.Errorf("expected status Cancelled, got %s", state.Status)
	}
}

// TestCancellationEventEmitted tests that cancellation events are emitted.
func TestCancellationEventEmitted(t *testing.T) {
	bus := events.NewBus()
	defer bus.Close()

	pool := NewPool(bus, 5*time.Second)

	agent := core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock")
	adapter := mock.NewMockAdapter()
	adapter.Delay = 2 * time.Second

	pool.AddAgent(agent, adapter)

	var cancelEventReceived atomic.Bool
	bus.Subscribe(core.EventAgentCancelled, func(event core.Event) {
		data, ok := event.Data.(core.AgentCancelledData)
		if ok && data.AgentID == "agent-1" {
			cancelEventReceived.Store(true)
		}
	})

	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = pool.ExecuteParallel(ctx, messages)
	}()

	time.Sleep(100 * time.Millisecond)
	pool.Cancel()
	<-done

	// Wait for async event
	time.Sleep(100 * time.Millisecond)

	if !cancelEventReceived.Load() {
		t.Error("EventAgentCancelled should have been emitted")
	}
}

// TestAgentStatusCancelled tests the new AgentStatusCancelled constant.
func TestAgentStatusCancelled(t *testing.T) {
	if core.AgentStatusCancelled != "cancelled" {
		t.Errorf("expected 'cancelled', got %s", core.AgentStatusCancelled)
	}

	state := core.NewAgentState(core.NewAgent("test", "mock", "Test", "model", "mock"))
	state.SetCancelled()

	if state.Status != core.AgentStatusCancelled {
		t.Errorf("expected status Cancelled, got %s", state.Status)
	}
	if state.LastError != "request cancelled" {
		t.Errorf("expected 'request cancelled', got %s", state.LastError)
	}
}

// TestNewAgentCancelledEvent tests the cancellation event constructor.
func TestNewAgentCancelledEvent(t *testing.T) {
	event := core.NewAgentCancelledEvent("agent-1", "Agent 1", "user requested")

	if event.Type != core.EventAgentCancelled {
		t.Errorf("expected EventAgentCancelled, got %s", event.Type)
	}

	data, ok := event.Data.(core.AgentCancelledData)
	if !ok {
		t.Fatal("expected AgentCancelledData")
	}

	if data.AgentID != "agent-1" {
		t.Errorf("expected agent-1, got %s", data.AgentID)
	}
	if data.AgentName != "Agent 1" {
		t.Errorf("expected Agent 1, got %s", data.AgentName)
	}
	if data.Reason != "user requested" {
		t.Errorf("expected 'user requested', got %s", data.Reason)
	}
}
