package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ASRagab/agentpipe/internal/registry"
	"github.com/ASRagab/agentpipe/pkg/adapters"
	"github.com/ASRagab/agentpipe/pkg/config"
	"github.com/ASRagab/agentpipe/pkg/core"

	// Import v2 adapters to register them for v2 doctor checks
	_ "github.com/ASRagab/agentpipe/pkg/adapters/api"
	_ "github.com/ASRagab/agentpipe/pkg/adapters/cli"
	_ "github.com/ASRagab/agentpipe/pkg/adapters/mock"
)

type AgentCheck struct {
	Name          string `json:"name"`
	Command       string `json:"command"`
	Available     bool   `json:"available"`
	Path          string `json:"path,omitempty"`
	Version       string `json:"version,omitempty"`
	Error         error  `json:"-"`
	ErrorMessage  string `json:"error,omitempty"`
	InstallCmd    string `json:"install_cmd,omitempty"`
	UpgradeCmd    string `json:"upgrade_cmd,omitempty"`
	Docs          string `json:"docs,omitempty"`
	Authenticated bool   `json:"authenticated"`
}

type SystemCheck struct {
	Name    string `json:"name"`
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Icon    string `json:"icon,omitempty"`
}

type DoctorOutput struct {
	SystemEnvironment []SystemCheck `json:"system_environment"`
	SupportedAgents   []AgentCheck  `json:"supported_agents"`
	AvailableAgents   []AgentCheck  `json:"available_agents"`
	Configuration     []SystemCheck `json:"configuration"`
	Summary           DoctorSummary `json:"summary"`
}

type DoctorSummary struct {
	TotalAgents    int      `json:"total_agents"`
	AvailableCount int      `json:"available_count"`
	MissingAgents  []string `json:"missing_agents,omitempty"`
	Ready          bool     `json:"ready"`
}

// V2AgentCheck represents a v2 agent health check result.
type V2AgentCheck struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Model        string `json:"model"`
	Adapter      string `json:"adapter"`
	Available    bool   `json:"available"`
	Healthy      bool   `json:"healthy"`
	APIKeySet    bool   `json:"api_key_set"`
	CLIAvailable bool   `json:"cli_available,omitempty"`
	ResponseTime string `json:"response_time,omitempty"`
	Error        string `json:"error,omitempty"`
}

// V2DoctorOutput contains v2-specific doctor output.
type V2DoctorOutput struct {
	V2Ready            bool           `json:"v2_ready"`
	RegisteredAdapters []string       `json:"registered_adapters"`
	AgentChecks        []V2AgentCheck `json:"agent_checks,omitempty"`
	ConfigFile         string         `json:"config_file,omitempty"`
	ConfigValid        bool           `json:"config_valid"`
	ConfigError        string         `json:"config_error,omitempty"`
	HealthyAgents      int            `json:"healthy_agents"`
	TotalAgents        int            `json:"total_agents"`
}

var (
	doctorJSON   bool
	doctorV2     bool
	doctorConfig string
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check if AI agent CLIs are installed and available",
	Long:  `Doctor command checks your system for installed AI agent CLIs, versions, and configuration.`,
	Run:   runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "Output results in JSON format")
	doctorCmd.Flags().BoolVar(&doctorV2, "v2", false, "Run v2 engine health checks")
	doctorCmd.Flags().StringVarP(&doctorConfig, "config", "c", "", "v2 config file to validate and test agents")
}

func runDoctor(cmd *cobra.Command, args []string) {
	// Check if v2 mode requested
	if doctorV2 {
		runDoctorV2(cmd, args)
		return
	}

	// Get all agents from registry
	registryAgents := registry.GetAll()

	// Perform system checks
	systemChecks := performSystemChecks()

	// Check all agents
	supportedAgents := make([]AgentCheck, 0, len(registryAgents))
	availableAgents := make([]AgentCheck, 0, len(registryAgents))
	unavailableAgents := make([]string, 0, len(registryAgents))

	for _, agent := range registryAgents {
		installCmd, _ := agent.GetInstallCommand()
		upgradeCmd, _ := agent.GetUpgradeCommand()

		check := checkAgent(agent.Command, installCmd)
		check.Name = agent.Name
		check.UpgradeCmd = upgradeCmd
		check.Docs = agent.Docs

		if check.Error != nil {
			check.ErrorMessage = check.Error.Error()
		}

		supportedAgents = append(supportedAgents, check)

		if check.Available {
			availableAgents = append(availableAgents, check)
		} else {
			unavailableAgents = append(unavailableAgents, agent.Name)
		}
	}

	// Configuration checks
	configChecks := performConfigChecks()

	// Build summary
	summary := DoctorSummary{
		TotalAgents:    len(registryAgents),
		AvailableCount: len(availableAgents),
		MissingAgents:  unavailableAgents,
		Ready:          len(availableAgents) > 0,
	}

	// Build complete output
	output := DoctorOutput{
		SystemEnvironment: systemChecks,
		SupportedAgents:   supportedAgents,
		AvailableAgents:   availableAgents,
		Configuration:     configChecks,
		Summary:           summary,
	}

	// Output in requested format
	if doctorJSON {
		jsonOutput, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating JSON output: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonOutput))
	} else {
		printHumanReadableOutput(output)
	}
}

