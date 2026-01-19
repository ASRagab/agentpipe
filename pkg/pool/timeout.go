// Package pool provides the timeout handling functionality for agent execution.
package pool

import (
	"context"
	stdErrors "errors"
	"fmt"
	"sync"
	"time"

	"github.com/ASRagab/agentpipe/pkg/core"
	"github.com/ASRagab/agentpipe/pkg/errors"
	"github.com/ASRagab/agentpipe/pkg/events"
	"github.com/ASRagab/agentpipe/pkg/log"
)

// TimeoutConfig contains timeout configuration for agent execution.
type TimeoutConfig struct {
	// DefaultAgentTimeout is the default timeout for individual agents.
	// If an agent doesn't have a specific timeout, this value is used.
	DefaultAgentTimeout time.Duration

	// GlobalConversationTimeout is the maximum time for all agents combined
	// to respond to a single user message. Set to 0 to disable.
	GlobalConversationTimeout time.Duration

	// GracePeriod is extra time allowed after timeout for cleanup.
	GracePeriod time.Duration

	// PreservePartialResponse determines whether to keep partial responses on timeout.
	PreservePartialResponse bool
}

// DefaultTimeoutConfig returns a default timeout configuration.
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		DefaultAgentTimeout:       30 * time.Second,
		GlobalConversationTimeout: 120 * time.Second, // 2 minutes for all agents
		GracePeriod:               2 * time.Second,
		PreservePartialResponse:   true,
	}
}

// AgentTimeoutConfig stores per-agent timeout configuration.
type AgentTimeoutConfig struct {
	// AgentID is the unique identifier of the agent.
	AgentID string
	// Timeout is the specific timeout for this agent.
	Timeout time.Duration
}

// TimeoutHandler manages timeouts for agent execution with support for
// per-agent timeouts, global conversation timeout, and partial response preservation.
type TimeoutHandler struct {
	config        TimeoutConfig
	agentTimeouts map[string]time.Duration
	eventBus      *events.Bus
	mu            sync.RWMutex
}

// NewTimeoutHandler creates a new timeout handler with the given configuration.
func NewTimeoutHandler(config TimeoutConfig, eventBus *events.Bus) *TimeoutHandler {
	return &TimeoutHandler{
		config:        config,
		agentTimeouts: make(map[string]time.Duration),
		eventBus:      eventBus,
	}
}

// SetAgentTimeout sets a specific timeout for an agent.
func (h *TimeoutHandler) SetAgentTimeout(agentID string, timeout time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.agentTimeouts[agentID] = timeout
}

// GetAgentTimeout returns the timeout for a specific agent.
// If no specific timeout is set, returns the default.
func (h *TimeoutHandler) GetAgentTimeout(agentID string) time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if timeout, ok := h.agentTimeouts[agentID]; ok {
		return timeout
	}
	return h.config.DefaultAgentTimeout
}

// SetAgentTimeouts sets multiple agent timeouts at once.
func (h *TimeoutHandler) SetAgentTimeouts(configs []AgentTimeoutConfig) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, cfg := range configs {
		h.agentTimeouts[cfg.AgentID] = cfg.Timeout
	}
}

// CreateAgentContext creates a context with the appropriate timeout for an agent.
// It returns the context, cancel function, and the actual timeout duration.
func (h *TimeoutHandler) CreateAgentContext(parent context.Context, agentID string) (context.Context, context.CancelFunc, time.Duration) {
	timeout := h.GetAgentTimeout(agentID)
	ctx, cancel := context.WithTimeout(parent, timeout)
	return ctx, cancel, timeout
}

// CreateGlobalContext creates a context for the global conversation timeout.
// If GlobalConversationTimeout is 0, returns the parent context with a no-op cancel.
func (h *TimeoutHandler) CreateGlobalContext(parent context.Context) (context.Context, context.CancelFunc) {
	if h.config.GlobalConversationTimeout <= 0 {
		return parent, func() {}
	}
	return context.WithTimeout(parent, h.config.GlobalConversationTimeout)
}

// TimeoutError represents a timeout error with detailed context.
type TimeoutError struct {
	// AgentID is the ID of the agent that timed out.
	AgentID string
	// AgentName is the display name of the agent.
	AgentName string
	// TimeoutDuration is how long we waited before timing out.
	TimeoutDuration time.Duration
	// ElapsedTime is how much time had passed when the timeout occurred.
	ElapsedTime time.Duration
	// PartialContent is any content received before the timeout (if preserved).
	PartialContent string
	// IsGlobalTimeout indicates if this was a global conversation timeout.
	IsGlobalTimeout bool
}

