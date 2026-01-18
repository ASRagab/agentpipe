package adapters

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/ASRagab/agentpipe/pkg/v2/core"
	pkgerrors "github.com/ASRagab/agentpipe/pkg/v2/errors"
)

// mockAdapterForCircuitBreaker is a test adapter that can be configured to fail or succeed.
type mockAdapterForCircuitBreaker struct {
	available       bool
	model           string
	failCount       int
	currentFailures int
	responses       []string
	responseIndex   int
	sendDelay       time.Duration
}

func newMockAdapterForCircuitBreaker() *mockAdapterForCircuitBreaker {
	return &mockAdapterForCircuitBreaker{
		available: true,
		model:     "test-model",
		responses: []string{"success response"},
	}
}

func (m *mockAdapterForCircuitBreaker) Initialize(_ core.Agent) error {
	return nil
}

func (m *mockAdapterForCircuitBreaker) IsAvailable() bool {
	return m.available
}

func (m *mockAdapterForCircuitBreaker) GetModel() string {
	return m.model
}

func (m *mockAdapterForCircuitBreaker) HealthCheck(_ context.Context) error {
	return nil
}

func (m *mockAdapterForCircuitBreaker) SendMessage(ctx context.Context, _ []core.Message) (string, *core.Metrics, error) {
	if m.sendDelay > 0 {
		select {
		case <-time.After(m.sendDelay):
		case <-ctx.Done():
			return "", nil, ctx.Err()
		}
	}

	if m.currentFailures < m.failCount {
		m.currentFailures++
		return "", nil, pkgerrors.NewNetworkError("test-agent", "Test Agent", errors.New("simulated failure"))
	}

	response := "success"
	if m.responseIndex < len(m.responses) {
		response = m.responses[m.responseIndex]
		m.responseIndex++
	}
	return response, &core.Metrics{Duration: time.Millisecond * 100}, nil
}

func (m *mockAdapterForCircuitBreaker) StreamMessage(ctx context.Context, messages []core.Message, writer io.Writer) (*core.Metrics, error) {
	response, metrics, err := m.SendMessage(ctx, messages)
	if err != nil {
		return nil, err
	}
	_, writeErr := writer.Write([]byte(response))
	return metrics, writeErr
}

