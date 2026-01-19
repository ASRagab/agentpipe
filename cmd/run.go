// Package cmd provides CLI commands for AgentPipe.
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ASRagab/agentpipe/pkg/config"
	"github.com/ASRagab/agentpipe/pkg/core"
	"github.com/ASRagab/agentpipe/pkg/events"
	"github.com/ASRagab/agentpipe/pkg/manager"
	v2tui "github.com/ASRagab/agentpipe/pkg/tui"
)

// Run command flags
var (
	configPath    string
	useTUI        bool
	jsonOutput    bool
	parallel      bool
	timeout       int
	saveDir       string
	resumeID      string
	exportPath    string
	autoSave      bool
	migrateConfig bool
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start a conversation between AI agents",
	Long: `Start a conversation between multiple AI agents using the v2 engine.

The v2 engine provides parallel execution, persistence, and graceful degradation.

FEATURES:
  • Parallel agent execution (faster responses)
  • Auto-save and resume conversations
  • Circuit breaker for agent failures
  • Graceful degradation when agents fail
  • Markdown export on exit

EXAMPLES:
  # Basic conversation
  agentpipe run -c config.yaml

  # With TUI interface
  agentpipe run -c config.yaml -t

  # With custom timeout and save directory
  agentpipe run -c config.yaml --timeout 90 --save-dir ./chats

  # Resume a previous conversation
  agentpipe run --resume latest
  agentpipe run --resume abc12345

  # Headless mode with piped input
  echo "What is 2+2?" | agentpipe run -c config.yaml

  # Export conversation to Markdown on exit
  agentpipe run -c config.yaml --export conversation.md

INTERACTIVE COMMANDS:
  When running in interactive mode (no piped input), use these commands:
    /save           Save conversation immediately
    /export [file]  Export to Markdown (default: conversation_<timestamp>.md)
    /status         Show agent status and circuit breaker states
    /retry          Retry failed agents (resets circuit breakers)
    /summary        Display conversation summary with metrics
    /help           Show available commands
    /quit           Exit conversation (also /exit, /q)

ENVIRONMENT VARIABLES:
  AGENTPIPE_CONFIG=path   Default config file path
  AGENTPIPE_SAVE_DIR=dir  Directory for conversation saves
  AGENTPIPE_TIMEOUT=60    Default agent timeout in seconds

TROUBLESHOOTING:
  "failed to resume conversation":
    - Check that the conversation ID exists in your save directory
    - Use 'latest' to resume the most recent conversation
    - Verify save-dir matches where the conversation was saved

  "all agents failed":
    - Use /status to see which agents failed and why
    - Use /retry to reset circuit breakers and try again
    - Check agent health with 'agentpipe doctor -c config.yaml'

  "timeout errors":
    - Increase timeout with --timeout flag
    - Check network connectivity to AI providers
    - Some agents (Claude, Cursor) need longer startup times`,
	Run: runConversation,
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Configuration
	runCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to YAML configuration file")

	// Display options
	runCmd.Flags().BoolVarP(&useTUI, "tui", "t", false, "Use TUI interface")
	runCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output events in JSON format (JSONL)")

	// Execution options
	runCmd.Flags().BoolVar(&parallel, "parallel", true, "Enable parallel agent execution")
	runCmd.Flags().IntVar(&timeout, "timeout", 60, "Default agent timeout in seconds")

	// Persistence options
	runCmd.Flags().StringVar(&saveDir, "save-dir", "", "Directory for conversation saves")
	runCmd.Flags().StringVar(&resumeID, "resume", "", "Resume a saved conversation (use 'latest' for most recent)")
	runCmd.Flags().StringVar(&exportPath, "export", "", "Export conversation to Markdown on exit")
	runCmd.Flags().BoolVar(&autoSave, "auto-save", true, "Enable auto-save")

	// Migration options
	runCmd.Flags().BoolVar(&migrateConfig, "migrate-config", false, "Migrate v1 config to v2 format and save")

	// Bind environment variables
	viper.SetDefault("AGENTPIPE_SAVE_DIR", "")
	viper.SetDefault("AGENTPIPE_TIMEOUT", 60)
}