// Error implements the error interface.
func (e *TimeoutError) Error() string {
	if e.IsGlobalTimeout {
		return fmt.Sprintf("global conversation timeout after %s for agent %s",
			e.TimeoutDuration.Round(time.Millisecond), e.AgentName)
	}
	return fmt.Sprintf("agent %s timed out after %s",
		e.AgentName, e.TimeoutDuration.Round(time.Millisecond))
}

// ToAgentError converts a TimeoutError to an AgentError.
func (e *TimeoutError) ToAgentError() *errors.AgentError {
	return errors.NewTimeoutError(e.AgentID, e.AgentName, e)
}

// HasPartialContent returns true if there's any partial content preserved.
func (e *TimeoutError) HasPartialContent() bool {
	return len(e.PartialContent) > 0
}

// TimeoutResult represents the result of a timeout-wrapped operation.
type TimeoutResult struct {
	// Success indicates the operation completed successfully.
	Success bool
	// Content is the response content (full or partial).
	Content string
	// Metrics contains any metrics collected.
	Metrics *core.Metrics
	// Error is any error that occurred.
	Error error
	// TimedOut indicates if a timeout occurred.
	TimedOut bool
	// ElapsedTime is how long the operation took.
	ElapsedTime time.Duration
	// PartialContent is preserved content from a timed-out operation.
	PartialContent string
}

// ExecuteWithTimeout executes a function with timeout handling and partial response preservation.
// The operation function receives a channel where it can send partial content for preservation.
func (h *TimeoutHandler) ExecuteWithTimeout(
	ctx context.Context,
	agentID, agentName string,
	operation func(ctx context.Context, partialChan chan<- string) (string, *core.Metrics, error),
) TimeoutResult {
	timeout := h.GetAgentTimeout(agentID)
	return h.executeWithDuration(ctx, agentID, agentName, timeout, false, operation)
}

// ExecuteWithGlobalTimeout executes with the global conversation timeout.
func (h *TimeoutHandler) ExecuteWithGlobalTimeout(
	ctx context.Context,
	agentID, agentName string,
	operation func(ctx context.Context, partialChan chan<- string) (string, *core.Metrics, error),
) TimeoutResult {
	timeout := h.config.GlobalConversationTimeout
	if timeout <= 0 {
		timeout = h.config.DefaultAgentTimeout
	}
	return h.executeWithDuration(ctx, agentID, agentName, timeout, true, operation)
}

// executeWithDuration handles the actual timeout execution logic.
func (h *TimeoutHandler) executeWithDuration(
	ctx context.Context,
	agentID, agentName string,
	timeout time.Duration,
	isGlobal bool,
	operation func(ctx context.Context, partialChan chan<- string) (string, *core.Metrics, error),
) TimeoutResult {
	startTime := time.Now()

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Channel for operation result
	type opResult struct {
		content string
		metrics *core.Metrics
		err     error
	}
	resultChan := make(chan opResult, 1)

	// Channel for partial content (buffered to avoid blocking)
	partialChan := make(chan string, 100)
	var partialContent string
	var partialMu sync.Mutex

	// Goroutine to collect partial content
	go func() {
		for partial := range partialChan {
			if h.config.PreservePartialResponse {
				partialMu.Lock()
				partialContent += partial
				partialMu.Unlock()
			}
		}
	}()

	// Execute operation in goroutine
	go func() {
		defer close(partialChan)
		content, metrics, err := operation(timeoutCtx, partialChan)
		resultChan <- opResult{content: content, metrics: metrics, err: err}
	}()

	// Wait for result or timeout
	select {
	case result := <-resultChan:
		elapsed := time.Since(startTime)

		if result.err != nil {
			if stdErrors.Is(result.err, context.Canceled) {
				return TimeoutResult{
					Success:     false,
					Content:     result.content,
					Metrics:     result.metrics,
					Error:       result.err,
					TimedOut:    false,
					ElapsedTime: elapsed,
				}
			}
			// Check if the error is a context-related timeout
			if isContextTimeout(result.err) {
				return h.handleTimeout(agentID, agentName, timeout, elapsed, partialContent, isGlobal)
			}
			return TimeoutResult{
				Success:     false,
				Content:     result.content,
				Metrics:     result.metrics,
				Error:       result.err,
				TimedOut:    false,
				ElapsedTime: elapsed,
			}
		}

		return TimeoutResult{
			Success:     true,
			Content:     result.content,
			Metrics:     result.metrics,
			Error:       nil,
			TimedOut:    false,
			ElapsedTime: elapsed,
		}

	case <-timeoutCtx.Done():
		elapsed := time.Since(startTime)

		// Get any partial content collected
		partialMu.Lock()
		preserved := partialContent
		partialMu.Unlock()

		return h.handleTimeout(agentID, agentName, timeout, elapsed, preserved, isGlobal)
	}
}

