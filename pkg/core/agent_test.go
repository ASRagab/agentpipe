package core

import (
	"encoding/json"
	"testing"
)

func TestNewAgent(t *testing.T) {
	agent := NewAgent("agent-1", "openrouter", "Test Agent", "gpt-4", "openrouter")

	if agent.ID != "agent-1" {
		t.Errorf("expected ID 'agent-1', got %q", agent.ID)
	}
	if agent.Type != "openrouter" {
		t.Errorf("expected Type 'openrouter', got %q", agent.Type)
	}
	if agent.Name != "Test Agent" {
		t.Errorf("expected Name 'Test Agent', got %q", agent.Name)
	}
	if agent.Model != "gpt-4" {
		t.Errorf("expected Model 'gpt-4', got %q", agent.Model)
	}
	if agent.AdapterName != "openrouter" {
		t.Errorf("expected AdapterName 'openrouter', got %q", agent.AdapterName)
	}
}

func TestAgent_WithConfig(t *testing.T) {
	agent := NewAgent("agent-1", "openrouter", "Test Agent", "gpt-4", "openrouter")

	config := AgentAdapterConfig{
		SystemPrompt: "You are a helpful assistant",
		Temperature:  0.7,
		MaxTokens:    1000,
		APIKeyEnvVar: "OPENROUTER_API_KEY",
	}

	agentWithConfig := agent.WithConfig(config)

	// Original should be unchanged (value semantics)
	if agent.Config.SystemPrompt != "" {
		t.Error("original agent should not have config")
	}

	// New agent should have config
	if agentWithConfig.Config.SystemPrompt != "You are a helpful assistant" {
		t.Errorf("expected SystemPrompt, got %q", agentWithConfig.Config.SystemPrompt)
	}
	if agentWithConfig.Config.Temperature != 0.7 {
		t.Errorf("expected Temperature=0.7, got %f", agentWithConfig.Config.Temperature)
	}
	if agentWithConfig.Config.MaxTokens != 1000 {
		t.Errorf("expected MaxTokens=1000, got %d", agentWithConfig.Config.MaxTokens)
	}
	if agentWithConfig.Config.APIKeyEnvVar != "OPENROUTER_API_KEY" {
		t.Errorf("expected APIKeyEnvVar, got %q", agentWithConfig.Config.APIKeyEnvVar)
	}
}

func TestAgent_WithSystemPrompt(t *testing.T) {
	agent := NewAgent("agent-1", "openrouter", "Test Agent", "gpt-4", "openrouter")
	agentWithPrompt := agent.WithSystemPrompt("Custom prompt")

	if agentWithPrompt.Config.SystemPrompt != "Custom prompt" {
		t.Errorf("expected SystemPrompt 'Custom prompt', got %q", agentWithPrompt.Config.SystemPrompt)
	}
}

func TestAgent_WithTemperature(t *testing.T) {
	agent := NewAgent("agent-1", "openrouter", "Test Agent", "gpt-4", "openrouter")
	agentWithTemp := agent.WithTemperature(0.5)

	if agentWithTemp.Config.Temperature != 0.5 {
		t.Errorf("expected Temperature=0.5, got %f", agentWithTemp.Config.Temperature)
	}
}

func TestAgent_WithMaxTokens(t *testing.T) {
	agent := NewAgent("agent-1", "openrouter", "Test Agent", "gpt-4", "openrouter")
	agentWithTokens := agent.WithMaxTokens(2000)

	if agentWithTokens.Config.MaxTokens != 2000 {
		t.Errorf("expected MaxTokens=2000, got %d", agentWithTokens.Config.MaxTokens)
	}
}

func TestAgent_Chaining(t *testing.T) {
	agent := NewAgent("agent-1", "openrouter", "Test Agent", "gpt-4", "openrouter").
		WithSystemPrompt("You are helpful").
		WithTemperature(0.8).
		WithMaxTokens(500)

	if agent.Config.SystemPrompt != "You are helpful" {
		t.Errorf("expected SystemPrompt, got %q", agent.Config.SystemPrompt)
	}
	if agent.Config.Temperature != 0.8 {
		t.Errorf("expected Temperature=0.8, got %f", agent.Config.Temperature)
	}
	if agent.Config.MaxTokens != 500 {
		t.Errorf("expected MaxTokens=500, got %d", agent.Config.MaxTokens)
	}
}

func TestAgent_JSONSerialization(t *testing.T) {
	original := NewAgent("agent-1", "openrouter", "Test Agent", "gpt-4", "openrouter").
		WithConfig(AgentAdapterConfig{
			SystemPrompt: "Test prompt",
			Temperature:  0.7,
			MaxTokens:    1000,
			Extra:        map[string]interface{}{"custom": "value"},
		})

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal agent: %v", err)
	}

	var restored Agent
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("failed to unmarshal agent: %v", err)
	}

	if restored.ID != original.ID {
		t.Errorf("ID mismatch: expected %q, got %q", original.ID, restored.ID)
	}
	if restored.Type != original.Type {
		t.Errorf("Type mismatch: expected %q, got %q", original.Type, restored.Type)
	}
	if restored.Name != original.Name {
		t.Errorf("Name mismatch: expected %q, got %q", original.Name, restored.Name)
	}
	if restored.Config.SystemPrompt != original.Config.SystemPrompt {
		t.Errorf("SystemPrompt mismatch: expected %q, got %q", original.Config.SystemPrompt, restored.Config.SystemPrompt)
	}
}

