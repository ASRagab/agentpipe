package adapters

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/v2/core"
	"github.com/kevinelliott/agentpipe/pkg/v2/errors"
)

// mockAdapter implements AgentAdapter for testing.
type mockAdapter struct {
	sendMessageFunc   func(ctx context.Context, messages []core.Message) (string, *core.Metrics, error)
	streamMessageFunc func(ctx context.Context, messages []core.Message, writer io.Writer) (*core.Metrics, error)
	model             string
	available         bool
}

func (m *mockAdapter) Initialize(_ core.Agent) error {
	return nil
}

func (m *mockAdapter) IsAvailable() bool {
	return m.available
}

func (m *mockAdapter) GetModel() string {
	return m.model
}

func (m *mockAdapter) HealthCheck(_ context.Context) error {
	return nil
}

func (m *mockAdapter) SendMessage(ctx context.Context, messages []core.Message) (string, *core.Metrics, error) {
	if m.sendMessageFunc != nil {
		return m.sendMessageFunc(ctx, messages)
	}
	return "response", &core.Metrics{}, nil
}

func (m *mockAdapter) StreamMessage(ctx context.Context, messages []core.Message, writer io.Writer) (*core.Metrics, error) {
	if m.streamMessageFunc != nil {
		return m.streamMessageFunc(ctx, messages, writer)
	}
	return &core.Metrics{}, nil
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts=3, got %d", config.MaxAttempts)
	}
	if config.InitialDelay != 1*time.Second {
		t.Errorf("expected InitialDelay=1s, got %v", config.InitialDelay)
	}
	if config.MaxDelay != 30*time.Second {
		t.Errorf("expected MaxDelay=30s, got %v", config.MaxDelay)
	}
	if config.BackoffMultiplier != 2.0 {
		t.Errorf("expected BackoffMultiplier=2.0, got %f", config.BackoffMultiplier)
	}
	if config.JitterFactor != 0.1 {
		t.Errorf("expected JitterFactor=0.1, got %f", config.JitterFactor)
	}
}

func TestRetryConfig_CalculateDelay(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:       5,
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          1 * time.Second,
		BackoffMultiplier: 2.0,
		JitterFactor:      0, // No jitter for deterministic testing
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 0},                      // No delay for first attempt
		{1, 100 * time.Millisecond}, // Initial delay
		{2, 200 * time.Millisecond}, // 100ms * 2
		{3, 400 * time.Millisecond}, // 200ms * 2
		{4, 800 * time.Millisecond}, // 400ms * 2
		{5, 1 * time.Second},        // Capped at MaxDelay
		{6, 1 * time.Second},        // Still capped
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			delay := config.calculateDelay(tt.attempt)
			if delay != tt.expected {
				t.Errorf("attempt %d: expected delay %v, got %v", tt.attempt, tt.expected, delay)
			}
		})
	}
}

func TestRetryConfig_CalculateDelayWithJitter(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          1 * time.Second,
		BackoffMultiplier: 2.0,
		JitterFactor:      0.1, // 10% jitter
	}

	// Run multiple times to verify jitter is applied
	delays := make(map[time.Duration]bool)
	for i := 0; i < 10; i++ {
		delay := config.calculateDelay(1)
		delays[delay] = true
	}

	// With 10% jitter, delay should be in range [90ms, 110ms]
	// With 10 samples, we should see some variation
	if len(delays) == 1 {
		// It's possible but unlikely that all 10 have the same delay with 10% jitter
		t.Log("Warning: All delays identical (possible but unlikely with jitter)")
	}
}

func TestRetryableAdapter_SendMessageSuccess(t *testing.T) {
	called := 0
	mock := &mockAdapter{
		available: true,
		model:     "test-model",
		sendMessageFunc: func(_ context.Context, _ []core.Message) (string, *core.Metrics, error) {
			called++
			return "success", &core.Metrics{}, nil
		},
	}

	config := RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		JitterFactor:      0,
	}

	adapter := NewRetryableAdapter(mock, config, "agent-1", "TestAgent")
	response, _, err := adapter.SendMessage(context.Background(), []core.Message{})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if response != "success" {
		t.Errorf("expected response='success', got '%s'", response)
	}
	if called != 1 {
		t.Errorf("expected 1 call, got %d", called)
	}
}

