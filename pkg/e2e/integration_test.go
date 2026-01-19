//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ASRagab/agentpipe/pkg/adapters/mock"
	"github.com/ASRagab/agentpipe/pkg/core"
	"github.com/ASRagab/agentpipe/pkg/events"
	"github.com/ASRagab/agentpipe/pkg/manager"
	"github.com/ASRagab/agentpipe/pkg/pool"
)

// Ensure mock adapter is registered
var _ = mock.NewMockAdapter()

func TestFullConversationFlow(t *testing.T) {
	// Create agents
	agents := []core.Agent{
		core.NewAgent("gpt", "mock", "GPT Agent", "gpt-4", "mock").
			WithSystemPrompt("You are GPT"),
		core.NewAgent("claude", "mock", "Claude Agent", "claude-3", "mock").
			WithSystemPrompt("You are Claude"),
	}

	// Create event bus
	bus := events.NewBus()
	defer bus.Close()

	// Track events
	var eventLog []core.EventType
	var mu sync.Mutex

	bus.SubscribeAll(func(event core.Event) {
		mu.Lock()
		eventLog = append(eventLog, event.Type)
		mu.Unlock()
	})

	// Create manager
	config := manager.Config{
		Timeout: 30 * time.Second,
	}

	mgr, err := manager.NewConversationManager(config, agents, bus)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer mgr.Close()

	// Start conversation
	mgr.Start()

	// Send user message
	ctx := context.Background()
	responses, err := mgr.SendUserMessage(ctx, "Hello, both of you!")
	if err != nil {
		t.Fatalf("SendUserMessage failed: %v", err)
	}

	// Verify responses from both agents
	if len(responses) != 2 {
		t.Errorf("expected 2 responses, got %d", len(responses))
	}

	// Complete conversation
	mgr.Complete()

	// Wait for events
	time.Sleep(100 * time.Millisecond)

	// Verify event flow
	mu.Lock()
	defer mu.Unlock()

	// Should have: ConversationStarted, MessageCreated (user), AgentTyping x2, AgentDone x2, MessageCreated x2, ConversationCompleted
	foundConversationStarted := false
	foundConversationCompleted := false
	agentTypingCount := 0
	agentDoneCount := 0
	messageCreatedCount := 0

	for _, et := range eventLog {
		switch et {
		case core.EventConversationStarted:
			foundConversationStarted = true
		case core.EventConversationCompleted:
			foundConversationCompleted = true
		case core.EventAgentTyping:
			agentTypingCount++
		case core.EventAgentDone:
			agentDoneCount++
		case core.EventMessageCreated:
			messageCreatedCount++
		}
	}

	if !foundConversationStarted {
		t.Error("expected ConversationStarted event")
	}
	if !foundConversationCompleted {
		t.Error("expected ConversationCompleted event")
	}
	if agentTypingCount != 2 {
		t.Errorf("expected 2 AgentTyping events, got %d", agentTypingCount)
	}
	if agentDoneCount != 2 {
		t.Errorf("expected 2 AgentDone events, got %d", agentDoneCount)
	}
	// User message + 2 agent messages
	if messageCreatedCount < 3 {
		t.Errorf("expected at least 3 MessageCreated events, got %d", messageCreatedCount)
	}

	// Verify final conversation state
	conv := mgr.GetConversation()
	if conv.Status != core.ConversationStatusCompleted {
		t.Errorf("expected status Completed, got %q", conv.Status)
	}
	if len(conv.Messages) < 3 {
		t.Errorf("expected at least 3 messages (1 user + 2 agent), got %d", len(conv.Messages))
	}
}

