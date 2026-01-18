// Package cmd provides CLI commands for AgentPipe.
package cmd

import (
	"os"
	"path/filepath"
	"testing"

	// Import v2 adapters to ensure they are registered in their init() functions
	_ "github.com/ASRagab/agentpipe/pkg/v2/adapters/api"
	_ "github.com/ASRagab/agentpipe/pkg/v2/adapters/mock"
)

func TestDoctorV2FlagsExist(t *testing.T) {
	// Verify that v2 flags are properly registered on doctor command
	flag := doctorCmd.Flags().Lookup("v2")
	if flag == nil {
		t.Error("--v2 flag not found on doctor command")
	}

	flag = doctorCmd.Flags().Lookup("config")
	if flag == nil {
		t.Error("--config flag not found on doctor command")
	}

	flag = doctorCmd.Flags().Lookup("json")
	if flag == nil {
		t.Error("--json flag not found on doctor command")
	}
}

func TestDoctorV2FlagDefaults(t *testing.T) {
	tests := []struct {
		name        string
		flagName    string
		expectedDef string
	}{
		{"v2 default", "v2", "false"},
		{"config default", "config", ""},
		{"json default", "json", "false"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			flag := doctorCmd.Flags().Lookup(tc.flagName)
			if flag == nil {
				t.Fatalf("--%s flag not found", tc.flagName)
			}
			if flag.DefValue != tc.expectedDef {
				t.Errorf("--%s default = %q, want %q", tc.flagName, flag.DefValue, tc.expectedDef)
			}
		})
	}
}

func TestPerformV2ChecksNoConfig(t *testing.T) {
	// Save and restore doctorConfig
	origConfig := doctorConfig
	defer func() {
		doctorConfig = origConfig
	}()

	doctorConfig = ""
	output := performV2Checks()

	// Should have registered adapters (at least openrouter from init)
	if len(output.RegisteredAdapters) == 0 {
		t.Error("expected at least one registered adapter")
	}

	// Without config, should still report basic v2 readiness
	// (based on registered adapters)
	if output.ConfigFile != "" {
		t.Errorf("expected empty ConfigFile, got %q", output.ConfigFile)
	}
}

func TestPerformV2ChecksInvalidConfig(t *testing.T) {
	// Save and restore doctorConfig
	origConfig := doctorConfig
	defer func() {
		doctorConfig = origConfig
	}()

	// Create a temp file with invalid YAML
	tmpDir := t.TempDir()
	invalidConfig := filepath.Join(tmpDir, "invalid.yaml")
	err := os.WriteFile(invalidConfig, []byte("invalid: yaml: content: :::"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	doctorConfig = invalidConfig
	output := performV2Checks()

	if output.ConfigValid {
		t.Error("expected ConfigValid to be false for invalid config")
	}
	if output.ConfigError == "" {
		t.Error("expected ConfigError to be set for invalid config")
	}
	if output.V2Ready {
		t.Error("expected V2Ready to be false for invalid config")
	}
}

func TestPerformV2ChecksValidConfig(t *testing.T) {
	// Save and restore doctorConfig
	origConfig := doctorConfig
	defer func() {
		doctorConfig = origConfig
	}()

	// Create a valid v2 config with openrouter adapter
	tmpDir := t.TempDir()
	validConfig := filepath.Join(tmpDir, "valid.yaml")
	configContent := `
conversation:
  timeout: 30s
  mode: parallel

agents:
  - id: test-agent
    type: openrouter
    name: "Test Agent"
    model: anthropic/claude-sonnet-4-5
    config:
      api_key_env: TEST_OPENROUTER_KEY
      system_prompt: "You are a test agent"
`
	err := os.WriteFile(validConfig, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	doctorConfig = validConfig
	output := performV2Checks()

	if !output.ConfigValid {
		t.Errorf("expected ConfigValid to be true, got error: %s", output.ConfigError)
	}

	if output.TotalAgents != 1 {
		t.Errorf("expected 1 agent, got %d", output.TotalAgents)
	}

	// Agent won't be healthy without API key, but check should run
	if len(output.AgentChecks) != 1 {
		t.Errorf("expected 1 agent check, got %d", len(output.AgentChecks))
	}
}

func TestV2AgentCheckStruct(t *testing.T) {
	check := V2AgentCheck{
		ID:           "test-id",
		Name:         "Test Name",
		Type:         "openrouter",
		Model:        "test-model",
		Adapter:      "openrouter",
		Available:    true,
		Healthy:      false,
		APIKeySet:    false,
		ResponseTime: "100ms",
		Error:        "test error",
	}

	if check.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got %q", check.ID)
	}
	if check.Healthy {
		t.Error("expected Healthy to be false")
	}
}

func TestV2DoctorOutputStruct(t *testing.T) {
	output := V2DoctorOutput{
		V2Ready:            true,
		RegisteredAdapters: []string{"openrouter", "claude-api"},
		ConfigFile:         "/path/to/config.yaml",
		ConfigValid:        true,
		HealthyAgents:      2,
		TotalAgents:        3,
	}

	if !output.V2Ready {
		t.Error("expected V2Ready to be true")
	}
	if len(output.RegisteredAdapters) != 2 {
		t.Errorf("expected 2 adapters, got %d", len(output.RegisteredAdapters))
	}
	if output.HealthyAgents != 2 {
		t.Errorf("expected 2 healthy agents, got %d", output.HealthyAgents)
	}
}

func TestLoadAndValidateV2ConfigMissingFile(t *testing.T) {
	_, err := loadAndValidateV2Config("/nonexistent/path/to/config.yaml")
	if err == nil {
		t.Error("expected error for missing config file")
	}
}

func TestLoadAndValidateV2ConfigEmptyAgents(t *testing.T) {
	tmpDir := t.TempDir()
	emptyConfig := filepath.Join(tmpDir, "empty.yaml")
	configContent := `
conversation:
  timeout: 30s
agents: []
`
	err := os.WriteFile(emptyConfig, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err = loadAndValidateV2Config(emptyConfig)
	if err == nil {
		t.Error("expected error for config with no agents")
	}
}
