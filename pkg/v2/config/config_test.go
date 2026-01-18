package config

import (
	"path/filepath"
	"testing"
	"time"

	_ "github.com/ASRagab/agentpipe/pkg/v2/adapters/mock" // Register mock adapter
)

func TestLoadConfig(t *testing.T) {
	path := filepath.Join("testdata", "valid.yaml")

	config, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	// Check conversation settings
	if config.Conversation.Timeout != 45*time.Second {
		t.Errorf("expected timeout 45s, got %v", config.Conversation.Timeout)
	}
	if config.Conversation.Mode != "parallel" {
		t.Errorf("expected mode 'parallel', got %q", config.Conversation.Mode)
	}
	if config.Conversation.MaxTurns != 10 {
		t.Errorf("expected max_turns 10, got %d", config.Conversation.MaxTurns)
	}

	// Check agents
	if len(config.Agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(config.Agents))
	}

	agent1 := config.Agents[0]
	if agent1.ID != "agent-1" {
		t.Errorf("expected agent ID 'agent-1', got %q", agent1.ID)
	}
	if agent1.Type != "mock" {
		t.Errorf("expected type 'mock', got %q", agent1.Type)
	}
	if agent1.Name != "Test Agent 1" {
		t.Errorf("expected name 'Test Agent 1', got %q", agent1.Name)
	}
	if agent1.Model != "mock-model" {
		t.Errorf("expected model 'mock-model', got %q", agent1.Model)
	}
	// Adapter should default to type
	if agent1.Adapter != "mock" {
		t.Errorf("expected adapter 'mock' (defaulted from type), got %q", agent1.Adapter)
	}
	if agent1.Config.SystemPrompt != "You are a helpful assistant" {
		t.Errorf("unexpected system prompt: %q", agent1.Config.SystemPrompt)
	}
	if agent1.Config.Temperature != 0.7 {
		t.Errorf("expected temperature 0.7, got %f", agent1.Config.Temperature)
	}
	if agent1.Config.MaxTokens != 1000 {
		t.Errorf("expected max_tokens 1000, got %d", agent1.Config.MaxTokens)
	}

	// Check TUI settings
	if !config.TUI.Enabled {
		t.Error("expected TUI enabled")
	}
	if config.TUI.Theme != "dark" {
		t.Errorf("expected theme 'dark', got %q", config.TUI.Theme)
	}
	if !config.TUI.ShowMetrics {
		t.Error("expected show_metrics true")
	}

	// Check logging settings
	if config.Logging.Level != "debug" {
		t.Errorf("expected level 'debug', got %q", config.Logging.Level)
	}
	if config.Logging.Format != "json" {
		t.Errorf("expected format 'json', got %q", config.Logging.Format)
	}
	if config.Logging.File != "/tmp/test.log" {
		t.Errorf("expected file '/tmp/test.log', got %q", config.Logging.File)
	}

	// Check persistence settings
	if config.Persistence.SaveDir != "/tmp/chats" {
		t.Errorf("expected save_dir '/tmp/chats', got %q", config.Persistence.SaveDir)
	}
	if !config.Persistence.AutoSave {
		t.Error("expected auto_save true")
	}
}

