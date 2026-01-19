package pool

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/ASRagab/agentpipe/pkg/core"
	"github.com/ASRagab/agentpipe/pkg/events"
)

// mockHealthAdapter is a mock adapter for testing health checks.
type mockHealthAdapter struct {
	healthy      bool
	responseTime time.Duration
	healthError  error
	model        string
	mu           sync.Mutex
}

func newMockHealthAdapter(healthy bool) *mockHealthAdapter {
	return &mockHealthAdapter{
		healthy:      healthy,
		responseTime: 10 * time.Millisecond,
		model:        "test-model",
	}
}

func (m *mockHealthAdapter) Initialize(_ core.Agent) error {
	return nil
}

func (m *mockHealthAdapter) SendMessage(_ context.Context, _ []core.Message, _ *core.ConversationContext) (string, *core.Metrics, error) {
	return "response", &core.Metrics{}, nil
}

func (m *mockHealthAdapter) StreamMessage(_ context.Context, _ []core.Message, _ io.Writer, _ *core.ConversationContext) (*core.Metrics, error) {
	return &core.Metrics{}, nil
}

func (m *mockHealthAdapter) IsAvailable() bool {
	return true
}

func (m *mockHealthAdapter) GetModel() string {
	return m.model
}

func (m *mockHealthAdapter) HealthCheck(ctx context.Context) error {
	m.mu.Lock()
	healthy := m.healthy
	responseTime := m.responseTime
	healthError := m.healthError
	m.mu.Unlock()

	select {
	case <-time.After(responseTime):
		if !healthy {
			if healthError != nil {
				return healthError
			}
			return errors.New("health check failed")
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *mockHealthAdapter) SetHealthy(healthy bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthy = healthy
}

func (m *mockHealthAdapter) SetResponseTime(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responseTime = d
}

func (m *mockHealthAdapter) SetHealthError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthError = err
}

func TestDefaultHealthConfig(t *testing.T) {
	config := DefaultHealthConfig()

	if config.PeriodicCheckInterval != 60*time.Second {
		t.Errorf("expected PeriodicCheckInterval=60s, got %v", config.PeriodicCheckInterval)
	}
	if !config.PreflightCheckEnabled {
		t.Error("expected PreflightCheckEnabled=true")
	}
	if config.PreflightCheckTimeout != 5*time.Second {
		t.Errorf("expected PreflightCheckTimeout=5s, got %v", config.PreflightCheckTimeout)
	}
	if config.HealthCheckTimeout != 5*time.Second {
		t.Errorf("expected HealthCheckTimeout=5s, got %v", config.HealthCheckTimeout)
	}
	if !config.WarnOnUnhealthy {
		t.Error("expected WarnOnUnhealthy=true")
	}
	if config.UnhealthyThreshold != 2 {
		t.Errorf("expected UnhealthyThreshold=2, got %d", config.UnhealthyThreshold)
	}
	if config.IdleThreshold != 30*time.Second {
		t.Errorf("expected IdleThreshold=30s, got %v", config.IdleThreshold)
	}
}

func TestAgentHealthStatus_IsHealthy(t *testing.T) {
	tests := []struct {
		status   AgentHealthStatus
		expected bool
	}{
		{HealthStatusUnknown, true},
		{HealthStatusHealthy, true},
		{HealthStatusDegraded, true},
		{HealthStatusUnhealthy, false},
		{HealthStatusChecking, false},
	}

	for _, tc := range tests {
		t.Run(string(tc.status), func(t *testing.T) {
			if tc.status.IsHealthy() != tc.expected {
				t.Errorf("expected IsHealthy()=%v for status %s", tc.expected, tc.status)
			}
		})
	}
}

func TestHealthMonitor_RegisterAgent(t *testing.T) {
	bus := events.NewBus()
	config := DefaultHealthConfig()
	monitor := NewHealthMonitor(config, bus)

	agent := core.NewAgent("agent-1", "test", "Agent 1", "model-1", "mock")
	adapter := newMockHealthAdapter(true)

	monitor.RegisterAgent(agent, adapter)

	info, ok := monitor.GetHealthStatus("agent-1")
	if !ok {
		t.Fatal("expected to find health status for agent-1")
	}
	if info.Status != HealthStatusUnknown {
		t.Errorf("expected initial status=unknown, got %s", info.Status)
	}
	if info.AgentID != "agent-1" {
		t.Errorf("expected AgentID=agent-1, got %s", info.AgentID)
	}
	if info.AgentName != "Agent 1" {
		t.Errorf("expected AgentName=Agent 1, got %s", info.AgentName)
	}
}

func TestHealthMonitor_UnregisterAgent(t *testing.T) {
	bus := events.NewBus()
	config := DefaultHealthConfig()
	monitor := NewHealthMonitor(config, bus)

	agent := core.NewAgent("agent-1", "test", "Agent 1", "model-1", "mock")
	adapter := newMockHealthAdapter(true)

	monitor.RegisterAgent(agent, adapter)
	monitor.UnregisterAgent("agent-1")

	_, ok := monitor.GetHealthStatus("agent-1")
	if ok {
		t.Error("expected agent to be unregistered")
	}
}

func TestHealthMonitor_CheckAgentHealthSync_Success(t *testing.T) {
	bus := events.NewBus()
	config := DefaultHealthConfig()
	config.HealthCheckTimeout = 1 * time.Second
	monitor := NewHealthMonitor(config, bus)

	agent := core.NewAgent("agent-1", "test", "Agent 1", "model-1", "mock")
	adapter := newMockHealthAdapter(true)
	adapter.SetResponseTime(10 * time.Millisecond)

	monitor.RegisterAgent(agent, adapter)

	ctx := context.Background()
	result := monitor.CheckAgentHealthSync(ctx, "agent-1")

	if !result.Success {
		t.Error("expected health check to succeed")
	}
	if result.Error != nil {
		t.Errorf("expected no error, got %v", result.Error)
	}
	if result.ResponseTime < 10*time.Millisecond {
		t.Errorf("expected response time >= 10ms, got %v", result.ResponseTime)
	}

	// Check that status was updated
	info, _ := monitor.GetHealthStatus("agent-1")
	if info.Status != HealthStatusHealthy {
		t.Errorf("expected status=healthy, got %s", info.Status)
	}
}

func TestHealthMonitor_CheckAgentHealthSync_Failure(t *testing.T) {
	bus := events.NewBus()
	config := DefaultHealthConfig()
	config.HealthCheckTimeout = 1 * time.Second
	config.UnhealthyThreshold = 1 // Mark unhealthy after 1 failure
	monitor := NewHealthMonitor(config, bus)

	agent := core.NewAgent("agent-1", "test", "Agent 1", "model-1", "mock")
	adapter := newMockHealthAdapter(false)
	adapter.SetHealthError(errors.New("connection refused"))

	monitor.RegisterAgent(agent, adapter)

	ctx := context.Background()
	result := monitor.CheckAgentHealthSync(ctx, "agent-1")

	if result.Success {
		t.Error("expected health check to fail")
	}
	if result.Error == nil {
		t.Error("expected an error")
	}

	// Check that status was updated to unhealthy
	info, _ := monitor.GetHealthStatus("agent-1")
	if info.Status != HealthStatusUnhealthy {
		t.Errorf("expected status=unhealthy, got %s", info.Status)
	}
	if info.LastError == "" {
		t.Error("expected LastError to be set")
	}
}

func TestHealthMonitor_CheckAgentHealthSync_Timeout(t *testing.T) {
	bus := events.NewBus()
	config := DefaultHealthConfig()
	config.HealthCheckTimeout = 50 * time.Millisecond
	monitor := NewHealthMonitor(config, bus)

	agent := core.NewAgent("agent-1", "test", "Agent 1", "model-1", "mock")
	adapter := newMockHealthAdapter(true)
	adapter.SetResponseTime(200 * time.Millisecond) // Slower than timeout

	monitor.RegisterAgent(agent, adapter)

	ctx := context.Background()
	result := monitor.CheckAgentHealthSync(ctx, "agent-1")

	if result.Success {
		t.Error("expected health check to timeout")
	}
}

func TestHealthMonitor_CheckAllHealthSync(t *testing.T) {
	bus := events.NewBus()
	config := DefaultHealthConfig()
	config.HealthCheckTimeout = 1 * time.Second
	monitor := NewHealthMonitor(config, bus)

	// Register 3 agents: 2 healthy, 1 unhealthy
	for i := 1; i <= 3; i++ {
		agent := core.NewAgent(
			"agent-"+string(rune('0'+i)),
			"test",
			"Agent "+string(rune('0'+i)),
			"model-1",
			"mock",
		)
		adapter := newMockHealthAdapter(i <= 2) // First 2 are healthy
		monitor.RegisterAgent(agent, adapter)
	}

	ctx := context.Background()
	results := monitor.CheckAllHealthSync(ctx)

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}
	if successCount != 2 {
		t.Errorf("expected 2 successful checks, got %d", successCount)
	}
}

