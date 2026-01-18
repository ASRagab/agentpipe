// Package pool provides the AgentPool for executing agent requests in parallel.
package pool

import (
	"context"
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

// AgentEntry pairs an agent with its adapter.
type AgentEntry struct {
	Agent   core.Agent
	Adapter adapters.AgentAdapter
	State   *core.AgentState
}

// Pool is the default implementation of AgentPool.
type Pool struct {
	agents   []AgentEntry
	eventBus *events.Bus
	timeout  time.Duration
	mu       sync.RWMutex
}

// NewPool creates a new agent pool.
func NewPool(eventBus *events.Bus, timeout time.Duration) *Pool {
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	return &Pool{
		agents:   make([]AgentEntry, 0),
		eventBus: eventBus,
		timeout:  timeout,
	}
}

// AddAgent adds an agent to the pool.
func (p *Pool) AddAgent(agent core.Agent, adapter adapters.AgentAdapter) {
	p.mu.Lock()
	defer p.mu.Unlock()

	state := core.NewAgentState(agent)
	p.agents = append(p.agents, AgentEntry{
		Agent:   agent,
		Adapter: adapter,
		State:   &state,
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

	// Create per-agent context with timeout
	agentCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	// Emit typing event
	if p.eventBus != nil {
		p.eventBus.Publish(core.NewAgentTypingEvent(agent.ID, agent.Name))
	}

	entry.State.SetTyping()

	log.WithFields(map[string]interface{}{
		"agent_id":   agent.ID,
		"agent_name": agent.Name,
		"model":      agent.Model,
	}).Debug("executing agent")

	startTime := time.Now()

	// Send message
	content, metrics, err := adapter.SendMessage(agentCtx, messages)
	duration := time.Since(startTime)

	if err != nil {
		entry.State.SetError(err.Error())

		// Emit error event
		if p.eventBus != nil {
			p.eventBus.Publish(core.NewAgentErrorEvent(agent.ID, agent.Name, err.Error()))
		}

		log.WithFields(map[string]interface{}{
			"agent_id":   agent.ID,
			"agent_name": agent.Name,
			"duration":   duration.String(),
		}).WithError(err).Error("agent execution failed")

		return Response{
			AgentID:   agent.ID,
			AgentName: agent.Name,
			Error:     err,
		}
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
