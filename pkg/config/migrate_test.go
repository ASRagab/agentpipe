package config

import (
	"os"
	"path/filepath"
	"testing"

	_ "github.com/ASRagab/agentpipe/pkg/adapters/cli"  // Register CLI adapters
	_ "github.com/ASRagab/agentpipe/pkg/adapters/mock" // Register mock adapter
)

func TestDetectV1Config_Orchestrator(t *testing.T) {
	// V1 config with orchestrator section
	data := []byte(`
version: "1.0"
agents:
  - id: agent-1
    type: claude
    name: Test Agent
    prompt: "You are helpful"
orchestrator:
  mode: free-form
  max_turns: 10
`)
	isV1, err := DetectV1Config(data)
	if err != nil {
		t.Fatalf("DetectV1Config returned error: %v", err)
	}
	if !isV1 {
		t.Error("expected to detect v1 config (has orchestrator section)")
	}
}

func TestDetectV1Config_FlatPrompt(t *testing.T) {
	// V1 config with flat prompt field (not nested in config)
	data := []byte(`
agents:
  - id: agent-1
    type: claude
    name: Test Agent
    prompt: "You are helpful"
    temperature: 0.7
`)
	isV1, err := DetectV1Config(data)
	if err != nil {
		t.Fatalf("DetectV1Config returned error: %v", err)
	}
	if !isV1 {
		t.Error("expected to detect v1 config (has flat prompt field)")
	}
}

func TestDetectV1Config_Version(t *testing.T) {
	// V1 config with explicit version
	data := []byte(`
version: "1.2"
agents:
  - id: agent-1
    type: mock
    name: Test Agent
`)
	isV1, err := DetectV1Config(data)
	if err != nil {
		t.Fatalf("DetectV1Config returned error: %v", err)
	}
	if !isV1 {
		t.Error("expected to detect v1 config (explicit version 1.2)")
	}
}

func TestDetectV1Config_V2Format(t *testing.T) {
	// V2 config should not be detected as v1
	data := []byte(`
conversation:
  timeout: 30s
  mode: parallel
agents:
  - id: agent-1
    type: mock
    adapter: mock
    name: Test Agent
    model: mock-model
    config:
      system_prompt: "You are helpful"
      temperature: 0.7
`)
	isV1, err := DetectV1Config(data)
	if err != nil {
		t.Fatalf("DetectV1Config returned error: %v", err)
	}
	if isV1 {
		t.Error("v2 config should not be detected as v1")
	}
}

func TestDetectV1Config_InvalidYAML(t *testing.T) {
	data := []byte(`
this is not valid yaml: [[[
`)
	_, err := DetectV1Config(data)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestMigrateV1Config_Claude(t *testing.T) {
	data := []byte(`
version: "1.0"
agents:
  - id: claude-agent
    type: claude
    name: "Claude Assistant"
    prompt: "You are a helpful assistant"
    model: claude-3-5-sonnet-20241022
    temperature: 0.8
    max_tokens: 1000
orchestrator:
  mode: round-robin
  max_turns: 5
  turn_timeout: 45s
`)
	result, err := MigrateV1Config(data)
	if err != nil {
		t.Fatalf("MigrateV1Config returned error: %v", err)
	}

	if !result.Migrated {
		t.Error("expected Migrated to be true")
	}
	if result.SourceVersion != "1.0" {
		t.Errorf("expected SourceVersion '1.0', got %q", result.SourceVersion)
	}

	cfg := result.Config
	if len(cfg.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(cfg.Agents))
	}

	agent := cfg.Agents[0]
	if agent.ID != "claude-agent" {
		t.Errorf("expected ID 'claude-agent', got %q", agent.ID)
	}
	if agent.Name != "Claude Assistant" {
		t.Errorf("expected Name 'Claude Assistant', got %q", agent.Name)
	}
	if agent.Type != "claude" {
		t.Errorf("expected Type 'claude', got %q", agent.Type)
	}
	// Adapter should be claude-cli (no API key set) or claude-api
	if agent.Adapter != "claude-cli" && agent.Adapter != "claude-api" {
		t.Errorf("expected Adapter 'claude-cli' or 'claude-api', got %q", agent.Adapter)
	}
	if agent.Model != "claude-3-5-sonnet-20241022" {
		t.Errorf("expected Model 'claude-3-5-sonnet-20241022', got %q", agent.Model)
	}
	if agent.Config.SystemPrompt != "You are a helpful assistant" {
		t.Errorf("expected SystemPrompt 'You are a helpful assistant', got %q", agent.Config.SystemPrompt)
	}
	if agent.Config.Temperature != 0.8 {
		t.Errorf("expected Temperature 0.8, got %f", agent.Config.Temperature)
	}
	if agent.Config.MaxTokens != 1000 {
		t.Errorf("expected MaxTokens 1000, got %d", agent.Config.MaxTokens)
	}

	// Check conversation settings (migrated from orchestrator)
	if cfg.Conversation.Mode != "round-robin" {
		t.Errorf("expected Conversation.Mode 'round-robin', got %q", cfg.Conversation.Mode)
	}
	if cfg.Conversation.MaxTurns != 5 {
		t.Errorf("expected Conversation.MaxTurns 5, got %d", cfg.Conversation.MaxTurns)
	}
}