func TestHealthMonitor_PreflightCheck(t *testing.T) {
	bus := events.NewBus()
	config := DefaultHealthConfig()
	config.PreflightCheckTimeout = 1 * time.Second
	config.HealthCheckTimeout = 500 * time.Millisecond
	monitor := NewHealthMonitor(config, bus)

	// Register 2 healthy agents
	for i := 1; i <= 2; i++ {
		agent := core.NewAgent(
			"agent-"+string(rune('0'+i)),
			"test",
			"Agent "+string(rune('0'+i)),
			"model-1",
			"mock",
		)
		adapter := newMockHealthAdapter(true)
		monitor.RegisterAgent(agent, adapter)
	}

	ctx := context.Background()
	result := monitor.PreflightCheck(ctx)

	if !result.AllHealthy {
		t.Error("expected AllHealthy=true")
	}
	if result.HealthyCount != 2 {
		t.Errorf("expected HealthyCount=2, got %d", result.HealthyCount)
	}
	if result.UnhealthyCount != 0 {
		t.Errorf("expected UnhealthyCount=0, got %d", result.UnhealthyCount)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings, got %v", result.Warnings)
	}
}

func TestHealthMonitor_PreflightCheck_WithUnhealthy(t *testing.T) {
	bus := events.NewBus()
	config := DefaultHealthConfig()
	config.PreflightCheckTimeout = 1 * time.Second
	config.HealthCheckTimeout = 500 * time.Millisecond
	monitor := NewHealthMonitor(config, bus)

	// Register 1 healthy, 1 unhealthy
	healthy := core.NewAgent("healthy-1", "test", "Healthy", "model-1", "mock")
	healthyAdapter := newMockHealthAdapter(true)
	monitor.RegisterAgent(healthy, healthyAdapter)

	unhealthy := core.NewAgent("unhealthy-1", "test", "Unhealthy", "model-1", "mock")
	unhealthyAdapter := newMockHealthAdapter(false)
	monitor.RegisterAgent(unhealthy, unhealthyAdapter)

	ctx := context.Background()
	result := monitor.PreflightCheck(ctx)

	if result.AllHealthy {
		t.Error("expected AllHealthy=false")
	}
	if result.HealthyCount != 1 {
		t.Errorf("expected HealthyCount=1, got %d", result.HealthyCount)
	}
	if result.UnhealthyCount != 1 {
		t.Errorf("expected UnhealthyCount=1, got %d", result.UnhealthyCount)
	}
	if len(result.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(result.Warnings))
	}
}

