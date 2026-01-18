// Package pool provides the AgentPool for executing agent requests in parallel.
package pool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/log"
	"github.com/kevinelliott/agentpipe/pkg/v2/adapters"
	"github.com/kevinelliott/agentpipe/pkg/v2/core"
	"github.com/kevinelliott/agentpipe/pkg/v2/events"
)

// Response represents the result of an agent's message processing.
type Response struct {
	// AgentID is the ID of the agent that responded.
	AgentID string
	// AgentName is the display name of the agent.
	AgentName string
	// Message is the completed message from the agent.
	Message *core.Message
	// Error is any error that occurred during processing.
	Error error
}

// AgentPool manages a collection of agents and executes requests in parallel.
type AgentPool interface {
	// ExecuteParallel sends a message to all agents in parallel and returns their responses.
	ExecuteParallel(ctx context.Context, messages []core.Message) []Response
	// GetStatus returns the status of all agents in the pool.
	GetStatus() map[string]core.AgentStatus
}

// AgentEntry pairs an agent with its adapter and circuit breaker.
type AgentEntry struct {
	Agent          core.Agent
	Adapter        adapters.AgentAdapter
	State          *core.AgentState
	CircuitBreaker *adapters.CircuitBreaker
}

// Pool is the default implementation of AgentPool.
type Pool struct {
	agents               []AgentEntry
	eventBus             *events.Bus
	timeout              time.Duration
	circuitBreakerConfig adapters.CircuitBreakerConfig
	timeoutHandler       *TimeoutHandler
	timeoutStats         *TimeoutStats
	cancellation         *CancellationManager
	mu                   sync.RWMutex
}

// NewPool creates a new agent pool.
func NewPool(eventBus *events.Bus, timeout time.Duration) *Pool {
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	timeoutConfig := DefaultTimeoutConfig()
	timeoutConfig.DefaultAgentTimeout = timeout

	return &Pool{
		agents:               make([]AgentEntry, 0),
		eventBus:             eventBus,
		timeout:              timeout,
		circuitBreakerConfig: adapters.DefaultCircuitBreakerConfig(),
		timeoutHandler:       NewTimeoutHandler(timeoutConfig, eventBus),
		timeoutStats:         NewTimeoutStats(),
		cancellation:         NewCancellationManager(eventBus),
	}
}

// NewPoolWithCircuitBreaker creates a new agent pool with custom circuit breaker config.
func NewPoolWithCircuitBreaker(eventBus *events.Bus, timeout time.Duration, cbConfig adapters.CircuitBreakerConfig) *Pool {
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	timeoutConfig := DefaultTimeoutConfig()
	timeoutConfig.DefaultAgentTimeout = timeout

	return &Pool{
		agents:               make([]AgentEntry, 0),
		eventBus:             eventBus,
		timeout:              timeout,
		circuitBreakerConfig: cbConfig,
		timeoutHandler:       NewTimeoutHandler(timeoutConfig, eventBus),
		timeoutStats:         NewTimeoutStats(),
		cancellation:         NewCancellationManager(eventBus),
	}
}

// NewPoolWithTimeoutConfig creates a new agent pool with custom timeout configuration.
func NewPoolWithTimeoutConfig(eventBus *events.Bus, timeoutConfig TimeoutConfig) *Pool {
	return &Pool{
		agents:               make([]AgentEntry, 0),
		eventBus:             eventBus,
		timeout:              timeoutConfig.DefaultAgentTimeout,
		circuitBreakerConfig: adapters.DefaultCircuitBreakerConfig(),
		timeoutHandler:       NewTimeoutHandler(timeoutConfig, eventBus),
		timeoutStats:         NewTimeoutStats(),
		cancellation:         NewCancellationManager(eventBus),
	}
}

// NewPoolWithFullConfig creates a new agent pool with both circuit breaker and timeout configuration.
func NewPoolWithFullConfig(eventBus *events.Bus, cbConfig adapters.CircuitBreakerConfig, timeoutConfig TimeoutConfig) *Pool {
	return &Pool{
		agents:               make([]AgentEntry, 0),
		eventBus:             eventBus,
		timeout:              timeoutConfig.DefaultAgentTimeout,
		circuitBreakerConfig: cbConfig,
		timeoutHandler:       NewTimeoutHandler(timeoutConfig, eventBus),
		timeoutStats:         NewTimeoutStats(),
		cancellation:         NewCancellationManager(eventBus),
	}
}

