// Package pool provides the AgentPool for executing agent requests in parallel.
package pool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ASRagab/agentpipe/pkg/adapters"
	"github.com/ASRagab/agentpipe/pkg/core"
	"github.com/ASRagab/agentpipe/pkg/events"
	"github.com/ASRagab/agentpipe/pkg/log"
)

// HealthConfig contains configuration for agent health monitoring.
type HealthConfig struct {
	// PeriodicCheckInterval is how often to check idle agents (default: 60s).
	PeriodicCheckInterval time.Duration
	// PreflightCheckEnabled enables health checks before starting conversations.
	PreflightCheckEnabled bool
	// PreflightCheckTimeout is the timeout for preflight health checks (default: 5s).
	PreflightCheckTimeout time.Duration
	// HealthCheckTimeout is the timeout for individual health checks (default: 5s).
	HealthCheckTimeout time.Duration
	// WarnOnUnhealthy warns user when attempting to message unhealthy agents.
	WarnOnUnhealthy bool
	// UnhealthyThreshold is consecutive failed checks before marking unhealthy (default: 2).
	UnhealthyThreshold int
	// IdleThreshold is how long an agent must be idle before periodic checks start (default: 30s).
	IdleThreshold time.Duration
}

// DefaultHealthConfig returns the default health monitoring configuration.
func DefaultHealthConfig() HealthConfig {
	return HealthConfig{
		PeriodicCheckInterval: 60 * time.Second,
		PreflightCheckEnabled: true,
		PreflightCheckTimeout: 5 * time.Second,
		HealthCheckTimeout:    5 * time.Second,
		WarnOnUnhealthy:       true,
		UnhealthyThreshold:    2,
		IdleThreshold:         30 * time.Second,
	}
}

// AgentHealthStatus represents the health state of an agent.
type AgentHealthStatus string

const (
	// HealthStatusUnknown means health has not been checked yet.
	HealthStatusUnknown AgentHealthStatus = "unknown"
	// HealthStatusHealthy means the agent is responding normally.
	HealthStatusHealthy AgentHealthStatus = "healthy"
	// HealthStatusUnhealthy means the agent has failed recent health checks.
	HealthStatusUnhealthy AgentHealthStatus = "unhealthy"
	// HealthStatusDegraded means the agent is slow but responding.
	HealthStatusDegraded AgentHealthStatus = "degraded"
	// HealthStatusChecking means a health check is in progress.
	HealthStatusChecking AgentHealthStatus = "checking"
)

// String returns the string representation of the health status.
func (s AgentHealthStatus) String() string {
	return string(s)
}

// IsHealthy returns true if the status indicates the agent can be used.
func (s AgentHealthStatus) IsHealthy() bool {
	return s == HealthStatusHealthy || s == HealthStatusDegraded || s == HealthStatusUnknown
}

// AgentHealthInfo contains detailed health information for an agent.
type AgentHealthInfo struct {
	// AgentID is the unique identifier of the agent.
	AgentID string
	// AgentName is the display name of the agent.
	AgentName string
	// Status is the current health status.
	Status AgentHealthStatus
	// LastCheck is when the last health check was performed.
	LastCheck time.Time
	// LastSuccess is when the last successful health check occurred.
	LastSuccess time.Time
	// LastError is the most recent health check error (if any).
	LastError string
	// ConsecutiveFailures is the number of consecutive failed health checks.
	ConsecutiveFailures int
	// ResponseTime is the last measured response time.
	ResponseTime time.Duration
	// CheckCount is the total number of health checks performed.
	CheckCount int
	// LastActivity is when the agent last had any activity.
	LastActivity time.Time
}

// HealthCheckResult contains the result of a single health check.
type HealthCheckResult struct {
	AgentID      string
	AgentName    string
	Success      bool
	ResponseTime time.Duration
	Error        error
	Timestamp    time.Time
}

// PreflightResult contains the results of a preflight health check.
type PreflightResult struct {
	AllHealthy     bool
	HealthyCount   int
	UnhealthyCount int
	Results        map[string]HealthCheckResult
	Duration       time.Duration
	Warnings       []string
}

// HealthMonitor manages health monitoring for agents in a pool.
type HealthMonitor struct {
	config     HealthConfig
	eventBus   *events.Bus
	healthInfo map[string]*AgentHealthInfo
	adapters   map[string]adapters.AgentAdapter
	stopChan   chan struct{}
	wg         sync.WaitGroup
	running    bool
	mu         sync.RWMutex
}

// NewHealthMonitor creates a new health monitor.
func NewHealthMonitor(config HealthConfig, eventBus *events.Bus) *HealthMonitor {
	return &HealthMonitor{
		config:     config,
		eventBus:   eventBus,
		healthInfo: make(map[string]*AgentHealthInfo),
		adapters:   make(map[string]adapters.AgentAdapter),
		stopChan:   make(chan struct{}),
	}
}

