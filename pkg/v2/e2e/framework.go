// Package e2e provides end-to-end testing framework for v2 AgentPipe.
// It includes test harnesses, helper functions, mock event collectors,
// and test fixtures for comprehensive integration testing.
package e2e

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ASRagab/agentpipe/pkg/v2/adapters"
	"github.com/ASRagab/agentpipe/pkg/v2/adapters/mock"
	"github.com/ASRagab/agentpipe/pkg/v2/core"
	"github.com/ASRagab/agentpipe/pkg/v2/events"
	"github.com/ASRagab/agentpipe/pkg/v2/manager"
)

// TestHarness provides a complete v2 stack for end-to-end testing.
type TestHarness struct {
	t            *testing.T
	Manager      *manager.ConversationManager
	EventBus     *events.Bus
	EventLog     *EventCollector
	MockAdapters map[string]*mock.MockAdapter
	config       manager.Config

	mu      sync.RWMutex
	cleanup []func()
}

// HarnessOption configures the test harness.
type HarnessOption func(*TestHarness)

// WithTimeout sets the conversation timeout.
func WithTimeout(d time.Duration) HarnessOption {
	return func(h *TestHarness) {
		h.config.Timeout = d
	}
}

// WithPersistence enables persistence with the given save directory.
func WithPersistence(saveDir string, interval time.Duration) HarnessOption {
	return func(h *TestHarness) {
		h.config.Persistence.Enabled = true
		h.config.Persistence.SaveDir = saveDir
		h.config.Persistence.SaveInterval = interval
	}
}

// WithGracefulDegradation configures graceful degradation behavior.
func WithGracefulDegradation(cfg manager.GracefulDegradationConfig) HarnessOption {
	return func(h *TestHarness) {
		h.config.GracefulDegradation = cfg
	}
}

// NewTestHarness creates a new test harness with the specified agents.
func NewTestHarness(t *testing.T, agents []core.Agent, opts ...HarnessOption) *TestHarness {
	t.Helper()

	h := &TestHarness{
		t:            t,
		EventBus:     events.NewBus(),
		EventLog:     NewEventCollector(),
		MockAdapters: make(map[string]*mock.MockAdapter),
		config:       manager.DefaultConfig(),
		cleanup:      make([]func(), 0),
	}

	// Apply options
	for _, opt := range opts {
		opt(h)
	}

	// Subscribe to all events
	h.EventBus.SubscribeAll(func(event core.Event) {
		h.EventLog.Record(event)
	})

	// Register mock adapters for each agent
	for i := range agents {
		adapter := mock.NewMockAdapter()
		adapter.Delay = 25 * time.Millisecond // Minimum delay for Windows timer resolution
		h.MockAdapters[agents[i].ID] = adapter

		// Register custom adapter factory that returns our specific mock
		adapterID := agents[i].ID
		adapters.Register("mock-"+adapterID, func() adapters.AgentAdapter {
			return h.MockAdapters[adapterID]
		})
		agents[i].AdapterName = "mock-" + adapterID
	}

	// Create manager
	mgr, err := manager.NewConversationManager(h.config, agents, h.EventBus)
	if err != nil {
		t.Fatalf("failed to create ConversationManager: %v", err)
	}
	h.Manager = mgr

	// Register cleanup
	h.cleanup = append(h.cleanup, func() {
		h.Manager.Close()
		h.EventBus.Close()
	})

	return h
}

// Cleanup releases all resources.
func (h *TestHarness) Cleanup() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, fn := range h.cleanup {
		fn()
	}
	h.cleanup = nil
}

// SetMockResponse sets the response for a specific agent.
func (h *TestHarness) SetMockResponse(agentID, response string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if adapter, ok := h.MockAdapters[agentID]; ok {
		adapter.Response = response
	}
}

// SetMockError sets an error for a specific agent.
func (h *TestHarness) SetMockError(agentID string, err error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if adapter, ok := h.MockAdapters[agentID]; ok {
		adapter.Error = err
	}
}