func TestMigrateV1Config_Gemini(t *testing.T) {
	data := []byte(`
agents:
  - id: gemini-agent
    type: gemini
    name: "Gemini Pro"
    prompt: "You are a creative assistant"
`)
	result, err := MigrateV1Config(data)
	if err != nil {
		t.Fatalf("MigrateV1Config returned error: %v", err)
	}

	agent := result.Config.Agents[0]
	if agent.Adapter != "gemini-cli" {
		t.Errorf("expected Adapter 'gemini-cli', got %q", agent.Adapter)
	}
	if agent.Config.SystemPrompt != "You are a creative assistant" {
		t.Errorf("expected SystemPrompt 'You are a creative assistant', got %q", agent.Config.SystemPrompt)
	}
}

func TestMigrateV1Config_OpenRouter(t *testing.T) {
	data := []byte(`
agents:
  - id: openrouter-agent
    type: openrouter
    name: "OpenRouter GPT"
    model: openai/gpt-4o-mini
    prompt: "You are a helpful AI"
`)
	result, err := MigrateV1Config(data)
	if err != nil {
		t.Fatalf("MigrateV1Config returned error: %v", err)
	}

	agent := result.Config.Agents[0]
	if agent.Adapter != "openrouter" {
		t.Errorf("expected Adapter 'openrouter', got %q", agent.Adapter)
	}
	if agent.Model != "openai/gpt-4o-mini" {
		t.Errorf("expected Model 'openai/gpt-4o-mini', got %q", agent.Model)
	}
}

func TestMigrateV1Config_CLIGeneric(t *testing.T) {
	// qwen, ollama, etc. should map to cli-generic
	data := []byte(`
agents:
  - id: qwen-agent
    type: qwen
    name: "Qwen"
    prompt: "You are Qwen"
`)
	result, err := MigrateV1Config(data)
	if err != nil {
		t.Fatalf("MigrateV1Config returned error: %v", err)
	}

	agent := result.Config.Agents[0]
	if agent.Adapter != "cli-generic" {
		t.Errorf("expected Adapter 'cli-generic', got %q", agent.Adapter)
	}
	// Should have a warning about using cli-generic
	if len(result.Warnings) == 0 {
		t.Error("expected warnings for cli-generic adapter")
	}
}