// RegisterAgent registers an agent with the health monitor.
func (m *HealthMonitor) RegisterAgent(agent core.Agent, adapter adapters.AgentAdapter) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.adapters[agent.ID] = adapter
	m.healthInfo[agent.ID] = &AgentHealthInfo{
		AgentID:      agent.ID,
		AgentName:    agent.Name,
		Status:       HealthStatusUnknown,
		LastActivity: time.Now(),
	}

	log.WithFields(map[string]interface{}{
		"agent_id":   agent.ID,
		"agent_name": agent.Name,
	}).Debug("agent registered with health monitor")
}

// UnregisterAgent removes an agent from the health monitor.
func (m *HealthMonitor) UnregisterAgent(agentID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.adapters, agentID)
	delete(m.healthInfo, agentID)
}

// Start begins periodic health monitoring.
func (m *HealthMonitor) Start() {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.stopChan = make(chan struct{})
	m.mu.Unlock()

	m.wg.Add(1)
	go m.runPeriodicChecks()

	log.WithFields(map[string]interface{}{
		"interval": m.config.PeriodicCheckInterval.String(),
	}).Info("health monitor started")
}

// Stop halts periodic health monitoring.
func (m *HealthMonitor) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	m.running = false
	close(m.stopChan)
	m.mu.Unlock()

	m.wg.Wait()
	log.Info("health monitor stopped")
}

// IsRunning returns true if the health monitor is actively running.
func (m *HealthMonitor) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// runPeriodicChecks runs the periodic health check loop.
func (m *HealthMonitor) runPeriodicChecks() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.PeriodicCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkIdleAgents()
		case <-m.stopChan:
			return
		}
	}
}

// checkIdleAgents performs health checks on agents that have been idle.
func (m *HealthMonitor) checkIdleAgents() {
	m.mu.RLock()
	agentIDs := make([]string, 0)
	now := time.Now()

	for agentID, info := range m.healthInfo {
		// Only check agents that have been idle for longer than threshold
		if now.Sub(info.LastActivity) > m.config.IdleThreshold {
			agentIDs = append(agentIDs, agentID)
		}
	}
	m.mu.RUnlock()

	if len(agentIDs) == 0 {
		return
	}

	log.WithFields(map[string]interface{}{
		"agent_count": len(agentIDs),
	}).Debug("performing periodic health checks on idle agents")

	ctx, cancel := context.WithTimeout(context.Background(), m.config.HealthCheckTimeout*time.Duration(len(agentIDs)))
	defer cancel()

	for _, agentID := range agentIDs {
		m.checkAgentHealth(ctx, agentID)
	}
}

// checkAgentHealth performs a health check on a single agent.
func (m *HealthMonitor) checkAgentHealth(ctx context.Context, agentID string) HealthCheckResult {
	m.mu.RLock()
	adapter, ok := m.adapters[agentID]
	info := m.healthInfo[agentID]
	if !ok || info == nil {
		m.mu.RUnlock()
		return HealthCheckResult{
			AgentID:   agentID,
			Success:   false,
			Error:     fmt.Errorf("agent not found"),
			Timestamp: time.Now(),
		}
	}
	agentName := info.AgentName
	m.mu.RUnlock()

	// Mark as checking
	m.mu.Lock()
	if info, exists := m.healthInfo[agentID]; exists {
		info.Status = HealthStatusChecking
	}
	m.mu.Unlock()

	// Perform the health check
	start := time.Now()
	checkCtx, cancel := context.WithTimeout(ctx, m.config.HealthCheckTimeout)
	defer cancel()

	err := adapter.HealthCheck(checkCtx)
	responseTime := time.Since(start)

	result := HealthCheckResult{
		AgentID:      agentID,
		AgentName:    agentName,
		Success:      err == nil,
		ResponseTime: responseTime,
		Error:        err,
		Timestamp:    time.Now(),
	}

	// Update health info
	m.mu.Lock()
	info, exists := m.healthInfo[agentID]
	if !exists {
		m.mu.Unlock()
		return result
	}

	info.LastCheck = result.Timestamp
	info.CheckCount++
	info.ResponseTime = responseTime

	if result.Success {
		info.LastSuccess = result.Timestamp
		info.ConsecutiveFailures = 0
		info.LastError = ""

		// Determine if degraded based on response time
		if responseTime > m.config.HealthCheckTimeout/2 {
			info.Status = HealthStatusDegraded
		} else {
			info.Status = HealthStatusHealthy
		}
	} else {
		info.ConsecutiveFailures++
		if result.Error != nil {
			info.LastError = result.Error.Error()
		}

		if info.ConsecutiveFailures >= m.config.UnhealthyThreshold {
			info.Status = HealthStatusUnhealthy
		}
	}
	m.mu.Unlock()

	// Emit events
	if m.eventBus != nil {
		if result.Success {
			m.eventBus.Publish(core.NewAgentHealthyEvent(agentID, agentName, responseTime))
		} else {
			errMsg := ""
			if result.Error != nil {
				errMsg = result.Error.Error()
			}
			m.eventBus.Publish(core.NewAgentUnhealthyEvent(agentID, agentName, errMsg))
		}
	}

	log.WithFields(map[string]interface{}{
		"agent_id":      agentID,
		"agent_name":    agentName,
		"success":       result.Success,
		"response_time": responseTime.String(),
		"status":        info.Status.String(),
	}).Debug("health check completed")

	return result
}

