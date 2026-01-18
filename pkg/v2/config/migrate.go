// Package config provides configuration loading for the v2 architecture.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/kevinelliott/agentpipe/pkg/log"
)

// V1Config represents the legacy v1 configuration format.
// This is used to detect and parse v1 configurations for migration.
type V1Config struct {
	Version string        `yaml:"version"`
	Agents  []V1AgentSpec `yaml:"agents"`
	// Orchestrator contains v1-style orchestrator settings.
	Orchestrator V1OrchestratorConfig `yaml:"orchestrator,omitempty"`
	// Logging contains v1-style logging settings.
	Logging V1LoggingConfig `yaml:"logging,omitempty"`
}

// V1AgentSpec represents a v1-style agent configuration.
// The key difference is that v1 uses 'type' to specify the agent type (claude, gemini, etc.)
// without an 'adapter' field, and has flattened config fields like 'prompt' instead of nested 'config'.
type V1AgentSpec struct {
	ID           string  `yaml:"id"`
	Type         string  `yaml:"type"`
	Name         string  `yaml:"name"`
	Prompt       string  `yaml:"prompt,omitempty"` // v1 uses "prompt" instead of nested "config.system_prompt"
	Model        string  `yaml:"model,omitempty"`
	Announcement string  `yaml:"announcement,omitempty"`
	Temperature  float64 `yaml:"temperature,omitempty"`
	MaxTokens    int     `yaml:"max_tokens,omitempty"`
}

// V1OrchestratorConfig represents v1-style orchestrator settings.
type V1OrchestratorConfig struct {
	Mode          string        `yaml:"mode,omitempty"`
	MaxTurns      int           `yaml:"max_turns,omitempty"`
	TurnTimeout   time.Duration `yaml:"turn_timeout,omitempty"`
	ResponseDelay time.Duration `yaml:"response_delay,omitempty"`
	InitialPrompt string        `yaml:"initial_prompt,omitempty"`
}

// V1LoggingConfig represents v1-style logging settings.
type V1LoggingConfig struct {
	Enabled     bool `yaml:"enabled,omitempty"`
	ShowMetrics bool `yaml:"show_metrics,omitempty"`
}

// MigrationResult contains the result of a configuration migration.
type MigrationResult struct {
	// Migrated indicates if migration was performed.
	Migrated bool
	// SourceVersion is the original config version.
	SourceVersion string
	// Warnings contains any migration warnings.
	Warnings []string
	// Config is the migrated v2 configuration.
	Config *Config
}

// DetectV1Config checks if the given raw config data is in v1 format.
// V1 configs are identified by:
// - Having agents with 'type' field but no 'adapter' field
// - Having 'orchestrator' section instead of 'conversation'
// - Having flattened agent config (prompt instead of config.system_prompt)
func DetectV1Config(data []byte) (bool, error) {
	// Parse into a generic map to inspect structure
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return false, fmt.Errorf("failed to parse YAML for v1 detection: %w", err)
	}

	// Check for v1 indicators

	// 1. Check for 'orchestrator' key (v1-specific)
	if _, hasOrchestrator := raw["orchestrator"]; hasOrchestrator {
		return true, nil
	}

	// 2. Check for v1-style agents (no 'adapter' field, has 'prompt' field directly)
	if agentsRaw, hasAgents := raw["agents"]; hasAgents {
		if agents, ok := agentsRaw.([]interface{}); ok && len(agents) > 0 {
			for _, agentRaw := range agents {
				if agent, ok := agentRaw.(map[string]interface{}); ok {
					// V1 agents have 'prompt' directly on the agent, not nested in 'config'
					_, hasPrompt := agent["prompt"]
					_, hasAdapter := agent["adapter"]
					_, hasConfig := agent["config"]

					// If has prompt but no config.system_prompt structure, it's v1
					if hasPrompt && !hasConfig {
						return true, nil
					}
					// If has type but no adapter field, and no config block, likely v1
					_, hasType := agent["type"]
					if hasType && !hasAdapter && !hasConfig {
						return true, nil
					}
				}
			}
		}
	}

	// 3. Check for explicit version field indicating v1
	if version, hasVersion := raw["version"]; hasVersion {
		if vstr, ok := version.(string); ok {
			if strings.HasPrefix(vstr, "1.") || vstr == "1" {
				return true, nil
			}
		}
	}

	return false, nil
}

