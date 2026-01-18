// Package config provides configuration loading for the v2 architecture.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/kevinelliott/agentpipe/pkg/log"
	"github.com/kevinelliott/agentpipe/pkg/v2/adapters"
	"github.com/kevinelliott/agentpipe/pkg/v2/core"
)

// Config represents the complete v2 configuration file.
type Config struct {
	// Conversation contains conversation-level settings.
	Conversation ConversationConfig `yaml:"conversation"`
	// Agents is the list of agent configurations.
	Agents []AgentConfig `yaml:"agents"`
	// TUI contains terminal UI settings.
	TUI TUIConfig `yaml:"tui,omitempty"`
	// Logging contains logging settings.
	Logging LoggingConfig `yaml:"logging,omitempty"`
	// Persistence contains save/load settings.
	Persistence PersistenceConfig `yaml:"persistence,omitempty"`
}

// ConversationConfig contains conversation-level settings.
type ConversationConfig struct {
	// Timeout is the default timeout for individual agent responses.
	Timeout time.Duration `yaml:"timeout"`
	// GlobalTimeout is the maximum time for all agents combined to respond.
	// If not set or 0, no global timeout is enforced.
	GlobalTimeout time.Duration `yaml:"global_timeout,omitempty"`
	// Mode is the conversation mode (parallel, round-robin, reactive).
	Mode string `yaml:"mode,omitempty"`
	// MaxTurns is the maximum number of conversation turns.
	MaxTurns int `yaml:"max_turns,omitempty"`
	// PreservePartialResponse determines whether to keep partial responses on timeout.
	PreservePartialResponse *bool `yaml:"preserve_partial_response,omitempty"`
}

// AgentConfig represents a single agent's configuration.
type AgentConfig struct {
	// ID is the unique identifier for this agent.
	ID string `yaml:"id"`
	// Type is the agent type (openrouter, claude-api, etc.).
	Type string `yaml:"type"`
	// Adapter is the adapter to use (defaults to type if not specified).
	Adapter string `yaml:"adapter,omitempty"`
	// Name is the display name.
	Name string `yaml:"name"`
	// Model is the AI model to use.
	Model string `yaml:"model"`
	// Timeout is the per-agent timeout (overrides conversation default).
	Timeout time.Duration `yaml:"timeout,omitempty"`
	// Config contains adapter-specific settings.
	Config AgentAdapterConfig `yaml:"config,omitempty"`
}

// AgentAdapterConfig contains adapter-specific configuration.
type AgentAdapterConfig struct {
	// SystemPrompt is the system prompt for the agent.
	SystemPrompt string `yaml:"system_prompt,omitempty"`
	// Temperature controls response randomness.
	Temperature float64 `yaml:"temperature,omitempty"`
	// MaxTokens limits response length.
	MaxTokens int `yaml:"max_tokens,omitempty"`
	// APIKeyEnv is the environment variable for the API key.
	APIKeyEnv string `yaml:"api_key_env,omitempty"`
}

// TUIConfig contains terminal UI settings.
type TUIConfig struct {
	// Enabled determines if TUI should be used.
	Enabled bool `yaml:"enabled,omitempty"`
	// Theme is the color theme.
	Theme string `yaml:"theme,omitempty"`
	// ShowMetrics determines if metrics should be displayed.
	ShowMetrics bool `yaml:"show_metrics,omitempty"`
}

// LoggingConfig contains logging settings.
type LoggingConfig struct {
	// Level is the log level (debug, info, warn, error).
	Level string `yaml:"level,omitempty"`
	// Format is the log format (json, text).
	Format string `yaml:"format,omitempty"`
	// File is the log file path (empty for stdout).
	File string `yaml:"file,omitempty"`
}

// PersistenceConfig contains save/load settings.
type PersistenceConfig struct {
	// SaveDir is the directory for saving conversations.
	SaveDir string `yaml:"save_dir,omitempty"`
	// AutoSave determines if conversations should be auto-saved.
	AutoSave bool `yaml:"auto_save,omitempty"`
}

// LoadConfig loads a configuration file from the given path.
// It automatically detects and migrates v1 configurations to v2 format.
func LoadConfig(path string) (*Config, error) {
	return LoadConfigWithOptions(path, LoadOptions{})
}

// LoadOptions controls configuration loading behavior.
type LoadOptions struct {
	// AutoMigrateV1 enables automatic migration of v1 configs (default: true when not set).
	AutoMigrateV1 *bool
	// SaveMigratedConfig saves the migrated config to disk (default: false).
	SaveMigratedConfig bool
}

// LoadConfigWithOptions loads a configuration file with custom options.
func LoadConfigWithOptions(path string, opts LoadOptions) (*Config, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Check for v1 configuration format
	autoMigrate := true
	if opts.AutoMigrateV1 != nil {
		autoMigrate = *opts.AutoMigrateV1
	}

	if autoMigrate {
		isV1, detectErr := DetectV1Config(data)
		if detectErr != nil {
			log.WithFields(map[string]interface{}{
				"path":  path,
				"error": detectErr.Error(),
			}).Warn("failed to detect v1 config, proceeding with v2 parsing")
		} else if isV1 {
			// Migrate v1 config
			result, migrateErr := MigrateV1Config(data)
			if migrateErr != nil {
				return nil, fmt.Errorf("failed to migrate v1 config: %w", migrateErr)
			}

			// Log deprecation warning
			log.WithFields(map[string]interface{}{
				"path":           path,
				"source_version": result.SourceVersion,
			}).Warn("v1 configuration format is deprecated; automatically migrated to v2 format")

			for _, warning := range result.Warnings {
				log.Warn(warning)
			}

			// Optionally save the migrated config
			if opts.SaveMigratedConfig {
				if saveErr := SaveMigratedConfig(path, result.Config); saveErr != nil {
					log.WithFields(map[string]interface{}{
						"path":  path,
						"error": saveErr.Error(),
					}).Warn("failed to save migrated config")
				}
			}

			// Apply defaults to migrated config
			result.Config.applyDefaults()

			// Validate
			if err := result.Config.Validate(); err != nil {
				return nil, fmt.Errorf("invalid migrated configuration: %w", err)
			}

			log.WithFields(map[string]interface{}{
				"path":        path,
				"agent_count": len(result.Config.Agents),
				"migrated":    true,
			}).Info("configuration loaded (migrated from v1)")

			return result.Config, nil
		}
	}

	// Parse YAML as v2 config
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults
	config.applyDefaults()

	// Validate
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	log.WithFields(map[string]interface{}{
		"path":        path,
		"agent_count": len(config.Agents),
	}).Info("configuration loaded")

	return &config, nil
}