// AddAgent adds an agent to the pool with automatic circuit breaker setup.
func (p *Pool) AddAgent(agent core.Agent, adapter adapters.AgentAdapter) {
	p.mu.Lock()
	defer p.mu.Unlock()

	state := core.NewAgentState(agent)
	cb := adapters.NewCircuitBreaker(p.circuitBreakerConfig, agent.ID, agent.Name)

	// Set up state change callback for logging
	cb.OnStateChange(func(oldState, newState adapters.CircuitState) {
		log.WithFields(map[string]interface{}{
			"agent_id":   agent.ID,
			"agent_name": agent.Name,
			"old_state":  oldState.String(),
			"new_state":  newState.String(),
		}).Info("Agent circuit breaker state changed")

		// Emit error event when circuit opens
		if newState == adapters.CircuitOpen && p.eventBus != nil {
			p.eventBus.Publish(core.NewAgentErrorEvent(
				agent.ID,
				agent.Name,
				fmt.Sprintf("circuit breaker opened after %d consecutive failures", p.circuitBreakerConfig.FailureThreshold),
			))
		}
	})

	p.agents = append(p.agents, AgentEntry{
		Agent:          agent,
		Adapter:        adapter,
		State:          &state,
		CircuitBreaker: cb,
	})
}

// GetAgents returns all agents in the pool.
func (p *Pool) GetAgents() []core.Agent {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]core.Agent, len(p.agents))
	for i, entry := range p.agents {
		result[i] = entry.Agent
	}
	return result
}

// ExecuteParallel sends messages to all agents in parallel.
// Agents with open circuit breakers are skipped with an error response.
func (p *Pool) ExecuteParallel(ctx context.Context, messages []core.Message) []Response {
	p.mu.RLock()
	agentsCopy := make([]AgentEntry, len(p.agents))
	copy(agentsCopy, p.agents)
	p.mu.RUnlock()

	var wg sync.WaitGroup
	responseChan := make(chan Response, len(agentsCopy))

	for _, entry := range agentsCopy {
		wg.Add(1)
		go func(e AgentEntry) {
			defer wg.Done()

			// Check circuit breaker before executing
			if e.CircuitBreaker != nil && !e.CircuitBreaker.AllowRequest() {
				timeUntil := e.CircuitBreaker.TimeUntilRetry()

				log.WithFields(map[string]interface{}{
					"agent_id":       e.Agent.ID,
					"agent_name":     e.Agent.Name,
					"circuit_state":  e.CircuitBreaker.State().String(),
					"retry_in":       timeUntil.String(),
				}).Warn("Skipping agent due to open circuit breaker")

				// Emit error event for circuit open
				if p.eventBus != nil {
					p.eventBus.Publish(core.NewAgentErrorEvent(
						e.Agent.ID,
						e.Agent.Name,
						fmt.Sprintf("circuit breaker open, retry in %s", timeUntil.Round(time.Second)),
					))
				}

				responseChan <- Response{
					AgentID:   e.Agent.ID,
					AgentName: e.Agent.Name,
					Error:     fmt.Errorf("circuit breaker open for %s, retry in %s", e.Agent.Name, timeUntil.Round(time.Second)),
				}
				return
			}

			resp := p.executeAgent(ctx, e, messages)
			responseChan <- resp
		}(entry)
	}

	// Wait for all agents to complete
	wg.Wait()
	close(responseChan)

	// Collect responses
	responses := make([]Response, 0, len(agentsCopy))
	for resp := range responseChan {
		responses = append(responses, resp)
	}

	return responses
}