// CheckAgentHealthSync performs a synchronous health check on a specific agent.
func (m *HealthMonitor) CheckAgentHealthSync(ctx context.Context, agentID string) HealthCheckResult {
	return m.checkAgentHealth(ctx, agentID)
}

// CheckAllHealthSync performs synchronous health checks on all registered agents.
func (m *HealthMonitor) CheckAllHealthSync(ctx context.Context) map[string]HealthCheckResult {
	m.mu.RLock()
	agentIDs := make([]string, 0, len(m.adapters))
	for id := range m.adapters {
		agentIDs = append(agentIDs, id)
	}
	m.mu.RUnlock()

	results := make(map[string]HealthCheckResult)
	var wg sync.WaitGroup
	var resultMu sync.Mutex

	for _, id := range agentIDs {
		wg.Add(1)
		go func(agentID string) {
			defer wg.Done()
			result := m.checkAgentHealth(ctx, agentID)
			resultMu.Lock()
			results[agentID] = result
			resultMu.Unlock()
		}(id)
	}

	wg.Wait()
	return results
}

// PreflightCheck performs health checks on all agents before starting a conversation.
// Returns a PreflightResult with the status of all agents and warnings.
func (m *HealthMonitor) PreflightCheck(ctx context.Context) PreflightResult {
	start := time.Now()

	checkCtx, cancel := context.WithTimeout(ctx, m.config.PreflightCheckTimeout*2)
	defer cancel()

	results := m.CheckAllHealthSync(checkCtx)

	healthyCount := 0
	unhealthyCount := 0
	var warnings []string

	for agentID, result := range results {
		if result.Success {
			healthyCount++
		} else {
			unhealthyCount++
			errMsg := "unknown error"
			if result.Error != nil {
				errMsg = result.Error.Error()
			}
			warnings = append(warnings, fmt.Sprintf("%s (%s): %s", result.AgentName, agentID, errMsg))
		}
	}

	return PreflightResult{
		AllHealthy:     unhealthyCount == 0,
		HealthyCount:   healthyCount,
		UnhealthyCount: unhealthyCount,
		Results:        results,
		Duration:       time.Since(start),
		Warnings:       warnings,
	}
}

// GetHealthStatus returns the current health status for an agent.
func (m *HealthMonitor) GetHealthStatus(agentID string) (AgentHealthInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info, ok := m.healthInfo[agentID]
	if !ok {
		return AgentHealthInfo{}, false
	}
	return *info, true
}

// GetAllHealthStatus returns the health status for all registered agents.
func (m *HealthMonitor) GetAllHealthStatus() []AgentHealthInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]AgentHealthInfo, 0, len(m.healthInfo))
	for _, info := range m.healthInfo {
		result = append(result, *info)
	}
	return result
}

// GetHealthyAgentCount returns the count of agents with healthy status.
func (m *HealthMonitor) GetHealthyAgentCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, info := range m.healthInfo {
		if info.Status.IsHealthy() {
			count++
		}
	}
	return count
}

// GetUnhealthyAgentCount returns the count of agents with unhealthy status.
func (m *HealthMonitor) GetUnhealthyAgentCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, info := range m.healthInfo {
		if info.Status == HealthStatusUnhealthy {
			count++
		}
	}
	return count
}

// IsAgentHealthy returns true if the specified agent is healthy.
func (m *HealthMonitor) IsAgentHealthy(agentID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info, ok := m.healthInfo[agentID]
	if !ok {
		return false
	}
	return info.Status.IsHealthy()
}

// RecordActivity records activity for an agent (resets idle timer).
func (m *HealthMonitor) RecordActivity(agentID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if info, ok := m.healthInfo[agentID]; ok {
		info.LastActivity = time.Now()
	}
}

// ResetAgent resets the health status for an agent to unknown.
func (m *HealthMonitor) ResetAgent(agentID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if info, ok := m.healthInfo[agentID]; ok {
		info.Status = HealthStatusUnknown
		info.ConsecutiveFailures = 0
		info.LastError = ""
		info.LastActivity = time.Now()
	}
}

// ShouldWarnBeforeMessage returns true if user should be warned about unhealthy agents.
func (m *HealthMonitor) ShouldWarnBeforeMessage() (bool, []string) {
	if !m.config.WarnOnUnhealthy {
		return false, nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var unhealthyAgents []string
	for _, info := range m.healthInfo {
		if info.Status == HealthStatusUnhealthy {
			unhealthyAgents = append(unhealthyAgents, info.AgentName)
		}
	}

	return len(unhealthyAgents) > 0, unhealthyAgents
}

// GetConfig returns the current health configuration.
func (m *HealthMonitor) GetConfig() HealthConfig {
	return m.config
}

// UpdateConfig updates the health configuration (requires restart for some settings).
func (m *HealthMonitor) UpdateConfig(config HealthConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = config
}
