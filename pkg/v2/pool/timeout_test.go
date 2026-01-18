package pool

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/v2/core"
	v2errors "github.com/kevinelliott/agentpipe/pkg/v2/errors"
	"github.com/kevinelliott/agentpipe/pkg/v2/events"
)

func TestDefaultTimeoutConfig(t *testing.T) {
	config := DefaultTimeoutConfig()

	if config.DefaultAgentTimeout != 30*time.Second {
		t.Errorf("DefaultAgentTimeout = %v, want %v", config.DefaultAgentTimeout, 30*time.Second)
	}
	if config.GlobalConversationTimeout != 120*time.Second {
		t.Errorf("GlobalConversationTimeout = %v, want %v", config.GlobalConversationTimeout, 120*time.Second)
	}
	if config.GracePeriod != 2*time.Second {
		t.Errorf("GracePeriod = %v, want %v", config.GracePeriod, 2*time.Second)
	}
	if !config.PreservePartialResponse {
		t.Error("PreservePartialResponse should be true by default")
	}
}

func TestTimeoutHandler_SetAndGetAgentTimeout(t *testing.T) {
	config := DefaultTimeoutConfig()
	handler := NewTimeoutHandler(config, nil)

	// Test default timeout
	timeout := handler.GetAgentTimeout("agent-1")
	if timeout != config.DefaultAgentTimeout {
		t.Errorf("GetAgentTimeout returned %v, want %v", timeout, config.DefaultAgentTimeout)
	}

	// Set specific timeout
	handler.SetAgentTimeout("agent-1", 45*time.Second)
	timeout = handler.GetAgentTimeout("agent-1")
	if timeout != 45*time.Second {
		t.Errorf("GetAgentTimeout returned %v, want %v", timeout, 45*time.Second)
	}

	// Other agent still gets default
	timeout = handler.GetAgentTimeout("agent-2")
	if timeout != config.DefaultAgentTimeout {
		t.Errorf("GetAgentTimeout for other agent returned %v, want %v", timeout, config.DefaultAgentTimeout)
	}
}

func TestTimeoutHandler_SetAgentTimeouts(t *testing.T) {
	config := DefaultTimeoutConfig()
	handler := NewTimeoutHandler(config, nil)

	configs := []AgentTimeoutConfig{
		{AgentID: "agent-1", Timeout: 10 * time.Second},
		{AgentID: "agent-2", Timeout: 20 * time.Second},
		{AgentID: "agent-3", Timeout: 30 * time.Second},
	}

	handler.SetAgentTimeouts(configs)

	if handler.GetAgentTimeout("agent-1") != 10*time.Second {
		t.Errorf("agent-1 timeout = %v, want %v", handler.GetAgentTimeout("agent-1"), 10*time.Second)
	}
	if handler.GetAgentTimeout("agent-2") != 20*time.Second {
		t.Errorf("agent-2 timeout = %v, want %v", handler.GetAgentTimeout("agent-2"), 20*time.Second)
	}
	if handler.GetAgentTimeout("agent-3") != 30*time.Second {
		t.Errorf("agent-3 timeout = %v, want %v", handler.GetAgentTimeout("agent-3"), 30*time.Second)
	}
}

func TestTimeoutHandler_CreateAgentContext(t *testing.T) {
	config := DefaultTimeoutConfig()
	handler := NewTimeoutHandler(config, nil)

	handler.SetAgentTimeout("agent-1", 5*time.Second)

	ctx := context.Background()
	agentCtx, cancel, duration := handler.CreateAgentContext(ctx, "agent-1")
	defer cancel()

	if duration != 5*time.Second {
		t.Errorf("duration = %v, want %v", duration, 5*time.Second)
	}

	// Context should have a deadline
	deadline, ok := agentCtx.Deadline()
	if !ok {
		t.Error("Context should have a deadline")
	}

	// Deadline should be roughly 5 seconds from now
	expectedDeadline := time.Now().Add(5 * time.Second)
	if deadline.Before(expectedDeadline.Add(-100*time.Millisecond)) || deadline.After(expectedDeadline.Add(100*time.Millisecond)) {
		t.Errorf("deadline %v is not within expected range around %v", deadline, expectedDeadline)
	}
}