func printHumanReadableOutput(output DoctorOutput) {
	fmt.Println("\n🔍 AgentPipe Doctor - System Health Check")
	fmt.Println(strings.Repeat("=", 61))

	// System environment checks
	fmt.Println("\n📋 SYSTEM ENVIRONMENT")
	fmt.Println(strings.Repeat("-", 61))
	for _, check := range output.SystemEnvironment {
		fmt.Printf("  %s %s: %s\n", check.Icon, check.Name, check.Message)
	}
	fmt.Println()

	// Agent checks
	fmt.Println("\n🤖 AI AGENT CLIS")
	fmt.Println(strings.Repeat("-", 61))

	for i, check := range output.SupportedAgents {
		statusIcon := "❌"
		if check.Available {
			statusIcon = "✅"
		}

		// Add spacing between agents (but not before the first one)
		if i > 0 {
			fmt.Println()
		}

		fmt.Printf("\n  %s %s\n", statusIcon, check.Name)
		fmt.Printf("     Command:  %s\n", check.Command)

		if check.Available {
			fmt.Printf("     Path:     %s\n", check.Path)
			if check.Version != "" {
				fmt.Printf("     Version:  %s\n", check.Version)
			}
			if check.UpgradeCmd != "" {
				fmt.Printf("     Upgrade:  %s\n", check.UpgradeCmd)
			}
			// Check authentication where applicable
			if check.Authenticated {
				fmt.Printf("     Auth:     ✅ Authenticated\n")
			} else if check.Name == "Claude" || check.Name == "Cursor" || check.Name == "Qoder" || check.Name == "Factory" {
				fmt.Printf("     Auth:     ⚠️  Not authenticated (run '%s' and authenticate)\n", check.Command)
			}
		} else {
			fmt.Printf("     Status:   Not installed\n")
			if check.InstallCmd != "" {
				fmt.Printf("     Install:  %s\n", check.InstallCmd)
			}
		}
		fmt.Printf("     Docs:     %s\n", check.Docs)
	}
	fmt.Println()

	// Configuration checks
	fmt.Println("\n⚙️  CONFIGURATION")
	fmt.Println(strings.Repeat("-", 61))
	for _, check := range output.Configuration {
		fmt.Printf("  %s %s: %s\n", check.Icon, check.Name, check.Message)
	}
	fmt.Println()

	// Summary
	fmt.Println("\n" + strings.Repeat("=", 61))
	fmt.Printf("\n📊 SUMMARY\n")
	fmt.Printf("   Available Agents: %d/%d\n", output.Summary.AvailableCount, output.Summary.TotalAgents)

	if len(output.Summary.MissingAgents) > 0 {
		fmt.Printf("   Missing Agents:   %s\n", strings.Join(output.Summary.MissingAgents, ", "))
	}

	if output.Summary.AvailableCount == 0 {
		fmt.Println()
		fmt.Println("⚠️  No AI agents found. Please install at least one agent CLI to use AgentPipe.")
		fmt.Println("   Visit the respective documentation pages above for installation instructions.")
	} else {
		fmt.Println()
		fmt.Printf("✨ AgentPipe is ready! You can use %d agent(s).\n", output.Summary.AvailableCount)
		fmt.Println("   Run 'agentpipe run --help' to start a conversation.")
	}

	fmt.Println()
}

