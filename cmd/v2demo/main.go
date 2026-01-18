// Package main provides a demonstration CLI for the v2 AgentPipe architecture.
// It loads a configuration file, initializes agents, and sends a test message
// to all agents in parallel, displaying the responses.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ASRagab/agentpipe/pkg/log"
	"github.com/ASRagab/agentpipe/pkg/v2/core"
	"github.com/ASRagab/agentpipe/pkg/v2/events"
	"github.com/ASRagab/agentpipe/pkg/v2/manager"
	"github.com/ASRagab/agentpipe/pkg/v2/persistence"

	// Import adapters to register them
	_ "github.com/ASRagab/agentpipe/pkg/v2/adapters/api"
	"github.com/ASRagab/agentpipe/pkg/v2/config"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "examples/v2/demo-config.yaml", "Path to configuration file")
	message := flag.String("message", "Explain async/await in JavaScript in one sentence.", "Message to send to agents")
	debug := flag.Bool("debug", false, "Enable debug logging")

	// Persistence flags
	save := flag.Bool("save", false, "Save conversation after completion")
	resume := flag.String("resume", "", "Resume from conversation ID, file path, or 'latest'")
	export := flag.String("export", "", "Export completed conversation to Markdown file")
	listSaved := flag.Bool("list", false, "List saved conversations and exit")

	flag.Parse()

	// Configure logging
	if *debug {
		log.SetGlobalLevel(log.ParseLevel("debug"))
	}

	// Handle list command
	if *listSaved {
		listConversations()
		return
	}

	// Load configuration
	fmt.Printf("📂 Loading configuration from: %s\n", *configPath)
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("❌ Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize agents
	fmt.Printf("🤖 Initializing %d agents...\n", len(cfg.Agents))
	agents, err := cfg.InitializeAgents()
	if err != nil {
		fmt.Printf("❌ Failed to initialize agents: %v\n", err)
		os.Exit(1)
	}

	// Print agent info
	for _, agent := range agents {
		fmt.Printf("   • %s (%s) - %s\n", agent.Name, agent.Type, agent.Model)
	}

	// Create event bus
	eventBus := events.NewBus()

	// Subscribe to events for display
	eventBus.SubscribeAll(func(event core.Event) {
		displayEvent(event)
	})

	// Create conversation manager with persistence enabled if saving
	timeout, saveDir := cfg.GetManagerConfig()
	mgrConfig := manager.Config{
		Timeout: timeout,
		SaveDir: saveDir,
		Persistence: manager.PersistenceConfig{
			Enabled: *save,
			SaveDir: persistence.DefaultSaveDir(),
		},
	}

	mgr, err := manager.NewConversationManager(mgrConfig, agents, eventBus)
	if err != nil {
		fmt.Printf("❌ Failed to create conversation manager: %v\n", err)
		os.Exit(1)
	}
	defer mgr.Close()

	// Handle resume if specified
	if *resume != "" {
		fmt.Printf("📂 Resuming conversation: %s\n", *resume)
		if err := mgr.Resume(*resume); err != nil {
			fmt.Printf("❌ Failed to resume conversation: %v\n", err)
			os.Exit(1)
		}
		printResumedSummary(mgr)
	}

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n⚠️  Interrupted, shutting down...")
		cancel()
	}()

	// Start conversation
	fmt.Println("\n🚀 Starting conversation...")
	fmt.Println(strings.Repeat("─", 60))
	mgr.Start()

	// Send test message
	fmt.Printf("\n📤 User: %s\n\n", *message)
	startTime := time.Now()
	responses, err := mgr.SendUserMessage(ctx, *message)
	totalDuration := time.Since(startTime)

	if err != nil {
		fmt.Printf("❌ Error sending message: %v\n", err)
	}

	// Display responses
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println("\n📥 Responses:")
	for _, msg := range responses {
		if msg.Status == core.MessageStatusError {
			fmt.Printf("\n❌ %s (error): %s\n", msg.AgentName, msg.Content)
		} else {
			fmt.Printf("\n💬 %s:\n%s\n", msg.AgentName, msg.Content)
			if msg.Metrics != nil {
				fmt.Printf("   ⏱️  %v | 🔢 %d tokens | 💰 $%.6f\n",
					msg.Metrics.Duration.Round(time.Millisecond),
					msg.Metrics.TotalTokens,
					msg.Metrics.Cost)
			}
		}
	}

	// Complete conversation
	mgr.Complete()

	// Print summary
	fmt.Println(strings.Repeat("─", 60))
	summary := mgr.Summary()
	fmt.Println("\n📊 Summary:")
	fmt.Printf("   • Messages: %d\n", summary.MessageCount)
	fmt.Printf("   • Total Tokens: %d\n", summary.TotalTokens)
	fmt.Printf("   • Total Cost: $%.6f\n", summary.TotalCost)
	fmt.Printf("   • Total Duration: %v\n", totalDuration.Round(time.Millisecond))
	fmt.Printf("   • Parallel Speedup: Executed %d agents in %v\n", len(agents), totalDuration.Round(time.Millisecond))

	// Save conversation if requested
	if *save {
		fmt.Println("\n💾 Saving conversation...")
		filePath, err := mgr.Save()
		if err != nil {
			fmt.Printf("❌ Failed to save conversation: %v\n", err)
		} else {
			fmt.Printf("✅ Conversation saved to: %s\n", filePath)
		}
	}

	// Export to Markdown if requested
	if *export != "" {
		fmt.Println("\n📝 Exporting to Markdown...")
		if err := mgr.ExportToMarkdown(*export); err != nil {
			fmt.Printf("❌ Failed to export conversation: %v\n", err)
		} else {
			fmt.Printf("✅ Conversation exported to: %s\n", *export)
		}
	}

	fmt.Println("\n✅ Demo complete!")
}