func TestTimeoutHandler_CreateGlobalContext(t *testing.T) {
	config := DefaultTimeoutConfig()
	handler := NewTimeoutHandler(config, nil)

	ctx := context.Background()
	globalCtx, cancel := handler.CreateGlobalContext(ctx)
	defer cancel()

	// Context should have a deadline
	deadline, ok := globalCtx.Deadline()
	if !ok {
		t.Error("Context should have a deadline for global timeout")
	}

	// Deadline should be roughly 120 seconds from now
	expectedDeadline := time.Now().Add(120 * time.Second)
	if deadline.Before(expectedDeadline.Add(-100*time.Millisecond)) || deadline.After(expectedDeadline.Add(100*time.Millisecond)) {
		t.Errorf("deadline %v is not within expected range around %v", deadline, expectedDeadline)
	}
}

func TestTimeoutHandler_CreateGlobalContext_Disabled(t *testing.T) {
	config := DefaultTimeoutConfig()
	config.GlobalConversationTimeout = 0 // Disabled
	handler := NewTimeoutHandler(config, nil)

	ctx := context.Background()
	globalCtx, cancel := handler.CreateGlobalContext(ctx)
	defer cancel()

	// Context should NOT have a deadline when disabled
	_, ok := globalCtx.Deadline()
	if ok {
		t.Error("Context should not have a deadline when global timeout is disabled")
	}
}

func TestTimeoutError(t *testing.T) {
	err := &TimeoutError{
		AgentID:         "agent-1",
		AgentName:       "Test Agent",
		TimeoutDuration: 30 * time.Second,
		ElapsedTime:     30 * time.Second,
		PartialContent:  "partial response",
		IsGlobalTimeout: false,
	}

	// Test Error() message
	expected := "agent Test Agent timed out after 30s"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}

	// Test HasPartialContent
	if !err.HasPartialContent() {
		t.Error("HasPartialContent() should return true")
	}

	// Test ToAgentError
	agentErr := err.ToAgentError()
	if agentErr.ErrorType != v2errors.ErrTypeTimeout {
		t.Errorf("ErrorType = %v, want %v", agentErr.ErrorType, v2errors.ErrTypeTimeout)
	}
}

func TestTimeoutError_GlobalTimeout(t *testing.T) {
	err := &TimeoutError{
		AgentID:         "agent-1",
		AgentName:       "Test Agent",
		TimeoutDuration: 120 * time.Second,
		ElapsedTime:     120 * time.Second,
		IsGlobalTimeout: true,
	}

	// Test Error() message for global timeout
	expected := "global conversation timeout after 2m0s for agent Test Agent"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}

	// No partial content
	if err.HasPartialContent() {
		t.Error("HasPartialContent() should return false")
	}
}

func TestTimeoutHandler_ExecuteWithTimeout_Success(t *testing.T) {
	config := DefaultTimeoutConfig()
	config.DefaultAgentTimeout = 5 * time.Second
	handler := NewTimeoutHandler(config, nil)

	ctx := context.Background()
	result := handler.ExecuteWithTimeout(ctx, "agent-1", "Test Agent",
		func(ctx context.Context, partialChan chan<- string) (string, *core.Metrics, error) {
			return "success response", &core.Metrics{TotalTokens: 100}, nil
		})

	if !result.Success {
		t.Error("Expected success")
	}
	if result.TimedOut {
		t.Error("Should not have timed out")
	}
	if result.Content != "success response" {
		t.Errorf("Content = %q, want %q", result.Content, "success response")
	}
	if result.Metrics == nil || result.Metrics.TotalTokens != 100 {
		t.Error("Metrics not properly returned")
	}
	if result.Error != nil {
		t.Errorf("Unexpected error: %v", result.Error)
	}
}

