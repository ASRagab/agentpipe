// Package pool provides the AgentPool for executing agent requests in parallel.
package pool

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ASRagab/agentpipe/pkg/adapters"
	"github.com/ASRagab/agentpipe/pkg/core"
	"github.com/ASRagab/agentpipe/pkg/errors"
	"github.com/ASRagab/agentpipe/pkg/events"
	"github.com/ASRagab/agentpipe/pkg/log"
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
	Error          error
	PartialContent string
}

// AgentPool manages a collection of agents and executes requests in parallel.
type AgentPool interface {
	// ExecuteParallel sends a message to all agents in parallel and returns their responses.
	ExecuteParallel(ctx context.Context, messages []core.Message, conversation *core.ConversationContext) []Response
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
	healthMonitor        *HealthMonitor
	healthConfig         HealthConfig
	mu                   sync.RWMutex
}

// NewPool creates a new agent pool.
func NewPool(eventBus *events.Bus, timeout time.Duration) *Pool {
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	timeoutConfig := DefaultTimeoutConfig()
	timeoutConfig.DefaultAgentTimeout = timeout
	healthConfig := DefaultHealthConfig()

	return &Pool{
		agents:               make([]AgentEntry, 0),
		eventBus:             eventBus,
		timeout:              timeout,
		circuitBreakerConfig: adapters.DefaultCircuitBreakerConfig(),
		timeoutHandler:       NewTimeoutHandler(timeoutConfig, eventBus),
		timeoutStats:         NewTimeoutStats(),
		cancellation:         NewCancellationManager(eventBus),
		healthMonitor:        NewHealthMonitor(healthConfig, eventBus),
		healthConfig:         healthConfig,
	}
}

// NewPoolWithCircuitBreaker creates a new agent pool with custom circuit breaker config.
func NewPoolWithCircuitBreaker(eventBus *events.Bus, timeout time.Duration, cbConfig adapters.CircuitBreakerConfig) *Pool {
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	timeoutConfig := DefaultTimeoutConfig()
	timeoutConfig.DefaultAgentTimeout = timeout
	healthConfig := DefaultHealthConfig()

	return &Pool{
		agents:               make([]AgentEntry, 0),
		eventBus:             eventBus,
		timeout:              timeout,
		circuitBreakerConfig: cbConfig,
		timeoutHandler:       NewTimeoutHandler(timeoutConfig, eventBus),
		timeoutStats:         NewTimeoutStats(),
		cancellation:         NewCancellationManager(eventBus),
		healthMonitor:        NewHealthMonitor(healthConfig, eventBus),
		healthConfig:         healthConfig,
	}
}

// NewPoolWithTimeoutConfig creates a new agent pool with custom timeout configuration.
func NewPoolWithTimeoutConfig(eventBus *events.Bus, timeoutConfig TimeoutConfig) *Pool {
	healthConfig := DefaultHealthConfig()
	return &Pool{
		agents:               make([]AgentEntry, 0),
		eventBus:             eventBus,
		timeout:              timeoutConfig.DefaultAgentTimeout,
		circuitBreakerConfig: adapters.DefaultCircuitBreakerConfig(),
		timeoutHandler:       NewTimeoutHandler(timeoutConfig, eventBus),
		timeoutStats:         NewTimeoutStats(),
		cancellation:         NewCancellationManager(eventBus),
		healthMonitor:        NewHealthMonitor(healthConfig, eventBus),
		healthConfig:         healthConfig,
	}
}

// NewPoolWithFullConfig creates a new agent pool with both circuit breaker and timeout configuration.
func NewPoolWithFullConfig(eventBus *events.Bus, cbConfig adapters.CircuitBreakerConfig, timeoutConfig TimeoutConfig) *Pool {
	healthConfig := DefaultHealthConfig()
	return &Pool{
		agents:               make([]AgentEntry, 0),
		eventBus:             eventBus,
		timeout:              timeoutConfig.DefaultAgentTimeout,
		circuitBreakerConfig: cbConfig,
		timeoutHandler:       NewTimeoutHandler(timeoutConfig, eventBus),
		timeoutStats:         NewTimeoutStats(),
		cancellation:         NewCancellationManager(eventBus),
		healthMonitor:        NewHealthMonitor(healthConfig, eventBus),
		healthConfig:         healthConfig,
	}
}