func performSystemChecks() []SystemCheck {
	checks := []SystemCheck{}

	// Go version check
	goVersion := runtime.Version()
	checks = append(checks, SystemCheck{
		Name:    "Go Runtime",
		Status:  true,
		Message: fmt.Sprintf("%s (%s/%s)", goVersion, runtime.GOOS, runtime.GOARCH),
		Icon:    "✅",
	})

	// Check PATH
	pathEnv := os.Getenv("PATH")
	pathCount := len(strings.Split(pathEnv, string(os.PathListSeparator)))
	checks = append(checks, SystemCheck{
		Name:    "PATH",
		Status:  pathCount > 0,
		Message: fmt.Sprintf("%d directories in PATH", pathCount),
		Icon:    "✅",
	})

	// Check home directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		checks = append(checks, SystemCheck{
			Name:    "Home Directory",
			Status:  true,
			Message: homeDir,
			Icon:    "✅",
		})
	}

	// Check agentpipe directories
	agentpipeDir := filepath.Join(homeDir, ".agentpipe")
	chatsDir := filepath.Join(agentpipeDir, "chats")
	statesDir := filepath.Join(agentpipeDir, "states")

	if _, err := os.Stat(chatsDir); err == nil {
		checks = append(checks, SystemCheck{
			Name:    "Chat Logs Directory",
			Status:  true,
			Message: chatsDir,
			Icon:    "✅",
		})
	} else {
		checks = append(checks, SystemCheck{
			Name:    "Chat Logs Directory",
			Status:  false,
			Message: "Will be created on first use",
			Icon:    "ℹ️",
		})
	}

	if _, err := os.Stat(statesDir); err == nil {
		checks = append(checks, SystemCheck{
			Name:    "States Directory",
			Status:  true,
			Message: statesDir,
			Icon:    "✅",
		})
	}

	return checks
}

func performConfigChecks() []SystemCheck {
	checks := []SystemCheck{}

	homeDir, _ := os.UserHomeDir()

	// Check for example configs
	exampleConfigPaths := []string{
		"examples/simple-conversation.yaml",
		"examples/brainstorm.yaml",
	}

	foundExamples := 0
	for _, path := range exampleConfigPaths {
		if _, err := os.Stat(path); err == nil {
			foundExamples++
		}
	}

	if foundExamples > 0 {
		checks = append(checks, SystemCheck{
			Name:    "Example Configs",
			Status:  true,
			Message: fmt.Sprintf("%d example configurations found", foundExamples),
			Icon:    "✅",
		})
	} else {
		checks = append(checks, SystemCheck{
			Name:    "Example Configs",
			Status:  false,
			Message: "No example configs found (expected in ./examples/)",
			Icon:    "ℹ️",
		})
	}

	// Check for user config
	configPath := filepath.Join(homeDir, ".agentpipe", "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		checks = append(checks, SystemCheck{
			Name:    "User Config",
			Status:  true,
			Message: configPath,
			Icon:    "✅",
		})
	} else {
		checks = append(checks, SystemCheck{
			Name:    "User Config",
			Status:  false,
			Message: "No user config (use 'agentpipe init' to create one)",
			Icon:    "ℹ️",
		})
	}

	return checks
}

func checkAgent(command string, installCmd string) AgentCheck {
	check := AgentCheck{
		Name:       command,
		Command:    command,
		InstallCmd: installCmd,
	}

	path, err := exec.LookPath(command)
	if err != nil {
		check.Error = err
		if err == exec.ErrNotFound {
			check.Available = false
		}
		return check
	}

	check.Available = true
	check.Path = path

	// Try to get version
	versionCmd := exec.Command(command, "--version")
	if output, err := versionCmd.CombinedOutput(); err == nil {
		version := strings.TrimSpace(string(output))
		// Clean up version output (take first line if multi-line)
		if lines := strings.Split(version, "\n"); len(lines) > 0 {
			check.Version = strings.TrimSpace(lines[0])
			// Limit version string length
			if len(check.Version) > 60 {
				check.Version = check.Version[:60] + "..."
			}
		}
	} else {
		// Try alternative version commands
		versionCmd = exec.Command(command, "version")
		if output, err := versionCmd.CombinedOutput(); err == nil {
			version := strings.TrimSpace(string(output))
			if lines := strings.Split(version, "\n"); len(lines) > 0 {
				check.Version = strings.TrimSpace(lines[0])
				if len(check.Version) > 60 {
					check.Version = check.Version[:60] + "..."
				}
			}
		}
	}

	// Check authentication status for specific agents
	check.Authenticated = checkAuthentication(command)

	return check
}