func TestStreamingFlow(t *testing.T) {
	// Create mock adapter with streaming
	adapter := mock.NewMockAdapter()
	adapter.StreamChunks = []string{"Hello ", "from ", "streaming ", "agent!"}
	adapter.StreamDelay = 20 * time.Millisecond

	// Create agent
	agent := core.NewAgent("streamer", "mock", "Streaming Agent", "model", "mock")
	if err := adapter.Initialize(agent); err != nil {
		t.Fatalf("failed to initialize adapter: %v", err)
	}

	// Create event bus
	bus := events.NewBus()
	defer bus.Close()

	// Test streaming
	ctx := context.Background()
	var buf bytes.Buffer
	messages := []core.Message{core.NewUserMessage("Stream to me")}

	metrics, err := adapter.StreamMessage(ctx, messages, &buf, nil)
	if err != nil {
		t.Fatalf("StreamMessage failed: %v", err)
	}

	// Verify streamed content
	expectedContent := "Hello from streaming agent!"
	if buf.String() != expectedContent {
		t.Errorf("expected content %q, got %q", expectedContent, buf.String())
	}

	// Verify metrics returned
	if metrics == nil {
		t.Error("expected non-nil metrics")
	}
}

func TestMultipleMessages(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock"),
	}

	mgr, err := manager.NewConversationManager(manager.DefaultConfig(), agents, nil)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer mgr.Close()

	ctx := context.Background()

	// Send multiple messages
	for i := 0; i < 5; i++ {
		_, err := mgr.SendUserMessage(ctx, "Message "+string(rune('1'+i)))
		if err != nil {
			t.Fatalf("SendUserMessage %d failed: %v", i, err)
		}
	}

	// Verify conversation grows
	messages := mgr.GetMessages()

	// 5 user messages + 5 agent responses = 10 messages
	if len(messages) != 10 {
		t.Errorf("expected 10 messages, got %d", len(messages))
	}

	// Verify alternating user/agent pattern
	for i, msg := range messages {
		if i%2 == 0 {
			// Even indices: user messages
			if msg.Role != core.RoleUser {
				t.Errorf("message %d: expected user role, got %q", i, msg.Role)
			}
		} else {
			// Odd indices: agent messages
			if msg.Role != core.RoleAgent {
				t.Errorf("message %d: expected agent role, got %q", i, msg.Role)
			}
		}
	}
}

func TestParallelAgentsWithDifferentSpeeds(t *testing.T) {
	// Create two agents with different response times
	bus := events.NewBus()
	defer bus.Close()

	// We need to create custom adapters with different delays
	// First, register custom adapters
	fastAdapter := mock.NewMockAdapter()
	fastAdapter.Response = "I'm fast!"
	fastAdapter.Delay = 50 * time.Millisecond

	slowAdapter := mock.NewMockAdapter()
	slowAdapter.Response = "I'm slow..."
	slowAdapter.Delay = 150 * time.Millisecond

	// Create pool directly
	agentPool := pool.NewPool(bus, 10*time.Second)

	fastAgent := core.NewAgent("fast", "mock", "Fast Agent", "model", "mock")
	slowAgent := core.NewAgent("slow", "mock", "Slow Agent", "model", "mock")

	agentPool.AddAgent(fastAgent, fastAdapter)
	agentPool.AddAgent(slowAgent, slowAdapter)

	// Execute parallel
	ctx := context.Background()
	messages := []core.Message{core.NewUserMessage("Hello")}

	start := time.Now()
	responses := agentPool.ExecuteParallel(ctx, messages, nil)
	elapsed := time.Since(start)

	// Verify both responded
	if len(responses) != 2 {
		t.Errorf("expected 2 responses, got %d", len(responses))
	}

	// Verify parallel execution (should be ~150ms, not 200ms)
	if elapsed > 250*time.Millisecond {
		t.Errorf("parallel execution took too long: %v", elapsed)
	}
	if elapsed < 150*time.Millisecond {
		t.Errorf("execution was too fast: %v", elapsed)
	}

	// Verify both agents responded with correct content
	responseMap := make(map[string]string)
	for _, resp := range responses {
		if resp.Message != nil {
			responseMap[resp.AgentID] = resp.Message.Content
		}
	}

	if responseMap["fast"] != "I'm fast!" {
		t.Errorf("fast agent response: %q", responseMap["fast"])
	}
	if responseMap["slow"] != "I'm slow..." {
		t.Errorf("slow agent response: %q", responseMap["slow"])
	}
}