func TestNewAgentState(t *testing.T) {
	agent := NewAgent("agent-1", "openrouter", "Test Agent", "gpt-4", "openrouter")
	state := NewAgentState(agent)

	if state.Agent.ID != agent.ID {
		t.Errorf("expected Agent.ID %q, got %q", agent.ID, state.Agent.ID)
	}
	if state.Status != AgentStatusIdle {
		t.Errorf("expected Status %q, got %q", AgentStatusIdle, state.Status)
	}
	if state.LastError != "" {
		t.Errorf("expected empty LastError, got %q", state.LastError)
	}
	if state.MessageCount != 0 {
		t.Errorf("expected MessageCount=0, got %d", state.MessageCount)
	}
	if state.TotalTokens != 0 {
		t.Errorf("expected TotalTokens=0, got %d", state.TotalTokens)
	}
	if state.TotalCost != 0 {
		t.Errorf("expected TotalCost=0, got %f", state.TotalCost)
	}
}

func TestAgentState_SetTyping(t *testing.T) {
	agent := NewAgent("agent-1", "openrouter", "Test Agent", "gpt-4", "openrouter")
	state := NewAgentState(agent)

	// Set an error first to verify it's cleared
	state.SetError("previous error")

	state.SetTyping()

	if state.Status != AgentStatusTyping {
		t.Errorf("expected Status %q, got %q", AgentStatusTyping, state.Status)
	}
	if state.LastError != "" {
		t.Errorf("expected LastError cleared, got %q", state.LastError)
	}
}

func TestAgentState_SetIdle(t *testing.T) {
	agent := NewAgent("agent-1", "openrouter", "Test Agent", "gpt-4", "openrouter")
	state := NewAgentState(agent)
	state.SetTyping()

	state.SetIdle()

	if state.Status != AgentStatusIdle {
		t.Errorf("expected Status %q, got %q", AgentStatusIdle, state.Status)
	}
}

func TestAgentState_SetError(t *testing.T) {
	agent := NewAgent("agent-1", "openrouter", "Test Agent", "gpt-4", "openrouter")
	state := NewAgentState(agent)

	state.SetError("connection failed")

	if state.Status != AgentStatusError {
		t.Errorf("expected Status %q, got %q", AgentStatusError, state.Status)
	}
	if state.LastError != "connection failed" {
		t.Errorf("expected LastError 'connection failed', got %q", state.LastError)
	}
}

func TestAgentState_RecordMessage(t *testing.T) {
	agent := NewAgent("agent-1", "openrouter", "Test Agent", "gpt-4", "openrouter")
	state := NewAgentState(agent)
	state.SetTyping()

	state.RecordMessage(100, 0.01)

	if state.Status != AgentStatusIdle {
		t.Errorf("expected Status %q after RecordMessage, got %q", AgentStatusIdle, state.Status)
	}
	if state.MessageCount != 1 {
		t.Errorf("expected MessageCount=1, got %d", state.MessageCount)
	}
	if state.TotalTokens != 100 {
		t.Errorf("expected TotalTokens=100, got %d", state.TotalTokens)
	}
	if state.TotalCost != 0.01 {
		t.Errorf("expected TotalCost=0.01, got %f", state.TotalCost)
	}

	// Record another message
	state.RecordMessage(50, 0.005)

	if state.MessageCount != 2 {
		t.Errorf("expected MessageCount=2, got %d", state.MessageCount)
	}
	if state.TotalTokens != 150 {
		t.Errorf("expected TotalTokens=150, got %d", state.TotalTokens)
	}
	if state.TotalCost != 0.015 {
		t.Errorf("expected TotalCost=0.015, got %f", state.TotalCost)
	}
}

func TestAgentStatus_Constants(t *testing.T) {
	statuses := []AgentStatus{
		AgentStatusIdle,
		AgentStatusTyping,
		AgentStatusError,
		AgentStatusOffline,
	}

	seen := make(map[AgentStatus]bool)
	for _, s := range statuses {
		if seen[s] {
			t.Errorf("duplicate status constant: %q", s)
		}
		seen[s] = true
	}
}

func TestAgentAdapterConfig_Extra(t *testing.T) {
	config := AgentAdapterConfig{
		SystemPrompt: "Test",
		Extra: map[string]interface{}{
			"custom_param": "custom_value",
			"number":       42,
		},
	}

	if config.Extra["custom_param"] != "custom_value" {
		t.Errorf("expected Extra['custom_param']='custom_value', got %v", config.Extra["custom_param"])
	}
	if config.Extra["number"] != 42 {
		t.Errorf("expected Extra['number']=42, got %v", config.Extra["number"])
	}
}