func TestHealthMonitor_StartAndStop(t *testing.T) {
	bus := events.NewBus()
	config := DefaultHealthConfig()
	config.PeriodicCheckInterval = 50 * time.Millisecond
	monitor := NewHealthMonitor(config, bus)

	if monitor.IsRunning() {
		t.Error("expected monitor not running initially")
	}

	monitor.Start()
	if !monitor.IsRunning() {
		t.Error("expected monitor to be running")
	}

	// Starting again should be a no-op
	monitor.Start()
	if !monitor.IsRunning() {
		t.Error("expected monitor to still be running")
	}

	monitor.Stop()
	if monitor.IsRunning() {
		t.Error("expected monitor to be stopped")
	}

	// Stopping again should be a no-op
	monitor.Stop()
	if monitor.IsRunning() {
		t.Error("expected monitor to remain stopped")
	}
}

func TestHealthMonitor_RecordActivity(t *testing.T) {
	bus := events.NewBus()
	config := DefaultHealthConfig()
	monitor := NewHealthMonitor(config, bus)

	agent := core.NewAgent("agent-1", "test", "Agent 1", "model-1", "mock")
	adapter := newMockHealthAdapter(true)
	monitor.RegisterAgent(agent, adapter)

	// Get initial activity time
	info1, _ := monitor.GetHealthStatus("agent-1")
	initialActivity := info1.LastActivity

	// Wait a bit and record activity
	time.Sleep(10 * time.Millisecond)
	monitor.RecordActivity("agent-1")

	info2, _ := monitor.GetHealthStatus("agent-1")
	if !info2.LastActivity.After(initialActivity) {
		t.Error("expected LastActivity to be updated")
	}
}