// MigrateV1Config converts a v1 configuration to v2 format.
// Migration rules:
// - type: claude -> adapter: claude-api (if ANTHROPIC_API_KEY set) or claude-cli
// - type: gemini -> adapter: gemini-cli
// - type: openrouter -> adapter: openrouter
// - type: qwen/ollama/codex/etc. -> adapter: cli-generic (with appropriate config)
// - orchestrator.mode -> conversation.mode
// - orchestrator.turn_timeout -> conversation.timeout
// - agent.prompt -> agent.config.system_prompt
func MigrateV1Config(data []byte) (*MigrationResult, error) {
	result := &MigrationResult{
		Migrated:      true,
		SourceVersion: "1.x",
		Warnings:      []string{},
	}

	// Parse v1 config
	var v1 V1Config
	if err := yaml.Unmarshal(data, &v1); err != nil {
		return nil, fmt.Errorf("failed to parse v1 config: %w", err)
	}

	// Track source version if specified
	if v1.Version != "" {
		result.SourceVersion = v1.Version
	}

	// Create v2 config
	v2Config := &Config{
		Agents: make([]AgentConfig, 0, len(v1.Agents)),
	}

	// Migrate orchestrator -> conversation
	v2Config.Conversation = migrateOrchestrator(v1.Orchestrator)

	// Migrate logging settings
	if v1.Logging.Enabled || v1.Logging.ShowMetrics {
		v2Config.TUI.ShowMetrics = v1.Logging.ShowMetrics
		if v1.Logging.Enabled {
			v2Config.Logging.Level = "info"
		}
	}

	// Migrate each agent
	for _, v1Agent := range v1.Agents {
		v2Agent, warnings := migrateAgent(v1Agent)
		v2Config.Agents = append(v2Config.Agents, v2Agent)
		result.Warnings = append(result.Warnings, warnings...)
	}

	result.Config = v2Config
	return result, nil
}

// migrateOrchestrator converts v1 orchestrator settings to v2 conversation settings.
func migrateOrchestrator(v1Orch V1OrchestratorConfig) ConversationConfig {
	v2Conv := ConversationConfig{}

	// Map mode names
	switch strings.ToLower(v1Orch.Mode) {
	case "free-form", "freeform":
		v2Conv.Mode = "reactive" // closest v2 equivalent
	case "round-robin", "roundrobin":
		v2Conv.Mode = "round-robin"
	case "reactive":
		v2Conv.Mode = "reactive"
	default:
		v2Conv.Mode = "parallel" // v2 default
	}

	// Migrate timeout
	if v1Orch.TurnTimeout > 0 {
		v2Conv.Timeout = v1Orch.TurnTimeout
	}

	// Migrate max turns
	v2Conv.MaxTurns = v1Orch.MaxTurns

	return v2Conv
}

// migrateAgent converts a v1 agent spec to v2 agent config.
func migrateAgent(v1Agent V1AgentSpec) (AgentConfig, []string) {
	var warnings []string

	v2Agent := AgentConfig{
		ID:   v1Agent.ID,
		Name: v1Agent.Name,
		Type: v1Agent.Type,
		Config: AgentAdapterConfig{
			Temperature: v1Agent.Temperature,
			MaxTokens:   v1Agent.MaxTokens,
		},
	}

	// Migrate prompt -> system_prompt
	if v1Agent.Prompt != "" {
		v2Agent.Config.SystemPrompt = v1Agent.Prompt
	}

	// Migrate model (set default if not specified)
	if v1Agent.Model != "" {
		v2Agent.Model = v1Agent.Model
	}

	// Determine adapter based on type
	adapter, model, warning := determineAdapter(v1Agent.Type, v1Agent.Model)
	v2Agent.Adapter = adapter
	if v2Agent.Model == "" {
		v2Agent.Model = model
	}
	if warning != "" {
		warnings = append(warnings, fmt.Sprintf("agent %s: %s", v1Agent.ID, warning))
	}

	return v2Agent, warnings
}