// TestCircuitBreakerState tests the string representation of circuit states.
func TestCircuitBreakerState(t *testing.T) {
	tests := []struct {
		state    CircuitState
		expected string
	}{
		{CircuitClosed, "closed"},
		{CircuitOpen, "open"},
		{CircuitHalfOpen, "half-open"},
		{CircuitState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("CircuitState.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestDefaultCircuitBreakerConfig tests the default configuration values.
func TestDefaultCircuitBreakerConfig(t *testing.T) {
	config := DefaultCircuitBreakerConfig()

	if config.FailureThreshold != 5 {
		t.Errorf("FailureThreshold = %d, want 5", config.FailureThreshold)
	}
	if config.CooldownPeriod != 30*time.Second {
		t.Errorf("CooldownPeriod = %v, want 30s", config.CooldownPeriod)
	}
	if config.SuccessThreshold != 1 {
		t.Errorf("SuccessThreshold = %d, want 1", config.SuccessThreshold)
	}
}

// TestCircuitBreakerInitialState tests that circuit breaker starts closed.
func TestCircuitBreakerInitialState(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig(), "test-agent", "Test Agent")

	if cb.State() != CircuitClosed {
		t.Errorf("Initial state = %v, want closed", cb.State())
	}
	if cb.FailureCount() != 0 {
		t.Errorf("Initial failure count = %d, want 0", cb.FailureCount())
	}
	if !cb.AllowRequest() {
		t.Error("Expected AllowRequest() to return true for closed circuit")
	}
}

// TestCircuitBreakerOpensAfterFailures tests that circuit opens after threshold.
func TestCircuitBreakerOpensAfterFailures(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 3,
		CooldownPeriod:   30 * time.Second,
		SuccessThreshold: 1,
	}
	cb := NewCircuitBreaker(config, "test-agent", "Test Agent")

	// Record failures up to threshold
	for i := 0; i < 3; i++ {
		if cb.State() != CircuitClosed {
			t.Errorf("State after %d failures = %v, want closed", i, cb.State())
		}
		cb.RecordFailure()
	}

	// Circuit should now be open
	if cb.State() != CircuitOpen {
		t.Errorf("State after threshold failures = %v, want open", cb.State())
	}
	if cb.FailureCount() != 3 {
		t.Errorf("Failure count = %d, want 3", cb.FailureCount())
	}
	if cb.AllowRequest() {
		t.Error("Expected AllowRequest() to return false for open circuit")
	}
}

// TestCircuitBreakerSuccessResets tests that success resets failure count.
func TestCircuitBreakerSuccessResets(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 3,
		CooldownPeriod:   30 * time.Second,
		SuccessThreshold: 1,
	}
	cb := NewCircuitBreaker(config, "test-agent", "Test Agent")

	// Record some failures (but not enough to open)
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.FailureCount() != 2 {
		t.Errorf("Failure count = %d, want 2", cb.FailureCount())
	}

	// Success should reset failure count
	cb.RecordSuccess()

	if cb.FailureCount() != 0 {
		t.Errorf("Failure count after success = %d, want 0", cb.FailureCount())
	}
	if cb.State() != CircuitClosed {
		t.Errorf("State = %v, want closed", cb.State())
	}
}

// TestCircuitBreakerHalfOpenAfterCooldown tests half-open transition.
func TestCircuitBreakerHalfOpenAfterCooldown(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 1,
		CooldownPeriod:   50 * time.Millisecond,
		SuccessThreshold: 1,
	}
	cb := NewCircuitBreaker(config, "test-agent", "Test Agent")

	// Open the circuit
	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Fatalf("State = %v, want open", cb.State())
	}

	// Wait for cooldown
	time.Sleep(100 * time.Millisecond)

	// AllowRequest should transition to half-open
	if !cb.AllowRequest() {
		t.Error("Expected AllowRequest() to return true after cooldown")
	}
	if cb.State() != CircuitHalfOpen {
		t.Errorf("State = %v, want half-open", cb.State())
	}
}

// TestCircuitBreakerClosesAfterSuccessInHalfOpen tests recovery.
func TestCircuitBreakerClosesAfterSuccessInHalfOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 1,
		CooldownPeriod:   10 * time.Millisecond,
		SuccessThreshold: 2, // Require 2 successes
	}
	cb := NewCircuitBreaker(config, "test-agent", "Test Agent")

	// Open and wait for half-open
	cb.RecordFailure()
	time.Sleep(20 * time.Millisecond)
	cb.AllowRequest() // Triggers transition to half-open

	if cb.State() != CircuitHalfOpen {
		t.Fatalf("State = %v, want half-open", cb.State())
	}

	// First success
	cb.RecordSuccess()
	if cb.State() != CircuitHalfOpen {
		t.Errorf("State after 1 success = %v, want half-open (need 2)", cb.State())
	}

	// Second success should close
	cb.RecordSuccess()
	if cb.State() != CircuitClosed {
		t.Errorf("State after 2 successes = %v, want closed", cb.State())
	}
}

// TestCircuitBreakerReopensOnFailureInHalfOpen tests immediate re-open.
func TestCircuitBreakerReopensOnFailureInHalfOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 1,
		CooldownPeriod:   10 * time.Millisecond,
		SuccessThreshold: 1,
	}
	cb := NewCircuitBreaker(config, "test-agent", "Test Agent")

	// Open and wait for half-open
	cb.RecordFailure()
	time.Sleep(20 * time.Millisecond)
	cb.AllowRequest()

	if cb.State() != CircuitHalfOpen {
		t.Fatalf("State = %v, want half-open", cb.State())
	}

	// Failure in half-open should immediately open
	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Errorf("State after failure in half-open = %v, want open", cb.State())
	}
}