func TestHealthMonitor_ResetAgent(t *testing.T) {
	bus := events.NewBus()
	config := DefaultHealthConfig()
	config.UnhealthyThreshold = 1
	config.HealthCheckTimeout = 1 * time.Second
	monitor := NewHealthMonitor(config, bus)

	agent := core.NewAgent("agent-1", "test", "Agent 1", "model-1", "mock")
	adapter := newMockHealthAdapter(false)
	monitor.RegisterAgent(agent, adapter)

	// Fail the health check to make it unhealthy
	ctx := context.Background()
	monitor.CheckAgentHealthSync(ctx, "agent-1")

	info1, _ := monitor.GetHealthStatus("agent-1")
	if info1.Status != HealthStatusUnhealthy {
		t.Errorf("expected status=unhealthy, got %s", info1.Status)
	}

	// Reset the agent
	monitor.ResetAgent("agent-1")

	info2, _ := monitor.GetHealthStatus("agent-1")
	if info2.Status != HealthStatusUnknown {
		t.Errorf("expected status=unknown after reset, got %s", info2.Status)
	}
	if info2.ConsecutiveFailures != 0 {
		t.Errorf("expected ConsecutiveFailures=0, got %d", info2.ConsecutiveFailures)
	}
}

func TestHealthMonitor_ShouldWarnBeforeMessage(t *testing.T) {
	bus := events.NewBus()
	config := DefaultHealthConfig()
	config.WarnOnUnhealthy = true
	config.UnhealthyThreshold = 1
	config.HealthCheckTimeout = 1 * time.Second
	monitor := NewHealthMonitor(config, bus)

	// Initially no warnings
	shouldWarn, agents := monitor.ShouldWarnBeforeMessage()
	if shouldWarn {
		t.Error("expected no warning with no agents")
	}

	// Add a healthy agent
	healthy := core.NewAgent("healthy-1", "test", "Healthy", "model-1", "mock")
	healthyAdapter := newMockHealthAdapter(true)
	monitor.RegisterAgent(healthy, healthyAdapter)

	ctx := context.Background()
	monitor.CheckAgentHealthSync(ctx, "healthy-1")

	shouldWarn, agents = monitor.ShouldWarnBeforeMessage()
	if shouldWarn {
		t.Error("expected no warning with healthy agent")
	}

	// Add an unhealthy agent
	unhealthy := core.NewAgent("unhealthy-1", "test", "Unhealthy", "model-1", "mock")
	unhealthyAdapter := newMockHealthAdapter(false)
	monitor.RegisterAgent(unhealthy, unhealthyAdapter)
	monitor.CheckAgentHealthSync(ctx, "unhealthy-1")

	shouldWarn, agents = monitor.ShouldWarnBeforeMessage()
	if !shouldWarn {
		t.Error("expected warning with unhealthy agent")
	}
	if len(agents) != 1 || agents[0] != "Unhealthy" {
		t.Errorf("expected [Unhealthy], got %v", agents)
	}
}