func TestMigrateV1Config_PreservesAllFields(t *testing.T) {
	data := []byte(`
version: "1.0"
agents:
  - id: test-agent
    type: claude
    name: "Test"
    prompt: "System prompt here"
    model: claude-3-haiku-20240307
    announcement: "Hello!"
    temperature: 0.5
    max_tokens: 500
orchestrator:
  mode: free-form
  max_turns: 12
  turn_timeout: 60s
  response_delay: 2s
logging:
  enabled: true
  show_metrics: true
`)
	result, err := MigrateV1Config(data)
	if err != nil {
		t.Fatalf("MigrateV1Config returned error: %v", err)
	}

	agent := result.Config.Agents[0]
	// Check all preserved fields
	if agent.ID != "test-agent" {
		t.Errorf("ID not preserved: got %q", agent.ID)
	}
	if agent.Name != "Test" {
		t.Errorf("Name not preserved: got %q", agent.Name)
	}
	if agent.Config.SystemPrompt != "System prompt here" {
		t.Errorf("Prompt not preserved: got %q", agent.Config.SystemPrompt)
	}
	if agent.Model != "claude-3-haiku-20240307" {
		t.Errorf("Model not preserved: got %q", agent.Model)
	}
	if agent.Config.Temperature != 0.5 {
		t.Errorf("Temperature not preserved: got %f", agent.Config.Temperature)
	}
	if agent.Config.MaxTokens != 500 {
		t.Errorf("MaxTokens not preserved: got %d", agent.Config.MaxTokens)
	}

	// Check TUI settings (migrated from logging.show_metrics)
	if !result.Config.TUI.ShowMetrics {
		t.Error("ShowMetrics not preserved from logging settings")
	}
}

func TestMigrateV1Config_WithAPIKey(t *testing.T) {
	// Set API key temporarily
	originalKey := os.Getenv("ANTHROPIC_API_KEY")
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	defer func() {
		if originalKey == "" {
			os.Unsetenv("ANTHROPIC_API_KEY")
		} else {
			os.Setenv("ANTHROPIC_API_KEY", originalKey)
		}
	}()

	data := []byte(`
agents:
  - id: claude-agent
    type: claude
    name: "Claude"
    prompt: "Test"
`)
	result, err := MigrateV1Config(data)
	if err != nil {
		t.Fatalf("MigrateV1Config returned error: %v", err)
	}

	agent := result.Config.Agents[0]
	// With API key set, should prefer claude-api
	if agent.Adapter != "claude-api" {
		t.Errorf("expected Adapter 'claude-api' when API key is set, got %q", agent.Adapter)
	}
}

func TestMigrateV1Config_FreeFormMode(t *testing.T) {
	data := []byte(`
agents:
  - id: agent
    type: claude
    name: "Agent"
orchestrator:
  mode: free-form
`)
	result, err := MigrateV1Config(data)
	if err != nil {
		t.Fatalf("MigrateV1Config returned error: %v", err)
	}

	// free-form should map to reactive
	if result.Config.Conversation.Mode != "reactive" {
		t.Errorf("expected Mode 'reactive' for v1 'free-form', got %q", result.Config.Conversation.Mode)
	}
}

func TestMigrateV1Config_MultipleAgents(t *testing.T) {
	data := []byte(`
agents:
  - id: agent-1
    type: claude
    name: "Claude"
    prompt: "You are Claude"
  - id: agent-2
    type: gemini
    name: "Gemini"
    prompt: "You are Gemini"
  - id: agent-3
    type: openrouter
    name: "OpenRouter"
    model: openai/gpt-4
`)
	result, err := MigrateV1Config(data)
	if err != nil {
		t.Fatalf("MigrateV1Config returned error: %v", err)
	}

	if len(result.Config.Agents) != 3 {
		t.Fatalf("expected 3 agents, got %d", len(result.Config.Agents))
	}

	// Check each agent has correct adapter
	adapters := map[string]string{
		"agent-1": "claude-cli", // or claude-api if key set
		"agent-2": "gemini-cli",
		"agent-3": "openrouter",
	}
	for _, agent := range result.Config.Agents {
		expected := adapters[agent.ID]
		// Skip claude check since it depends on env
		if agent.ID == "agent-1" && (agent.Adapter == "claude-cli" || agent.Adapter == "claude-api") {
			continue
		}
		if agent.ID != "agent-1" && agent.Adapter != expected {
			t.Errorf("agent %s: expected adapter %q, got %q", agent.ID, expected, agent.Adapter)
		}
	}
}