// TestCircuitBreakerReset tests manual reset.
func TestCircuitBreakerReset(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 1,
		CooldownPeriod:   30 * time.Second,
		SuccessThreshold: 1,
	}
	cb := NewCircuitBreaker(config, "test-agent", "Test Agent")

	// Open the circuit
	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Fatalf("State = %v, want open", cb.State())
	}

	// Reset should close it
	cb.Reset()
	if cb.State() != CircuitClosed {
		t.Errorf("State after reset = %v, want closed", cb.State())
	}
	if cb.FailureCount() != 0 {
		t.Errorf("Failure count after reset = %d, want 0", cb.FailureCount())
	}
	if !cb.AllowRequest() {
		t.Error("Expected AllowRequest() to return true after reset")
	}
}

// TestCircuitBreakerTimeUntilRetry tests retry timing calculation.
func TestCircuitBreakerTimeUntilRetry(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 1,
		CooldownPeriod:   100 * time.Millisecond,
		SuccessThreshold: 1,
	}
	cb := NewCircuitBreaker(config, "test-agent", "Test Agent")

	// Closed circuit should return 0
	if cb.TimeUntilRetry() != 0 {
		t.Errorf("TimeUntilRetry for closed = %v, want 0", cb.TimeUntilRetry())
	}

	// Open the circuit
	cb.RecordFailure()

	// Should return remaining cooldown time
	remaining := cb.TimeUntilRetry()
	if remaining < 50*time.Millisecond || remaining > 100*time.Millisecond {
		t.Errorf("TimeUntilRetry = %v, want between 50ms and 100ms", remaining)
	}

	// Wait for cooldown
	time.Sleep(110 * time.Millisecond)
	if cb.TimeUntilRetry() != 0 {
		t.Errorf("TimeUntilRetry after cooldown = %v, want 0", cb.TimeUntilRetry())
	}
}

// TestCircuitBreakerStateChangeCallback tests the callback mechanism.
func TestCircuitBreakerStateChangeCallback(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 1,
		CooldownPeriod:   10 * time.Millisecond,
		SuccessThreshold: 1,
	}
	cb := NewCircuitBreaker(config, "test-agent", "Test Agent")

	var transitions []struct {
		old, new CircuitState
	}

	cb.OnStateChange(func(old, new CircuitState) {
		transitions = append(transitions, struct{ old, new CircuitState }{old, new})
	})

	// Trigger transitions: closed -> open -> half-open -> closed
	cb.RecordFailure() // closed -> open
	time.Sleep(20 * time.Millisecond)
	cb.AllowRequest()  // open -> half-open
	cb.RecordSuccess() // half-open -> closed

	if len(transitions) != 3 {
		t.Errorf("Got %d transitions, want 3", len(transitions))
	}

	expected := []struct{ old, new CircuitState }{
		{CircuitClosed, CircuitOpen},
		{CircuitOpen, CircuitHalfOpen},
		{CircuitHalfOpen, CircuitClosed},
	}

	for i, trans := range transitions {
		if trans.old != expected[i].old || trans.new != expected[i].new {
			t.Errorf("Transition %d: got %v->%v, want %v->%v",
				i, trans.old, trans.new, expected[i].old, expected[i].new)
		}
	}
}

// TestCircuitBreakerConcurrency tests thread safety.
func TestCircuitBreakerConcurrency(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold: 100,
		CooldownPeriod:   30 * time.Second,
		SuccessThreshold: 10,
	}
	cb := NewCircuitBreaker(config, "test-agent", "Test Agent")

	var wg sync.WaitGroup
	operations := 100

	// Concurrent successes
	for i := 0; i < operations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cb.RecordSuccess()
		}()
	}

	// Concurrent failures
	for i := 0; i < operations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cb.RecordFailure()
		}()
	}

	// Concurrent state checks
	for i := 0; i < operations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = cb.AllowRequest()
			_ = cb.State()
			_ = cb.FailureCount()
		}()
	}

	wg.Wait()

	// Should not panic and state should be valid
	state := cb.State()
	if state != CircuitClosed && state != CircuitOpen && state != CircuitHalfOpen {
		t.Errorf("Invalid state after concurrent operations: %v", state)
	}
}

