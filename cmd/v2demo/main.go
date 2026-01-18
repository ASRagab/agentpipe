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
	"strings"
	"syscall"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/log"
	"github.com/kevinelliott/agentpipe/pkg/v2/core"
	"github.com/kevinelliott/agentpipe/pkg/v2/events"
	"github.com/kevinelliott/agentpipe/pkg/v2/manager"

	// Import adapters to register them
	_ "github.com/kevinelliott/agentpipe/pkg/v2/adapters/api"
	"github.com/kevinelliott/agentpipe/pkg/v2/config"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "examples/v2/demo-config.yaml", "Path to configuration file")
	message := flag.String("message", "Explain async/await in JavaScript in one sentence.", "Message to send to agents")
	debug := flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()

	// Configure logging
	if *debug {
		log.SetGlobalLevel(log.ParseLevel("debug"))
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

	// Create conversation manager
	timeout, saveDir := cfg.GetManagerConfig()
	mgr, err := manager.NewConversationManager(
		manager.Config{
			Timeout: timeout,
			SaveDir: saveDir,
		},
		agents,
		eventBus,
	)
	if err != nil {
		fmt.Printf("❌ Failed to create conversation manager: %v\n", err)
		os.Exit(1)
	}
	defer mgr.Close()

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
	}
}