func TestTimeoutHandler_ExecuteWithTimeout_Timeout(t *testing.T) {
	config := DefaultTimeoutConfig()
	config.DefaultAgentTimeout = 50 * time.Millisecond
	config.PreservePartialResponse = true
	eventBus := events.NewBus()
	handler := NewTimeoutHandler(config, eventBus)

	// Track emitted events
	var receivedEvents []core.Event
	var eventMu sync.Mutex
	eventBus.Subscribe(core.EventAgentError, func(event core.Event) {
		eventMu.Lock()
		receivedEvents = append(receivedEvents, event)
		eventMu.Unlock()
	})

	ctx := context.Background()
	result := handler.ExecuteWithTimeout(ctx, "agent-1", "Test Agent",
		func(ctx context.Context, partialChan chan<- string) (string, *core.Metrics, error) {
			// Send some partial content
			partialChan <- "partial "
			partialChan <- "content"

			// Wait longer than timeout
			select {
			case <-ctx.Done():
				return "", nil, ctx.Err()
			case <-time.After(200 * time.Millisecond):
				return "complete", nil, nil
			}
		})

	if result.Success {
		t.Error("Should not be successful on timeout")
	}
	if !result.TimedOut {
		t.Error("Should have timed out")
	}
	if result.Error == nil {
		t.Error("Expected timeout error")
	}

	// Check partial content was preserved
	if result.PartialContent != "partial content" {
		t.Errorf("PartialContent = %q, want %q", result.PartialContent, "partial content")
	}

	// Wait for event to be processed
	time.Sleep(10 * time.Millisecond)

	// Check event was emitted
	eventMu.Lock()
	if len(receivedEvents) == 0 {
		t.Error("Expected EventAgentError to be emitted")
	}
	eventMu.Unlock()
}

func TestTimeoutHandler_ExecuteWithTimeout_NonRetryableError(t *testing.T) {
	config := DefaultTimeoutConfig()
	handler := NewTimeoutHandler(config, nil)

	ctx := context.Background()
	expectedErr := errors.New("authentication failed")
	result := handler.ExecuteWithTimeout(ctx, "agent-1", "Test Agent",
		func(ctx context.Context, partialChan chan<- string) (string, *core.Metrics, error) {
			return "", nil, expectedErr
		})

	if result.Success {
		t.Error("Should not be successful on error")
	}
	if result.TimedOut {
		t.Error("Should not be marked as timeout for non-timeout error")
	}
	if result.Error == nil {
		t.Error("Expected error")
	}
}

func TestTimeoutStats(t *testing.T) {
	stats := NewTimeoutStats()

	// Record some timeouts
	stats.RecordTimeout("agent-1", 30*time.Second, false, false)
	stats.RecordTimeout("agent-1", 25*time.Second, true, false)
	stats.RecordTimeout("agent-2", 35*time.Second, false, true)

	result := stats.GetStats()

	if result.TotalTimeouts != 3 {
		t.Errorf("TotalTimeouts = %d, want %d", result.TotalTimeouts, 3)
	}
	if result.TimeoutsByAgent["agent-1"] != 2 {
		t.Errorf("agent-1 timeouts = %d, want %d", result.TimeoutsByAgent["agent-1"], 2)
	}
	if result.TimeoutsByAgent["agent-2"] != 1 {
		t.Errorf("agent-2 timeouts = %d, want %d", result.TimeoutsByAgent["agent-2"], 1)
	}
	if result.PartialResponsesPreserved != 1 {
		t.Errorf("PartialResponsesPreserved = %d, want %d", result.PartialResponsesPreserved, 1)
	}
	if result.GlobalTimeouts != 1 {
		t.Errorf("GlobalTimeouts = %d, want %d", result.GlobalTimeouts, 1)
	}
}

func TestTimeoutStats_Concurrent(t *testing.T) {
	stats := NewTimeoutStats()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			agentID := "agent-" + string(rune('a'+idx%5))
			stats.RecordTimeout(agentID, time.Duration(idx)*time.Millisecond, idx%2 == 0, idx%10 == 0)
		}(i)
	}

	wg.Wait()

	result := stats.GetStats()
	if result.TotalTimeouts != 100 {
		t.Errorf("TotalTimeouts = %d, want %d", result.TotalTimeouts, 100)
	}
}

