package manager

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/v2/adapters"
	"github.com/kevinelliott/agentpipe/pkg/v2/adapters/mock"
	"github.com/kevinelliott/agentpipe/pkg/v2/core"
	"github.com/kevinelliott/agentpipe/pkg/v2/events"
)

// Ensure mock adapter is registered
var _ = mock.NewMockAdapter()

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Timeout != 60*time.Second {
		t.Errorf("expected timeout 60s, got %v", config.Timeout)
	}
	if config.SaveDir != "" {
		t.Errorf("expected empty SaveDir, got %q", config.SaveDir)
	}
}

func TestNewConversationManager(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model-1", "mock"),
		core.NewAgent("agent-2", "mock", "Agent 2", "model-2", "mock"),
	}

	bus := events.NewBus()
	defer bus.Close()

	config := Config{
		Timeout: 30 * time.Second,
	}

	manager, err := NewConversationManager(config, agents, bus)
	if err != nil {
		t.Fatalf("NewConversationManager returned error: %v", err)
	}
	defer manager.Close()

	if manager == nil {
		t.Fatal("expected non-nil manager")
	}

	conv := manager.GetConversation()
	if conv == nil {
		t.Fatal("expected non-nil conversation")
	}
	if len(conv.Agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(conv.Agents))
	}
}

func TestNewConversationManager_NilEventBus(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock"),
	}

	// Should create its own event bus
	manager, err := NewConversationManager(DefaultConfig(), agents, nil)
	if err != nil {
		t.Fatalf("NewConversationManager returned error: %v", err)
	}
	defer manager.Close()

	if manager.eventBus == nil {
		t.Error("expected event bus to be created")
	}
}

func TestNewConversationManager_UnknownAdapter(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "unknown-type", "Agent 1", "model", "nonexistent-adapter"),
	}

	_, err := NewConversationManager(DefaultConfig(), agents, nil)
	if err == nil {
		t.Error("expected error for unknown adapter")
	}
}

func TestStart(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock"),
	}

	bus := events.NewBus()
	defer bus.Close()

	var conversationStarted atomic.Bool
	bus.Subscribe(core.EventConversationStarted, func(event core.Event) {
		conversationStarted.Store(true)
	})

	manager, err := NewConversationManager(DefaultConfig(), agents, bus)
	if err != nil {
		t.Fatalf("NewConversationManager returned error: %v", err)
	}
	defer manager.Close()

	manager.Start()

	time.Sleep(50 * time.Millisecond)

	if !conversationStarted.Load() {
		t.Error("expected ConversationStarted event to be emitted")
	}
}

func TestSendUserMessage(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock"),
	}

	bus := events.NewBus()
	defer bus.Close()

	manager, err := NewConversationManager(DefaultConfig(), agents, bus)
	if err != nil {
		t.Fatalf("NewConversationManager returned error: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()
	agentMessages, err := manager.SendUserMessage(ctx, "Hello, agents!")
	if err != nil {
		t.Fatalf("SendUserMessage returned error: %v", err)
	}

	// Should get one response (one agent)
	if len(agentMessages) != 1 {
		t.Errorf("expected 1 agent message, got %d", len(agentMessages))
	}

	// Verify messages are in history
	messages := manager.GetMessages()
	if len(messages) != 2 {
		t.Errorf("expected 2 messages (user + agent), got %d", len(messages))
	}

	// First should be user message
	if messages[0].Role != core.RoleUser {
		t.Errorf("expected first message to be user, got %q", messages[0].Role)
	}
	if messages[0].Content != "Hello, agents!" {
		t.Errorf("expected user content 'Hello, agents!', got %q", messages[0].Content)
	}

	// Second should be agent message
	if messages[1].Role != core.RoleAgent {
		t.Errorf("expected second message to be agent, got %q", messages[1].Role)
	}
}

func TestSendUserMessage_EventEmission(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock"),
	}

	bus := events.NewBus()
	defer bus.Close()

	var messageEvents []core.EventType
	var mu sync.Mutex

	bus.SubscribeAll(func(event core.Event) {
		mu.Lock()
		messageEvents = append(messageEvents, event.Type)
		mu.Unlock()
	})

	manager, err := NewConversationManager(DefaultConfig(), agents, bus)
	if err != nil {
		t.Fatalf("NewConversationManager returned error: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()
	_, _ = manager.SendUserMessage(ctx, "Hello")

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// Should have at least: user message created, agent typing, agent done, agent message created
	foundUserMessage := false
	for _, et := range messageEvents {
		if et == core.EventMessageCreated {
			foundUserMessage = true
			break
		}
	}
	if !foundUserMessage {
		t.Error("expected EventMessageCreated to be emitted")
	}
}

func TestParallelAgentResponses(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock"),
		core.NewAgent("agent-2", "mock", "Agent 2", "model", "mock"),
	}

	manager, err := NewConversationManager(DefaultConfig(), agents, nil)
	if err != nil {
		t.Fatalf("NewConversationManager returned error: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()
	agentMessages, err := manager.SendUserMessage(ctx, "Hello to all")
	if err != nil {
		t.Fatalf("SendUserMessage returned error: %v", err)
	}

	// Should get responses from both agents
	if len(agentMessages) != 2 {
		t.Errorf("expected 2 agent messages, got %d", len(agentMessages))
	}

	// All messages should have content
	for _, msg := range agentMessages {
		if msg.Content == "" && msg.Status != core.MessageStatusError {
			t.Errorf("agent %s returned empty content", msg.AgentID)
		}
	}
}

func TestGetMessages_ThreadSafe(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock"),
	}

	manager, err := NewConversationManager(DefaultConfig(), agents, nil)
	if err != nil {
		t.Fatalf("NewConversationManager returned error: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()
	_, _ = manager.SendUserMessage(ctx, "Hello")

	messages1 := manager.GetMessages()
	messages2 := manager.GetMessages()

	// Modifications to returned slice should not affect original
	if len(messages1) > 0 {
		messages1[0].Content = "Modified"

		messages3 := manager.GetMessages()
		if messages3[0].Content == "Modified" {
			t.Error("GetMessages should return a copy, mutations should not affect original")
		}
	}

	// Both slices should be independent
	_ = messages2
}

func TestGetConversation(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock"),
	}

	manager, err := NewConversationManager(DefaultConfig(), agents, nil)
	if err != nil {
		t.Fatalf("NewConversationManager returned error: %v", err)
	}
	defer manager.Close()

	conv := manager.GetConversation()
	if conv == nil {
		t.Fatal("expected non-nil conversation")
	}
	if conv.Status != core.ConversationStatusActive {
		t.Errorf("expected status Active, got %q", conv.Status)
	}
}

func TestGetAgents(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock"),
		core.NewAgent("agent-2", "mock", "Agent 2", "model", "mock"),
	}

	manager, err := NewConversationManager(DefaultConfig(), agents, nil)
	if err != nil {
		t.Fatalf("NewConversationManager returned error: %v", err)
	}
	defer manager.Close()

	retrievedAgents := manager.GetAgents()
	if len(retrievedAgents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(retrievedAgents))
	}
}

func TestGetAgentStatus(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock"),
	}

	manager, err := NewConversationManager(DefaultConfig(), agents, nil)
	if err != nil {
		t.Fatalf("NewConversationManager returned error: %v", err)
	}
	defer manager.Close()

	status := manager.GetAgentStatus()
	if len(status) != 1 {
		t.Errorf("expected 1 agent status, got %d", len(status))
	}
	if status["agent-1"] != core.AgentStatusIdle {
		t.Errorf("expected status Idle, got %q", status["agent-1"])
	}
}

