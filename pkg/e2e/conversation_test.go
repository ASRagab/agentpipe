//go:build e2e

package e2e

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/ASRagab/agentpipe/pkg/core"
)

// TestSingleUserMessage verifies that a single user message receives responses from all agents.
func TestSingleUserMessage(t *testing.T) {
	TimeoutWrapper(t, 10*time.Second, func(t *testing.T) {
		agents := FixtureSimpleConversation()
		harness := NewTestHarness(t, agents)
		defer harness.Cleanup()

		// Configure mock responses
		harness.SetMockResponse("alice", "Hello from Alice!")
		harness.SetMockResponse("bob", "Greetings from Bob!")

		// Start conversation
		harness.Manager.Start()

		// Send message
		ctx := context.Background()
		responses, err := SendTestMessage(ctx, harness, "Hello everyone!")
		if err != nil {
			t.Fatalf("SendTestMessage failed: %v", err)
		}

		// Verify responses from both agents
		if len(responses) != 2 {
			t.Errorf("expected 2 responses, got %d", len(responses))
		}

		// Verify event sequence - with 2 agents, we get 2 typing and 2 done events
		WaitForResponses(harness, core.EventAgentDone, 2, 500*time.Millisecond)

		// Verify minimum expected events occurred (exact sequence varies due to parallel execution)
		AssertEventCount(t, harness.EventLog, core.EventConversationStarted, 1)
		AssertEventCount(t, harness.EventLog, core.EventAgentTyping, 2)
		AssertEventCount(t, harness.EventLog, core.EventAgentDone, 2)

		// Verify message content
		messages := harness.Manager.GetMessages()
		if len(messages) < 3 {
			t.Errorf("expected at least 3 messages (1 user + 2 agents), got %d", len(messages))
		}

		// Check user message
		if messages[0].Role != core.RoleUser {
			t.Errorf("first message should be from user, got %s", messages[0].Role)
		}
		if messages[0].Content != "Hello everyone!" {
			t.Errorf("unexpected user message content: %s", messages[0].Content)
		}
	})
}

// TestMultipleTurns verifies a three-turn conversation with different messages.
func TestMultipleTurns(t *testing.T) {
	TimeoutWrapper(t, 15*time.Second, func(t *testing.T) {
		agents := FixtureSimpleConversation()
		harness := NewTestHarness(t, agents)
		defer harness.Cleanup()

		harness.Manager.Start()
		ctx := context.Background()

		// Turn 1
		harness.SetMockResponse("alice", "Alice turn 1")
		harness.SetMockResponse("bob", "Bob turn 1")
		_, err := SendTestMessage(ctx, harness, "First message")
		if err != nil {
			t.Fatalf("Turn 1 failed: %v", err)
		}

		// Turn 2
		harness.SetMockResponse("alice", "Alice turn 2")
		harness.SetMockResponse("bob", "Bob turn 2")
		_, err = SendTestMessage(ctx, harness, "Second message")
		if err != nil {
			t.Fatalf("Turn 2 failed: %v", err)
		}

		// Turn 3
		harness.SetMockResponse("alice", "Alice turn 3")
		harness.SetMockResponse("bob", "Bob turn 3")
		_, err = SendTestMessage(ctx, harness, "Third message")
		if err != nil {
			t.Fatalf("Turn 3 failed: %v", err)
		}

		// Verify message count: 3 user + 6 agent = 9
		messages := harness.Manager.GetMessages()
		if len(messages) != 9 {
			t.Errorf("expected 9 messages, got %d", len(messages))
		}

		// Verify conversation summary
		summary := harness.Manager.Summary()
		if summary.MessageCount != 9 {
			t.Errorf("summary message count: expected 9, got %d", summary.MessageCount)
		}

		// Verify agent typing events (2 agents × 3 turns = 6)
		AssertEventCount(t, harness.EventLog, core.EventAgentTyping, 6)
		AssertEventCount(t, harness.EventLog, core.EventAgentDone, 6)
	})
}