func TestAgentErrorRecovery(t *testing.T) {
	bus := events.NewBus()
	defer bus.Close()

	// Track error events
	var errorEvents atomic.Int32
	bus.Subscribe(core.EventAgentError, func(event core.Event) {
		errorEvents.Add(1)
	})

	// Create pool with one failing and one succeeding agent
	agentPool := pool.NewPool(bus, 10*time.Second)

	successAdapter := mock.NewMockAdapter()
	successAdapter.Response = "I worked!"
	successAdapter.Delay = 20 * time.Millisecond

	failAdapter := mock.NewMockAdapter()
	failAdapter.Error = errors.New("adapter not available") // Simulating error

	agentPool.AddAgent(core.NewAgent("success", "mock", "Success", "model", "mock"), successAdapter)
	agentPool.AddAgent(core.NewAgent("fail", "mock", "Fail", "model", "mock"), failAdapter)

	ctx := context.Background()
	responses := agentPool.ExecuteParallel(ctx, []core.Message{core.NewUserMessage("Test")}, nil)

	// Wait for events
	time.Sleep(100 * time.Millisecond)

	// Verify we still get responses for both
	if len(responses) != 2 {
		t.Errorf("expected 2 responses, got %d", len(responses))
	}

	// Verify error event was emitted
	if errorEvents.Load() != 1 {
		t.Errorf("expected 1 error event, got %d", errorEvents.Load())
	}

	// Verify partial success
	var successCount, failCount int
	for _, resp := range responses {
		if resp.Error != nil {
			failCount++
		} else {
			successCount++
		}
	}

	if successCount != 1 {
		t.Errorf("expected 1 success, got %d", successCount)
	}
	if failCount != 1 {
		t.Errorf("expected 1 failure, got %d", failCount)
	}
}

func TestEventBusIntegration(t *testing.T) {
	bus := events.NewBus()
	defer bus.Close()

	// Subscribe to specific events
	var typingAgents []string
	var doneAgents []string
	var mu sync.Mutex

	bus.Subscribe(core.EventAgentTyping, func(event core.Event) {
		if data, ok := event.Data.(core.AgentTypingData); ok {
			mu.Lock()
			typingAgents = append(typingAgents, data.AgentName)
			mu.Unlock()
		}
	})

	bus.Subscribe(core.EventAgentDone, func(event core.Event) {
		if data, ok := event.Data.(core.AgentDoneData); ok {
			mu.Lock()
			doneAgents = append(doneAgents, data.AgentName)
			mu.Unlock()
		}
	})

	// Create pool
	agentPool := pool.NewPool(bus, 10*time.Second)

	adapter := mock.NewMockAdapter()
	adapter.Response = "Hello!"
	adapter.Delay = 30 * time.Millisecond

	agentPool.AddAgent(core.NewAgent("agent-1", "mock", "Test Agent", "model", "mock"), adapter)

	// Execute
	ctx := context.Background()
	_ = agentPool.ExecuteParallel(ctx, []core.Message{core.NewUserMessage("Test")}, nil)

	// Wait for events
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(typingAgents) != 1 || typingAgents[0] != "Test Agent" {
		t.Errorf("unexpected typing agents: %v", typingAgents)
	}
	if len(doneAgents) != 1 || doneAgents[0] != "Test Agent" {
		t.Errorf("unexpected done agents: %v", doneAgents)
	}
}

func TestConversationPersistence(t *testing.T) {
	agents := []core.Agent{
		core.NewAgent("agent-1", "mock", "Agent 1", "model", "mock"),
	}

	mgr, err := manager.NewConversationManager(manager.DefaultConfig(), agents, nil)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer mgr.Close()

	ctx := context.Background()

	// Send messages
	_, _ = mgr.SendUserMessage(ctx, "First message")
	_, _ = mgr.SendUserMessage(ctx, "Second message")

	// Get conversation and verify it persists across calls
	conv1 := mgr.GetConversation()
	conv2 := mgr.GetConversation()

	if conv1.ID != conv2.ID {
		t.Error("conversation ID should be consistent")
	}

	// Verify summary
	summary := mgr.Summary()
	if summary.MessageCount != 4 { // 2 user + 2 agent
		t.Errorf("expected 4 messages, got %d", summary.MessageCount)
	}
}