func TestSubscribe(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock"),
	}

	manager, err := NewConversationManager(DefaultConfig(), agents, nil)
	if err != nil {
		t.Fatalf("NewConversationManager returned error: %v", err)
	}
	defer manager.Close()

	var received atomic.Bool
	unsub := manager.Subscribe(core.EventMessageCreated, func(event core.Event) {
		received.Store(true)
	})
	defer unsub()

	ctx := context.Background()
	_, _ = manager.SendUserMessage(ctx, "Test")

	time.Sleep(50 * time.Millisecond)

	if !received.Load() {
		t.Error("subscription should receive events")
	}
}

func TestSubscribeAll(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock"),
	}

	manager, err := NewConversationManager(DefaultConfig(), agents, nil)
	if err != nil {
		t.Fatalf("NewConversationManager returned error: %v", err)
	}
	defer manager.Close()

	var eventCount atomic.Int32
	unsub := manager.SubscribeAll(func(event core.Event) {
		eventCount.Add(1)
	})
	defer unsub()

	ctx := context.Background()
	_, _ = manager.SendUserMessage(ctx, "Test")

	time.Sleep(100 * time.Millisecond)

	if eventCount.Load() < 2 {
		t.Errorf("expected at least 2 events, got %d", eventCount.Load())
	}
}

func TestComplete(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock"),
	}

	bus := events.NewBus()
	defer bus.Close()

	var conversationCompleted atomic.Bool
	bus.Subscribe(core.EventConversationCompleted, func(event core.Event) {
		conversationCompleted.Store(true)
	})

	manager, err := NewConversationManager(DefaultConfig(), agents, bus)
	if err != nil {
		t.Fatalf("NewConversationManager returned error: %v", err)
	}
	defer manager.Close()

	manager.Complete()

	time.Sleep(50 * time.Millisecond)

	if !conversationCompleted.Load() {
		t.Error("expected ConversationCompleted event to be emitted")
	}

	conv := manager.GetConversation()
	if conv.Status != core.ConversationStatusCompleted {
		t.Errorf("expected status Completed, got %q", conv.Status)
	}
}

func TestSummary(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock"),
	}

	manager, err := NewConversationManager(DefaultConfig(), agents, nil)
	if err != nil {
		t.Fatalf("NewConversationManager returned error: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()
	_, _ = manager.SendUserMessage(ctx, "Hello")

	summary := manager.Summary()

	if summary.MessageCount < 2 {
		t.Errorf("expected at least 2 messages, got %d", summary.MessageCount)
	}
	if summary.AgentCount != 1 {
		t.Errorf("expected 1 agent, got %d", summary.AgentCount)
	}
}

func TestClose(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock"),
	}

	manager, err := NewConversationManager(DefaultConfig(), agents, nil)
	if err != nil {
		t.Fatalf("NewConversationManager returned error: %v", err)
	}

	// Should not panic
	manager.Close()
}

// Test that mock adapter is properly registered
func TestMockAdapterIsRegistered(t *testing.T) {
	if !adapters.Has("mock") {
		t.Fatal("mock adapter should be registered")
	}
}