func TestLoadV1Config_EndToEnd(t *testing.T) {
	// Create temporary v1 config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "v1-config.yaml")

	v1Config := []byte(`
version: "1.0"
agents:
  - id: mock-agent
    type: mock
    name: "Mock Agent"
    prompt: "You are a mock agent"
    model: mock-model
    temperature: 0.5
orchestrator:
  mode: round-robin
  max_turns: 5
`)
	if err := os.WriteFile(configPath, v1Config, 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Load the config (should auto-detect and migrate)
	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	// Verify migration was successful
	if len(config.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(config.Agents))
	}

	agent := config.Agents[0]
	if agent.ID != "mock-agent" {
		t.Errorf("expected ID 'mock-agent', got %q", agent.ID)
	}
	if agent.Adapter != "mock" {
		t.Errorf("expected Adapter 'mock', got %q", agent.Adapter)
	}
	if agent.Config.SystemPrompt != "You are a mock agent" {
		t.Errorf("expected SystemPrompt 'You are a mock agent', got %q", agent.Config.SystemPrompt)
	}

	// Verify conversation settings were migrated
	if config.Conversation.Mode != "round-robin" {
		t.Errorf("expected Mode 'round-robin', got %q", config.Conversation.Mode)
	}
	if config.Conversation.MaxTurns != 5 {
		t.Errorf("expected MaxTurns 5, got %d", config.Conversation.MaxTurns)
	}
}

func TestLoadConfigWithOptions_DisableMigration(t *testing.T) {
	// Create temporary v1 config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "v1-config.yaml")

	v1Config := []byte(`
version: "1.0"
agents:
  - id: agent
    type: claude
    prompt: "test"
orchestrator:
  mode: free-form
`)
	if err := os.WriteFile(configPath, v1Config, 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Disable migration
	autoMigrate := false
	_, err := LoadConfigWithOptions(configPath, LoadOptions{AutoMigrateV1: &autoMigrate})

	// Should fail because v1 config won't parse as v2 (missing required fields)
	if err == nil {
		t.Error("expected error when migration is disabled for v1 config")
	}
}

func TestSaveMigratedConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write original content
	original := []byte("original: content\n")
	if err := os.WriteFile(configPath, original, 0644); err != nil {
		t.Fatalf("failed to write original config: %v", err)
	}

	// Create a test config to save
	cfg := &Config{
		Conversation: ConversationConfig{
			Mode: "parallel",
		},
		Agents: []AgentConfig{
			{
				ID:      "test",
				Type:    "mock",
				Adapter: "mock",
				Name:    "Test",
				Model:   "model",
			},
		},
	}

	// Save migrated config
	if err := SaveMigratedConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveMigratedConfig returned error: %v", err)
	}

	// Check backup was created
	backupPath := configPath + ".v1.backup"
	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("failed to read backup: %v", err)
	}
	if string(backupData) != "original: content\n" {
		t.Errorf("backup content mismatch: got %q", string(backupData))
	}

	// Check new config was written
	newData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read new config: %v", err)
	}
	if !contains(string(newData), "Migrated from v1") {
		t.Error("migrated config should have migration header")
	}
	if !contains(string(newData), "test") {
		t.Error("migrated config should contain agent data")
	}
}

func TestMigrateV1Config_UnknownType(t *testing.T) {
	data := []byte(`
agents:
  - id: unknown-agent
    type: unknown_provider
    name: "Unknown"
    prompt: "test"
`)
	result, err := MigrateV1Config(data)
	if err != nil {
		t.Fatalf("MigrateV1Config returned error: %v", err)
	}

	agent := result.Config.Agents[0]
	// Unknown types should fall back to cli-generic
	if agent.Adapter != "cli-generic" {
		t.Errorf("expected Adapter 'cli-generic' for unknown type, got %q", agent.Adapter)
	}
	// Should have a warning
	hasWarning := false
	for _, w := range result.Warnings {
		if contains(w, "unknown type") {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Error("expected warning about unknown type")
	}
}

func TestMigrateV1Config_DefaultModels(t *testing.T) {
	data := []byte(`
agents:
  - id: claude-no-model
    type: claude
    name: "Claude"
  - id: gemini-no-model
    type: gemini
    name: "Gemini"
  - id: openrouter-no-model
    type: openrouter
    name: "OpenRouter"
`)
	result, err := MigrateV1Config(data)
	if err != nil {
		t.Fatalf("MigrateV1Config returned error: %v", err)
	}

	// Check default models were assigned
	for _, agent := range result.Config.Agents {
		if agent.Model == "" {
			t.Errorf("agent %s should have a default model", agent.ID)
		}
	}
}
