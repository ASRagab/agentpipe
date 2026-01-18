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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kevinelliott/agentpipe/pkg/v2/config"
	"github.com/kevinelliott/agentpipe/pkg/v2/core"
	"github.com/kevinelliott/agentpipe/pkg/v2/events"
	"github.com/kevinelliott/agentpipe/pkg/v2/manager"
	"github.com/kevinelliott/agentpipe/pkg/v2/persistence"
	v2tui "github.com/kevinelliott/agentpipe/pkg/v2/tui"
)

// V2 flags
var (
	v2Enabled      bool
	v2Parallel     bool
	v2Timeout      int
	v2SaveDir      string
	v2Resume       string
	v2Export       string
	v2AutoSave     bool
	v2MigrateConfig bool
)

func init() {
	// Add v2 flags to run command
	runCmd.Flags().BoolVar(&v2Enabled, "v2", false, "Use v2 engine for parallel execution")
	runCmd.Flags().BoolVar(&v2Parallel, "parallel", true, "Enable parallel agent execution (v2 only)")
	runCmd.Flags().IntVar(&v2Timeout, "v2-timeout", 60, "Default agent timeout in seconds (v2 only)")
	runCmd.Flags().StringVar(&v2SaveDir, "save-dir", "", "Directory for conversation saves (v2 only)")
	runCmd.Flags().StringVar(&v2Resume, "resume", "", "Resume a saved conversation (v2 only, use 'latest' for most recent)")
	runCmd.Flags().StringVar(&v2Export, "export", "", "Export conversation to Markdown on exit (v2 only)")
	runCmd.Flags().BoolVar(&v2AutoSave, "auto-save", true, "Enable auto-save (v2 only)")
	runCmd.Flags().BoolVar(&v2MigrateConfig, "migrate-config", false, "Migrate v1 config to v2 format and save")

	// Bind environment variables
	viper.SetDefault("AGENTPIPE_V2", false)
	viper.SetDefault("AGENTPIPE_SAVE_DIR", "")
	viper.SetDefault("AGENTPIPE_TIMEOUT", 60)
}

// shouldUseV2 checks if v2 engine should be used based on flags and environment.
func shouldUseV2() bool {
	// Explicit flag takes precedence
	if v2Enabled {
		return true
	}
	// Check environment variable
	if viper.GetBool("AGENTPIPE_V2") || os.Getenv("AGENTPIPE_V2") == "true" {
		return true
	}
	// Check if resume flag is set (implies v2)
	if v2Resume != "" {
		return true
	}
	return false
}