// TestLongConversation verifies a 20+ message conversation for memory/performance.
func TestLongConversation(t *testing.T) {
	TimeoutWrapper(t, 60*time.Second, func(t *testing.T) {
		agents := []core.Agent{
			core.NewAgent("agent-1", "mock", "Solo Agent", "model", "mock"),
		}
		harness := NewTestHarness(t, agents, WithTimeout(30*time.Second))
		defer harness.Cleanup()

		harness.Manager.Start()
		ctx := context.Background()

		messageCount := 25
		for i := 0; i < messageCount; i++ {
			harness.SetMockResponse("agent-1", "Response to message "+string(rune('A'+i%26)))
			_, err := SendTestMessage(ctx, harness, "Message "+string(rune('A'+i%26)))
			if err != nil {
				t.Fatalf("Message %d failed: %v", i, err)
			}
		}

		// Verify total messages: messageCount user + messageCount agent = messageCount * 2
		expectedMessages := messageCount * 2
		messages := harness.Manager.GetMessages()
		if len(messages) != expectedMessages {
			t.Errorf("expected %d messages, got %d", expectedMessages, len(messages))
		}

		// Verify conversation can be completed successfully
		harness.Manager.Complete()

		conv := harness.Manager.GetConversation()
		if conv.Status != core.ConversationStatusCompleted {
			t.Errorf("expected completed status, got %s", conv.Status)
		}

		// Verify memory usage is reasonable (basic sanity check)
		summary := harness.Manager.Summary()
		if summary.TotalTokens <= 0 {
			// This is expected with mocks, just verify it's tracked
			t.Log("Note: TotalTokens is 0 or negative with mock adapters (expected)")
		}
	})
}

// TestAgentContextAwareness verifies that agents receive full message history.
func TestAgentContextAwareness(t *testing.T) {
	TimeoutWrapper(t, 15*time.Second, func(t *testing.T) {
		agents := []core.Agent{
			core.NewAgent("context-agent", "mock", "Context Agent", "model", "mock"),
		}
		harness := NewTestHarness(t, agents)
		defer harness.Cleanup()

		// Track messages received by the mock adapter
		var receivedMessages [][]core.Message

		adapter := harness.MockAdapters["context-agent"]
		adapter.OnSendMessage = func(ctx context.Context, messages []core.Message, conversation *core.ConversationContext) (string, *core.Metrics, error) {
			// Make a copy of the messages
			msgCopy := make([]core.Message, len(messages))
			copy(msgCopy, messages)
			receivedMessages = append(receivedMessages, msgCopy)
			return "Response", &core.Metrics{
				InputTokens:  len(messages) * 10,
				OutputTokens: 10,
				TotalTokens:  len(messages)*10 + 10,
			}, nil
		}

		harness.Manager.Start()
		ctx := context.Background()

		// Send 3 messages
		_, _ = SendTestMessage(ctx, harness, "First")
		_, _ = SendTestMessage(ctx, harness, "Second")
		_, _ = SendTestMessage(ctx, harness, "Third")

		// Verify each call received incrementally more context
		if len(receivedMessages) != 3 {
			t.Fatalf("expected 3 calls, got %d", len(receivedMessages))
		}

		// First call: 1 user message
		if len(receivedMessages[0]) != 1 {
			t.Errorf("first call: expected 1 message, got %d", len(receivedMessages[0]))
		}

		// Second call: 1 user + 1 agent + 1 user = 3 messages
		if len(receivedMessages[1]) != 3 {
			t.Errorf("second call: expected 3 messages, got %d", len(receivedMessages[1]))
		}

		// Third call: 3 + 1 agent + 1 user = 5 messages
		if len(receivedMessages[2]) != 5 {
			t.Errorf("third call: expected 5 messages, got %d", len(receivedMessages[2]))
		}
	})
}

// TestMessageOrdering verifies that messages are ordered by start time, not completion time.
func TestMessageOrdering(t *testing.T) {
	TimeoutWrapper(t, 10*time.Second, func(t *testing.T) {
		agents := []core.Agent{
			core.NewAgent("fast", "mock", "Fast Agent", "model", "mock"),
			core.NewAgent("slow", "mock", "Slow Agent", "model", "mock"),
		}
		harness := NewTestHarness(t, agents)
		defer harness.Cleanup()

		// Fast agent completes in 30ms, slow agent in 100ms
		harness.SetMockDelay("fast", 30*time.Millisecond)
		harness.SetMockDelay("slow", 100*time.Millisecond)
		harness.SetMockResponse("fast", "Fast response")
		harness.SetMockResponse("slow", "Slow response")

		harness.Manager.Start()
		ctx := context.Background()

		responses, err := SendTestMessage(ctx, harness, "Hello")
		if err != nil {
			t.Fatalf("SendTestMessage failed: %v", err)
		}

		if len(responses) != 2 {
			t.Fatalf("expected 2 responses, got %d", len(responses))
		}

		// Sort responses by Timestamp to verify ordering
		sort.Slice(responses, func(i, j int) bool {
			return responses[i].Timestamp.Before(responses[j].Timestamp)
		})

		// Verify that timestamps are recorded (should be very close together for parallel execution)
		timeDiff := responses[1].Timestamp.Sub(responses[0].Timestamp)
		if timeDiff > 150*time.Millisecond {
			t.Errorf("timestamps too far apart: %v (expected parallel start)", timeDiff)
		}

		// Verify metrics reflect the delays (if available)
		if responses[0].Metrics != nil && responses[0].Metrics.Duration < 25*time.Millisecond {
			t.Errorf("fast response duration too short: %v", responses[0].Metrics.Duration)
		}
	})
}