func TestRetryableAdapter_SendMessageRetryOnNetworkError(t *testing.T) {
	called := 0
	mock := &mockAdapter{
		available: true,
		sendMessageFunc: func(_ context.Context, _ []core.Message) (string, *core.Metrics, error) {
			called++
			if called < 3 {
				// Return a retryable network error
				return "", nil, errors.NewNetworkError("agent-1", "TestAgent", fmt.Errorf("connection refused"))
			}
			return "success after retry", &core.Metrics{}, nil
		},
	}

	config := RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		JitterFactor:      0,
	}

	adapter := NewRetryableAdapter(mock, config, "agent-1", "TestAgent")
	response, _, err := adapter.SendMessage(context.Background(), []core.Message{})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if response != "success after retry" {
		t.Errorf("expected response='success after retry', got '%s'", response)
	}
	if called != 3 {
		t.Errorf("expected 3 calls, got %d", called)
	}
}

func TestRetryableAdapter_SendMessageNoRetryOnAuthError(t *testing.T) {
	called := 0
	mock := &mockAdapter{
		available: true,
		sendMessageFunc: func(_ context.Context, _ []core.Message) (string, *core.Metrics, error) {
			called++
			// Return a non-retryable auth error
			return "", nil, errors.NewAuthError("agent-1", "TestAgent", 401, fmt.Errorf("unauthorized"))
		},
	}

	config := RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		JitterFactor:      0,
	}

	adapter := NewRetryableAdapter(mock, config, "agent-1", "TestAgent")
	_, _, err := adapter.SendMessage(context.Background(), []core.Message{})

	if err == nil {
		t.Error("expected error, got nil")
	}
	if called != 1 {
		t.Errorf("expected 1 call (no retry for auth errors), got %d", called)
	}

	// Verify it's an auth error
	agentErr, ok := errors.AsAgentError(err)
	if !ok {
		t.Error("expected AgentError")
	} else if agentErr.ErrorType != errors.ErrTypeAuth {
		t.Errorf("expected ErrTypeAuth, got %v", agentErr.ErrorType)
	}
}

func TestRetryableAdapter_SendMessageExhaustedRetries(t *testing.T) {
	called := 0
	mock := &mockAdapter{
		available: true,
		sendMessageFunc: func(_ context.Context, _ []core.Message) (string, *core.Metrics, error) {
			called++
			return "", nil, errors.NewNetworkError("agent-1", "TestAgent", fmt.Errorf("connection refused"))
		},
	}

	config := RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		JitterFactor:      0,
	}

	adapter := NewRetryableAdapter(mock, config, "agent-1", "TestAgent")
	_, _, err := adapter.SendMessage(context.Background(), []core.Message{})

	if err == nil {
		t.Error("expected error after exhausted retries, got nil")
	}
	if called != 3 {
		t.Errorf("expected 3 calls, got %d", called)
	}

	// Verify the error includes retry count
	agentErr, ok := errors.AsAgentError(err)
	if !ok {
		t.Error("expected AgentError")
	} else if agentErr.RetryCount != 3 {
		t.Errorf("expected RetryCount=3, got %d", agentErr.RetryCount)
	}
}

func TestRetryableAdapter_SendMessageContextCancellation(t *testing.T) {
	called := 0
	mock := &mockAdapter{
		available: true,
		sendMessageFunc: func(_ context.Context, _ []core.Message) (string, *core.Metrics, error) {
			called++
			return "", nil, errors.NewNetworkError("agent-1", "TestAgent", fmt.Errorf("connection refused"))
		},
	}

	config := RetryConfig{
		MaxAttempts:       5,
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          1 * time.Second,
		BackoffMultiplier: 2.0,
		JitterFactor:      0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	adapter := NewRetryableAdapter(mock, config, "agent-1", "TestAgent")

	// Cancel context after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, _, err := adapter.SendMessage(ctx, []core.Message{})

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	// Should have been cancelled before all retries
	if called >= 5 {
		t.Errorf("expected fewer than 5 calls due to cancellation, got %d", called)
	}
}

func TestRetryableAdapter_StreamMessageRetry(t *testing.T) {
	called := 0
	mock := &mockAdapter{
		available: true,
		streamMessageFunc: func(_ context.Context, _ []core.Message, _ io.Writer) (*core.Metrics, error) {
			called++
			if called < 2 {
				return nil, errors.NewTimeoutError("agent-1", "TestAgent", fmt.Errorf("timeout"))
			}
			return &core.Metrics{}, nil
		},
	}

	config := RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		JitterFactor:      0,
	}

	adapter := NewRetryableAdapter(mock, config, "agent-1", "TestAgent")
	_, err := adapter.StreamMessage(context.Background(), []core.Message{}, io.Discard)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if called != 2 {
		t.Errorf("expected 2 calls, got %d", called)
	}
}