func TestHealthMonitor_ShouldWarnBeforeMessage_Disabled(t *testing.T) {
	bus := events.NewBus()
	config := DefaultHealthConfig()
	config.WarnOnUnhealthy = false
	config.UnhealthyThreshold = 1
	config.HealthCheckTimeout = 1 * time.Second
	monitor := NewHealthMonitor(config, bus)

	// Add unhealthy agent
	unhealthy := core.NewAgent("unhealthy-1", "test", "Unhealthy", "model-1", "mock")
	unhealthyAdapter := newMockHealthAdapter(false)
	monitor.RegisterAgent(unhealthy, unhealthyAdapter)

	ctx := context.Background()
	monitor.CheckAgentHealthSync(ctx, "unhealthy-1")

	shouldWarn, _ := monitor.ShouldWarnBeforeMessage()
	if shouldWarn {
		t.Error("expected no warning when WarnOnUnhealthy=false")
	}
}

func TestHealthMonitor_GetHealthyAgentCount(t *testing.T) {
	bus := events.NewBus()
	config := DefaultHealthConfig()
	config.UnhealthyThreshold = 1
	config.HealthCheckTimeout = 1 * time.Second
	monitor := NewHealthMonitor(config, bus)

	// Add 2 healthy, 1 unhealthy
	for i := 1; i <= 2; i++ {
		agent := core.NewAgent("healthy-"+string(rune('0'+i)), "test", "Healthy "+string(rune('0'+i)), "model-1", "mock")
		adapter := newMockHealthAdapter(true)
		monitor.RegisterAgent(agent, adapter)
	}
	unhealthy := core.NewAgent("unhealthy-1", "test", "Unhealthy", "model-1", "mock")
	unhealthyAdapter := newMockHealthAdapter(false)
	monitor.RegisterAgent(unhealthy, unhealthyAdapter)

	ctx := context.Background()

	// Before checking, all should count as "healthy" (unknown is treated as healthy)
	if monitor.GetHealthyAgentCount() != 3 {
		t.Errorf("expected 3 healthy (unknown counts as healthy), got %d", monitor.GetHealthyAgentCount())
	}

	// Check all agents
	for _, id := range []string{"healthy-1", "healthy-2", "unhealthy-1"} {
		monitor.CheckAgentHealthSync(ctx, id)
	}

	if monitor.GetHealthyAgentCount() != 2 {
		t.Errorf("expected 2 healthy, got %d", monitor.GetHealthyAgentCount())
	}
	if monitor.GetUnhealthyAgentCount() != 1 {
		t.Errorf("expected 1 unhealthy, got %d", monitor.GetUnhealthyAgentCount())
	}
}

func TestHealthMonitor_Events(t *testing.T) {
	bus := events.NewBus()
	config := DefaultHealthConfig()
	config.HealthCheckTimeout = 1 * time.Second
	config.UnhealthyThreshold = 1
	monitor := NewHealthMonitor(config, bus)

	var healthyEvents, unhealthyEvents int
	var mu sync.Mutex

	bus.Subscribe(core.EventAgentHealthy, func(e core.Event) {
		mu.Lock()
		healthyEvents++
		mu.Unlock()
	})
	bus.Subscribe(core.EventAgentUnhealthy, func(e core.Event) {
		mu.Lock()
		unhealthyEvents++
		mu.Unlock()
	})

	// Add and check a healthy agent
	healthy := core.NewAgent("healthy-1", "test", "Healthy", "model-1", "mock")
	healthyAdapter := newMockHealthAdapter(true)
	monitor.RegisterAgent(healthy, healthyAdapter)

	ctx := context.Background()
	monitor.CheckAgentHealthSync(ctx, "healthy-1")

	// Add and check an unhealthy agent
	unhealthy := core.NewAgent("unhealthy-1", "test", "Unhealthy", "model-1", "mock")
	unhealthyAdapter := newMockHealthAdapter(false)
	monitor.RegisterAgent(unhealthy, unhealthyAdapter)
	monitor.CheckAgentHealthSync(ctx, "unhealthy-1")

	// Give events time to propagate
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if healthyEvents != 1 {
		t.Errorf("expected 1 healthy event, got %d", healthyEvents)
	}
	if unhealthyEvents != 1 {
		t.Errorf("expected 1 unhealthy event, got %d", unhealthyEvents)
	}
}