// SetMockDelay sets the response delay for a specific agent.
func (h *TestHarness) SetMockDelay(agentID string, delay time.Duration) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if adapter, ok := h.MockAdapters[agentID]; ok {
		adapter.Delay = delay
	}
}

// SetMockStreamChunks sets the streaming chunks for a specific agent.
func (h *TestHarness) SetMockStreamChunks(agentID string, chunks []string, delay time.Duration) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if adapter, ok := h.MockAdapters[agentID]; ok {
		adapter.StreamChunks = chunks
		adapter.StreamDelay = delay
	}
}

// CreateTestManager is a convenience function to create a manager with default settings.
func CreateTestManager(t *testing.T, agentCount int) *TestHarness {
	t.Helper()

	agents := make([]core.Agent, agentCount)
	for i := 0; i < agentCount; i++ {
		agents[i] = core.NewAgent(
			AgentID(i),
			"mock",
			AgentName(i),
			"test-model",
			"mock",
		).WithSystemPrompt("You are test agent " + AgentName(i))
	}

	return NewTestHarness(t, agents)
}

// SendTestMessage sends a test message and returns responses.
func SendTestMessage(ctx context.Context, h *TestHarness, content string) ([]core.Message, error) {
	return h.Manager.SendUserMessage(ctx, content)
}

// WaitForResponses waits until the expected number of events are received or timeout.
func WaitForResponses(h *TestHarness, eventType core.EventType, count int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if h.EventLog.CountByType(eventType) >= count {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// WaitForEvent waits for a specific event type to appear.
func WaitForEvent(h *TestHarness, eventType core.EventType, timeout time.Duration) bool {
	return WaitForResponses(h, eventType, 1, timeout)
}

// AgentID generates a test agent ID.
func AgentID(index int) string {
	return testAgentIDs[index%len(testAgentIDs)]
}

// AgentName generates a test agent name.
func AgentName(index int) string {
	return testAgentNames[index%len(testAgentNames)]
}

// Test agent identifiers.
var testAgentIDs = []string{
	"alice", "bob", "charlie", "diana", "eve",
	"frank", "grace", "henry", "iris", "jack",
}

var testAgentNames = []string{
	"Alice", "Bob", "Charlie", "Diana", "Eve",
	"Frank", "Grace", "Henry", "Iris", "Jack",
}

// EventCollector collects and tracks events for testing.
type EventCollector struct {
	mu     sync.RWMutex
	events []core.Event
}

// NewEventCollector creates a new event collector.
func NewEventCollector() *EventCollector {
	return &EventCollector{
		events: make([]core.Event, 0, 100),
	}
}

// Record adds an event to the collector.
func (c *EventCollector) Record(event core.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, event)
}

// Events returns a copy of all recorded events.
func (c *EventCollector) Events() []core.Event {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]core.Event, len(c.events))
	copy(result, c.events)
	return result
}

// EventsByType returns events of a specific type.
func (c *EventCollector) EventsByType(eventType core.EventType) []core.Event {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]core.Event, 0)
	for _, e := range c.events {
		if e.Type == eventType {
			result = append(result, e)
		}
	}
	return result
}

// CountByType returns the count of events of a specific type.
func (c *EventCollector) CountByType(eventType core.EventType) int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	count := 0
	for _, e := range c.events {
		if e.Type == eventType {
			count++
		}
	}
	return count
}

// Count returns the total number of recorded events.
func (c *EventCollector) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.events)
}

// Clear removes all recorded events.
func (c *EventCollector) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = c.events[:0]
}

// ContainsSequence checks if events of the given types appear in order.
func (c *EventCollector) ContainsSequence(types ...core.EventType) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	typeIdx := 0
	for _, e := range c.events {
		if e.Type == types[typeIdx] {
			typeIdx++
			if typeIdx == len(types) {
				return true
			}
		}
	}
	return false
}