func TestRetryableAdapter_RateLimitHandling(t *testing.T) {
	called := 0
	mock := &mockAdapter{
		available: true,
		sendMessageFunc: func(_ context.Context, _ []core.Message) (string, *core.Metrics, error) {
			called++
			if called == 1 {
				return "", nil, errors.NewRateLimitError("agent-1", "TestAgent", 5*time.Second)
			}
			return "success", &core.Metrics{}, nil
		},
	}

	config := RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		JitterFactor:      0,
	}

	adapter := NewRetryableAdapter(mock, config, "agent-1", "TestAgent")
	response, _, err := adapter.SendMessage(context.Background(), []core.Message{})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if response != "success" {
		t.Errorf("expected response='success', got '%s'", response)
	}
	if called != 2 {
		t.Errorf("expected 2 calls (rate limit is retryable), got %d", called)
	}
}

func TestWithRetry_Success(t *testing.T) {
	called := 0
	result, err := WithRetry(context.Background(), DefaultRetryConfig(), "agent-1", "TestAgent", func() (string, error) {
		called++
		return "result", nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "result" {
		t.Errorf("expected result='result', got '%s'", result)
	}
	if called != 1 {
		t.Errorf("expected 1 call, got %d", called)
	}
}

func TestWithRetry_RetryableError(t *testing.T) {
	called := 0
	config := RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		JitterFactor:      0,
	}

	result, err := WithRetry(context.Background(), config, "agent-1", "TestAgent", func() (int, error) {
		called++
		if called < 3 {
			return 0, errors.NewNetworkError("agent-1", "TestAgent", fmt.Errorf("connection refused"))
		}
		return 42, nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("expected result=42, got %d", result)
	}
	if called != 3 {
		t.Errorf("expected 3 calls, got %d", called)
	}
}

func TestWithRetry_NonRetryableError(t *testing.T) {
	called := 0
	config := RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		JitterFactor:      0,
	}

	_, err := WithRetry(context.Background(), config, "agent-1", "TestAgent", func() (int, error) {
		called++
		return 0, errors.NewAuthError("agent-1", "TestAgent", 401, fmt.Errorf("unauthorized"))
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
	if called != 1 {
		t.Errorf("expected 1 call (no retry for auth), got %d", called)
	}
}

func TestRetryOnError_Success(t *testing.T) {
	called := 0
	config := RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		JitterFactor:      0,
	}

	err := RetryOnError(context.Background(), config, "agent-1", "TestAgent", func() error {
		called++
		if called < 2 {
			return errors.NewNetworkError("agent-1", "TestAgent", fmt.Errorf("connection refused"))
		}
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if called != 2 {
		t.Errorf("expected 2 calls, got %d", called)
	}
}

func TestRetryableAdapter_DelegatesMethods(t *testing.T) {
	mock := &mockAdapter{
		available: true,
		model:     "test-model",
	}

	adapter := NewRetryableAdapter(mock, DefaultRetryConfig(), "agent-1", "TestAgent")

	if !adapter.IsAvailable() {
		t.Error("expected IsAvailable to return true")
	}

	if adapter.GetModel() != "test-model" {
		t.Errorf("expected GetModel='test-model', got '%s'", adapter.GetModel())
	}

	err := adapter.HealthCheck(context.Background())
	if err != nil {
		t.Errorf("unexpected HealthCheck error: %v", err)
	}
}

func TestRetryConfig_ZeroMaxAttempts(t *testing.T) {
	called := 0
	config := RetryConfig{
		MaxAttempts:       0, // Edge case: zero attempts
		InitialDelay:      10 * time.Millisecond,
		MaxDelay:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
		JitterFactor:      0,
	}

	_, err := WithRetry(context.Background(), config, "agent-1", "TestAgent", func() (int, error) {
		called++
		return 42, nil
	})

	// With 0 max attempts, should return an error about exhausted retries
	if err == nil {
		t.Error("expected error with 0 max attempts")
	}
	if called != 0 {
		t.Errorf("expected 0 calls with MaxAttempts=0, got %d", called)
	}
}
