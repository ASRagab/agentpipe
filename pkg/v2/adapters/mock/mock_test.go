package mock

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/v2/adapters"
	"github.com/kevinelliott/agentpipe/pkg/v2/core"
)

func TestNewMockAdapter(t *testing.T) {
	m := NewMockAdapter()

	if m == nil {
		t.Fatal("NewMockAdapter returned nil")
	}
	if m.Response != "Mock response" {
		t.Errorf("expected default response 'Mock response', got %q", m.Response)
	}
	if !m.Available {
		t.Error("expected Available to be true by default")
	}
	if m.Model != "mock-model" {
		t.Errorf("expected default model 'mock-model', got %q", m.Model)
	}
}

func TestMockAdapter_Initialize(t *testing.T) {
	m := NewMockAdapter()
	agent := core.Agent{
		ID:    "test-agent",
		Name:  "Test Agent",
		Model: "test-model",
	}

	err := m.Initialize(agent)
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	storedAgent := m.GetAgent()
	if storedAgent.ID != agent.ID {
		t.Errorf("expected agent ID %q, got %q", agent.ID, storedAgent.ID)
	}
	if storedAgent.Name != agent.Name {
		t.Errorf("expected agent Name %q, got %q", agent.Name, storedAgent.Name)
	}
}

func TestMockAdapter_SendMessage(t *testing.T) {
	m := NewMockAdapter()
	m.Response = "Hello from mock!"
	ctx := context.Background()

	messages := []core.Message{
		core.NewUserMessage("Hi there"),
	}

	response, metrics, err := m.SendMessage(ctx, messages)
	if err != nil {
		t.Fatalf("SendMessage returned error: %v", err)
	}

	if response != "Hello from mock!" {
		t.Errorf("expected response 'Hello from mock!', got %q", response)
	}
	if metrics == nil {
		t.Fatal("expected metrics, got nil")
	}
	if m.SendMessageCalls != 1 {
		t.Errorf("expected SendMessageCalls=1, got %d", m.SendMessageCalls)
	}
	if len(m.LastMessages) != 1 {
		t.Errorf("expected 1 message stored, got %d", len(m.LastMessages))
	}
}

func TestMockAdapter_SendMessage_WithDelay(t *testing.T) {
	m := NewMockAdapter()
	m.Delay = 50 * time.Millisecond
	ctx := context.Background()

	start := time.Now()
	_, _, err := m.SendMessage(ctx, nil)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("SendMessage returned error: %v", err)
	}
	if elapsed < 50*time.Millisecond {
		t.Errorf("expected at least 50ms delay, got %v", elapsed)
	}
}