// runConversation runs a conversation using the v2 engine.
func runConversation(cmd *cobra.Command, args []string) {
	if err := executeConversation(cmd, configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// executeConversation runs the conversation with the v2 engine.
func executeConversation(cmd *cobra.Command, cfgPath string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Suppress console logging in TUI mode EARLY to prevent startup logs
	// from appearing in terminal history after TUI exits.
	if useTUI {
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Fprintln(os.Stderr, "\n\nInterrupted. Shutting down gracefully...")
		cancel()
	}()

	// Load configuration
	cfg, err := loadConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Apply CLI overrides
	if timeout > 0 {
		cfg.Conversation.Timeout = time.Duration(timeout) * time.Second
	}
	if saveDir != "" {
		cfg.Persistence.SaveDir = saveDir
	}
	if envSaveDir := os.Getenv("AGENTPIPE_SAVE_DIR"); envSaveDir != "" && saveDir == "" {
		cfg.Persistence.SaveDir = envSaveDir
	}
	if envTimeout := os.Getenv("AGENTPIPE_TIMEOUT"); envTimeout != "" && timeout == 60 {
		if t, parseErr := time.ParseDuration(envTimeout + "s"); parseErr == nil {
			cfg.Conversation.Timeout = t
		}
	}
	cfg.Persistence.AutoSave = autoSave

	// Initialize agents from config
	agents, err := cfg.InitializeAgents()
	if err != nil {
		return fmt.Errorf("failed to initialize agents: %w", err)
	}

	if len(agents) == 0 {
		return fmt.Errorf("no agents configured")
	}

	// Create event bus
	eventBus := events.NewBus()
	defer eventBus.Close()

	// Create manager config
	managerCfg := manager.Config{
		Timeout:          cfg.Conversation.Timeout,
		SaveDir:          cfg.Persistence.SaveDir,
		ConversationMode: cfg.Conversation.Mode,
		MaxTurns:         cfg.Conversation.MaxTurns,
		Persistence: manager.PersistenceConfig{
			Enabled: cfg.Persistence.AutoSave,
			SaveDir: cfg.Persistence.SaveDir,
		},
		GracefulDegradation: manager.DefaultGracefulDegradationConfig(),
	}

	// Create conversation manager
	mgr, err := manager.NewConversationManager(managerCfg, agents, eventBus)
	if err != nil {
		return fmt.Errorf("failed to create conversation manager: %w", err)
	}
	defer mgr.Close()

	// Resume if requested
	if resumeID != "" {
		if err := mgr.Resume(resumeID); err != nil {
			return fmt.Errorf("failed to resume conversation: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Resumed conversation from: %s\n", resumeID)
	}

	// Start conversation
	mgr.Start()

	// Run TUI or headless mode
	if useTUI {
		return v2tui.RunWithContext(ctx, mgr, eventBus)
	}

	return runHeadless(ctx, mgr, eventBus, cfg)
}

// loadConfig loads and possibly migrates configuration.
func loadConfig(cfgPath string) (*config.Config, error) {
	if cfgPath == "" {
		// Check environment variable
		if envConfig := os.Getenv("AGENTPIPE_CONFIG"); envConfig != "" {
			cfgPath = envConfig
		} else {
			return nil, fmt.Errorf("config file required (use -c or AGENTPIPE_CONFIG)")
		}
	}

	// Check for v1 config and display user-visible warning
	if err := checkAndWarnV1Config(cfgPath); err != nil {
		// Non-fatal - just log the detection error
		fmt.Fprintf(os.Stderr, "Warning: Could not check config version: %v\n", err)
	}

	opts := config.LoadOptions{
		SaveMigratedConfig: migrateConfig,
	}

	cfg, err := config.LoadConfigWithOptions(cfgPath, opts)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// checkAndWarnV1Config checks if the config file is in v1 format and warns the user.
func checkAndWarnV1Config(cfgPath string) error {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	isV1, err := config.DetectV1Config(data)
	if err != nil {
		return fmt.Errorf("failed to detect config version: %w", err)
	}

	if isV1 {
		// Display user-visible deprecation warning
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "┌──────────────────────────────────────────────────────────────────┐")
		fmt.Fprintln(os.Stderr, "│  ⚠️  DEPRECATION WARNING: v1 configuration format detected        │")
		fmt.Fprintln(os.Stderr, "├──────────────────────────────────────────────────────────────────┤")
		fmt.Fprintln(os.Stderr, "│  Your config file uses the legacy v1 format which is deprecated. │")
		fmt.Fprintln(os.Stderr, "│  The config will be automatically migrated in memory for now.    │")
		fmt.Fprintln(os.Stderr, "│                                                                  │")
		fmt.Fprintln(os.Stderr, "│  To permanently migrate your config file, run:                   │")
		fmt.Fprintf(os.Stderr, "│    agentpipe run --migrate-config -c %s\n", truncateForBox(cfgPath, 20))
		fmt.Fprintln(os.Stderr, "│                                                                  │")
		fmt.Fprintln(os.Stderr, "│  This will:                                                      │")
		fmt.Fprintln(os.Stderr, "│    • Backup your original config to <filename>.v1.backup        │")
		fmt.Fprintln(os.Stderr, "│    • Convert to v2 format with parallel execution support       │")
		fmt.Fprintln(os.Stderr, "│    • Add new v2 features: timeouts, persistence, etc.           │")
		fmt.Fprintln(os.Stderr, "│                                                                  │")
		fmt.Fprintln(os.Stderr, "│  See: agentpipe run --help for options                           │")
		fmt.Fprintln(os.Stderr, "└──────────────────────────────────────────────────────────────────┘")
		fmt.Fprintln(os.Stderr, "")

		// If --migrate-config flag was passed, confirm the migration
		if migrateConfig {
			fmt.Fprintln(os.Stderr, "📁 Migrating config file with backup...")
		}
	}

	return nil
}

// truncateForBox truncates a string to fit in the warning box.
func truncateForBox(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return "..." + s[len(s)-maxLen:]
}

// runHeadless runs in headless (non-TUI) mode.
func runHeadless(ctx context.Context, mgr *manager.ConversationManager, eventBus *events.Bus, cfg *config.Config) error {
	verbose := viper.GetBool("verbose")

	// Print startup info
	if !jsonOutput {
		fmt.Fprintln(os.Stderr, "🚀 AgentPipe")
		fmt.Fprintf(os.Stderr, "Mode: parallel | Timeout: %s | Agents: %d\n",
			cfg.Conversation.Timeout.String(), len(mgr.GetAgents()))
		if mgr.IsResumed() {
			fmt.Fprintln(os.Stderr, "📂 Resumed from saved conversation")
		}
		fmt.Fprintln(os.Stderr, strings.Repeat("=", 60))
	}

	// Subscribe to events for output
	eventBus.Subscribe(core.EventMessageCreated, func(event core.Event) {
		if msg, ok := event.Data.(core.Message); ok {
			printMessage(msg, verbose)
		}
	})

	// Check if we have piped input
	stat, _ := os.Stdin.Stat()
	isPiped := (stat.Mode() & os.ModeCharDevice) == 0

	if isPiped {
		return runPipedMode(ctx, mgr)
	}

	return runInteractiveMode(ctx, mgr, eventBus)
}

// runPipedMode handles piped input (non-interactive).
func runPipedMode(ctx context.Context, mgr *manager.ConversationManager) error {
	// Read all input
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	content := strings.TrimSpace(string(input))
	if content == "" {
		return fmt.Errorf("no input provided")
	}

	// Send message and wait for responses
	responses, err := mgr.SendUserMessage(ctx, content)
	if err != nil {
		// Print responses even if some agents failed
		for _, msg := range responses {
			printMessage(msg, false)
		}
		return fmt.Errorf("error during message processing: %w", err)
	}

	// Mark complete and print summary
	mgr.Complete()
	printSummary(mgr)

	return nil
}

// runInteractiveMode runs the interactive headless mode.
func runInteractiveMode(ctx context.Context, mgr *manager.ConversationManager, eventBus *events.Bus) error {
	reader := bufio.NewReader(os.Stdin)

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			mgr.Complete()
			printSummary(mgr)
			return nil
		default:
		}

		// Prompt for input
		if !jsonOutput {
			fmt.Fprint(os.Stderr, "\n💬 You: ")
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				mgr.Complete()
				printSummary(mgr)
				return nil
			}
			return fmt.Errorf("failed to read input: %w", err)
		}

		content := strings.TrimSpace(line)
		if content == "" {
			continue
		}

		// Handle special commands
		if handleCommand(content, mgr) {
			continue
		}

		// Check for exit commands
		if content == "/quit" || content == "/exit" || content == "/q" {
			mgr.Complete()
			printSummary(mgr)
			return nil
		}

		// Send message
		if !jsonOutput {
			fmt.Fprintln(os.Stderr, "")
		}

		_, err = mgr.SendUserMessage(ctx, content)
		if err != nil {
			if err == manager.ErrAllAgentsFailed {
				fmt.Fprintln(os.Stderr, "⚠️  All agents failed. Conversation paused.")
				fmt.Fprintln(os.Stderr, "Type /retry to retry failed agents, or /status to see agent status.")
				continue
			}
			// Non-fatal error, continue
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}
}

// handleCommand handles special commands.
// Returns true if a command was handled.
func handleCommand(input string, mgr *manager.ConversationManager) bool {
	if !strings.HasPrefix(input, "/") {
		return false
	}

	parts := strings.SplitN(input, " ", 2)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "/save":
		path, err := mgr.Save()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to save: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "💾 Saved to: %s\n", path)
		}
		return true

	case "/export":
		var outputPath string
		if len(parts) > 1 {
			outputPath = strings.TrimSpace(parts[1])
		} else {
			outputPath = fmt.Sprintf("conversation_%s.md", time.Now().Format("2006-01-02_15-04-05"))
		}
		if err := mgr.ExportToMarkdown(outputPath); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to export: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "📝 Exported to: %s\n", outputPath)
		}
		return true

	case "/status":
		printAgentStatus(mgr)
		return true

	case "/retry":
		if mgr.IsPaused() {
			mgr.ResetCircuitBreakers()
			mgr.ResumeConversation(true)
			fmt.Fprintln(os.Stderr, "🔄 Conversation resumed. Circuit breakers reset.")
		} else {
			fmt.Fprintln(os.Stderr, "ℹ️  Conversation is not paused.")
		}
		return true

	case "/summary":
		printSummary(mgr)
		return true

	case "/help":
		printHelp()
		return true
	}

	return false
}