// WaitForCount waits until the event count reaches the expected value.
func (c *EventCollector) WaitForCount(expected int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if c.Count() >= expected {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// TimeoutWrapper wraps a test function with a timeout.
func TimeoutWrapper(t *testing.T, timeout time.Duration, fn func(t *testing.T)) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		defer close(done)
		fn(t)
	}()

	select {
	case <-done:
		// Test completed
	case <-time.After(timeout):
		t.Fatalf("test timed out after %v", timeout)
	}
}

// Common test fixtures

// FixtureSimpleConversation returns agents for a simple 2-agent conversation.
func FixtureSimpleConversation() []core.Agent {
	return []core.Agent{
		core.NewAgent("alice", "mock", "Alice", "model-a", "mock").
			WithSystemPrompt("You are Alice, a helpful assistant."),
		core.NewAgent("bob", "mock", "Bob", "model-b", "mock").
			WithSystemPrompt("You are Bob, a knowledgeable expert."),
	}
}

// FixtureLargeGroup returns agents for a large group conversation (5+ agents).
func FixtureLargeGroup() []core.Agent {
	agents := make([]core.Agent, 5)
	for i := 0; i < 5; i++ {
		agents[i] = core.NewAgent(
			AgentID(i),
			"mock",
			AgentName(i),
			"model",
			"mock",
		).WithSystemPrompt("You are " + AgentName(i))
	}
	return agents
}

// FixtureMixedReliability returns agents with different reliability profiles.
func FixtureMixedReliability() (agents []core.Agent, errors map[string]error) {
	agents = []core.Agent{
		core.NewAgent("reliable", "mock", "Reliable", "model", "mock"),
		core.NewAgent("flaky", "mock", "Flaky", "model", "mock"),
		core.NewAgent("slow", "mock", "Slow", "model", "mock"),
	}
	errors = map[string]error{
		"reliable": nil,
		"flaky":    ErrIntermittentFailure,
		"slow":     nil,
	}
	return
}

// Common test errors
var (
	ErrIntermittentFailure = errors.New("intermittent failure")
	ErrNetworkTimeout      = errors.New("network timeout")
	ErrRateLimit           = errors.New("rate limit exceeded")
	ErrAuthFailure         = errors.New("authentication failed")
)

// AssertEventSequence verifies that events appear in the expected order.
func AssertEventSequence(t *testing.T, collector *EventCollector, types ...core.EventType) {
	t.Helper()

	if !collector.ContainsSequence(types...) {
		events := collector.Events()
		t.Errorf("expected event sequence %v, got events:", types)
		for i, e := range events {
			t.Errorf("  [%d] %s", i, e.Type)
		}
	}
}

// AssertEventCount verifies the count of a specific event type.
func AssertEventCount(t *testing.T, collector *EventCollector, eventType core.EventType, expected int) {
	t.Helper()

	actual := collector.CountByType(eventType)
	if actual != expected {
		t.Errorf("expected %d %s events, got %d", expected, eventType, actual)
	}
}

// AssertNoErrors verifies that no error events were recorded.
func AssertNoErrors(t *testing.T, collector *EventCollector) {
	t.Helper()

	errorEvents := collector.EventsByType(core.EventAgentError)
	if len(errorEvents) > 0 {
		t.Errorf("expected no error events, got %d:", len(errorEvents))
		for _, e := range errorEvents {
			if data, ok := e.Data.(core.AgentErrorData); ok {
				t.Errorf("  - %s: %s", data.AgentName, data.Error)
			}
		}
	}
}

// AssertMessageCount verifies the total message count in the conversation.
func AssertMessageCount(t *testing.T, harness *TestHarness, expected int) {
	t.Helper()

	messages := harness.Manager.GetMessages()
	if len(messages) != expected {
		t.Errorf("expected %d messages, got %d", expected, len(messages))
	}
}
