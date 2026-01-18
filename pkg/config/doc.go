// Package config provides configuration loading and validation for the v2 architecture.
//
// This package handles loading YAML configuration files, automatic migration from
// v1 format, validation, and conversion to runtime types.
//
// # Configuration File Format
//
// The v2 configuration uses YAML with the following structure:
//
//	# v2 Configuration Example
//	conversation:
//	  timeout: 30s              # Per-agent timeout
//	  global_timeout: 2m        # Optional: max time for all agents
//	  mode: parallel            # parallel, round-robin, or reactive
//	  max_turns: 10             # Optional: limit conversation turns
//
//	agents:
//	  - id: claude-1
//	    type: openrouter
//	    name: Claude
//	    model: anthropic/claude-3-sonnet-20240229
//	    timeout: 45s            # Optional: override default timeout
//	    config:
//	      system_prompt: "You are a helpful assistant."
//	      temperature: 0.7
//	      max_tokens: 4096
//	      api_key_env: OPENROUTER_API_KEY
//
//	  - id: gpt-1
//	    type: openrouter
//	    name: GPT-4
//	    model: openai/gpt-4-turbo
//	    config:
//	      system_prompt: "You are a creative writer."
//	      temperature: 0.9
//
//	tui:
//	  enabled: true
//	  show_metrics: true
//	  theme: default
//
//	logging:
//	  level: info
//	  format: text
//
//	persistence:
//	  save_dir: ~/.agentpipe/v2/chats
//	  auto_save: true
//
// # Loading Configuration
//
// Basic loading with automatic v1 migration:
//
//	cfg, err := config.LoadConfig("conversation.yaml")
//	if err != nil {
//		log.Fatal(err)
//	}
//
// Advanced loading with options:
//
//	opts := config.LoadOptions{
//		AutoMigrateV1:      boolPtr(true),   // Enable v1 migration
//		SaveMigratedConfig: true,             // Save migrated config
//	}
//	cfg, err := config.LoadConfigWithOptions("conversation.yaml", opts)
//
// # Initializing Agents
//
// Convert config to runtime agents:
//
//	agents, err := cfg.InitializeAgents()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Use with ConversationManager
//	mgr, err := manager.NewConversationManager(managerCfg, agents, nil)
//
// # Configuration Types
//
// ## ConversationConfig
//
// Controls conversation-level behavior:
//
//	type ConversationConfig struct {
//		Timeout                 time.Duration  // Per-agent timeout
//		GlobalTimeout           time.Duration  // Max time for all agents
//		Mode                    string         // parallel, round-robin, reactive
//		MaxTurns                int            // Limit conversation turns
//		PreservePartialResponse *bool          // Keep partial on timeout
//	}
//
// ## AgentConfig
//
// Defines an AI agent:
//
//	type AgentConfig struct {
//		ID      string               // Unique identifier
//		Type    string               // Agent type (openrouter, claude-api, etc.)
//		Adapter string               // Adapter name (defaults to Type)
//		Name    string               // Display name
//		Model   string               // AI model identifier
//		Timeout time.Duration        // Per-agent timeout override
//		Config  AgentAdapterConfig   // Adapter-specific settings
//	}
//
// ## AgentAdapterConfig
//
// Adapter-specific settings:
//
//	type AgentAdapterConfig struct {
//		SystemPrompt string   // Initial system prompt
//		Temperature  float64  // Response randomness (0.0-2.0)
//		MaxTokens    int      // Maximum response length
//		APIKeyEnv    string   // Environment variable for API key
//	}
//
// # Timeout Configuration
//
// Extract timeout settings for the pool:
//
//	timeoutCfg := cfg.GetTimeoutConfig()
//	fmt.Printf("Default timeout: %s\n", timeoutCfg.DefaultAgentTimeout)
//	fmt.Printf("Global timeout: %s\n", timeoutCfg.GlobalTimeout)
//	fmt.Printf("Preserve partial: %v\n", timeoutCfg.PreservePartialResponse)
//
//	// Per-agent timeouts
//	for agentID, timeout := range timeoutCfg.PerAgentTimeouts {
//		fmt.Printf("Agent %s timeout: %s\n", agentID, timeout)
//	}
//
//	// Get specific agent timeout
//	timeout := cfg.GetAgentTimeout("claude-1")
//
// # V1 Migration
//
// V1 configurations are automatically detected and migrated:
//
//	# v1 format (deprecated)
//	orchestration:
//	  mode: round-robin
//	  turn_limit: 5
//	  timeout: 30
//
//	agents:
//	  - name: Claude
//	    type: claude
//	    cli_path: claude
//	    model_hint: claude-3-sonnet
//
// Becomes:
//
//	# v2 format (migrated)
//	conversation:
//	  mode: round-robin
//	  timeout: 30s
//	  max_turns: 5
//
//	agents:
//	  - id: agent-1
//	    type: claude
//	    name: Claude
//	    model: claude-3-sonnet
//
// Detect v1 format manually:
//
//	data, _ := os.ReadFile("config.yaml")
//	isV1, err := config.DetectV1Config(data)
//	if isV1 {
//		result, err := config.MigrateV1Config(data)
//		if err != nil {
//			log.Fatal(err)
//		}
//		for _, warning := range result.Warnings {
//			log.Warn(warning)
//		}
//	}
//
// # Validation
//
// Configuration is validated automatically on load:
//
//	err := cfg.Validate()
//	if err != nil {
//		// Error messages indicate the specific problem:
//		// - "at least one agent must be configured"
//		// - "agent claude-1: type is required"
//		// - "agent claude-1: unknown adapter: foo"
//		log.Fatal(err)
//	}
//
// Validation checks:
//   - At least one agent is configured
//   - Each agent has required fields (id, type, name, model)
//   - Each agent's adapter is registered
//
// # Defaults
//
// Default values are applied automatically:
//
//	cfg, _ := config.LoadConfig("minimal.yaml")
//	// cfg.Conversation.Timeout = 30s (if not set)
//	// cfg.Conversation.Mode = "parallel" (if not set)
//	// cfg.Logging.Level = "info" (if not set)
//	// cfg.Logging.Format = "text" (if not set)
//	// Agent.Adapter = Agent.Type (if not set)
//
// # Environment Variables
//
// API keys are read from environment variables specified in config:
//
//	agents:
//	  - id: claude-1
//	    config:
//	      api_key_env: OPENROUTER_API_KEY  # Uses $OPENROUTER_API_KEY
//
// The adapter reads the key at runtime:
//
//	apiKey := os.Getenv(agent.Config.APIKeyEnvVar)
//
// # Getting Manager Config
//
// Extract settings for ConversationManager:
//
//	timeout, saveDir := cfg.GetManagerConfig()
//	managerCfg := manager.Config{
//		Timeout: timeout,
//		Persistence: manager.PersistenceConfig{
//			Enabled: cfg.Persistence.AutoSave,
//			SaveDir: saveDir,
//		},
//	}
package config