// displayEvent formats and prints an event to the console.
func displayEvent(event core.Event) {
	timestamp := event.Timestamp.Format("15:04:05.000")

	switch event.Type {
	case core.EventAgentTyping:
		if data, ok := event.Data.(core.AgentTypingData); ok {
			fmt.Printf("[%s] ⌨️  %s is typing...\n", timestamp, data.AgentName)
		}
	case core.EventAgentDone:
		if data, ok := event.Data.(core.AgentDoneData); ok {
			fmt.Printf("[%s] ✅ %s finished responding\n", timestamp, data.AgentName)
		}
	case core.EventAgentError:
		if data, ok := event.Data.(core.AgentErrorData); ok {
			fmt.Printf("[%s] ❌ %s error: %s\n", timestamp, data.AgentName, data.Error)
		}
	case core.EventConversationStarted:
		if data, ok := event.Data.(core.ConversationStartedData); ok {
			fmt.Printf("[%s] 🎬 Conversation started with %d agents\n", timestamp, len(data.Agents))
		}
	case core.EventConversationCompleted:
		if data, ok := event.Data.(core.ConversationCompletedData); ok {
			fmt.Printf("[%s] 🏁 Conversation completed: %d messages, %d tokens\n",
				timestamp, data.Summary.MessageCount, data.Summary.TotalTokens)
		}
	case core.EventConversationSaved:
		if data, ok := event.Data.(core.ConversationSavedData); ok {
			fmt.Printf("[%s] 💾 Conversation saved: %s\n", timestamp, filepath.Base(data.FilePath))
		}
	}
}

// listConversations lists all saved conversations.
func listConversations() {
	fmt.Println("📂 Saved Conversations:")
	fmt.Println(strings.Repeat("─", 60))

	metadata, err := persistence.ListConversations("")
	if err != nil {
		fmt.Printf("❌ Failed to list conversations: %v\n", err)
		return
	}

	if len(metadata) == 0 {
		fmt.Println("   No saved conversations found.")
		return
	}

	for _, m := range metadata {
		fmt.Printf("\n   📝 %s\n", m.ID[:8])
		fmt.Printf("      Started: %s\n", m.Started.Format(time.RFC3339))
		fmt.Printf("      Messages: %d | Agents: %d\n", m.MessageCount, m.AgentCount)
		fmt.Printf("      Participants: %s\n", strings.Join(m.AgentNames, ", "))
		fmt.Printf("      Status: %s\n", m.Status)
		fmt.Printf("      File: %s\n", filepath.Base(m.FilePath))
	}

	fmt.Println()
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Total: %d conversations\n", len(metadata))
}

// printResumedSummary prints a summary of the resumed conversation.
func printResumedSummary(mgr *manager.ConversationManager) {
	fmt.Println("\n📋 Resumed Conversation Summary:")
	fmt.Println(strings.Repeat("─", 60))

	conv := mgr.GetConversation()
	fmt.Printf("   ID: %s\n", conv.ID)
	fmt.Printf("   Started: %s\n", conv.Started.Format(time.RFC3339))
	fmt.Printf("   Messages: %d\n", len(conv.Messages))
	fmt.Printf("   Agents: %d\n", len(conv.Agents))

	// Show agent names
	var agentNames []string
	for _, agent := range conv.Agents {
		agentNames = append(agentNames, agent.Name)
	}
	fmt.Printf("   Participants: %s\n", strings.Join(agentNames, ", "))

	// Show last few messages
	fmt.Println("\n   Recent Messages:")
	messages := conv.Messages
	start := 0
	if len(messages) > 3 {
		start = len(messages) - 3
	}
	for i := start; i < len(messages); i++ {
		msg := messages[i]
		content := msg.Content
		if len(content) > 50 {
			content = content[:50] + "..."
		}
		switch msg.Role {
		case core.RoleUser:
			fmt.Printf("      👤 User: %s\n", content)
		case core.RoleAgent:
			fmt.Printf("      🤖 %s: %s\n", msg.AgentName, content)
		case core.RoleSystem:
			fmt.Printf("      ⚙️  System: %s\n", content)
		}
	}
	fmt.Println(strings.Repeat("─", 60))
}