// applyDefaults sets default values for unspecified fields.
func (c *Config) applyDefaults() {
	// Conversation defaults
	if c.Conversation.Timeout == 0 {
		c.Conversation.Timeout = 30 * time.Second
	}
	if c.Conversation.Mode == "" {
		c.Conversation.Mode = "parallel"
	}

	// Agent defaults
	for i := range c.Agents {
		if c.Agents[i].Adapter == "" {
			c.Agents[i].Adapter = c.Agents[i].Type
		}
	}

	// Logging defaults
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "text"
	}

	// Persistence defaults
	if c.Persistence.SaveDir == "" {
		homeDir, _ := os.UserHomeDir()
		c.Persistence.SaveDir = filepath.Join(homeDir, ".agentpipe", "v2", "chats")
	}
}

// Validate checks the configuration for errors.
func (c *Config) Validate() error {
	if len(c.Agents) == 0 {
		return fmt.Errorf("at least one agent must be configured")
	}

	for i, agent := range c.Agents {
		if agent.ID == "" {
			return fmt.Errorf("agent %d: id is required", i)
		}
		if agent.Type == "" {
			return fmt.Errorf("agent %s: type is required", agent.ID)
		}
		if agent.Name == "" {
			return fmt.Errorf("agent %s: name is required", agent.ID)
		}
		if agent.Model == "" {
			return fmt.Errorf("agent %s: model is required", agent.ID)
		}

		// Check if adapter is registered
		adapterName := agent.Adapter
		if adapterName == "" {
			adapterName = agent.Type
		}
		if !adapters.Has(adapterName) {
			return fmt.Errorf("agent %s: unknown adapter: %s", agent.ID, adapterName)
		}
	}

	return nil
}

// InitializeAgents creates and initializes agents from the configuration.
func (c *Config) InitializeAgents() ([]core.Agent, error) {
	agents := make([]core.Agent, 0, len(c.Agents))

	for _, agentCfg := range c.Agents {
		agent := core.Agent{
			ID:          agentCfg.ID,
			Type:        agentCfg.Type,
			Name:        agentCfg.Name,
			Model:       agentCfg.Model,
			AdapterName: agentCfg.Adapter,
			Config: core.AgentAdapterConfig{
				SystemPrompt: agentCfg.Config.SystemPrompt,
				Temperature:  agentCfg.Config.Temperature,
				MaxTokens:    agentCfg.Config.MaxTokens,
				APIKeyEnvVar: agentCfg.Config.APIKeyEnv,
			},
		}

		agents = append(agents, agent)

		log.WithFields(map[string]interface{}{
			"agent_id":   agent.ID,
			"agent_name": agent.Name,
			"adapter":    agent.AdapterName,
			"model":      agent.Model,
		}).Debug("agent initialized from config")
	}

	return agents, nil
}

// GetManagerConfig returns the manager configuration.
func (c *Config) GetManagerConfig() (timeout time.Duration, saveDir string) {
	return c.Conversation.Timeout, c.Persistence.SaveDir
}

// TimeoutConfigInfo contains timeout configuration details extracted from config.
type TimeoutConfigInfo struct {
	// DefaultAgentTimeout is the default timeout for agents without specific timeout.
	DefaultAgentTimeout time.Duration
	// GlobalTimeout is the maximum time for all agents combined (0 = disabled).
	GlobalTimeout time.Duration
	// PreservePartialResponse determines if partial responses should be saved on timeout.
	PreservePartialResponse bool
	// PerAgentTimeouts maps agent IDs to their specific timeouts.
	PerAgentTimeouts map[string]time.Duration
}

// GetTimeoutConfig extracts timeout configuration from the config.
func (c *Config) GetTimeoutConfig() TimeoutConfigInfo {
	info := TimeoutConfigInfo{
		DefaultAgentTimeout:     c.Conversation.Timeout,
		GlobalTimeout:           c.Conversation.GlobalTimeout,
		PreservePartialResponse: true, // default
		PerAgentTimeouts:        make(map[string]time.Duration),
	}

	// Check if preserve partial response is explicitly set
	if c.Conversation.PreservePartialResponse != nil {
		info.PreservePartialResponse = *c.Conversation.PreservePartialResponse
	}

	// Collect per-agent timeouts
	for _, agentCfg := range c.Agents {
		if agentCfg.Timeout > 0 {
			info.PerAgentTimeouts[agentCfg.ID] = agentCfg.Timeout
		}
	}

	return info
}

// GetAgentTimeout returns the timeout for a specific agent.
// Returns the agent-specific timeout if set, otherwise the default.
func (c *Config) GetAgentTimeout(agentID string) time.Duration {
	for _, agentCfg := range c.Agents {
		if agentCfg.ID == agentID && agentCfg.Timeout > 0 {
			return agentCfg.Timeout
		}
	}
	return c.Conversation.Timeout
}