func TestHealthMonitor_DegradedStatus(t *testing.T) {
	bus := events.NewBus()
	config := DefaultHealthConfig()
	config.HealthCheckTimeout = 200 * time.Millisecond // 200ms timeout
	monitor := NewHealthMonitor(config, bus)

	agent := core.NewAgent("agent-1", "test", "Agent 1", "model-1", "mock")
	adapter := newMockHealthAdapter(true)
	// Response time > timeout/2 (100ms) but < timeout should be degraded
	adapter.SetResponseTime(120 * time.Millisecond)
	monitor.RegisterAgent(agent, adapter)

	ctx := context.Background()
	result := monitor.CheckAgentHealthSync(ctx, "agent-1")

	if !result.Success {
		t.Error("expected health check to succeed")
	}

	info, _ := monitor.GetHealthStatus("agent-1")
	if info.Status != HealthStatusDegraded {
		t.Errorf("expected status=degraded for slow response, got %s", info.Status)
	}
}

func TestHealthMonitor_ConsecutiveFailures(t *testing.T) {
	bus := events.NewBus()
	config := DefaultHealthConfig()
	config.HealthCheckTimeout = 1 * time.Second
	config.UnhealthyThreshold = 3 // Need 3 failures to be unhealthy
	monitor := NewHealthMonitor(config, bus)

	agent := core.NewAgent("agent-1", "test", "Agent 1", "model-1", "mock")
	adapter := newMockHealthAdapter(false)
	monitor.RegisterAgent(agent, adapter)

	ctx := context.Background()

	// First failure - should not be unhealthy yet
	monitor.CheckAgentHealthSync(ctx, "agent-1")
	info1, _ := monitor.GetHealthStatus("agent-1")
	if info1.Status == HealthStatusUnhealthy {
		t.Error("should not be unhealthy after 1 failure")
	}
	if info1.ConsecutiveFailures != 1 {
		t.Errorf("expected 1 consecutive failure, got %d", info1.ConsecutiveFailures)
	}

	// Second failure
	monitor.CheckAgentHealthSync(ctx, "agent-1")
	info2, _ := monitor.GetHealthStatus("agent-1")
	if info2.Status == HealthStatusUnhealthy {
		t.Error("should not be unhealthy after 2 failures")
	}

	// Third failure - should be unhealthy now
	monitor.CheckAgentHealthSync(ctx, "agent-1")
	info3, _ := monitor.GetHealthStatus("agent-1")
	if info3.Status != HealthStatusUnhealthy {
		t.Errorf("expected unhealthy after 3 failures, got %s", info3.Status)
	}
}

func TestHealthMonitor_ConsecutiveFailuresReset(t *testing.T) {
	bus := events.NewBus()
	config := DefaultHealthConfig()
	config.HealthCheckTimeout = 1 * time.Second
	config.UnhealthyThreshold = 3
	monitor := NewHealthMonitor(config, bus)

	agent := core.NewAgent("agent-1", "test", "Agent 1", "model-1", "mock")
	adapter := newMockHealthAdapter(false)
	monitor.RegisterAgent(agent, adapter)

	ctx := context.Background()

	// Fail twice
	monitor.CheckAgentHealthSync(ctx, "agent-1")
	monitor.CheckAgentHealthSync(ctx, "agent-1")

	info1, _ := monitor.GetHealthStatus("agent-1")
	if info1.ConsecutiveFailures != 2 {
		t.Errorf("expected 2 consecutive failures, got %d", info1.ConsecutiveFailures)
	}

	// Now succeed
	adapter.SetHealthy(true)
	monitor.CheckAgentHealthSync(ctx, "agent-1")

	info2, _ := monitor.GetHealthStatus("agent-1")
	if info2.ConsecutiveFailures != 0 {
		t.Errorf("expected 0 consecutive failures after success, got %d", info2.ConsecutiveFailures)
	}
	if info2.Status != HealthStatusHealthy {
		t.Errorf("expected healthy after success, got %s", info2.Status)
	}
}