// handleTimeout creates the timeout result and emits appropriate events.
func (h *TimeoutHandler) handleTimeout(
	agentID, agentName string,
	timeout, elapsed time.Duration,
	partialContent string,
	isGlobal bool,
) TimeoutResult {
	timeoutErr := &TimeoutError{
		AgentID:         agentID,
		AgentName:       agentName,
		TimeoutDuration: timeout,
		ElapsedTime:     elapsed,
		PartialContent:  partialContent,
		IsGlobalTimeout: isGlobal,
	}

	// Log the timeout
	logFields := map[string]interface{}{
		"agent_id":          agentID,
		"agent_name":        agentName,
		"timeout":           timeout.String(),
		"elapsed":           elapsed.String(),
		"has_partial":       len(partialContent) > 0,
		"is_global_timeout": isGlobal,
	}
	if len(partialContent) > 0 {
		logFields["partial_length"] = len(partialContent)
	}
	log.WithFields(logFields).Warn("Agent timed out")

	// Emit error event
	if h.eventBus != nil {
		errMsg := timeoutErr.Error()
		if len(partialContent) > 0 {
			errMsg += fmt.Sprintf(" (preserved %d chars of partial response)", len(partialContent))
		}
		h.eventBus.Publish(core.NewAgentErrorEvent(agentID, agentName, errMsg))
	}

	return TimeoutResult{
		Success:        false,
		Content:        "",
		Metrics:        nil,
		Error:          timeoutErr.ToAgentError(),
		TimedOut:       true,
		ElapsedTime:    elapsed,
		PartialContent: partialContent,
	}
}

// isContextTimeout checks if an error is a context timeout/cancellation.
func isContextTimeout(err error) bool {
	if err == nil {
		return false
	}
	return err == context.DeadlineExceeded || err == context.Canceled ||
		errors.ClassifyError(err) == errors.ErrTypeTimeout
}

// TimeoutStats tracks timeout statistics across agents.
type TimeoutStats struct {
	// TotalTimeouts is the total number of timeouts.
	TotalTimeouts int
	// TimeoutsByAgent tracks timeouts per agent.
	TimeoutsByAgent map[string]int
	// AverageTimeoutDuration is the average time before timeout.
	AverageTimeoutDuration time.Duration
	// PartialResponsesPreserved is the count of partial responses saved.
	PartialResponsesPreserved int
	// GlobalTimeouts is the count of global conversation timeouts.
	GlobalTimeouts int
	mu             sync.RWMutex
}

// NewTimeoutStats creates a new timeout statistics tracker.
func NewTimeoutStats() *TimeoutStats {
	return &TimeoutStats{
		TimeoutsByAgent: make(map[string]int),
	}
}

// RecordTimeout records a timeout event.
func (s *TimeoutStats) RecordTimeout(agentID string, elapsed time.Duration, hasPartial, isGlobal bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TotalTimeouts++
	s.TimeoutsByAgent[agentID]++

	// Update average
	n := float64(s.TotalTimeouts)
	currentAvg := float64(s.AverageTimeoutDuration)
	newAvg := currentAvg + (float64(elapsed)-currentAvg)/n
	s.AverageTimeoutDuration = time.Duration(newAvg)

	if hasPartial {
		s.PartialResponsesPreserved++
	}
	if isGlobal {
		s.GlobalTimeouts++
	}
}

// GetStats returns a copy of the current statistics.
func (s *TimeoutStats) GetStats() TimeoutStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a copy
	copy := TimeoutStats{
		TotalTimeouts:             s.TotalTimeouts,
		TimeoutsByAgent:           make(map[string]int),
		AverageTimeoutDuration:    s.AverageTimeoutDuration,
		PartialResponsesPreserved: s.PartialResponsesPreserved,
		GlobalTimeouts:            s.GlobalTimeouts,
	}
	for k, v := range s.TimeoutsByAgent {
		copy.TimeoutsByAgent[k] = v
	}
	return copy
}
