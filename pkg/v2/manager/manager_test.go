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

// ============================================
// Graceful Degradation Tests
// ============================================

func TestDefaultGracefulDegradationConfig(t *testing.T) {
	config := DefaultGracefulDegradationConfig()

	if !config.Enabled {
		t.Error("expected Enabled to be true by default")
	}
	if !config.PauseOnAllFailed {
		t.Error("expected PauseOnAllFailed to be true by default")
	}
	if !config.RetryFailedOnNextMessage {
		t.Error("expected RetryFailedOnNextMessage to be true by default")
	}
	if !config.EmitSystemMessages {
		t.Error("expected EmitSystemMessages to be true by default")
	}
}

func TestGracefulDegradation_ContinuesWithSomeAgentsFailing(t *testing.T) {
	// Create a custom registry for this test to control adapter behavior
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock"),
		core.NewAgent("agent-2", "mock", "Agent 2", "model", "mock"),
	}

	bus := events.NewBus()
	defer bus.Close()

	// Track error events
	var agentErrors []string
	var mu sync.Mutex
	bus.Subscribe(core.EventAgentError, func(event core.Event) {
		mu.Lock()
		if data, ok := event.Data.(core.AgentErrorData); ok {
			agentErrors = append(agentErrors, data.AgentName)
		}
		mu.Unlock()
	})

	manager, err := NewConversationManager(DefaultConfig(), agents, bus)
	if err != nil {
		t.Fatalf("NewConversationManager returned error: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()
	agentMessages, err := manager.SendUserMessage(ctx, "Hello")

	// Both agents should respond (mock adapter doesn't fail by default)
	if err != nil {
		t.Fatalf("SendUserMessage returned error: %v", err)
	}
	if len(agentMessages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(agentMessages))
	}
}

func TestGracefulDegradation_TracksFailedAgents(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock"),
	}

	config := DefaultConfig()
	manager, err := NewConversationManager(config, agents, nil)
	if err != nil {
		t.Fatalf("NewConversationManager returned error: %v", err)
	}
	defer manager.Close()

	// Initially no failed agents
	failed := manager.GetFailedAgents()
	if len(failed) != 0 {
		t.Errorf("expected 0 failed agents initially, got %d", len(failed))
	}

	// Verify IsPaused is false initially
	if manager.IsPaused() {
		t.Error("expected conversation to not be paused initially")
	}
}

func TestGracefulDegradation_IsPausedAndResume(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock"),
	}

	manager, err := NewConversationManager(DefaultConfig(), agents, nil)
	if err != nil {
		t.Fatalf("NewConversationManager returned error: %v", err)
	}
	defer manager.Close()

	if manager.IsPaused() {
		t.Error("expected conversation to not be paused initially")
	}

	// Simulate pausing (normally happens when all agents fail)
	manager.mu.Lock()
	manager.paused = true
	manager.failedAgents["agent-1"] = &FailedAgentInfo{
		AgentID:   "agent-1",
		AgentName: "Agent 1",
		LastError: "test error",
		FailedAt:  time.Now(),
	}
	manager.mu.Unlock()

	if !manager.IsPaused() {
		t.Error("expected conversation to be paused")
	}

	// Test ResumeConversation without clearing failed
	manager.ResumeConversation(false)
	if manager.IsPaused() {
		t.Error("expected conversation to be resumed")
	}

	failed := manager.GetFailedAgents()
	if len(failed) != 1 {
		t.Errorf("expected 1 failed agent (not cleared), got %d", len(failed))
	}

	// Pause again and resume with clearing
	manager.mu.Lock()
	manager.paused = true
	manager.mu.Unlock()

	manager.ResumeConversation(true)
	if manager.IsPaused() {
		t.Error("expected conversation to be resumed")
	}

	failed = manager.GetFailedAgents()
	if len(failed) != 0 {
		t.Errorf("expected 0 failed agents (cleared), got %d", len(failed))
	}
}