func TestMockAdapter_SendMessage_WithError(t *testing.T) {
	m := NewMockAdapter()
	expectedErr := errors.New("mock error")
	m.Error = expectedErr
	ctx := context.Background()

	_, _, err := m.SendMessage(ctx, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != expectedErr.Error() {
		t.Errorf("expected error %q, got %q", expectedErr.Error(), err.Error())
	}
}

func TestMockAdapter_SendMessage_ContextCancellation(t *testing.T) {
	m := NewMockAdapter()
	m.Delay = 500 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	_, _, err := m.SendMessage(ctx, nil)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestMockAdapter_SendMessage_WithCustomMetrics(t *testing.T) {
	m := NewMockAdapter()
	m.Metrics = &core.Metrics{
		InputTokens:  100,
		OutputTokens: 200,
		TotalTokens:  300,
		Cost:         0.01,
		Model:        "custom-model",
	}
	ctx := context.Background()

	_, metrics, err := m.SendMessage(ctx, nil)
	if err != nil {
		t.Fatalf("SendMessage returned error: %v", err)
	}
	if metrics.TotalTokens != 300 {
		t.Errorf("expected TotalTokens=300, got %d", metrics.TotalTokens)
	}
	if metrics.Cost != 0.01 {
		t.Errorf("expected Cost=0.01, got %f", metrics.Cost)
	}
}

func TestMockAdapter_StreamMessage(t *testing.T) {
	m := NewMockAdapter()
	m.StreamChunks = []string{"Hello ", "world", "!"}
	ctx := context.Background()

	var buf bytes.Buffer
	metrics, err := m.StreamMessage(ctx, nil, &buf)

	if err != nil {
		t.Fatalf("StreamMessage returned error: %v", err)
	}
	if buf.String() != "Hello world!" {
		t.Errorf("expected 'Hello world!', got %q", buf.String())
	}
	if metrics == nil {
		t.Fatal("expected metrics, got nil")
	}
	if m.StreamMessageCalls != 1 {
		t.Errorf("expected StreamMessageCalls=1, got %d", m.StreamMessageCalls)
	}
}

func TestMockAdapter_StreamMessage_WithDelay(t *testing.T) {
	m := NewMockAdapter()
	m.StreamChunks = []string{"a", "b", "c"}
	m.StreamDelay = 30 * time.Millisecond
	ctx := context.Background()

	var buf bytes.Buffer
	start := time.Now()
	_, err := m.StreamMessage(ctx, nil, &buf)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("StreamMessage returned error: %v", err)
	}
	// 3 chunks with 30ms delay between first and second, second and third = 60ms
	if elapsed < 60*time.Millisecond {
		t.Errorf("expected at least 60ms for streaming, got %v", elapsed)
	}
}

func TestMockAdapter_StreamMessage_Fallback(t *testing.T) {
	m := NewMockAdapter()
	m.Response = "Fallback response"
	// Don't set StreamChunks - should fallback to Response
	ctx := context.Background()

	var buf bytes.Buffer
	_, err := m.StreamMessage(ctx, nil, &buf)

	if err != nil {
		t.Fatalf("StreamMessage returned error: %v", err)
	}
	if buf.String() != "Fallback response" {
		t.Errorf("expected 'Fallback response', got %q", buf.String())
	}
}

func TestMockAdapter_StreamMessage_ContextCancellation(t *testing.T) {
	m := NewMockAdapter()
	m.StreamChunks = []string{"a", "b", "c", "d", "e"}
	m.StreamDelay = 100 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after first chunk
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	var buf bytes.Buffer
	_, err := m.StreamMessage(ctx, nil, &buf)

	if err == nil {
		t.Fatal("expected context cancellation error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestMockAdapter_IsAvailable(t *testing.T) {
	m := NewMockAdapter()

	if !m.IsAvailable() {
		t.Error("expected IsAvailable to return true by default")
	}

	m.Available = false
	if m.IsAvailable() {
		t.Error("expected IsAvailable to return false when set")
	}
}

func TestMockAdapter_GetModel(t *testing.T) {
	m := NewMockAdapter()

	if m.GetModel() != "mock-model" {
		t.Errorf("expected 'mock-model', got %q", m.GetModel())
	}

	m.Model = "custom-model"
	if m.GetModel() != "custom-model" {
		t.Errorf("expected 'custom-model', got %q", m.GetModel())
	}
}

func TestMockAdapter_HealthCheck(t *testing.T) {
	m := NewMockAdapter()
	ctx := context.Background()

	err := m.HealthCheck(ctx)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	expectedErr := errors.New("health check failed")
	m.HealthCheckError = expectedErr
	err = m.HealthCheck(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != expectedErr.Error() {
		t.Errorf("expected error %q, got %q", expectedErr.Error(), err.Error())
	}
}

func TestMockAdapter_Reset(t *testing.T) {
	m := NewMockAdapter()
	ctx := context.Background()

	// Make some calls
	_, _, _ = m.SendMessage(ctx, []core.Message{core.NewUserMessage("test")})
	var buf bytes.Buffer
	_, _ = m.StreamMessage(ctx, nil, &buf)

	// Verify calls were tracked
	if m.SendMessageCalls != 1 {
		t.Errorf("expected SendMessageCalls=1, got %d", m.SendMessageCalls)
	}
	if m.StreamMessageCalls != 1 {
		t.Errorf("expected StreamMessageCalls=1, got %d", m.StreamMessageCalls)
	}

	// Reset
	m.Reset()

	if m.SendMessageCalls != 0 {
		t.Errorf("expected SendMessageCalls=0 after reset, got %d", m.SendMessageCalls)
	}
	if m.StreamMessageCalls != 0 {
		t.Errorf("expected StreamMessageCalls=0 after reset, got %d", m.StreamMessageCalls)
	}
	if m.LastMessages != nil {
		t.Error("expected LastMessages=nil after reset")
	}
}

func TestMockAdapter_RegisteredWithDefaultRegistry(t *testing.T) {
	// The mock adapter should be registered with the default registry via init()
	adapter, err := adapters.Get("mock")
	if err != nil {
		t.Fatalf("failed to get mock adapter from registry: %v", err)
	}
	if adapter == nil {
		t.Fatal("expected adapter, got nil")
	}

	// Verify it's a MockAdapter
	_, ok := adapter.(*MockAdapter)
	if !ok {
		t.Error("expected *MockAdapter from registry")
	}
}

func TestMockAdapter_ImplementsInterface(t *testing.T) {
	var _ adapters.AgentAdapter = (*MockAdapter)(nil)
}
