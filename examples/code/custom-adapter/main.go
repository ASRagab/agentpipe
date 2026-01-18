// Package main demonstrates using a custom adapter with AgentPipe v2.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ASRagab/agentpipe/pkg/core"
	"github.com/ASRagab/agentpipe/pkg/events"
	"github.com/ASRagab/agentpipe/pkg/manager"
)

func main() {
	fmt.Println("=== AgentPipe v2 Custom Adapter Example ===")

	// The EchoAdapter is registered automatically via init() in adapter.go
	// You can verify it's available:
	// adapters.Has("echo") // returns true

	// Create agents using the custom adapter
	agents := []core.Agent{
		{
			ID:          "echo-1",
			Type:        "echo",
			Name:        "Echo Agent",
			Model:       "echo-v1",
			AdapterName: "echo", // Uses our custom adapter
			Config: core.AgentAdapterConfig{
				SystemPrompt: "Friendly Assistant",
				Temperature:  0.7,
				MaxTokens:    1024,
			},
		},
		{
			ID:          "echo-2",
			Type:        "echo",
			Name:        "Echo Agent 2",
			Model:       "echo-v1",
			AdapterName: "echo",
			Config: core.AgentAdapterConfig{
				SystemPrompt: "Expert Advisor",
				Temperature:  0.3,
				MaxTokens:    2048,
			},
		},
	}

	// Create event bus for receiving events
	eventBus := events.NewBus()

	// Subscribe to events to see what's happening
	eventBus.SubscribeAll(func(event core.Event) {
		switch event.Type {
		case core.EventConversationStarted:
			fmt.Println("[Event] Conversation started")
		case core.EventAgentTyping:
			if data, ok := event.Data.(core.AgentTypingData); ok {
				fmt.Printf("[Event] %s is typing...\n", data.AgentName)
			}
		case core.EventMessageCreated:
			if msg, ok := event.Data.(core.Message); ok {
				if msg.Role == core.RoleAgent {
					fmt.Printf("[Event] %s responded (tokens: %d)\n",
						msg.AgentName,
						safeTokens(msg.Metrics))
				}
			}
		case core.EventConversationCompleted:
			fmt.Println("[Event] Conversation completed")
		}
	})

	// Create the conversation manager
	config := manager.Config{
		Timeout: 30 * time.Second,
		GracefulDegradation: manager.GracefulDegradationConfig{
			Enabled:                  true,
			PauseOnAllFailed:         true,
			RetryFailedOnNextMessage: true,
			EmitSystemMessages:       true,
		},
	}

	mgr, err := manager.NewConversationManager(config, agents, eventBus)
	if err != nil {
		fmt.Printf("Error creating manager: %v\n", err)
		os.Exit(1)
	}
	defer mgr.Close()

	// Start the conversation
	mgr.Start()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Send a user message and get responses from all agents
	fmt.Println("\n[User] Hello, how are you today?")

	responses, err := mgr.SendUserMessage(ctx, "Hello, how are you today?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Print the responses
	fmt.Println("\n--- Agent Responses ---")
	for _, msg := range responses {
		if msg.Role == core.RoleAgent {
			fmt.Printf("\n[%s]: %s\n", msg.AgentName, msg.Content)
			if msg.Metrics != nil {
				fmt.Printf("  Duration: %v, Tokens: %d, Cost: $%.6f\n",
					msg.Metrics.Duration,
					msg.Metrics.TotalTokens,
					msg.Metrics.Cost)
			}
		}
	}

	// Complete the conversation
	mgr.Complete()

	// Print summary
	summary := mgr.Summary()
	fmt.Printf("\n--- Conversation Summary ---\n")
	fmt.Printf("Messages: %d\n", summary.MessageCount)
	fmt.Printf("Total Tokens: %d\n", summary.TotalTokens)
	fmt.Printf("Total Cost: $%.6f\n", summary.TotalCost)
	fmt.Printf("Duration: %v\n", summary.Duration)
}

func safeTokens(metrics *core.Metrics) int {
	if metrics == nil {
		return 0
	}
	return metrics.TotalTokens
}