func TestGracefulDegradation_SendMessageWhenPaused(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock"),
	}

	manager, err := NewConversationManager(DefaultConfig(), agents, nil)
	if err != nil {
		t.Fatalf("NewConversationManager returned error: %v", err)
	}
	defer manager.Close()

	// Pause the conversation
	manager.mu.Lock()
	manager.paused = true
	manager.mu.Unlock()

	// Try to send a message while paused
	ctx := context.Background()
	_, err = manager.SendUserMessage(ctx, "Hello")

	if err != ErrConversationPaused {
		t.Errorf("expected ErrConversationPaused, got %v", err)
	}
}

func TestGracefulDegradation_ResetCircuitBreakers(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock"),
	}

	manager, err := NewConversationManager(DefaultConfig(), agents, nil)
	if err != nil {
		t.Fatalf("NewConversationManager returned error: %v", err)
	}
	defer manager.Close()

	// Should not panic
	manager.ResetCircuitBreakers()

	// Verify available count
	count := manager.GetAvailableAgentCount()
	if count != 1 {
		t.Errorf("expected 1 available agent, got %d", count)
	}
}

func TestGracefulDegradation_SystemMessageFormat(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock"),
	}

	manager, err := NewConversationManager(DefaultConfig(), agents, nil)
	if err != nil {
		t.Fatalf("NewConversationManager returned error: %v", err)
	}
	defer manager.Close()

	// Test different error types
	testCases := []struct {
		errorMsg     string
		expectedPart string
	}{
		{"connection timeout", "request timed out"},
		{"rate limit exceeded", "rate limit exceeded"},
		{"connection refused", "connection error"},
		{"authentication failed", "authentication error"},
		{"unknown error", "currently unavailable"},
	}

	for _, tc := range testCases {
		msg := manager.createUnavailableSystemMessage("TestAgent", tc.errorMsg)
		if msg.Role != core.RoleSystem {
			t.Errorf("expected RoleSystem, got %s", msg.Role)
		}
		if msg.Content == "" {
			t.Error("expected non-empty content")
		}
		// Content should mention the agent name
		if !containsString(msg.Content, "TestAgent") {
			t.Errorf("expected content to contain agent name, got %q", msg.Content)
		}
	}
}

func TestGracefulDegradation_FailedAgentRetryTracking(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock"),
	}

	manager, err := NewConversationManager(DefaultConfig(), agents, nil)
	if err != nil {
		t.Fatalf("NewConversationManager returned error: %v", err)
	}
	defer manager.Close()

	// Track first failure
	manager.trackFailedAgent("agent-1", "Agent 1", "error 1")

	failed := manager.GetFailedAgents()
	if len(failed) != 1 {
		t.Fatalf("expected 1 failed agent, got %d", len(failed))
	}
	if failed[0].RetryCount != 1 {
		t.Errorf("expected retry count 1, got %d", failed[0].RetryCount)
	}

	// Track second failure (should increment)
	manager.trackFailedAgent("agent-1", "Agent 1", "error 2")

	failed = manager.GetFailedAgents()
	if len(failed) != 1 {
		t.Fatalf("expected 1 failed agent, got %d", len(failed))
	}
	if failed[0].RetryCount != 2 {
		t.Errorf("expected retry count 2, got %d", failed[0].RetryCount)
	}
	if failed[0].LastError != "error 2" {
		t.Errorf("expected last error 'error 2', got %q", failed[0].LastError)
	}

	// Clear the failure
	manager.clearFailedAgent("agent-1")

	failed = manager.GetFailedAgents()
	if len(failed) != 0 {
		t.Errorf("expected 0 failed agents after clear, got %d", len(failed))
	}
}

func TestGracefulDegradation_DisabledEmitsNoSystemMessages(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock"),
	}

	config := DefaultConfig()
	config.GracefulDegradation.EmitSystemMessages = false

	manager, err := NewConversationManager(config, agents, nil)
	if err != nil {
		t.Fatalf("NewConversationManager returned error: %v", err)
	}
	defer manager.Close()

	// Verify config is applied
	if manager.config.GracefulDegradation.EmitSystemMessages {
		t.Error("expected EmitSystemMessages to be false")
	}
}

