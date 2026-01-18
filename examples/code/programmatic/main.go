// Package main demonstrates programmatic usage of the AgentPipe v2 engine.
//
// This example shows how to:
// - Create agents programmatically
// - Set up the conversation manager
// - Subscribe to events
// - Send messages and handle responses
// - Access conversation metrics
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	// Import the API adapter package to register the openrouter adapter
	_ "github.com/ASRagab/agentpipe/pkg/v2/adapters/api"

	"github.com/ASRagab/agentpipe/pkg/v2/core"
	"github.com/ASRagab/agentpipe/pkg/v2/events"
	"github.com/ASRagab/agentpipe/pkg/v2/manager"
)

func main() {
	fmt.Println("=== AgentPipe v2 Programmatic Usage Example ===")

	// Check for API key
	if os.Getenv("OPENROUTER_API_KEY") == "" {
		fmt.Println("Warning: OPENROUTER_API_KEY not set")
		fmt.Println("Set it with: export OPENROUTER_API_KEY='your-key-here'")
		fmt.Println("\nRunning with mock responses instead...")
		runMockExample()
		return
	}

	// Run the real example with OpenRouter
	runOpenRouterExample()
}

// runOpenRouterExample demonstrates usage with real API calls.
func runOpenRouterExample() {
	// Define agents programmatically
	agents := []core.Agent{
		{
			ID:          "claude-haiku",
			Type:        "openrouter",
			Name:        "Claude Haiku",
			Model:       "anthropic/claude-3-haiku",
			AdapterName: "openrouter",
			Config: core.AgentAdapterConfig{
				SystemPrompt: "You are a helpful, concise assistant. Keep responses brief.",
				Temperature:  0.7,
				MaxTokens:    256,
			},
		},
		{
			ID:          "gpt-mini",
			Type:        "openrouter",
			Name:        "GPT-4o Mini",
			Model:       "openai/gpt-4o-mini",
			AdapterName: "openrouter",
			Config: core.AgentAdapterConfig{
				SystemPrompt: "You are a precise, analytical assistant. Be direct.",
				Temperature:  0.3,
				MaxTokens:    256,
			},
		},
	}

	// Create event bus
	eventBus := events.NewBus()

	// Subscribe to events for real-time feedback
	setupEventSubscriptions(eventBus)

	// Configure the conversation manager
	config := manager.Config{
		Timeout: 60 * time.Second,
		GracefulDegradation: manager.GracefulDegradationConfig{
			Enabled:                  true,
			PauseOnAllFailed:         true,
			RetryFailedOnNextMessage: true,
			EmitSystemMessages:       true,
		},
	}

	// Create the conversation manager
	mgr, err := manager.NewConversationManager(config, agents, eventBus)
	if err != nil {
		fmt.Printf("Error creating manager: %v\n", err)
		os.Exit(1)
	}
	defer mgr.Close()

	// Start the conversation
	fmt.Println("Starting conversation with 2 agents...")
	mgr.Start()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// === Send first message ===
	fmt.Println("\n[User] What is the capital of France?")
	responses, err := mgr.SendUserMessage(ctx, "What is the capital of France?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	printResponses(responses)

	// === Send follow-up message ===
	fmt.Println("\n[User] What's one famous landmark there?")
	responses, err = mgr.SendUserMessage(ctx, "What's one famous landmark there?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	printResponses(responses)

	// Complete the conversation
	mgr.Complete()

	// Print final summary
	printSummary(mgr)
}

// runMockExample demonstrates the API without real API calls.
func runMockExample() {
	// Import the mock adapter
	// In real code, you would use a real adapter

	fmt.Println("This example requires the OPENROUTER_API_KEY environment variable.")
	fmt.Println("\nTo see the programmatic API in action:")
	fmt.Println("  1. Get an API key from https://openrouter.ai")
	fmt.Println("  2. export OPENROUTER_API_KEY='your-key-here'")
	fmt.Println("  3. Run this example again")
	fmt.Println("\nAlternatively, see the custom-adapter example for a working mock.")
}

// setupEventSubscriptions demonstrates subscribing to v2 events.
func setupEventSubscriptions(eventBus *events.Bus) {
	// Subscribe to typing events
	eventBus.Subscribe(core.EventAgentTyping, func(event core.Event) {
		if data, ok := event.Data.(core.AgentTypingData); ok {
			fmt.Printf("  [%s is typing...]\n", data.AgentName)
		}
	})

	// Subscribe to agent done events
	eventBus.Subscribe(core.EventAgentDone, func(event core.Event) {
		if data, ok := event.Data.(core.AgentDoneData); ok {
			metrics := ""
			if data.Message.Metrics != nil {
				metrics = fmt.Sprintf(" (%v, %d tokens, $%.6f)",
					data.Message.Metrics.Duration,
					data.Message.Metrics.TotalTokens,
					data.Message.Metrics.Cost)
			}
			fmt.Printf("  [%s finished%s]\n", data.AgentName, metrics)
		}
	})

	// Subscribe to error events
	eventBus.Subscribe(core.EventAgentError, func(event core.Event) {
		if data, ok := event.Data.(core.AgentErrorData); ok {
			fmt.Printf("  [ERROR from %s: %s]\n", data.AgentName, data.Error)
		}
	})
}

// printResponses prints agent responses.
func printResponses(responses []core.Message) {
	fmt.Println("\n--- Responses ---")
	for _, msg := range responses {
		if msg.Role == core.RoleAgent {
			fmt.Printf("\n[%s]:\n%s\n", msg.AgentName, msg.Content)
		}
	}
}

// printSummary prints the conversation summary.
func printSummary(mgr *manager.ConversationManager) {
	summary := mgr.Summary()
	fmt.Printf("\n=== Conversation Summary ===\n")
	fmt.Printf("ID:           %s\n", summary.ID)
	fmt.Printf("Status:       %s\n", summary.Status)
	fmt.Printf("Messages:     %d\n", summary.MessageCount)
	fmt.Printf("Agents:       %d\n", summary.AgentCount)
	fmt.Printf("Total Tokens: %d\n", summary.TotalTokens)
	fmt.Printf("Total Cost:   $%.6f\n", summary.TotalCost)
	fmt.Printf("Duration:     %v\n", summary.Duration)
}