// TestCircuitBreakerAdapterSendMessage tests the adapter wrapper.
func TestCircuitBreakerAdapterSendMessage(t *testing.T) {
	mock := newMockAdapterForCircuitBreaker()
	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		CooldownPeriod:   10 * time.Millisecond,
		SuccessThreshold: 1,
	}
	adapter := NewCircuitBreakerAdapter(mock, config, "test-agent", "Test Agent")

	ctx := context.Background()
	messages := []core.Message{}

	// Successful request
	response, metrics, err := adapter.SendMessage(ctx, messages)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if response != "success response" {
		t.Errorf("Response = %q, want %q", response, "success response")
	}
	if metrics == nil {
		t.Error("Expected metrics, got nil")
	}
}

// TestCircuitBreakerAdapterOpens tests circuit opens after failures.
func TestCircuitBreakerAdapterOpens(t *testing.T) {
	mock := newMockAdapterForCircuitBreaker()
	mock.failCount = 10 // Always fail
	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		CooldownPeriod:   1 * time.Second,
		SuccessThreshold: 1,
	}
	adapter := NewCircuitBreakerAdapter(mock, config, "test-agent", "Test Agent")

	ctx := context.Background()
	messages := []core.Message{}

	// First two requests should fail normally
	for i := 0; i < 2; i++ {
		_, _, err := adapter.SendMessage(ctx, messages)
		if err == nil {
			t.Errorf("Request %d: expected error", i+1)
		}
	}

	// Circuit should now be open
	if adapter.CircuitBreaker().State() != CircuitOpen {
		t.Errorf("Circuit state = %v, want open", adapter.CircuitBreaker().State())
	}

	// Next request should fail with circuit open error
	_, _, err := adapter.SendMessage(ctx, messages)
	if err == nil {
		t.Error("Expected error for open circuit")
	}
	if !IsCircuitOpenError(err) {
		t.Errorf("Expected circuit open error, got: %v", err)
	}
}

// TestCircuitBreakerAdapterRecovery tests circuit recovery.
func TestCircuitBreakerAdapterRecovery(t *testing.T) {
	mock := newMockAdapterForCircuitBreaker()
	mock.failCount = 2 // Fail first 2, then succeed
	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		CooldownPeriod:   10 * time.Millisecond,
		SuccessThreshold: 1,
	}
	adapter := NewCircuitBreakerAdapter(mock, config, "test-agent", "Test Agent")

	ctx := context.Background()
	messages := []core.Message{}

	// Trigger circuit open
	for i := 0; i < 2; i++ {
		_, _, _ = adapter.SendMessage(ctx, messages)
	}

	if adapter.CircuitBreaker().State() != CircuitOpen {
		t.Fatalf("Circuit state = %v, want open", adapter.CircuitBreaker().State())
	}

	// Wait for cooldown
	time.Sleep(20 * time.Millisecond)

	// Next request should succeed (mock is now past its fail count)
	response, _, err := adapter.SendMessage(ctx, messages)
	if err != nil {
		t.Errorf("Unexpected error after recovery: %v", err)
	}
	if response != "success response" {
		t.Errorf("Response = %q, want %q", response, "success response")
	}

	// Circuit should be closed
	if adapter.CircuitBreaker().State() != CircuitClosed {
		t.Errorf("Circuit state = %v, want closed", adapter.CircuitBreaker().State())
	}
}