func TestLoadConfig_Minimal(t *testing.T) {
	path := filepath.Join("testdata", "minimal.yaml")

	config, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	// Check defaults are applied
	if config.Conversation.Timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", config.Conversation.Timeout)
	}
	if config.Conversation.Mode != "parallel" {
		t.Errorf("expected default mode 'parallel', got %q", config.Conversation.Mode)
	}
	if config.Logging.Level != "info" {
		t.Errorf("expected default level 'info', got %q", config.Logging.Level)
	}
	if config.Logging.Format != "text" {
		t.Errorf("expected default format 'text', got %q", config.Logging.Format)
	}

	// Check agent adapter defaults to type
	if len(config.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(config.Agents))
	}
	if config.Agents[0].Adapter != "mock" {
		t.Errorf("expected adapter 'mock' (defaulted from type), got %q", config.Agents[0].Adapter)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := LoadConfig("testdata/nonexistent.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
	// Error should mention file reading issue
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	path := filepath.Join("testdata", "invalid.yaml")

	_, err := LoadConfig(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadConfig_NoAgents(t *testing.T) {
	path := filepath.Join("testdata", "no_agents.yaml")

	_, err := LoadConfig(path)
	if err == nil {
		t.Error("expected error for empty agents list")
	}
}

func TestLoadConfig_UnknownAdapter(t *testing.T) {
	path := filepath.Join("testdata", "unknown_adapter.yaml")

	_, err := LoadConfig(path)
	if err == nil {
		t.Error("expected error for unknown adapter")
	}
}

func TestLoadConfig_MissingRequiredFields(t *testing.T) {
	path := filepath.Join("testdata", "missing_fields.yaml")

	_, err := LoadConfig(path)
	if err == nil {
		t.Error("expected error for missing required fields")
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				Agents: []AgentConfig{
					{ID: "agent-1", Type: "mock", Name: "Agent 1", Model: "model"},
				},
			},
			wantError: false,
		},
		{
			name: "no agents",
			config: Config{
				Agents: []AgentConfig{},
			},
			wantError: true,
			errorMsg:  "at least one agent",
		},
		{
			name: "missing agent ID",
			config: Config{
				Agents: []AgentConfig{
					{Type: "mock", Name: "Agent", Model: "model"},
				},
			},
			wantError: true,
			errorMsg:  "id is required",
		},
		{
			name: "missing agent type",
			config: Config{
				Agents: []AgentConfig{
					{ID: "agent-1", Name: "Agent", Model: "model"},
				},
			},
			wantError: true,
			errorMsg:  "type is required",
		},
		{
			name: "missing agent name",
			config: Config{
				Agents: []AgentConfig{
					{ID: "agent-1", Type: "mock", Model: "model"},
				},
			},
			wantError: true,
			errorMsg:  "name is required",
		},
		{
			name: "missing agent model",
			config: Config{
				Agents: []AgentConfig{
					{ID: "agent-1", Type: "mock", Name: "Agent"},
				},
			},
			wantError: true,
			errorMsg:  "model is required",
		},
		{
			name: "unknown adapter",
			config: Config{
				Agents: []AgentConfig{
					{ID: "agent-1", Type: "nonexistent", Name: "Agent", Model: "model"},
				},
			},
			wantError: true,
			errorMsg:  "unknown adapter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestInitializeAgents(t *testing.T) {
	config := &Config{
		Agents: []AgentConfig{
			{
				ID:      "agent-1",
				Type:    "mock",
				Adapter: "mock",
				Name:    "Agent 1",
				Model:   "model-1",
				Config: AgentAdapterConfig{
					SystemPrompt: "You are helpful",
					Temperature:  0.5,
					MaxTokens:    500,
					APIKeyEnv:    "API_KEY",
				},
			},
			{
				ID:      "agent-2",
				Type:    "mock",
				Adapter: "mock",
				Name:    "Agent 2",
				Model:   "model-2",
			},
		},
	}

	agents, err := config.InitializeAgents()
	if err != nil {
		t.Fatalf("InitializeAgents returned error: %v", err)
	}

	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}

	// Check first agent
	agent1 := agents[0]
	if agent1.ID != "agent-1" {
		t.Errorf("expected ID 'agent-1', got %q", agent1.ID)
	}
	if agent1.Type != "mock" {
		t.Errorf("expected Type 'mock', got %q", agent1.Type)
	}
	if agent1.Name != "Agent 1" {
		t.Errorf("expected Name 'Agent 1', got %q", agent1.Name)
	}
	if agent1.Model != "model-1" {
		t.Errorf("expected Model 'model-1', got %q", agent1.Model)
	}
	if agent1.AdapterName != "mock" {
		t.Errorf("expected AdapterName 'mock', got %q", agent1.AdapterName)
	}
	if agent1.Config.SystemPrompt != "You are helpful" {
		t.Errorf("expected SystemPrompt 'You are helpful', got %q", agent1.Config.SystemPrompt)
	}
	if agent1.Config.Temperature != 0.5 {
		t.Errorf("expected Temperature 0.5, got %f", agent1.Config.Temperature)
	}
	if agent1.Config.MaxTokens != 500 {
		t.Errorf("expected MaxTokens 500, got %d", agent1.Config.MaxTokens)
	}
	if agent1.Config.APIKeyEnvVar != "API_KEY" {
		t.Errorf("expected APIKeyEnvVar 'API_KEY', got %q", agent1.Config.APIKeyEnvVar)
	}
}

func TestGetManagerConfig(t *testing.T) {
	config := &Config{
		Conversation: ConversationConfig{
			Timeout: 45 * time.Second,
		},
		Persistence: PersistenceConfig{
			SaveDir: "/custom/path",
		},
	}

	timeout, saveDir := config.GetManagerConfig()

	if timeout != 45*time.Second {
		t.Errorf("expected timeout 45s, got %v", timeout)
	}
	if saveDir != "/custom/path" {
		t.Errorf("expected saveDir '/custom/path', got %q", saveDir)
	}
}

func TestApplyDefaults(t *testing.T) {
	config := &Config{
		Agents: []AgentConfig{
			{
				ID:    "agent-1",
				Type:  "mock",
				Name:  "Agent",
				Model: "model",
				// Adapter is empty - should default to Type
			},
		},
	}

	config.applyDefaults()

	// Check conversation defaults
	if config.Conversation.Timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", config.Conversation.Timeout)
	}
	if config.Conversation.Mode != "parallel" {
		t.Errorf("expected default mode 'parallel', got %q", config.Conversation.Mode)
	}

	// Check logging defaults
	if config.Logging.Level != "info" {
		t.Errorf("expected default level 'info', got %q", config.Logging.Level)
	}
	if config.Logging.Format != "text" {
		t.Errorf("expected default format 'text', got %q", config.Logging.Format)
	}

	// Check agent adapter defaults to type
	if config.Agents[0].Adapter != "mock" {
		t.Errorf("expected adapter 'mock' (defaulted from type), got %q", config.Agents[0].Adapter)
	}

	// Check persistence default
	if config.Persistence.SaveDir == "" {
		t.Error("expected default save_dir to be set")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