// executeAgent runs a single agent with timeout handling.
func (p *Pool) executeAgent(ctx context.Context, entry AgentEntry, messages []core.Message) Response {
	agent := entry.Agent
	adapter := entry.Adapter

	// Get per-agent timeout from handler
	agentTimeout := p.timeout
	if p.timeoutHandler != nil {
		agentTimeout = p.timeoutHandler.GetAgentTimeout(agent.ID)
	}

	// Create per-agent context with timeout
	agentCtx, cancel := context.WithTimeout(ctx, agentTimeout)
	defer cancel()

	// Register request with cancellation manager for external cancellation support
	if p.cancellation != nil {
		p.cancellation.RegisterRequest(agent.ID, agent.Name, cancel)
		defer p.cancellation.UnregisterRequest(agent.ID)
	}

	// Emit typing event
	if p.eventBus != nil {
		p.eventBus.Publish(core.NewAgentTypingEvent(agent.ID, agent.Name))
	}

	entry.State.SetTyping()

	log.WithFields(map[string]interface{}{
		"agent_id":   agent.ID,
		"agent_name": agent.Name,
		"model":      agent.Model,
		"timeout":    agentTimeout.String(),
	}).Debug("executing agent")

	startTime := time.Now()

	// Send message
	content, metrics, err := adapter.SendMessage(agentCtx, messages)
	duration := time.Since(startTime)

	if err != nil {
		// Check if this was a cancellation
		isCancelled := IsCancellationError(err)
		if isCancelled {
			entry.State.SetCancelled()

			log.WithFields(map[string]interface{}{
				"agent_id":   agent.ID,
				"agent_name": agent.Name,
				"duration":   duration.String(),
			}).Info("agent request cancelled")

			// Note: cancellation event already emitted by CancellationManager
			return Response{
				AgentID:   agent.ID,
				AgentName: agent.Name,
				Error:     ErrAgentCancelled,
			}
		}

		entry.State.SetError(err.Error())

		// Check if this was a timeout error
		isTimeout := isContextTimeout(err)
		if isTimeout && p.timeoutStats != nil {
			p.timeoutStats.RecordTimeout(agent.ID, duration, false, false)
		}

		// Record failure in circuit breaker
		if entry.CircuitBreaker != nil {
			entry.CircuitBreaker.RecordFailure()
		}

		// Emit error event with enhanced timeout details
		if p.eventBus != nil {
			errMsg := err.Error()
			if isTimeout {
				errMsg = fmt.Sprintf("agent timed out after %s (limit: %s)",
					duration.Round(time.Millisecond), agentTimeout.Round(time.Millisecond))
			}
			p.eventBus.Publish(core.NewAgentErrorEvent(agent.ID, agent.Name, errMsg))
		}

		log.WithFields(map[string]interface{}{
			"agent_id":   agent.ID,
			"agent_name": agent.Name,
			"duration":   duration.String(),
			"timeout":    agentTimeout.String(),
			"is_timeout": isTimeout,
		}).WithError(err).Error("agent execution failed")

		return Response{
			AgentID:   agent.ID,
			AgentName: agent.Name,
			Error:     err,
		}
	}

	// Record success in circuit breaker
	if entry.CircuitBreaker != nil {
		entry.CircuitBreaker.RecordSuccess()
	}

	// Ensure metrics has duration
	if metrics == nil {
		metrics = &core.Metrics{}
	}
	if metrics.Duration == 0 {
		metrics.Duration = duration
	}

	// Create the message
	msg := core.NewAgentMessage(agent.ID, agent.Name, content, metrics)

	// Update agent state
	entry.State.RecordMessage(metrics.TotalTokens, metrics.Cost)

	// Emit done event
	if p.eventBus != nil {
		p.eventBus.Publish(core.NewAgentDoneEvent(agent.ID, agent.Name, msg))
	}

	log.WithFields(map[string]interface{}{
		"agent_id":     agent.ID,
		"agent_name":   agent.Name,
		"duration":     duration.String(),
		"total_tokens": metrics.TotalTokens,
	}).Info("agent execution completed")

	return Response{
		AgentID:   agent.ID,
		AgentName: agent.Name,
		Message:   &msg,
	}
}

// GetStatus returns the status of all agents.
func (p *Pool) GetStatus() map[string]core.AgentStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	status := make(map[string]core.AgentStatus)
	for _, entry := range p.agents {
		status[entry.Agent.ID] = entry.State.Status
	}
	return status
}

// AgentCount returns the number of agents in the pool.
func (p *Pool) AgentCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.agents)
}

// GetAgentState returns the state of a specific agent.
func (p *Pool) GetAgentState(agentID string) (*core.AgentState, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, entry := range p.agents {
		if entry.Agent.ID == agentID {
			return entry.State, true
		}
	}
	return nil, false
}

// CircuitBreakerInfo contains circuit breaker status for an agent.
type CircuitBreakerInfo struct {
	AgentID       string
	AgentName     string
	State         adapters.CircuitState
	FailureCount  int
	TimeUntilRetry time.Duration
	LastFailure   time.Time
}

// GetCircuitBreakerStatus returns the circuit breaker status for a specific agent.
func (p *Pool) GetCircuitBreakerStatus(agentID string) (*CircuitBreakerInfo, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, entry := range p.agents {
		if entry.Agent.ID == agentID && entry.CircuitBreaker != nil {
			return &CircuitBreakerInfo{
				AgentID:        entry.Agent.ID,
				AgentName:      entry.Agent.Name,
				State:          entry.CircuitBreaker.State(),
				FailureCount:   entry.CircuitBreaker.FailureCount(),
				TimeUntilRetry: entry.CircuitBreaker.TimeUntilRetry(),
				LastFailure:    entry.CircuitBreaker.LastFailure(),
			}, true
		}
	}
	return nil, false
}