// NewPoolWithHealthConfig creates a new agent pool with custom health monitoring configuration.
func NewPoolWithHealthConfig(eventBus *events.Bus, timeout time.Duration, healthConfig HealthConfig) *Pool {
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
		healthMonitor:        NewHealthMonitor(healthConfig, eventBus),
		healthConfig:         healthConfig,
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

	// Register with health monitor
	if p.healthMonitor != nil {
		p.healthMonitor.RegisterAgent(agent, adapter)
	}
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
func (p *Pool) ExecuteParallel(ctx context.Context, messages []core.Message, conversation *core.ConversationContext) []Response {
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
					"agent_id":      e.Agent.ID,
					"agent_name":    e.Agent.Name,
					"circuit_state": e.CircuitBreaker.State().String(),
					"retry_in":      timeUntil.String(),
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

			resp := p.executeAgent(ctx, e, messages, conversation)
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
func (p *Pool) executeAgent(ctx context.Context, entry AgentEntry, messages []core.Message, conversation *core.ConversationContext) Response {
	agent := entry.Agent
	adapter := entry.Adapter

	// Get per-agent timeout from handler
	agentTimeout := p.timeout
	if p.timeoutHandler != nil {
		agentTimeout = p.timeoutHandler.GetAgentTimeout(agent.ID)
	}

	agentCtx, cancel := context.WithTimeout(ctx, agentTimeout)
	defer cancel()

	if p.cancellation != nil {
		p.cancellation.RegisterRequest(agent.ID, agent.Name, cancel)
		defer p.cancellation.UnregisterRequest(agent.ID)
	}

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

	if p.isStreamingEnabled() {
		return p.executeAgentStreaming(agentCtx, entry, messages, startTime, conversation)
	}

	if p.timeoutHandler == nil {
		content, metrics, err := adapter.SendMessage(agentCtx, messages, conversation)
		duration := time.Since(startTime)
		if err != nil {
			return p.handleAgentError(entry, agentTimeout, duration, err)
		}

		msg := p.handleAgentSuccess(entry, duration, content, metrics)
		return Response{
			AgentID:   entry.Agent.ID,
			AgentName: entry.Agent.Name,
			Message:   &msg,
		}
	}

	result := p.timeoutHandler.ExecuteWithTimeout(agentCtx, entry.Agent.ID, entry.Agent.Name,
		func(execCtx context.Context, partialChan chan<- string) (string, *core.Metrics, error) {
			return adapter.SendMessage(execCtx, messages, conversation)
		},
	)
	duration := time.Since(startTime)

	if result.Error != nil {
		return p.handleTimeoutResult(entry, agentTimeout, duration, result)
	}

	msg := p.handleAgentSuccess(entry, duration, result.Content, result.Metrics)
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

func (p *Pool) isStreamingEnabled() bool {
	return p.eventBus != nil
}

func newMessageID() string {
	return fmt.Sprintf("msg-%d", time.Now().UnixNano())
}

func (p *Pool) executeAgentStreaming(ctx context.Context, entry AgentEntry, messages []core.Message, startTime time.Time, conversation *core.ConversationContext) Response {
	messageID := newMessageID()
	buffer := &chunkCollector{
		agentID:    entry.Agent.ID,
		agentName:  entry.Agent.Name,
		messageID:  messageID,
		eventBus:   p.eventBus,
		chunkIndex: 0,
	}
	if p.eventBus != nil {
		p.eventBus.Publish(core.NewMessageChunkEvent(core.MessageChunk{
			MessageID: messageID,
			AgentID:   entry.Agent.ID,
			AgentName: entry.Agent.Name,
			Content:   "",
			Index:     0,
			IsFinal:   false,
		}))
	}

	result := p.timeoutHandler.ExecuteWithTimeout(ctx, entry.Agent.ID, entry.Agent.Name,
		func(execCtx context.Context, partialChan chan<- string) (string, *core.Metrics, error) {
			writer := &streamWriter{
				collector:   buffer,
				partialChan: partialChan,
			}
			metrics, err := entry.Adapter.StreamMessage(execCtx, messages, writer, conversation)
			return buffer.String(), metrics, err
		},
	)

	elapsed := time.Since(startTime)
	if result.Error != nil {
		return p.handleStreamingError(entry, elapsed, result, messageID, buffer.chunkIndex)
	}

	if buffer.chunkIndex == 0 {
		buffer.chunkIndex = 1
	}

	if result.Metrics == nil {
		result.Metrics = &core.Metrics{}
	}
	if result.Metrics.Duration == 0 {
		result.Metrics.Duration = elapsed
	}

	entry.State.RecordMessage(result.Metrics.TotalTokens, result.Metrics.Cost)

	msg := core.NewAgentMessage(entry.Agent.ID, entry.Agent.Name, result.Content, result.Metrics)
	msg.ID = messageID
	if p.eventBus != nil {
		p.eventBus.Publish(core.NewMessageChunkEvent(core.MessageChunk{
			MessageID: messageID,
			AgentID:   entry.Agent.ID,
			AgentName: entry.Agent.Name,
			Content:   "",
			Index:     buffer.chunkIndex,
			IsFinal:   true,
		}))
		p.eventBus.Publish(core.NewAgentDoneEvent(entry.Agent.ID, entry.Agent.Name, msg))
	}

	log.WithFields(map[string]interface{}{
		"agent_id":     entry.Agent.ID,
		"agent_name":   entry.Agent.Name,
		"duration":     elapsed.String(),
		"total_tokens": result.Metrics.TotalTokens,
	}).Info("agent execution completed")

	return Response{
		AgentID:   entry.Agent.ID,
		AgentName: entry.Agent.Name,
		Message:   &msg,
	}

}

func (p *Pool) handleAgentSuccess(entry AgentEntry, duration time.Duration, content string, metrics *core.Metrics) core.Message {
	if entry.CircuitBreaker != nil {
		entry.CircuitBreaker.RecordSuccess()
	}

	if metrics == nil {
		metrics = &core.Metrics{}
	}
	if metrics.Duration == 0 {
		metrics.Duration = duration
	}

	entry.State.RecordMessage(metrics.TotalTokens, metrics.Cost)

	msg := core.NewAgentMessage(entry.Agent.ID, entry.Agent.Name, content, metrics)
	if p.eventBus != nil {
		p.eventBus.Publish(core.NewAgentDoneEvent(entry.Agent.ID, entry.Agent.Name, msg))
	}

	log.WithFields(map[string]interface{}{
		"agent_id":     entry.Agent.ID,
		"agent_name":   entry.Agent.Name,
		"duration":     duration.String(),
		"total_tokens": metrics.TotalTokens,
	}).Info("agent execution completed")

	return msg
}

func (p *Pool) handleAgentError(entry AgentEntry, agentTimeout, duration time.Duration, err error) Response {
	if IsCancellationError(err) {
		entry.State.SetCancelled()

		if p.eventBus != nil {
			p.eventBus.Publish(core.NewAgentCancelledEvent(
				entry.Agent.ID,
				entry.Agent.Name,
				"request cancelled",
			))
		}

		log.WithFields(map[string]interface{}{
			"agent_id":   entry.Agent.ID,
			"agent_name": entry.Agent.Name,
			"duration":   duration.String(),
		}).Info("agent request cancelled")

		return Response{
			AgentID:   entry.Agent.ID,
			AgentName: entry.Agent.Name,
			Error:     ErrAgentCancelled,
		}
	}

	if p.cancellation != nil && p.cancellation.IsCancelled(entry.Agent.ID) {
		entry.State.SetCancelled()

		if p.eventBus != nil {
			p.eventBus.Publish(core.NewAgentCancelledEvent(
				entry.Agent.ID,
				entry.Agent.Name,
				"request cancelled",
			))
		}

		log.WithFields(map[string]interface{}{
			"agent_id":   entry.Agent.ID,
			"agent_name": entry.Agent.Name,
			"duration":   duration.String(),
		}).Info("agent request cancelled")

		return Response{
			AgentID:   entry.Agent.ID,
			AgentName: entry.Agent.Name,
			Error:     ErrAgentCancelled,
		}
	}

	baseErr := err
	if agentErr, ok := errors.AsAgentError(err); ok {
		baseErr = agentErr.Unwrap()
	}
	if baseErr == nil {
		baseErr = err
	}
	agentErr := errors.WrapError(entry.Agent.ID, entry.Agent.Name, baseErr)
	entry.State.SetError(agentErr.Error())

	isTimeout := isContextTimeout(err)
	var responseErr error = agentErr
	if isTimeout {
		responseErr = err
	} else if baseErr != nil {
		responseErr = baseErr
	}
	if isTimeout && p.timeoutStats != nil {
		p.timeoutStats.RecordTimeout(entry.Agent.ID, duration, false, false)
	}

	if entry.CircuitBreaker != nil {
		entry.CircuitBreaker.RecordFailure()
	}

	if p.eventBus != nil {
		errMsg := agentErr.Error()
		if isTimeout {
			errMsg = fmt.Sprintf("agent timed out after %s (limit: %s)",
				duration.Round(time.Millisecond), agentTimeout.Round(time.Millisecond))
		}
		p.eventBus.Publish(core.NewAgentErrorEvent(entry.Agent.ID, entry.Agent.Name, errMsg))
	}

	log.WithFields(map[string]interface{}{
		"agent_id":   entry.Agent.ID,
		"agent_name": entry.Agent.Name,
		"duration":   duration.String(),
		"timeout":    agentTimeout.String(),
		"is_timeout": isTimeout,
	}).WithError(agentErr).Error("agent execution failed")

	return Response{
		AgentID:   entry.Agent.ID,
		AgentName: entry.Agent.Name,
		Error:     responseErr,
	}
}

func (p *Pool) handleTimeoutResult(entry AgentEntry, agentTimeout, duration time.Duration, result TimeoutResult) Response {
	result.Error = unwrapAgentError(result.Error)
	if result.Error == nil {
		return Response{
			AgentID:   entry.Agent.ID,
			AgentName: entry.Agent.Name,
		}
	}

	if IsCancellationError(result.Error) || (p.cancellation != nil && p.cancellation.IsCancelled(entry.Agent.ID)) {
		entry.State.SetCancelled()
		if p.eventBus != nil {
			p.eventBus.Publish(core.NewAgentCancelledEvent(
				entry.Agent.ID,
				entry.Agent.Name,
				"request cancelled",
			))
		}
		return Response{
			AgentID:   entry.Agent.ID,
			AgentName: entry.Agent.Name,
			Error:     ErrAgentCancelled,
		}
	}

	if result.TimedOut {
		timeoutErr := context.DeadlineExceeded
		entry.State.SetError(timeoutErr.Error())
		if p.timeoutStats != nil {
			p.timeoutStats.RecordTimeout(entry.Agent.ID, duration, result.PartialContent != "", false)
		}
		if entry.CircuitBreaker != nil {
			entry.CircuitBreaker.RecordFailure()
		}
		if p.eventBus != nil {
			p.eventBus.Publish(core.NewAgentErrorEvent(
				entry.Agent.ID,
				entry.Agent.Name,
				fmt.Sprintf("agent timed out after %s (limit: %s)",
					duration.Round(time.Millisecond), agentTimeout.Round(time.Millisecond)),
			))
		}
		log.WithFields(map[string]interface{}{
			"agent_id":   entry.Agent.ID,
			"agent_name": entry.Agent.Name,
			"duration":   duration.String(),
			"timeout":    agentTimeout.String(),
			"is_timeout": result.TimedOut,
		}).WithError(timeoutErr).Error("agent execution failed")
		return Response{
			AgentID:   entry.Agent.ID,
			AgentName: entry.Agent.Name,
			Error:     timeoutErr,
		}
	}

	agentErr := errors.WrapError(entry.Agent.ID, entry.Agent.Name, result.Error)
	entry.State.SetError(agentErr.Error())
	responseErr := result.Error

	if result.TimedOut && p.timeoutStats != nil {
		p.timeoutStats.RecordTimeout(entry.Agent.ID, duration, result.PartialContent != "", false)
	}

	if entry.CircuitBreaker != nil {
		entry.CircuitBreaker.RecordFailure()
	}

	if p.eventBus != nil {
		errMsg := agentErr.Error()
		if result.TimedOut {
			errMsg = fmt.Sprintf("agent timed out after %s (limit: %s)",
				duration.Round(time.Millisecond), agentTimeout.Round(time.Millisecond))
		}
		p.eventBus.Publish(core.NewAgentErrorEvent(entry.Agent.ID, entry.Agent.Name, errMsg))
	}

	log.WithFields(map[string]interface{}{
		"agent_id":   entry.Agent.ID,
		"agent_name": entry.Agent.Name,
		"duration":   duration.String(),
		"timeout":    agentTimeout.String(),
		"is_timeout": result.TimedOut,
	}).WithError(agentErr).Error("agent execution failed")

	return Response{
		AgentID:   entry.Agent.ID,
		AgentName: entry.Agent.Name,
		Error:     responseErr,
	}
}

func unwrapAgentError(err error) error {
	if err == nil {
		return nil
	}
	if agentErr, ok := errors.AsAgentError(err); ok {
		return agentErr.Unwrap()
	}
	return err
}

func (p *Pool) handleStreamingError(entry AgentEntry, elapsed time.Duration, result TimeoutResult, messageID string, finalIndex int) Response {
	result.Error = unwrapAgentError(result.Error)
	if IsCancellationError(result.Error) || (p.cancellation != nil && p.cancellation.IsCancelled(entry.Agent.ID)) {
		entry.State.SetCancelled()
		if p.eventBus != nil {
			p.eventBus.Publish(core.NewAgentCancelledEvent(
				entry.Agent.ID,
				entry.Agent.Name,
				"request cancelled",
			))
		}
		return Response{
			AgentID:   entry.Agent.ID,
			AgentName: entry.Agent.Name,
			Error:     ErrAgentCancelled,
		}
	}

	agentErr := errors.WrapError(entry.Agent.ID, entry.Agent.Name, result.Error)
	entry.State.SetError(agentErr.Error())
	responseErr := result.Error

	if result.TimedOut && p.timeoutStats != nil {
		p.timeoutStats.RecordTimeout(entry.Agent.ID, elapsed, result.PartialContent != "", false)
	}

	if entry.CircuitBreaker != nil {
		entry.CircuitBreaker.RecordFailure()
	}

	if p.eventBus != nil {
		errMsg := agentErr.Error()
		if result.TimedOut {
			errMsg = fmt.Sprintf("agent timed out after %s", elapsed.Round(time.Millisecond))
		}
		if result.PartialContent == "" {
			p.eventBus.Publish(core.NewMessageChunkEvent(core.MessageChunk{
				MessageID: messageID,
				AgentID:   entry.Agent.ID,
				AgentName: entry.Agent.Name,
				Content:   "",
				Index:     finalIndex,
				IsFinal:   true,
			}))
			p.eventBus.Publish(core.NewAgentErrorEvent(entry.Agent.ID, entry.Agent.Name, errMsg))
		}
	}

	log.WithFields(map[string]interface{}{
		"agent_id":   entry.Agent.ID,
		"agent_name": entry.Agent.Name,
		"duration":   elapsed.String(),
		"is_timeout": result.TimedOut,
	}).WithError(agentErr).Error("agent execution failed")

	if result.PartialContent != "" {
		metrics := result.Metrics
		if metrics == nil {
			metrics = &core.Metrics{}
		}
		if metrics.Duration == 0 {
			metrics.Duration = elapsed
		}

		msg := core.NewAgentMessage(entry.Agent.ID, entry.Agent.Name, result.PartialContent, metrics)
		msg.ID = messageID
		if p.eventBus != nil {
			p.eventBus.Publish(core.NewMessageChunkEvent(core.MessageChunk{
				MessageID: messageID,
				AgentID:   entry.Agent.ID,
				AgentName: entry.Agent.Name,
				Content:   "",
				Index:     finalIndex,
				IsFinal:   true,
			}))
			p.eventBus.Publish(core.NewAgentDoneEvent(entry.Agent.ID, entry.Agent.Name, msg))
		}
		return Response{
			AgentID:   entry.Agent.ID,
			AgentName: entry.Agent.Name,
			Message:   &msg,
			Error:     responseErr,
		}

	}

	return Response{
		AgentID:        entry.Agent.ID,
		AgentName:      entry.Agent.Name,
		Error:          responseErr,
		PartialContent: result.PartialContent,
	}
}

type chunkCollector struct {
	agentID    string
	agentName  string
	messageID  string
	eventBus   *events.Bus
	builder    strings.Builder
	chunkIndex int
}

func (c *chunkCollector) appendChunk(chunk string) {
	if chunk == "" {
		return
	}
	c.builder.WriteString(chunk)
	if c.eventBus != nil {
		c.eventBus.Publish(core.NewMessageChunkEvent(core.MessageChunk{
			MessageID: c.messageID,
			AgentID:   c.agentID,
			AgentName: c.agentName,
			Content:   chunk,
			Index:     c.chunkIndex,
			IsFinal:   false,
		}))
		c.chunkIndex++
	}
}

func (c *chunkCollector) String() string {
	return c.builder.String()
}

type streamWriter struct {
	collector   *chunkCollector
	partialChan chan<- string
}

func (w *streamWriter) Write(p []byte) (int, error) {
	chunk := string(p)
	if chunk == "" {
		return len(p), nil
	}

	w.collector.appendChunk(chunk)
	w.partialChan <- chunk
	return len(p), nil
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
	AgentID        string
	AgentName      string
	State          adapters.CircuitState
	FailureCount   int
	TimeUntilRetry time.Duration
	LastFailure    time.Time
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

// StartHealthMonitoring starts periodic health monitoring of agents.
func (p *Pool) StartHealthMonitoring() {
	if p.healthMonitor != nil {
		p.healthMonitor.Start()
	}
}

// StopHealthMonitoring stops periodic health monitoring.
func (p *Pool) StopHealthMonitoring() {
	if p.healthMonitor != nil {
		p.healthMonitor.Stop()
	}
}

// IsHealthMonitoringRunning returns true if health monitoring is active.
func (p *Pool) IsHealthMonitoringRunning() bool {
	if p.healthMonitor == nil {
		return false
	}
	return p.healthMonitor.IsRunning()
}

// PreflightCheck performs health checks on all agents before starting a conversation.
// Returns the preflight result with health status and warnings.
func (p *Pool) PreflightCheck(ctx context.Context) PreflightResult {
	if p.healthMonitor == nil {
		return PreflightResult{AllHealthy: true}
	}
	result := p.healthMonitor.PreflightCheck(ctx)

	// Emit preflight completed event
	if p.eventBus != nil {
		p.eventBus.Publish(core.NewPreflightCompletedEvent(
			result.AllHealthy,
			result.HealthyCount,
			result.UnhealthyCount,
			result.Duration,
			result.Warnings,
		))
	}

	return result
}

// CheckAgentHealth performs a health check on a specific agent.
func (p *Pool) CheckAgentHealth(ctx context.Context, agentID string) HealthCheckResult {
	if p.healthMonitor == nil {
		return HealthCheckResult{
			AgentID: agentID,
			Success: true,
		}
	}
	return p.healthMonitor.CheckAgentHealthSync(ctx, agentID)
}

// GetAgentHealthStatus returns the health status for a specific agent.
func (p *Pool) GetAgentHealthStatus(agentID string) (AgentHealthInfo, bool) {
	if p.healthMonitor == nil {
		return AgentHealthInfo{}, false
	}
	return p.healthMonitor.GetHealthStatus(agentID)
}

// GetAllAgentHealthStatus returns health status for all agents.
func (p *Pool) GetAllAgentHealthStatus() []AgentHealthInfo {
	if p.healthMonitor == nil {
		return nil
	}
	return p.healthMonitor.GetAllHealthStatus()
}

// GetHealthyAgentCount returns the count of healthy agents.
func (p *Pool) GetHealthyAgentCount() int {
	if p.healthMonitor == nil {
		return p.AgentCount()
	}
	return p.healthMonitor.GetHealthyAgentCount()
}

// GetUnhealthyAgentCount returns the count of unhealthy agents.
func (p *Pool) GetUnhealthyAgentCount() int {
	if p.healthMonitor == nil {
		return 0
	}
	return p.healthMonitor.GetUnhealthyAgentCount()
}

// IsAgentHealthy returns true if the specified agent is healthy.
func (p *Pool) IsAgentHealthy(agentID string) bool {
	if p.healthMonitor == nil {
		return true
	}
	return p.healthMonitor.IsAgentHealthy(agentID)
}

// ShouldWarnBeforeMessage returns true if user should be warned about unhealthy agents.
func (p *Pool) ShouldWarnBeforeMessage() (bool, []string) {
	if p.healthMonitor == nil {
		return false, nil
	}
	return p.healthMonitor.ShouldWarnBeforeMessage()
}

// RecordAgentActivity records activity for an agent (resets idle timer for health checks).
func (p *Pool) RecordAgentActivity(agentID string) {
	if p.healthMonitor != nil {
		p.healthMonitor.RecordActivity(agentID)
	}
}

// ResetAgentHealth resets the health status for an agent to unknown.
func (p *Pool) ResetAgentHealth(agentID string) {
	if p.healthMonitor != nil {
		p.healthMonitor.ResetAgent(agentID)
	}
}

// GetHealthMonitor returns the health monitor for advanced configuration.
func (p *Pool) GetHealthMonitor() *HealthMonitor {
	return p.healthMonitor
}

// GetHealthConfig returns the health configuration.
func (p *Pool) GetHealthConfig() HealthConfig {
	return p.healthConfig
}