func TestIsContextTimeout(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"deadline exceeded", context.DeadlineExceeded, true},
		{"canceled", context.Canceled, true},
		{"timeout in message", errors.New("request timed out"), true},
		{"generic error", errors.New("some error"), false},
		{"network error", errors.New("connection refused"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isContextTimeout(tt.err)
			if result != tt.expected {
				t.Errorf("isContextTimeout(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestPool_SetAgentTimeout(t *testing.T) {
	eventBus := events.NewBus()
	pool := NewPool(eventBus, 60*time.Second)

	// Set per-agent timeout
	pool.SetAgentTimeout("agent-1", 45*time.Second)

	timeout := pool.GetAgentTimeout("agent-1")
	if timeout != 45*time.Second {
		t.Errorf("GetAgentTimeout = %v, want %v", timeout, 45*time.Second)
	}

	// Other agent gets default
	timeout = pool.GetAgentTimeout("agent-2")
	if timeout != 60*time.Second {
		t.Errorf("GetAgentTimeout for other agent = %v, want %v", timeout, 60*time.Second)
	}
}

func TestPool_GetTimeoutStats(t *testing.T) {
	eventBus := events.NewBus()
	pool := NewPool(eventBus, 60*time.Second)

	stats := pool.GetTimeoutStats()
	if stats.TotalTimeouts != 0 {
		t.Errorf("Initial TotalTimeouts = %d, want 0", stats.TotalTimeouts)
	}
	if stats.TimeoutsByAgent == nil {
		t.Error("TimeoutsByAgent should not be nil")
	}
}

func TestNewPoolWithTimeoutConfig(t *testing.T) {
	eventBus := events.NewBus()
	timeoutConfig := TimeoutConfig{
		DefaultAgentTimeout:       45 * time.Second,
		GlobalConversationTimeout: 180 * time.Second,
		PreservePartialResponse:   true,
	}

	pool := NewPoolWithTimeoutConfig(eventBus, timeoutConfig)

	if pool.timeout != 45*time.Second {
		t.Errorf("pool.timeout = %v, want %v", pool.timeout, 45*time.Second)
	}

	timeout := pool.GetAgentTimeout("any-agent")
	if timeout != 45*time.Second {
		t.Errorf("GetAgentTimeout = %v, want %v", timeout, 45*time.Second)
	}

	handler := pool.GetTimeoutHandler()
	if handler == nil {
		t.Error("TimeoutHandler should not be nil")
	}
}

func TestPool_SetAgentTimeouts(t *testing.T) {
	eventBus := events.NewBus()
	pool := NewPool(eventBus, 60*time.Second)

	configs := []AgentTimeoutConfig{
		{AgentID: "agent-1", Timeout: 10 * time.Second},
		{AgentID: "agent-2", Timeout: 20 * time.Second},
	}

	pool.SetAgentTimeouts(configs)

	if pool.GetAgentTimeout("agent-1") != 10*time.Second {
		t.Errorf("agent-1 timeout = %v, want %v", pool.GetAgentTimeout("agent-1"), 10*time.Second)
	}
	if pool.GetAgentTimeout("agent-2") != 20*time.Second {
		t.Errorf("agent-2 timeout = %v, want %v", pool.GetAgentTimeout("agent-2"), 20*time.Second)
	}
}

func TestTimeoutResult_Fields(t *testing.T) {
	result := TimeoutResult{
		Success:        false,
		Content:        "partial",
		Metrics:        &core.Metrics{TotalTokens: 50},
		Error:          errors.New("timeout"),
		TimedOut:       true,
		ElapsedTime:    30 * time.Second,
		PartialContent: "partial response",
	}

	if result.Success {
		t.Error("Success should be false")
	}
	if !result.TimedOut {
		t.Error("TimedOut should be true")
	}
	if result.PartialContent != "partial response" {
		t.Errorf("PartialContent = %q, want %q", result.PartialContent, "partial response")
	}
}