// GetAllCircuitBreakerStatus returns the circuit breaker status for all agents.
func (p *Pool) GetAllCircuitBreakerStatus() []CircuitBreakerInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	infos := make([]CircuitBreakerInfo, 0, len(p.agents))
	for _, entry := range p.agents {
		if entry.CircuitBreaker != nil {
			infos = append(infos, CircuitBreakerInfo{
				AgentID:        entry.Agent.ID,
				AgentName:      entry.Agent.Name,
				State:          entry.CircuitBreaker.State(),
				FailureCount:   entry.CircuitBreaker.FailureCount(),
				TimeUntilRetry: entry.CircuitBreaker.TimeUntilRetry(),
				LastFailure:    entry.CircuitBreaker.LastFailure(),
			})
		}
	}
	return infos
}

// ResetCircuitBreaker resets the circuit breaker for a specific agent.
func (p *Pool) ResetCircuitBreaker(agentID string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, entry := range p.agents {
		if entry.Agent.ID == agentID && entry.CircuitBreaker != nil {
			entry.CircuitBreaker.Reset()
			log.WithFields(map[string]interface{}{
				"agent_id":   entry.Agent.ID,
				"agent_name": entry.Agent.Name,
			}).Info("Circuit breaker manually reset")
			return true
		}
	}
	return false
}

// ResetAllCircuitBreakers resets all circuit breakers in the pool.
func (p *Pool) ResetAllCircuitBreakers() {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, entry := range p.agents {
		if entry.CircuitBreaker != nil {
			entry.CircuitBreaker.Reset()
		}
	}

	log.Info("All circuit breakers reset")
}

// GetAvailableAgentCount returns the count of agents whose circuits allow requests.
func (p *Pool) GetAvailableAgentCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	count := 0
	for _, entry := range p.agents {
		if entry.CircuitBreaker == nil || entry.CircuitBreaker.AllowRequest() {
			count++
		}
	}
	return count
}

// SetAgentTimeout sets a specific timeout for an agent.
func (p *Pool) SetAgentTimeout(agentID string, timeout time.Duration) {
	if p.timeoutHandler != nil {
		p.timeoutHandler.SetAgentTimeout(agentID, timeout)
		log.WithFields(map[string]interface{}{
			"agent_id": agentID,
			"timeout":  timeout.String(),
		}).Info("Agent timeout configured")
	}
}

// GetAgentTimeout returns the timeout configuration for an agent.
func (p *Pool) GetAgentTimeout(agentID string) time.Duration {
	if p.timeoutHandler != nil {
		return p.timeoutHandler.GetAgentTimeout(agentID)
	}
	return p.timeout
}

// SetAgentTimeouts sets timeouts for multiple agents at once.
func (p *Pool) SetAgentTimeouts(configs []AgentTimeoutConfig) {
	if p.timeoutHandler != nil {
		p.timeoutHandler.SetAgentTimeouts(configs)
	}
}

// GetTimeoutStats returns timeout statistics for all agents.
func (p *Pool) GetTimeoutStats() TimeoutStats {
	if p.timeoutStats != nil {
		return p.timeoutStats.GetStats()
	}
	return TimeoutStats{TimeoutsByAgent: make(map[string]int)}
}

// GetTimeoutHandler returns the timeout handler for advanced configuration.
func (p *Pool) GetTimeoutHandler() *TimeoutHandler {
	return p.timeoutHandler
}

// Cancel cancels all pending agent requests.
// Returns the number of requests that were cancelled.
// This provides clean cancellation without goroutine leaks.
func (p *Pool) Cancel() int {
	if p.cancellation == nil {
		return 0
	}
	return p.cancellation.CancelAll()
}

// CancelAgent cancels a specific agent's pending request.
// Returns ErrAgentNotFound if the agent has no active request.
func (p *Pool) CancelAgent(agentID string) error {
	if p.cancellation == nil {
		return ErrAgentNotFound
	}
	return p.cancellation.CancelAgent(agentID)
}

// GetActiveRequests returns information about all currently active agent requests.
func (p *Pool) GetActiveRequests() []CancellationInfo {
	if p.cancellation == nil {
		return nil
	}
	return p.cancellation.GetActiveRequests()
}

// HasActiveRequest checks if a specific agent has an active request in progress.
func (p *Pool) HasActiveRequest(agentID string) bool {
	if p.cancellation == nil {
		return false
	}
	return p.cancellation.HasActiveRequest(agentID)
}

// ActiveRequestCount returns the number of currently active (non-cancelled) agent requests.
func (p *Pool) ActiveRequestCount() int {
	if p.cancellation == nil {
		return 0
	}
	return p.cancellation.ActiveRequestCount()
}

// IsAgentCancelled checks if a specific agent's request was cancelled.
func (p *Pool) IsAgentCancelled(agentID string) bool {
	if p.cancellation == nil {
		return false
	}
	return p.cancellation.IsCancelled(agentID)
}
