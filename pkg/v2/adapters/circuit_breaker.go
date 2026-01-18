// Package adapters provides the interface and implementations for AI agent adapters.
package adapters

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/log"
	"github.com/kevinelliott/agentpipe/pkg/v2/core"
	"github.com/kevinelliott/agentpipe/pkg/v2/errors"
)

// CircuitState represents the current state of the circuit breaker.
type CircuitState int

const (
	// CircuitClosed indicates normal operation - requests are allowed.
	CircuitClosed CircuitState = iota
	// CircuitOpen indicates circuit is tripped - requests are blocked.
	CircuitOpen
	// CircuitHalfOpen indicates circuit is testing - limited requests allowed.
	CircuitHalfOpen
)

// String returns the string representation of the circuit state.
func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig defines parameters for circuit breaker behavior.
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of consecutive failures before opening the circuit.
	// Default: 5
	FailureThreshold int

	// CooldownPeriod is the time to wait before transitioning from open to half-open.
	// Default: 30s
	CooldownPeriod time.Duration

	// SuccessThreshold is the number of consecutive successes in half-open state
	// required to close the circuit.
	// Default: 1
	SuccessThreshold int
}

// DefaultCircuitBreakerConfig returns the default circuit breaker configuration.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		CooldownPeriod:   30 * time.Second,
		SuccessThreshold: 1,
	}
}

// CircuitBreaker implements the circuit breaker pattern to prevent
// cascading failures when an agent becomes unresponsive.
type CircuitBreaker struct {
	mu sync.RWMutex

	config          CircuitBreakerConfig
	state           CircuitState
	failureCount    int
	successCount    int // consecutive successes in half-open state
	lastFailure     time.Time
	lastStateChange time.Time

	// Metadata for logging
	agentID   string
	agentName string

	// onStateChange is an optional callback for state change notifications.
	onStateChange func(oldState, newState CircuitState)
}

// NewCircuitBreaker creates a new CircuitBreaker with the given configuration.
func NewCircuitBreaker(config CircuitBreakerConfig, agentID, agentName string) *CircuitBreaker {
	return &CircuitBreaker{
		config:          config,
		state:           CircuitClosed,
		agentID:         agentID,
		agentName:       agentName,
		lastStateChange: time.Now(),
	}
}

// OnStateChange sets a callback to be invoked when the circuit state changes.
func (cb *CircuitBreaker) OnStateChange(fn func(oldState, newState CircuitState)) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.onStateChange = fn
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// FailureCount returns the current consecutive failure count.
func (cb *CircuitBreaker) FailureCount() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failureCount
}

// LastFailure returns the time of the last recorded failure.
func (cb *CircuitBreaker) LastFailure() time.Time {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.lastFailure
}

// LastStateChange returns the time of the last state change.
func (cb *CircuitBreaker) LastStateChange() time.Time {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.lastStateChange
}

// AllowRequest checks if a request should be allowed based on circuit state.
// Returns true if the request can proceed, false if it should be blocked.
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		// Normal operation - allow all requests
		return true

	case CircuitOpen:
		// Check if cooldown period has elapsed
		if time.Since(cb.lastFailure) >= cb.config.CooldownPeriod {
			// Transition to half-open
			cb.transitionToState(CircuitHalfOpen)
			log.WithFields(map[string]interface{}{
				"agent_id":   cb.agentID,
				"agent_name": cb.agentName,
				"cooldown":   cb.config.CooldownPeriod.String(),
			}).Info("Circuit breaker transitioning to half-open after cooldown")
			return true
		}
		// Still in cooldown - block request
		return false

	case CircuitHalfOpen:
		// Allow limited probe requests
		return true

	default:
		return false
	}
}

// RecordSuccess records a successful request.
// In half-open state, enough successes will close the circuit.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		// Reset failure count on success
		cb.failureCount = 0

	case CircuitHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.config.SuccessThreshold {
			// Enough successes - close the circuit
			cb.transitionToState(CircuitClosed)
			cb.failureCount = 0
			cb.successCount = 0
			log.WithFields(map[string]interface{}{
				"agent_id":   cb.agentID,
				"agent_name": cb.agentName,
				"successes":  cb.config.SuccessThreshold,
			}).Info("Circuit breaker closed after successful probe")
		}

	case CircuitOpen:
		// Should not receive success in open state, but handle gracefully
		cb.transitionToState(CircuitHalfOpen)
	}
}

// RecordFailure records a failed request.
// Enough consecutive failures will open the circuit.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailure = time.Now()
	cb.failureCount++

	switch cb.state {
	case CircuitClosed:
		if cb.failureCount >= cb.config.FailureThreshold {
			// Too many failures - open the circuit
			cb.transitionToState(CircuitOpen)
			log.WithFields(map[string]interface{}{
				"agent_id":   cb.agentID,
				"agent_name": cb.agentName,
				"failures":   cb.failureCount,
				"threshold":  cb.config.FailureThreshold,
			}).Warn("Circuit breaker opened due to consecutive failures")
		}

	case CircuitHalfOpen:
		// Failure during probe - immediately open the circuit again
		cb.transitionToState(CircuitOpen)
		cb.successCount = 0
		log.WithFields(map[string]interface{}{
			"agent_id":   cb.agentID,
			"agent_name": cb.agentName,
		}).Warn("Circuit breaker re-opened after failed probe")

	case CircuitOpen:
		// Already open, just update failure time
	}
}