func TestPool_HealthMonitorIntegration(t *testing.T) {
	bus := events.NewBus()
	healthConfig := DefaultHealthConfig()
	healthConfig.HealthCheckTimeout = 1 * time.Second
	pool := NewPoolWithHealthConfig(bus, 30*time.Second, healthConfig)

	agent := core.NewAgent("agent-1", "test", "Agent 1", "model-1", "mock")
	adapter := newMockHealthAdapter(true)
	pool.AddAgent(agent, adapter)

	// Verify health monitor was initialized
	if pool.GetHealthMonitor() == nil {
		t.Fatal("expected health monitor to be initialized")
	}

	// Check health status via pool
	ctx := context.Background()
	result := pool.CheckAgentHealth(ctx, "agent-1")
	if !result.Success {
		t.Error("expected health check to succeed")
	}

	// Check agent is healthy
	if !pool.IsAgentHealthy("agent-1") {
		t.Error("expected agent to be healthy")
	}

	// Check counts
	if pool.GetHealthyAgentCount() != 1 {
		t.Errorf("expected 1 healthy agent, got %d", pool.GetHealthyAgentCount())
	}
	if pool.GetUnhealthyAgentCount() != 0 {
		t.Errorf("expected 0 unhealthy agents, got %d", pool.GetUnhealthyAgentCount())
	}
}

func TestPool_PreflightCheck(t *testing.T) {
	bus := events.NewBus()
	healthConfig := DefaultHealthConfig()
	healthConfig.HealthCheckTimeout = 1 * time.Second
	healthConfig.PreflightCheckTimeout = 2 * time.Second
	pool := NewPoolWithHealthConfig(bus, 30*time.Second, healthConfig)

	// Add 2 agents
	for i := 1; i <= 2; i++ {
		agent := core.NewAgent("agent-"+string(rune('0'+i)), "test", "Agent "+string(rune('0'+i)), "model-1", "mock")
		adapter := newMockHealthAdapter(true)
		pool.AddAgent(agent, adapter)
	}

	ctx := context.Background()
	result := pool.PreflightCheck(ctx)

	if !result.AllHealthy {
		t.Error("expected all agents to be healthy")
	}
	if result.HealthyCount != 2 {
		t.Errorf("expected 2 healthy, got %d", result.HealthyCount)
	}
}

func TestPool_StartStopHealthMonitoring(t *testing.T) {
	bus := events.NewBus()
	pool := NewPool(bus, 30*time.Second)

	if pool.IsHealthMonitoringRunning() {
		t.Error("expected health monitoring not running initially")
	}

	pool.StartHealthMonitoring()
	if !pool.IsHealthMonitoringRunning() {
		t.Error("expected health monitoring to be running")
	}

	pool.StopHealthMonitoring()
	if pool.IsHealthMonitoringRunning() {
		t.Error("expected health monitoring to be stopped")
	}
}

func TestPool_ShouldWarnBeforeMessage(t *testing.T) {
	bus := events.NewBus()
	healthConfig := DefaultHealthConfig()
	healthConfig.WarnOnUnhealthy = true
	healthConfig.UnhealthyThreshold = 1
	healthConfig.HealthCheckTimeout = 1 * time.Second
	pool := NewPoolWithHealthConfig(bus, 30*time.Second, healthConfig)

	// Add an unhealthy agent
	agent := core.NewAgent("unhealthy-1", "test", "Unhealthy Agent", "model-1", "mock")
	adapter := newMockHealthAdapter(false)
	pool.AddAgent(agent, adapter)

	ctx := context.Background()
	pool.CheckAgentHealth(ctx, "unhealthy-1")

	shouldWarn, agents := pool.ShouldWarnBeforeMessage()
	if !shouldWarn {
		t.Error("expected warning for unhealthy agent")
	}
	if len(agents) != 1 {
		t.Errorf("expected 1 unhealthy agent, got %d", len(agents))
	}
}