// printMessage prints a message to stdout.
func printMessage(msg core.Message, verbose bool) {
	switch msg.Role {
	case core.RoleAgent:
		if jsonOutput {
			return
		}
		fmt.Printf("\n🤖 %s:\n", msg.AgentName)
		fmt.Println(msg.Content)

		if verbose && msg.Metrics != nil {
			fmt.Fprintf(os.Stderr, "  [%s | %d tokens | $%.4f]\n",
				msg.Metrics.Duration.Round(time.Millisecond),
				msg.Metrics.TotalTokens,
				msg.Metrics.Cost)
		}

	case core.RoleSystem:
		if !jsonOutput {
			fmt.Fprintf(os.Stderr, "ℹ️  %s\n", msg.Content)
		}

	case core.RoleUser:
		// User messages already shown at input
	}
}

// printAgentStatus prints the status of all agents.
func printAgentStatus(mgr *manager.ConversationManager) {
	fmt.Fprintln(os.Stderr, "\n📊 Agent Status")
	fmt.Fprintln(os.Stderr, strings.Repeat("-", 40))

	agents := mgr.GetAgents()
	status := mgr.GetAgentStatus()
	failedAgents := mgr.GetFailedAgents()

	failedMap := make(map[string]*manager.FailedAgentInfo)
	for _, info := range failedAgents {
		infoCopy := info
		failedMap[info.AgentID] = &infoCopy
	}

	for _, agent := range agents {
		statusIcon := "✅"
		statusStr := string(status[agent.ID])

		if failed, ok := failedMap[agent.ID]; ok {
			statusIcon = "❌"
			statusStr = fmt.Sprintf("error (retries: %d)", failed.RetryCount)
		} else if status[agent.ID] == core.AgentStatusTyping {
			statusIcon = "⏳"
		}

		fmt.Fprintf(os.Stderr, "%s %s (%s): %s\n", statusIcon, agent.Name, agent.Model, statusStr)
	}

	if mgr.IsPaused() {
		fmt.Fprintln(os.Stderr, "\n⚠️  Conversation is PAUSED due to all agents failing.")
		fmt.Fprintln(os.Stderr, "Type /retry to resume.")
	}
}