// determineAdapter determines the v2 adapter to use based on v1 type.
// Returns (adapter name, default model, optional warning).
func determineAdapter(v1Type, v1Model string) (string, string, string) {
	v1Type = strings.ToLower(v1Type)

	switch v1Type {
	case "claude":
		// Check if ANTHROPIC_API_KEY is set to prefer API over CLI
		if os.Getenv("ANTHROPIC_API_KEY") != "" {
			return "claude-api", getDefaultModel(v1Type, v1Model), ""
		}
		return "claude-cli", getDefaultModel(v1Type, v1Model), "using claude-cli; set ANTHROPIC_API_KEY for API adapter"

	case "gemini":
		return "gemini-cli", getDefaultModel(v1Type, v1Model), ""

	case "openrouter":
		return "openrouter", getDefaultModel(v1Type, v1Model), ""

	case "mock":
		// Mock adapter for testing - maps directly
		return "mock", v1Model, ""

	case "qwen", "ollama", "codex", "aider", "continue", "amp", "cursor", "copilot":
		// These are CLI-based agents in v1, map to cli-generic
		return "cli-generic", getDefaultModel(v1Type, v1Model), fmt.Sprintf("using cli-generic adapter for %s", v1Type)

	default:
		// Unknown type, try cli-generic as fallback
		return "cli-generic", v1Model, fmt.Sprintf("unknown type '%s', falling back to cli-generic", v1Type)
	}
}

// getDefaultModel returns a sensible default model for the agent type.
func getDefaultModel(agentType, specifiedModel string) string {
	if specifiedModel != "" {
		return specifiedModel
	}

	switch strings.ToLower(agentType) {
	case "claude":
		return "claude-3-5-sonnet-20241022"
	case "gemini":
		return "gemini-2.0-flash"
	case "openrouter":
		return "openai/gpt-4o-mini"
	default:
		return "default"
	}
}

// SaveMigratedConfig writes the migrated config to disk with a backup of the original.
func SaveMigratedConfig(originalPath string, config *Config) error {
	// Create backup
	backupPath := originalPath + ".v1.backup"
	originalData, err := os.ReadFile(originalPath)
	if err != nil {
		return fmt.Errorf("failed to read original config for backup: %w", err)
	}

	if err := os.WriteFile(backupPath, originalData, 0644); err != nil {
		return fmt.Errorf("failed to create backup at %s: %w", backupPath, err)
	}

	log.WithFields(map[string]interface{}{
		"backup_path": backupPath,
	}).Info("created backup of v1 config")

	// Marshal v2 config
	newData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal migrated config: %w", err)
	}

	// Add header comment
	header := "# Migrated from v1 format to v2 format\n" +
		"# Original config backed up to: " + filepath.Base(backupPath) + "\n\n"

	if err := os.WriteFile(originalPath, []byte(header+string(newData)), 0644); err != nil {
		return fmt.Errorf("failed to write migrated config: %w", err)
	}

	log.WithFields(map[string]interface{}{
		"path": originalPath,
	}).Info("saved migrated v2 config")

	return nil
}

// AutoMigrateConfig detects and migrates v1 config if needed.
// Returns the config (migrated or original) and migration result.
// If autoSave is true, the migrated config is written to disk.
func AutoMigrateConfig(path string, autoSave bool) (*Config, *MigrationResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read config: %w", err)
	}

	isV1, err := DetectV1Config(data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to detect config version: %w", err)
	}

	if !isV1 {
		// Not v1, parse normally
		return nil, &MigrationResult{Migrated: false}, nil
	}

	// Migrate v1 config
	result, err := MigrateV1Config(data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to migrate v1 config: %w", err)
	}

	// Log deprecation warning
	log.WithFields(map[string]interface{}{
		"path":           path,
		"source_version": result.SourceVersion,
		"warnings":       len(result.Warnings),
	}).Warn("v1 configuration format is deprecated, migrated to v2 format")

	for _, warning := range result.Warnings {
		log.Warn(warning)
	}

	// Optionally save the migrated config
	if autoSave {
		if err := SaveMigratedConfig(path, result.Config); err != nil {
			return nil, nil, fmt.Errorf("failed to save migrated config: %w", err)
		}
	}

	return result.Config, result, nil
}