// Reset resets the circuit breaker to its initial closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	oldState := cb.state
	cb.state = CircuitClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.lastStateChange = time.Now()

	if cb.onStateChange != nil && oldState != CircuitClosed {
		cb.onStateChange(oldState, CircuitClosed)
	}

	log.WithFields(map[string]interface{}{
		"agent_id":   cb.agentID,
		"agent_name": cb.agentName,
	}).Info("Circuit breaker reset to closed state")
}

// transitionToState changes the circuit state (must be called with lock held).
func (cb *CircuitBreaker) transitionToState(newState CircuitState) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState
	cb.lastStateChange = time.Now()

	if cb.onStateChange != nil {
		cb.onStateChange(oldState, newState)
	}
}

// TimeUntilRetry returns how long until the circuit will try to recover.
// Returns 0 if the circuit is not open.
func (cb *CircuitBreaker) TimeUntilRetry() time.Duration {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.state != CircuitOpen {
		return 0
	}

	elapsed := time.Since(cb.lastFailure)
	if elapsed >= cb.config.CooldownPeriod {
		return 0
	}

	return cb.config.CooldownPeriod - elapsed
}

// CircuitBreakerAdapter wraps an AgentAdapter with circuit breaker protection.
type CircuitBreakerAdapter struct {
	adapter        AgentAdapter
	circuitBreaker *CircuitBreaker
	agentID        string
	agentName      string
}

// NewCircuitBreakerAdapter creates a new adapter with circuit breaker protection.
func NewCircuitBreakerAdapter(
	adapter AgentAdapter,
	config CircuitBreakerConfig,
	agentID, agentName string,
) *CircuitBreakerAdapter {
	return &CircuitBreakerAdapter{
		adapter:        adapter,
		circuitBreaker: NewCircuitBreaker(config, agentID, agentName),
		agentID:        agentID,
		agentName:      agentName,
	}
}

// CircuitBreaker returns the underlying circuit breaker for inspection.
func (cba *CircuitBreakerAdapter) CircuitBreaker() *CircuitBreaker {
	return cba.circuitBreaker
}

// Initialize delegates to the wrapped adapter.
func (cba *CircuitBreakerAdapter) Initialize(agent core.Agent) error {
	return cba.adapter.Initialize(agent)
}

// IsAvailable returns true if the adapter is available AND circuit allows requests.
func (cba *CircuitBreakerAdapter) IsAvailable() bool {
	if !cba.adapter.IsAvailable() {
		return false
	}
	return cba.circuitBreaker.AllowRequest()
}

// GetModel delegates to the wrapped adapter.
func (cba *CircuitBreakerAdapter) GetModel() string {
	return cba.adapter.GetModel()
}

// HealthCheck delegates to the wrapped adapter (no circuit breaker for health checks).
func (cba *CircuitBreakerAdapter) HealthCheck(ctx context.Context) error {
	return cba.adapter.HealthCheck(ctx)
}

// SendMessage sends a message with circuit breaker protection.
func (cba *CircuitBreakerAdapter) SendMessage(ctx context.Context, messages []core.Message) (string, *core.Metrics, error) {
	if !cba.circuitBreaker.AllowRequest() {
		timeUntil := cba.circuitBreaker.TimeUntilRetry()
		return "", nil, errors.NewAgentError(
			cba.agentID,
			cba.agentName,
			errors.ErrTypeNetwork,
			"circuit breaker open",
			nil,
		).WithRetryAfter(timeUntil)
	}

	response, metrics, err := cba.adapter.SendMessage(ctx, messages)
	if err != nil {
		cba.circuitBreaker.RecordFailure()
		return response, metrics, err
	}

	cba.circuitBreaker.RecordSuccess()
	return response, metrics, nil
}

// StreamMessage streams a message with circuit breaker protection.
func (cba *CircuitBreakerAdapter) StreamMessage(ctx context.Context, messages []core.Message, writer io.Writer) (*core.Metrics, error) {
	if !cba.circuitBreaker.AllowRequest() {
		timeUntil := cba.circuitBreaker.TimeUntilRetry()
		return nil, errors.NewAgentError(
			cba.agentID,
			cba.agentName,
			errors.ErrTypeNetwork,
			"circuit breaker open",
			nil,
		).WithRetryAfter(timeUntil)
	}

	metrics, err := cba.adapter.StreamMessage(ctx, messages, writer)
	if err != nil {
		cba.circuitBreaker.RecordFailure()
		return metrics, err
	}

	cba.circuitBreaker.RecordSuccess()
	return metrics, nil
}

// ErrCircuitOpen is returned when the circuit breaker is open.
var ErrCircuitOpen = &errors.AgentError{
	ErrorType: errors.ErrTypeNetwork,
	Message:   "circuit breaker open",
}

// IsCircuitOpenError checks if an error indicates the circuit is open.
func IsCircuitOpenError(err error) bool {
	agentErr, ok := errors.AsAgentError(err)
	if !ok {
		return false
	}
	return agentErr.Message == "circuit breaker open"
}