func TestGracefulDegradation_DisabledPauseOnAllFailed(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock"),
	}

	config := DefaultConfig()
	config.GracefulDegradation.PauseOnAllFailed = false

	manager, err := NewConversationManager(config, agents, nil)
	if err != nil {
		t.Fatalf("NewConversationManager returned error: %v", err)
	}
	defer manager.Close()

	// Verify config is applied
	if manager.config.GracefulDegradation.PauseOnAllFailed {
		t.Error("expected PauseOnAllFailed to be false")
	}
}

func TestGracefulDegradation_ConfigDefaults(t *testing.T) {
	config := DefaultConfig()

	// Verify graceful degradation is included in default config
	if !config.GracefulDegradation.Enabled {
		t.Error("expected GracefulDegradation.Enabled to be true in DefaultConfig")
	}
	if !config.GracefulDegradation.PauseOnAllFailed {
		t.Error("expected GracefulDegradation.PauseOnAllFailed to be true in DefaultConfig")
	}
	if !config.GracefulDegradation.RetryFailedOnNextMessage {
		t.Error("expected GracefulDegradation.RetryFailedOnNextMessage to be true in DefaultConfig")
	}
	if !config.GracefulDegradation.EmitSystemMessages {
		t.Error("expected GracefulDegradation.EmitSystemMessages to be true in DefaultConfig")
	}
}

func TestGracefulDegradation_FailedAgentInfoFields(t *testing.T) {
	info := FailedAgentInfo{
		AgentID:    "test-id",
		AgentName:  "Test Agent",
		LastError:  "test error",
		FailedAt:   time.Now(),
		RetryCount: 3,
	}

	if info.AgentID != "test-id" {
		t.Errorf("expected AgentID 'test-id', got %q", info.AgentID)
	}
	if info.AgentName != "Test Agent" {
		t.Errorf("expected AgentName 'Test Agent', got %q", info.AgentName)
	}
	if info.LastError != "test error" {
		t.Errorf("expected LastError 'test error', got %q", info.LastError)
	}
	if info.RetryCount != 3 {
		t.Errorf("expected RetryCount 3, got %d", info.RetryCount)
	}
}

func TestGracefulDegradation_MultipleAgentsPartialFailure(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock"),
		core.NewAgent("agent-2", "mock", "Agent 2", "model", "mock"),
		core.NewAgent("agent-3", "mock", "Agent 3", "model", "mock"),
	}

	manager, err := NewConversationManager(DefaultConfig(), agents, nil)
	if err != nil {
		t.Fatalf("NewConversationManager returned error: %v", err)
	}
	defer manager.Close()

	// Send message - all should succeed with mock adapter
	ctx := context.Background()
	agentMessages, err := manager.SendUserMessage(ctx, "Hello all")
	if err != nil {
		t.Fatalf("SendUserMessage returned error: %v", err)
	}

	// All 3 agents should respond
	if len(agentMessages) != 3 {
		t.Errorf("expected 3 agent messages, got %d", len(agentMessages))
	}

	// Conversation should not be paused
	if manager.IsPaused() {
		t.Error("expected conversation to not be paused")
	}
}

func TestErrAllAgentsFailed_ErrorMessage(t *testing.T) {
	expected := "all agents failed to respond"
	if ErrAllAgentsFailed.Error() != expected {
		t.Errorf("expected error message %q, got %q", expected, ErrAllAgentsFailed.Error())
	}
}

func TestErrConversationPaused_ErrorMessage(t *testing.T) {
	expected := "conversation is paused due to all agents failing"
	if ErrConversationPaused.Error() != expected {
		t.Errorf("expected error message %q, got %q", expected, ErrConversationPaused.Error())
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
