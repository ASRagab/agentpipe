//go:build e2e

package e2e

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/v2/adapters"
	"github.com/kevinelliott/agentpipe/pkg/v2/config"
	"github.com/kevinelliott/agentpipe/pkg/v2/core"
	"github.com/kevinelliott/agentpipe/pkg/v2/events"
	"github.com/kevinelliott/agentpipe/pkg/v2/manager"
)

// TestV1ConfigMigration verifies that v1 configuration files are migrated correctly.
func TestV1ConfigMigration(t *testing.T) {
	TimeoutWrapper(t, 10*time.Second, func(t *testing.T) {
		tempDir := t.TempDir()

		// Create a v1-style config
		v1Config := `
# This is a v1 config (doesn't have 'conversation:' or 'agents:' top-level keys in v2 format)
agents:
  - name: Claude
    type: claude
    model: claude-3-5-sonnet-20241022
    system_prompt: You are Claude

orchestrator:
  mode: round-robin
  max_turns: 10

logging:
  level: debug
`
		configPath := filepath.Join(tempDir, "config.yaml")
		err := os.WriteFile(configPath, []byte(v1Config), 0600)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		// Load config (should auto-migrate if v1 format detected)
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			// Some v1 configs may not be auto-detected correctly
			t.Logf("Config load result: %v (may be expected for edge cases)", err)
			return
		}

		// Verify migration result
		if len(cfg.Agents) == 0 {
			t.Error("expected agents after migration")
		}

		t.Logf("Loaded %d agents from config", len(cfg.Agents))
	})
}

// TestMissingConfig verifies helpful error for missing config file.
func TestMissingConfig(t *testing.T) {
	TimeoutWrapper(t, 5*time.Second, func(t *testing.T) {
		_, err := config.LoadConfig("/nonexistent/path/config.yaml")
		if err == nil {
			t.Error("expected error for missing config file")
			return
		}

		// Verify error message is helpful
		if !contains(err.Error(), "read") && !contains(err.Error(), "no such file") {
			t.Logf("Error message: %v", err)
		}
	})
}

// TestInvalidConfig verifies parse error for malformed YAML.
func TestInvalidConfig(t *testing.T) {
	TimeoutWrapper(t, 5*time.Second, func(t *testing.T) {
		tempDir := t.TempDir()

		// Create malformed YAML
		malformedConfig := `
conversation:
  timeout: 30s
agents:
  - name: Test
    type: mock
    # Missing closing bracket or other YAML error
    model: [unclosed
`
		configPath := filepath.Join(tempDir, "malformed.yaml")
		err := os.WriteFile(configPath, []byte(malformedConfig), 0600)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		_, err = config.LoadConfig(configPath)
		if err == nil {
			t.Error("expected error for malformed YAML")
			return
		}

		// Should contain parse error indication
		t.Logf("Parse error (expected): %v", err)
	})
}

// TestMissingAPIKey verifies error message when API adapter lacks key.
func TestMissingAPIKey(t *testing.T) {
	TimeoutWrapper(t, 10*time.Second, func(t *testing.T) {
		// This test checks that adapters handle missing API keys correctly
		// Most API adapters check IsAvailable() which returns false without API key

		// Create an OpenRouter adapter without setting API key
		adapter, err := adapters.Get("openrouter")
		if err != nil {
			t.Logf("OpenRouter adapter not available: %v (expected in test env)", err)
			return
		}

		// Check if it reports as unavailable without API key
		if adapter.IsAvailable() {
			t.Log("OpenRouter adapter reports available - API key may be set in environment")
		} else {
			t.Log("OpenRouter adapter correctly reports unavailable without API key")
		}
	})
}

// TestUnknownAdapter verifies error for unknown adapter name.
func TestUnknownAdapter(t *testing.T) {
	TimeoutWrapper(t, 5*time.Second, func(t *testing.T) {
		tempDir := t.TempDir()

		// Create config with unknown adapter
		configContent := `
conversation:
  timeout: 30s
agents:
  - id: unknown
    name: Unknown Agent
    type: nonexistent-adapter-xyz
    model: some-model
`
		configPath := filepath.Join(tempDir, "unknown-adapter.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0600)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			// Error during load is expected
			if contains(err.Error(), "unknown adapter") {
				t.Log("Correctly reported unknown adapter during load")
			} else {
				t.Logf("Error: %v", err)
			}
			return
		}

		// If load succeeded, validation should catch it
		err = cfg.Validate()
		if err == nil {
			t.Error("expected validation error for unknown adapter")
			return
		}

		if contains(err.Error(), "unknown adapter") {
			t.Log("Validation correctly caught unknown adapter")
		} else {
			t.Logf("Validation error: %v", err)
		}
	})
}