// TestCircuitBreakerAdapterStreamMessage tests streaming with circuit breaker.
func TestCircuitBreakerAdapterStreamMessage(t *testing.T) {
	mock := newMockAdapterForCircuitBreaker()
	mock.responses = []string{"streamed content"}
	config := DefaultCircuitBreakerConfig()
	adapter := NewCircuitBreakerAdapter(mock, config, "test-agent", "Test Agent")

	ctx := context.Background()
	messages := []core.Message{}
	var buf bytes.Buffer

	metrics, err := adapter.StreamMessage(ctx, messages, &buf)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if metrics == nil {
		t.Error("Expected metrics, got nil")
	}
	if buf.String() != "streamed content" {
		t.Errorf("Streamed content = %q, want %q", buf.String(), "streamed content")
	}
}

// TestCircuitBreakerAdapterIsAvailable tests availability checking.
func TestCircuitBreakerAdapterIsAvailable(t *testing.T) {
	mock := newMockAdapterForCircuitBreaker()
	mock.failCount = 10
	config := CircuitBreakerConfig{
		FailureThreshold: 1,
		CooldownPeriod:   1 * time.Second,
		SuccessThreshold: 1,
	}
	adapter := NewCircuitBreakerAdapter(mock, config, "test-agent", "Test Agent")

	// Initially available
	if !adapter.IsAvailable() {
		t.Error("Expected adapter to be available initially")
	}

	// Open the circuit
	ctx := context.Background()
	_, _, _ = adapter.SendMessage(ctx, []core.Message{})

	// Should now be unavailable (circuit open)
	if adapter.IsAvailable() {
		t.Error("Expected adapter to be unavailable with open circuit")
	}

	// Make underlying adapter unavailable
	adapter.CircuitBreaker().Reset()
	mock.available = false
	if adapter.IsAvailable() {
		t.Error("Expected adapter to be unavailable when underlying is unavailable")
	}
}

// TestCircuitBreakerAdapterGetModel tests model delegation.
func TestCircuitBreakerAdapterGetModel(t *testing.T) {
	mock := newMockAdapterForCircuitBreaker()
	mock.model = "claude-3-opus"
	adapter := NewCircuitBreakerAdapter(mock, DefaultCircuitBreakerConfig(), "test-agent", "Test Agent")

	if adapter.GetModel() != "claude-3-opus" {
		t.Errorf("GetModel() = %q, want %q", adapter.GetModel(), "claude-3-opus")
	}
}

// TestCircuitBreakerAdapterHealthCheck tests health check delegation.
func TestCircuitBreakerAdapterHealthCheck(t *testing.T) {
	mock := newMockAdapterForCircuitBreaker()
	adapter := NewCircuitBreakerAdapter(mock, DefaultCircuitBreakerConfig(), "test-agent", "Test Agent")

	ctx := context.Background()
	if err := adapter.HealthCheck(ctx); err != nil {
		t.Errorf("HealthCheck() error = %v", err)
	}
}

// TestCircuitBreakerAdapterInitialize tests initialization delegation.
func TestCircuitBreakerAdapterInitialize(t *testing.T) {
	mock := newMockAdapterForCircuitBreaker()
	adapter := NewCircuitBreakerAdapter(mock, DefaultCircuitBreakerConfig(), "test-agent", "Test Agent")

	if err := adapter.Initialize(core.Agent{}); err != nil {
		t.Errorf("Initialize() error = %v", err)
	}
}

// TestIsCircuitOpenError tests error type checking.
func TestIsCircuitOpenError(t *testing.T) {
	// Circuit open error
	openErr := pkgerrors.NewAgentError("test", "Test", pkgerrors.ErrTypeNetwork, "circuit breaker open", nil)
	if !IsCircuitOpenError(openErr) {
		t.Error("Expected IsCircuitOpenError to return true for circuit open error")
	}

	// Other error
	otherErr := pkgerrors.NewNetworkError("test", "Test", errors.New("network down"))
	if IsCircuitOpenError(otherErr) {
		t.Error("Expected IsCircuitOpenError to return false for other errors")
	}

	// Non-AgentError
	if IsCircuitOpenError(errors.New("random error")) {
		t.Error("Expected IsCircuitOpenError to return false for non-AgentError")
	}
}