// printSummary prints the conversation summary.
func printSummary(mgr *manager.ConversationManager) {
	if jsonOutput {
		return
	}

	summary := mgr.Summary()

	// Calculate turn count from user messages
	turnCount := 0
	for _, msg := range mgr.GetMessages() {
		if msg.Role == core.RoleUser {
			turnCount++
		}
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, strings.Repeat("=", 60))
	fmt.Fprintln(os.Stderr, "📊 Conversation Summary")
	fmt.Fprintln(os.Stderr, strings.Repeat("-", 40))
	fmt.Fprintf(os.Stderr, "Messages:     %d\n", summary.MessageCount)
	fmt.Fprintf(os.Stderr, "Turns:        %d\n", turnCount)
	fmt.Fprintf(os.Stderr, "Total Tokens: %d\n", summary.TotalTokens)
	fmt.Fprintf(os.Stderr, "Total Cost:   $%.4f\n", summary.TotalCost)
	fmt.Fprintf(os.Stderr, "Duration:     %s\n", summary.Duration.Round(time.Millisecond))

	// Export if requested
	if exportPath != "" {
		if err := mgr.ExportToMarkdown(exportPath); err != nil {
			fmt.Fprintf(os.Stderr, "\n❌ Failed to export: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "\n📝 Exported to: %s\n", exportPath)
		}
	}

	// Print save location if auto-save was enabled
	if autoSave {
		saveID := mgr.GetConversation().ID
		if saveID != "" {
			fmt.Fprintf(os.Stderr, "\n💾 Conversation ID: %s\n", saveID[:8])
			fmt.Fprintf(os.Stderr, "   Use --resume %s to continue later\n", saveID[:8])
		}
	}
}

// printHelp prints available commands.
func printHelp() {
	fmt.Fprintln(os.Stderr, "\n📖 Available Commands")
	fmt.Fprintln(os.Stderr, strings.Repeat("-", 40))
	fmt.Fprintln(os.Stderr, "/save           Save conversation")
	fmt.Fprintln(os.Stderr, "/export [file]  Export to Markdown")
	fmt.Fprintln(os.Stderr, "/status         Show agent status")
	fmt.Fprintln(os.Stderr, "/retry          Retry failed agents")
	fmt.Fprintln(os.Stderr, "/summary        Show conversation summary")
	fmt.Fprintln(os.Stderr, "/quit           Exit conversation")
	fmt.Fprintln(os.Stderr, "/help           Show this help")
}