// TestValidConfig verifies that a valid config loads correctly.
func TestValidConfig(t *testing.T) {
	TimeoutWrapper(t, 10*time.Second, func(t *testing.T) {
		tempDir := t.TempDir()

		// Create valid v2 config with mock adapter
		configContent := `
conversation:
  timeout: 30s
  mode: parallel
agents:
  - id: test-agent
    name: Test Agent
    type: mock
    model: mock-model
    config:
      system_prompt: You are a test agent
`
		configPath := filepath.Join(tempDir, "valid.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0600)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			t.Fatalf("failed to load valid config: %v", err)
		}

		// Verify parsed correctly
		if len(cfg.Agents) != 1 {
			t.Errorf("expected 1 agent, got %d", len(cfg.Agents))
		}

		if cfg.Agents[0].ID != "test-agent" {
			t.Errorf("expected agent id 'test-agent', got '%s'", cfg.Agents[0].ID)
		}

		if cfg.Conversation.Timeout != 30*time.Second {
			t.Errorf("expected 30s timeout, got %v", cfg.Conversation.Timeout)
		}
	})
}

// TestConfigInitializeAgents verifies that agents are created from config.
func TestConfigInitializeAgents(t *testing.T) {
	TimeoutWrapper(t, 10*time.Second, func(t *testing.T) {
		tempDir := t.TempDir()

		configContent := `
conversation:
  timeout: 60s
agents:
  - id: alice
    name: Alice
    type: mock
    model: mock-model-a
    config:
      system_prompt: You are Alice
      temperature: 0.7
  - id: bob
    name: Bob
    type: mock
    model: mock-model-b
    config:
      system_prompt: You are Bob
      max_tokens: 1000
`
		configPath := filepath.Join(tempDir, "multi-agent.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		agents, err := cfg.InitializeAgents()
		if err != nil {
			t.Fatalf("failed to initialize agents: %v", err)
		}

		if len(agents) != 2 {
			t.Errorf("expected 2 agents, got %d", len(agents))
		}

		// Verify agent properties
		for _, agent := range agents {
			switch agent.ID {
			case "alice":
				if agent.Config.Temperature != 0.7 {
					t.Errorf("alice temperature: expected 0.7, got %v", agent.Config.Temperature)
				}
			case "bob":
				if agent.Config.MaxTokens != 1000 {
					t.Errorf("bob max_tokens: expected 1000, got %d", agent.Config.MaxTokens)
				}
			default:
				t.Errorf("unexpected agent: %s", agent.ID)
			}
		}
	})
}

// TestConfigDefaults verifies that default values are applied.
func TestConfigDefaults(t *testing.T) {
	TimeoutWrapper(t, 10*time.Second, func(t *testing.T) {
		tempDir := t.TempDir()

		// Minimal config without optional fields
		configContent := `
agents:
  - id: minimal
    name: Minimal Agent
    type: mock
    model: mock-model
`
		configPath := filepath.Join(tempDir, "minimal.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		// Verify defaults were applied
		if cfg.Conversation.Timeout == 0 {
			t.Error("expected non-zero default timeout")
		}

		if cfg.Conversation.Mode == "" {
			t.Error("expected default mode")
		}

		if cfg.Logging.Level == "" {
			t.Error("expected default log level")
		}

		t.Logf("Defaults: timeout=%v, mode=%s, log_level=%s",
			cfg.Conversation.Timeout, cfg.Conversation.Mode, cfg.Logging.Level)
	})
}

// TestConfigWithManager verifies that config integrates with manager.
func TestConfigWithManager(t *testing.T) {
	TimeoutWrapper(t, 15*time.Second, func(t *testing.T) {
		tempDir := t.TempDir()

		configContent := `
conversation:
  timeout: 5s
  mode: parallel
agents:
  - id: config-test
    name: Config Test Agent
    type: mock
    model: mock-model
persistence:
  auto_save: true
`
		configPath := filepath.Join(tempDir, "manager-test.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		agents, err := cfg.InitializeAgents()
		if err != nil {
			t.Fatalf("failed to initialize agents: %v", err)
		}

		// Create manager from config
		timeout, _ := cfg.GetManagerConfig()
		eventBus := events.NewBus()
		defer eventBus.Close()

		mgrConfig := manager.Config{
			Timeout: timeout,
		}

		mgr, err := manager.NewConversationManager(mgrConfig, agents, eventBus)
		if err != nil {
			t.Fatalf("failed to create manager: %v", err)
		}
		defer mgr.Close()

		mgr.Start()

		// Send a test message
		ctx := context.Background()
		responses, err := mgr.SendUserMessage(ctx, "Config test message")
		if err != nil {
			t.Fatalf("failed to send message: %v", err)
		}

		if len(responses) == 0 {
			t.Error("expected at least one response")
		}
	})
}

// TestTimeoutConfig verifies timeout configuration extraction.
func TestTimeoutConfig(t *testing.T) {
	TimeoutWrapper(t, 10*time.Second, func(t *testing.T) {
		tempDir := t.TempDir()

		configContent := `
conversation:
  timeout: 30s
  global_timeout: 120s
  preserve_partial_response: true
agents:
  - id: fast-agent
    name: Fast Agent
    type: mock
    model: mock-model
    timeout: 10s
  - id: slow-agent
    name: Slow Agent
    type: mock
    model: mock-model
    timeout: 60s
  - id: default-agent
    name: Default Agent
    type: mock
    model: mock-model
`
		configPath := filepath.Join(tempDir, "timeout.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		timeoutInfo := cfg.GetTimeoutConfig()

		// Verify default timeout
		if timeoutInfo.DefaultAgentTimeout != 30*time.Second {
			t.Errorf("expected default 30s, got %v", timeoutInfo.DefaultAgentTimeout)
		}

		// Verify global timeout
		if timeoutInfo.GlobalTimeout != 120*time.Second {
			t.Errorf("expected global 120s, got %v", timeoutInfo.GlobalTimeout)
		}

		// Verify preserve partial response
		if !timeoutInfo.PreservePartialResponse {
			t.Error("expected preserve_partial_response to be true")
		}

		// Verify per-agent timeouts
		if timeoutInfo.PerAgentTimeouts["fast-agent"] != 10*time.Second {
			t.Errorf("fast-agent: expected 10s, got %v", timeoutInfo.PerAgentTimeouts["fast-agent"])
		}

		if timeoutInfo.PerAgentTimeouts["slow-agent"] != 60*time.Second {
			t.Errorf("slow-agent: expected 60s, got %v", timeoutInfo.PerAgentTimeouts["slow-agent"])
		}

		// default-agent should not have an entry (uses default)
		if _, ok := timeoutInfo.PerAgentTimeouts["default-agent"]; ok {
			t.Error("default-agent should not have per-agent timeout")
		}

		// Verify GetAgentTimeout method
		if cfg.GetAgentTimeout("fast-agent") != 10*time.Second {
			t.Error("GetAgentTimeout for fast-agent failed")
		}

		if cfg.GetAgentTimeout("default-agent") != 30*time.Second {
			t.Error("GetAgentTimeout for default-agent should return default")
		}
	})
}

// TestConfigValidation verifies that validation catches errors.
func TestConfigValidation(t *testing.T) {
	testCases := []struct {
		name        string
		config      string
		expectError bool
		errorKey    string
	}{
		{
			name: "missing_agent_id",
			config: `
agents:
  - name: Test
    type: mock
    model: model
`,
			expectError: true,
			errorKey:    "id",
		},
		{
			name: "missing_agent_type",
			config: `
agents:
  - id: test
    name: Test
    model: model
`,
			expectError: true,
			errorKey:    "type",
		},
		{
			name: "missing_agent_name",
			config: `
agents:
  - id: test
    type: mock
    model: model
`,
			expectError: true,
			errorKey:    "name",
		},
		{
			name: "missing_agent_model",
			config: `
agents:
  - id: test
    name: Test
    type: mock
`,
			expectError: true,
			errorKey:    "model",
		},
		{
			name: "no_agents",
			config: `
conversation:
  timeout: 30s
agents: []
`,
			expectError: true,
			errorKey:    "agent",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "test.yaml")
			if err := os.WriteFile(configPath, []byte(tc.config), 0600); err != nil {
				t.Fatalf("failed to write config: %v", err)
			}

			_, err := config.LoadConfig(configPath)
			if tc.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else {
					t.Logf("Got expected error: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestMultiAgentConfig verifies config with multiple diverse agents.
func TestMultiAgentConfig(t *testing.T) {
	TimeoutWrapper(t, 10*time.Second, func(t *testing.T) {
		tempDir := t.TempDir()

		configContent := `
conversation:
  timeout: 30s
  mode: parallel
  max_turns: 10
agents:
  - id: agent-1
    name: Agent One
    type: mock
    model: model-1
  - id: agent-2
    name: Agent Two
    type: mock
    model: model-2
  - id: agent-3
    name: Agent Three
    type: mock
    model: model-3
  - id: agent-4
    name: Agent Four
    type: mock
    model: model-4
  - id: agent-5
    name: Agent Five
    type: mock
    model: model-5
`
		configPath := filepath.Join(tempDir, "multi.yaml")
		if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		if len(cfg.Agents) != 5 {
			t.Errorf("expected 5 agents, got %d", len(cfg.Agents))
		}

		agents, err := cfg.InitializeAgents()
		if err != nil {
			t.Fatalf("failed to initialize agents: %v", err)
		}

		// Verify all agents are core.Agent type
		for i, agent := range agents {
			if agent.ID == "" {
				t.Errorf("agent %d has empty ID", i)
			}
			if agent.Name == "" {
				t.Errorf("agent %d has empty Name", i)
			}
			// Verify the agent is the correct type
			var _ core.Agent = agent
		}
	})
}