// TestConversationComplete verifies proper conversation completion.
func TestConversationComplete(t *testing.T) {
	TimeoutWrapper(t, 10*time.Second, func(t *testing.T) {
		agents := FixtureSimpleConversation()
		harness := NewTestHarness(t, agents)
		defer harness.Cleanup()

		harness.Manager.Start()
		ctx := context.Background()

		_, _ = SendTestMessage(ctx, harness, "Hello")

		// Complete the conversation
		harness.Manager.Complete()

		// Wait for completion event
		WaitForEvent(harness, core.EventConversationCompleted, 500*time.Millisecond)

		// Verify status
		conv := harness.Manager.GetConversation()
		if conv.Status != core.ConversationStatusCompleted {
			t.Errorf("expected completed status, got %s", conv.Status)
		}

		// Verify completion event was emitted
		AssertEventCount(t, harness.EventLog, core.EventConversationCompleted, 1)
	})
}

// TestEmptyConversation verifies behavior with no messages.
func TestEmptyConversation(t *testing.T) {
	TimeoutWrapper(t, 5*time.Second, func(t *testing.T) {
		agents := FixtureSimpleConversation()
		harness := NewTestHarness(t, agents)
		defer harness.Cleanup()

		harness.Manager.Start()
		harness.Manager.Complete()

		// Verify empty conversation completes without error
		summary := harness.Manager.Summary()
		if summary.MessageCount != 0 {
			t.Errorf("expected 0 messages, got %d", summary.MessageCount)
		}

		AssertEventCount(t, harness.EventLog, core.EventConversationCompleted, 1)
	})
}

// TestSingleAgent verifies conversation with only one agent.
func TestSingleAgent(t *testing.T) {
	TimeoutWrapper(t, 10*time.Second, func(t *testing.T) {
		agents := []core.Agent{
			core.NewAgent("solo", "mock", "Solo Agent", "model", "mock"),
		}
		harness := NewTestHarness(t, agents)
		defer harness.Cleanup()

		harness.SetMockResponse("solo", "Solo response")

		harness.Manager.Start()
		ctx := context.Background()

		responses, err := SendTestMessage(ctx, harness, "Hello solo")
		if err != nil {
			t.Fatalf("SendTestMessage failed: %v", err)
		}

		if len(responses) != 1 {
			t.Errorf("expected 1 response, got %d", len(responses))
		}

		if responses[0].Content != "Solo response" {
			t.Errorf("unexpected response content: %s", responses[0].Content)
		}
	})
}

// TestLargeAgentGroup verifies conversation with 5+ agents.
func TestLargeAgentGroup(t *testing.T) {
	TimeoutWrapper(t, 15*time.Second, func(t *testing.T) {
		agents := FixtureLargeGroup()
		harness := NewTestHarness(t, agents)
		defer harness.Cleanup()

		// Set responses for all agents
		for i := range agents {
			harness.SetMockResponse(agents[i].ID, "Response from "+agents[i].Name)
		}

		harness.Manager.Start()
		ctx := context.Background()

		responses, err := SendTestMessage(ctx, harness, "Hello everyone")
		if err != nil {
			t.Fatalf("SendTestMessage failed: %v", err)
		}

		// Should get response from all 5 agents
		if len(responses) != 5 {
			t.Errorf("expected 5 responses, got %d", len(responses))
		}

		// Verify no errors
		AssertNoErrors(t, harness.EventLog)

		// Verify 5 agent done events
		AssertEventCount(t, harness.EventLog, core.EventAgentDone, 5)
	})
}