// runV2Conversation runs a conversation using the v2 engine.
func runV2Conversation(cmd *cobra.Command, configPath string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Fprintln(os.Stderr, "\n\nInterrupted. Shutting down gracefully...")
		cancel()
	}()

	// Load configuration with v2 format detection and migration
	cfg, err := loadV2Config(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Apply CLI overrides
	if v2Timeout > 0 {
		cfg.Conversation.Timeout = time.Duration(v2Timeout) * time.Second
	}
	if v2SaveDir != "" {
		cfg.Persistence.SaveDir = v2SaveDir
	}
	if envSaveDir := os.Getenv("AGENTPIPE_SAVE_DIR"); envSaveDir != "" && v2SaveDir == "" {
		cfg.Persistence.SaveDir = envSaveDir
	}
	if envTimeout := os.Getenv("AGENTPIPE_TIMEOUT"); envTimeout != "" && v2Timeout == 60 {
		if t, parseErr := time.ParseDuration(envTimeout + "s"); parseErr == nil {
			cfg.Conversation.Timeout = t
		}
	}
	cfg.Persistence.AutoSave = v2AutoSave

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
		Timeout: cfg.Conversation.Timeout,
		SaveDir: cfg.Persistence.SaveDir,
		Persistence: manager.PersistenceConfig{
			Enabled:  cfg.Persistence.AutoSave,
			SaveDir:  cfg.Persistence.SaveDir,
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
	if v2Resume != "" {
		if err := mgr.Resume(v2Resume); err != nil {
			return fmt.Errorf("failed to resume conversation: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Resumed conversation from: %s\n", v2Resume)
	}

	// Start conversation
	mgr.Start()

	// Run TUI or headless mode
	if useTUI {
		return runV2TUI(ctx, mgr, eventBus)
	}

	return runV2Headless(ctx, mgr, eventBus, cfg)
}

// loadV2Config loads and possibly migrates configuration.
func loadV2Config(configPath string) (*config.Config, error) {
	if configPath == "" {
		// Check environment variable
		if envConfig := os.Getenv("AGENTPIPE_CONFIG"); envConfig != "" {
			configPath = envConfig
		} else {
			return nil, fmt.Errorf("config file required for v2 mode (use -c or AGENTPIPE_CONFIG)")
		}
	}

	// Check for v1 config and display user-visible warning
	if err := checkAndWarnV1Config(configPath); err != nil {
		// Non-fatal - just log the detection error
		fmt.Fprintf(os.Stderr, "Warning: Could not check config version: %v\n", err)
	}

	opts := config.LoadOptions{
		SaveMigratedConfig: v2MigrateConfig,
	}

	cfg, err := config.LoadConfigWithOptions(configPath, opts)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// checkAndWarnV1Config checks if the config file is in v1 format and warns the user.
func checkAndWarnV1Config(configPath string) error {
	data, err := os.ReadFile(configPath)
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
		fmt.Fprintf(os.Stderr, "│    agentpipe run --v2 --migrate-config -c %s\n", truncateForBox(configPath, 20))
		fmt.Fprintln(os.Stderr, "│                                                                  │")
		fmt.Fprintln(os.Stderr, "│  This will:                                                      │")
		fmt.Fprintln(os.Stderr, "│    • Backup your original config to <filename>.v1.backup        │")
		fmt.Fprintln(os.Stderr, "│    • Convert to v2 format with parallel execution support       │")
		fmt.Fprintln(os.Stderr, "│    • Add new v2 features: timeouts, persistence, etc.           │")
		fmt.Fprintln(os.Stderr, "│                                                                  │")
		fmt.Fprintln(os.Stderr, "│  See: agentpipe run --help for v2 options                        │")
		fmt.Fprintln(os.Stderr, "└──────────────────────────────────────────────────────────────────┘")
		fmt.Fprintln(os.Stderr, "")

		// If --migrate-config flag was passed, confirm the migration
		if v2MigrateConfig {
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

// runV2TUI runs the v2 TUI mode.
func runV2TUI(ctx context.Context, mgr *manager.ConversationManager, eventBus *events.Bus) error {
	return v2tui.RunWithContext(ctx, mgr, eventBus)
}

// runV2Headless runs v2 in headless (non-TUI) mode.
func runV2Headless(ctx context.Context, mgr *manager.ConversationManager, eventBus *events.Bus, cfg *config.Config) error {
	verbose := viper.GetBool("verbose")

	// Print startup info
	if !jsonOutput {
		fmt.Fprintln(os.Stderr, "🚀 AgentPipe v2 Engine")
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
			printV2Message(msg, verbose)
		}
	})

	// Check if we have piped input
	stat, _ := os.Stdin.Stat()
	isPiped := (stat.Mode() & os.ModeCharDevice) == 0

	if isPiped {
		// Read piped input
		return runV2PipedMode(ctx, mgr)
	}

	// Interactive mode
	return runV2InteractiveMode(ctx, mgr, eventBus)
}

// runV2PipedMode handles piped input (non-interactive).
func runV2PipedMode(ctx context.Context, mgr *manager.ConversationManager) error {
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
			printV2Message(msg, false)
		}
		return fmt.Errorf("error during message processing: %w", err)
	}

	// Mark complete and print summary
	mgr.Complete()
	printV2Summary(mgr)

	return nil
}

// runV2InteractiveMode runs the interactive headless mode.
func runV2InteractiveMode(ctx context.Context, mgr *manager.ConversationManager, eventBus *events.Bus) error {
	reader := bufio.NewReader(os.Stdin)

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			mgr.Complete()
			printV2Summary(mgr)
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
				// End of input
				mgr.Complete()
				printV2Summary(mgr)
				return nil
			}
			return fmt.Errorf("failed to read input: %w", err)
		}

		content := strings.TrimSpace(line)
		if content == "" {
			continue
		}

		// Handle special commands
		if handleV2Command(content, mgr) {
			continue
		}

		// Check for exit commands
		if content == "/quit" || content == "/exit" || content == "/q" {
			mgr.Complete()
			printV2Summary(mgr)
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

// handleV2Command handles special v2 commands.
// Returns true if a command was handled.
func handleV2Command(input string, mgr *manager.ConversationManager) bool {
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
		printV2AgentStatus(mgr)
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
		printV2Summary(mgr)
		return true

	case "/help":
		printV2Help()
		return true
	}

	return false
}

// printV2Message prints a v2 message to stdout.
func printV2Message(msg core.Message, verbose bool) {
	switch msg.Role {
	case core.RoleAgent:
		if jsonOutput {
			// JSON mode handled by event bus
			return
		}
		// Print agent response
		fmt.Printf("\n🤖 %s:\n", msg.AgentName)
		fmt.Println(msg.Content)

		// Print metrics if verbose
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

// printV2AgentStatus prints the status of all agents.
func printV2AgentStatus(mgr *manager.ConversationManager) {
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

// printV2Summary prints the conversation summary.
func printV2Summary(mgr *manager.ConversationManager) {
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
	if v2Export != "" {
		if err := mgr.ExportToMarkdown(v2Export); err != nil {
			fmt.Fprintf(os.Stderr, "\n❌ Failed to export: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "\n📝 Exported to: %s\n", v2Export)
		}
	}

	// Print save location if auto-save was enabled
	if v2AutoSave {
		saveDir := mgr.GetConversation().ID
		if saveDir != "" {
			fmt.Fprintf(os.Stderr, "\n💾 Conversation ID: %s\n", saveDir[:8])
			fmt.Fprintf(os.Stderr, "   Use --resume %s to continue later\n", saveDir[:8])
		}
	}
}

// printV2Help prints available v2 commands.
func printV2Help() {
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

// listV2Conversations lists saved v2 conversations.
func listV2Conversations(saveDir string) error {
	if saveDir == "" {
		saveDir = persistence.DefaultSaveDir()
	}

	conversations, err := persistence.ListConversations(saveDir)
	if err != nil {
		return fmt.Errorf("failed to list conversations: %w", err)
	}

	if len(conversations) == 0 {
		fmt.Println("No saved conversations found.")
		return nil
	}

	fmt.Println("Saved Conversations:")
	fmt.Println(strings.Repeat("-", 60))

	for _, conv := range conversations {
		fmt.Printf("  %s  %s  (%d messages)\n",
			conv.ID[:8],
			conv.Started.Format("2006-01-02 15:04"),
			conv.MessageCount)
	}

	return nil
}