func checkAuthentication(command string) bool {
	switch command {
	case "claude":
		// Try a simple command that requires auth
		cmd := exec.Command(command, "--help")
		return cmd.Run() == nil
	case "cursor-agent":
		// Check status command
		cmd := exec.Command(command, "status")
		output, _ := cmd.CombinedOutput()
		return !strings.Contains(strings.ToLower(string(output)), "not logged in")
	case "qodercli":
		// Qoder might need specific auth check
		cmd := exec.Command(command, "--help")
		return cmd.Run() == nil
	case "droid":
		// Factory CLI requires authentication
		cmd := exec.Command(command, "--help")
		return cmd.Run() == nil
	default:
		// Default: assume authenticated if command exists
		return true
	}
}

// runDoctorV2 runs v2-specific health checks.
func runDoctorV2(cmd *cobra.Command, args []string) {
	v2Output := performV2Checks()

	if doctorJSON {
		jsonOutput, err := json.MarshalIndent(v2Output, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating JSON output: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonOutput))
	} else {
		printV2HumanReadableOutput(v2Output)
	}
}

// performV2Checks runs all v2-specific health checks.
func performV2Checks() V2DoctorOutput {
	output := V2DoctorOutput{
		RegisteredAdapters: adapters.List(),
		V2Ready:            true,
	}

	// Check registered adapters
	if len(output.RegisteredAdapters) == 0 {
		output.V2Ready = false
	}

	// If config file provided, validate and test agents
	if doctorConfig != "" {
		output.ConfigFile = doctorConfig
		cfg, err := loadAndValidateV2Config(doctorConfig)
		if err != nil {
			output.ConfigValid = false
			output.ConfigError = err.Error()
			output.V2Ready = false
		} else {
			output.ConfigValid = true
			// Test each agent in the config
			agentChecks := checkV2ConfigAgents(cfg)
			output.AgentChecks = agentChecks
			output.TotalAgents = len(agentChecks)

			healthyCount := 0
			for _, check := range agentChecks {
				if check.Healthy {
					healthyCount++
				}
			}
			output.HealthyAgents = healthyCount

			// V2 is ready if at least one agent is healthy
			output.V2Ready = healthyCount > 0
		}
	}

	return output
}

// loadAndValidateV2Config loads and validates a v2 configuration file.
func loadAndValidateV2Config(configPath string) (*config.Config, error) {
	opts := config.LoadOptions{
		SaveMigratedConfig: false,
	}
	return config.LoadConfigWithOptions(configPath, opts)
}

// checkV2ConfigAgents checks all agents defined in a v2 config file.
func checkV2ConfigAgents(cfg *config.Config) []V2AgentCheck {
	agents, err := cfg.InitializeAgents()
	if err != nil {
		return nil
	}

	checks := make([]V2AgentCheck, 0, len(agents))

	for _, agent := range agents {
		check := checkV2Agent(agent)
		checks = append(checks, check)
	}

	return checks
}

// checkV2Agent performs a health check on a single v2 agent.
func checkV2Agent(agent core.Agent) V2AgentCheck {
	check := V2AgentCheck{
		ID:      agent.ID,
		Name:    agent.Name,
		Type:    agent.Type,
		Model:   agent.Model,
		Adapter: agent.AdapterName,
	}

	// Get the adapter
	adapter, err := adapters.Get(agent.AdapterName)
	if err != nil {
		check.Available = false
		check.Error = fmt.Sprintf("adapter not found: %s", agent.AdapterName)
		return check
	}

	check.Available = true

	// Initialize the adapter
	if err := adapter.Initialize(agent); err != nil {
		check.Error = fmt.Sprintf("initialization failed: %v", err)
		// Check if it's an API key issue
		if strings.Contains(err.Error(), "environment variable") {
			check.APIKeySet = false
		}
		return check
	}

	// Check if adapter is available (API key set for API adapters, CLI binary for CLI adapters)
	if !adapter.IsAvailable() {
		check.Error = "adapter not available (check API key or CLI binary)"
		return check
	}

	check.APIKeySet = true

	// Check if it's a CLI adapter (by checking type)
	if agent.Type == "claude" || agent.Type == "gemini" || agent.Type == "qwen" ||
		agent.Type == "ollama" || agent.Type == "codex" || agent.Type == "continue" {
		// Check CLI availability
		cliPath, cliErr := exec.LookPath(agent.Type)
		check.CLIAvailable = cliErr == nil
		if cliErr != nil {
			check.Error = fmt.Sprintf("CLI binary not found: %s", agent.Type)
			return check
		}
		_ = cliPath // Path found but not displayed in check
	}

	// Perform health check with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	startTime := time.Now()
	healthErr := adapter.HealthCheck(ctx)
	responseTime := time.Since(startTime)

	if healthErr != nil {
		check.Healthy = false
		check.Error = healthErr.Error()
		check.ResponseTime = responseTime.String()
	} else {
		check.Healthy = true
		check.ResponseTime = responseTime.String()
	}

	return check
}

