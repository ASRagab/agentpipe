// Package main provides the v2 TUI demo command for AgentPipe.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ASRagab/agentpipe/pkg/v2/config"
	"github.com/ASRagab/agentpipe/pkg/v2/core"
	"github.com/ASRagab/agentpipe/pkg/v2/events"
	"github.com/ASRagab/agentpipe/pkg/v2/manager"
	"github.com/ASRagab/agentpipe/pkg/v2/tui"
)

func main() {
	// Parse flags
	configPath := flag.String("config", "", "Path to configuration file")
	demo := flag.Bool("demo", false, "Run with demo mock agents")
	flag.Parse()

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	var agents []core.Agent

	if *demo {
		// Create demo mock agents
		agents = createDemoAgents()
	} else if *configPath != "" {
		// Load from config file
		cfg, err := config.LoadConfig(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
		agents, err = cfg.InitializeAgents()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing agents: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Default: use demo agents
		agents = createDemoAgents()
	}

	// Create event bus
	eventBus := events.NewBus()
	defer eventBus.Close()

	// Create manager config
	mgrConfig := manager.DefaultConfig()
	mgrConfig.Persistence.Enabled = false // Disable auto-save for demo

	// Create conversation manager
	mgr, err := manager.NewConversationManager(mgrConfig, agents, eventBus)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating manager: %v\n", err)
		os.Exit(1)
	}
	defer mgr.Close()

	// Start the conversation
	mgr.Start()

	// Run the TUI
	if err := tui.RunWithContext(ctx, mgr, eventBus); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}

	// Complete the conversation
	mgr.Complete()
}

// createDemoAgents creates demo mock agents for testing the TUI.
func createDemoAgents() []core.Agent {
	return []core.Agent{
		core.NewAgent("demo-claude", "mock", "Claude", "claude-3-opus", "mock").
			WithSystemPrompt("You are Claude, a helpful AI assistant created by Anthropic.").
			WithTemperature(0.7),
		core.NewAgent("demo-gemini", "mock", "Gemini", "gemini-pro", "mock").
			WithSystemPrompt("You are Gemini, a helpful AI assistant created by Google.").
			WithTemperature(0.8),
		core.NewAgent("demo-gpt4", "mock", "GPT-4", "gpt-4-turbo", "mock").
			WithSystemPrompt("You are GPT-4, a helpful AI assistant created by OpenAI.").
			WithTemperature(0.7),
	}
}
