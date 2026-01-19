// Package pool provides the AgentPool for executing agent requests in parallel.
package pool

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ASRagab/agentpipe/pkg/core"
	"github.com/ASRagab/agentpipe/pkg/log"
)

// ErrAgentCancelled is returned when an agent's request is cancelled.
var ErrAgentCancelled = errors.New("agent request cancelled")

// ErrAgentNotFound is returned when the specified agent is not found.
var ErrAgentNotFound = errors.New("agent not found")

// ErrNoActiveRequests is returned when there are no active requests to cancel.
var ErrNoActiveRequests = errors.New("no active requests to cancel")

// CancellationInfo provides information about a pending cancellation.
type CancellationInfo struct {
	// AgentID is the ID of the agent with an active request.
	AgentID string
	// AgentName is the display name of the agent.
	AgentName string
	// StartTime is when the request started.
	StartTime time.Time
	// Cancelled indicates if cancellation was requested.
	Cancelled bool
}

// activeRequest tracks an in-flight agent request.
type activeRequest struct {
	agentID   string
	agentName string
	cancel    context.CancelFunc
	startTime time.Time
	cancelled bool
}

// EventBusRef is a reference to the event bus for emitting events.
type EventBusRef interface {
	Publish(event core.Event)
}

// CancellationManager tracks and manages cancellation of agent requests.
type CancellationManager struct {
	requests map[string]*activeRequest
	mu       sync.RWMutex
	eventBus EventBusRef
}

// NewCancellationManager creates a new cancellation manager.
func NewCancellationManager(eventBus EventBusRef) *CancellationManager {
	return &CancellationManager{
		requests: make(map[string]*activeRequest),
		eventBus: eventBus,
	}
}

// RegisterRequest registers a new active request for an agent.
// Returns the cancel function for the caller to use.
func (cm *CancellationManager) RegisterRequest(agentID, agentName string, cancel context.CancelFunc) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.requests[agentID] = &activeRequest{
		agentID:   agentID,
		agentName: agentName,
		cancel:    cancel,
		startTime: time.Now(),
		cancelled: false,
	}

	log.WithFields(map[string]interface{}{
		"agent_id":   agentID,
		"agent_name": agentName,
	}).Debug("Registered active request")
}

// UnregisterRequest removes a request from tracking.
func (cm *CancellationManager) UnregisterRequest(agentID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.requests[agentID]; exists {
		delete(cm.requests, agentID)
		log.WithFields(map[string]interface{}{
			"agent_id": agentID,
		}).Debug("Unregistered active request")
	}
}

// CancelAgent cancels a specific agent's request.
// Returns ErrAgentNotFound if the agent has no active request.
func (cm *CancellationManager) CancelAgent(agentID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	req, exists := cm.requests[agentID]
	if !exists {
		return ErrAgentNotFound
	}

	if req.cancelled {
		// Already cancelled
		return nil
	}

	req.cancelled = true
	req.cancel()

	log.WithFields(map[string]interface{}{
		"agent_id":   agentID,
		"agent_name": req.agentName,
		"elapsed":    time.Since(req.startTime).String(),
	}).Info("Cancelled agent request")

	// Emit cancellation event
	if cm.eventBus != nil {
		cm.eventBus.Publish(core.NewAgentCancelledEvent(
			agentID,
			req.agentName,
			"user requested cancellation",
		))
	}

	return nil
}

// CancelAll cancels all active agent requests.
// Returns the number of requests cancelled.
func (cm *CancellationManager) CancelAll() int {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cancelled := 0
	for agentID, req := range cm.requests {
		if !req.cancelled {
			req.cancelled = true
			req.cancel()
			cancelled++

			log.WithFields(map[string]interface{}{
				"agent_id":   agentID,
				"agent_name": req.agentName,
				"elapsed":    time.Since(req.startTime).String(),
			}).Info("Cancelled agent request (bulk cancellation)")

			// Emit cancellation event
			if cm.eventBus != nil {
				cm.eventBus.Publish(core.NewAgentCancelledEvent(
					agentID,
					req.agentName,
					"bulk cancellation requested",
				))
			}
		}
	}

	if cancelled > 0 {
		log.WithFields(map[string]interface{}{
			"count": cancelled,
		}).Info("Bulk cancellation completed")
	}

	return cancelled
}

// GetActiveRequests returns information about all active requests.
func (cm *CancellationManager) GetActiveRequests() []CancellationInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make([]CancellationInfo, 0, len(cm.requests))
	for _, req := range cm.requests {
		result = append(result, CancellationInfo{
			AgentID:   req.agentID,
			AgentName: req.agentName,
			StartTime: req.startTime,
			Cancelled: req.cancelled,
		})
	}
	return result
}

// HasActiveRequest checks if a specific agent has an active request.
func (cm *CancellationManager) HasActiveRequest(agentID string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	_, exists := cm.requests[agentID]
	return exists
}

// ActiveRequestCount returns the number of active (non-cancelled) requests.
func (cm *CancellationManager) ActiveRequestCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	count := 0
	for _, req := range cm.requests {
		if !req.cancelled {
			count++
		}
	}
	return count
}

// IsCancelled checks if a specific agent's request was cancelled.
func (cm *CancellationManager) IsCancelled(agentID string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if req, exists := cm.requests[agentID]; exists {
		return req.cancelled
	}
	return false
}

// Clear removes all tracked requests (used after execution completes).
func (cm *CancellationManager) Clear() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.requests = make(map[string]*activeRequest)
}

// IsCancellationError checks if an error is a cancellation error.
func IsCancellationError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrAgentCancelled) || errors.Is(err, context.Canceled)
}