// printV2HumanReadableOutput prints v2 doctor output in human-readable format.
func printV2HumanReadableOutput(output V2DoctorOutput) {
	fmt.Println("\n🔍 AgentPipe Doctor - V2 Engine Health Check")
	fmt.Println(strings.Repeat("=", 61))

	// Registered adapters
	fmt.Println("\n📦 REGISTERED ADAPTERS")
	fmt.Println(strings.Repeat("-", 61))
	if len(output.RegisteredAdapters) == 0 {
		fmt.Println("  ⚠️  No adapters registered")
	} else {
		for _, adapter := range output.RegisteredAdapters {
			fmt.Printf("  ✅ %s\n", adapter)
		}
	}

	// Config validation
	if output.ConfigFile != "" {
		fmt.Println("\n📄 CONFIGURATION")
		fmt.Println(strings.Repeat("-", 61))
		fmt.Printf("  File: %s\n", output.ConfigFile)
		if output.ConfigValid {
			fmt.Println("  ✅ Configuration is valid")
		} else {
			fmt.Printf("  ❌ Configuration error: %s\n", output.ConfigError)
		}
	}

	// Agent checks
	if len(output.AgentChecks) > 0 {
		fmt.Println("\n🤖 AGENT HEALTH CHECKS")
		fmt.Println(strings.Repeat("-", 61))

		for _, check := range output.AgentChecks {
			statusIcon := "❌"
			if check.Healthy {
				statusIcon = "✅"
			} else if check.Available {
				statusIcon = "⚠️"
			}

			fmt.Printf("\n  %s %s (%s)\n", statusIcon, check.Name, check.ID)
			fmt.Printf("     Type:     %s\n", check.Type)
			fmt.Printf("     Model:    %s\n", check.Model)
			fmt.Printf("     Adapter:  %s\n", check.Adapter)

			if check.Available {
				fmt.Println("     Status:   Adapter available")
			} else {
				fmt.Println("     Status:   Adapter not found")
			}

			if check.APIKeySet {
				fmt.Println("     API Key:  ✅ Set")
			} else if check.Error != "" && strings.Contains(check.Error, "API key") {
				fmt.Println("     API Key:  ❌ Not set")
			}

			if check.ResponseTime != "" {
				fmt.Printf("     Response: %s\n", check.ResponseTime)
			}

			if check.Healthy {
				fmt.Println("     Health:   ✅ Healthy")
			} else if check.Error != "" {
				fmt.Printf("     Error:    %s\n", check.Error)
			}
		}
	}

	// Summary
	fmt.Println("\n" + strings.Repeat("=", 61))
	fmt.Println("📊 V2 SUMMARY")

	if len(output.AgentChecks) > 0 {
		fmt.Printf("   Healthy Agents: %d/%d\n", output.HealthyAgents, output.TotalAgents)
	}
	fmt.Printf("   Registered Adapters: %d\n", len(output.RegisteredAdapters))

	if output.V2Ready {
		fmt.Println("\n✨ V2 engine is ready!")
		fmt.Println("   Run 'agentpipe run --v2 -c <config>' to start a v2 conversation.")
	} else {
		fmt.Println("\n⚠️  V2 engine is not ready.")
		if output.ConfigFile == "" {
			fmt.Println("   Provide a config file with --config/-c to check agent health.")
		} else if !output.ConfigValid {
			fmt.Println("   Fix the configuration errors above.")
		} else if output.HealthyAgents == 0 {
			fmt.Println("   No healthy agents found. Check API keys and network connectivity.")
		}
	}

	fmt.Println()
}
