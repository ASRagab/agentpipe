// Package main demonstrates webhook integration with AgentPipe v2.
//
// This example shows how to:
// - Subscribe to conversation events
// - Forward events to an external webhook
// - Handle webhook responses
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	// Import to register adapters
	_ "github.com/ASRagab/agentpipe/pkg/v2/adapters/api"

	"github.com/ASRagab/agentpipe/pkg/v2/core"
	"github.com/ASRagab/agentpipe/pkg/v2/events"
	"github.com/ASRagab/agentpipe/pkg/v2/manager"
)

func main() {
	fmt.Println("=== AgentPipe v2 Webhook Integration Example ===")

	// Webhook URL - in production, this would be your external service
	// For testing, run the server.go in this directory
	webhookURL := os.Getenv("WEBHOOK_URL")
	if webhookURL == "" {
		webhookURL = "http://localhost:8080/webhook"
		fmt.Printf("WEBHOOK_URL not set, using: %s\n", webhookURL)
		fmt.Println("To test, run 'go run server.go' in another terminal")
	}

	// Check for API key
	if os.Getenv("OPENROUTER_API_KEY") == "" {
		fmt.Println("Running mock example (no API key set)")
		runMockWebhookExample(webhookURL)
		return
	}

	runWebhookExample(webhookURL)
}

// runWebhookExample demonstrates webhook integration with real API calls.
func runWebhookExample(webhookURL string) {
	// Create webhook handler
	handler := NewWebhookHandler(webhookURL, WebhookConfig{
		MaxRetries:  3,
		Timeout:     10 * time.Second,
		Async:       true,
		FilterTypes: nil, // Send all event types
	})
	defer handler.Close()

	// Create event bus
	eventBus := events.NewBus()

	// Subscribe to all events and forward to webhook
	eventBus.SubscribeAll(func(event core.Event) {
		if err := handler.Handle(event); err != nil {
			fmt.Printf("[Webhook Error] %v\n", err)
		}
	})

	// Create agents
	agents := []core.Agent{
		{
			ID:          "assistant",
			Type:        "openrouter",
			Name:        "Assistant",
			Model:       "anthropic/claude-3-haiku",
			AdapterName: "openrouter",
			Config: core.AgentAdapterConfig{
				SystemPrompt: "Be brief and helpful.",
				MaxTokens:    100,
			},
		},
	}

	// Create manager
	config := manager.Config{
		Timeout: 30 * time.Second,
	}

	mgr, err := manager.NewConversationManager(config, agents, eventBus)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer mgr.Close()

	// Start conversation
	fmt.Println("Starting conversation...")
	mgr.Start()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Send a message
	fmt.Println("\n[User] Hello!")
	responses, err := mgr.SendUserMessage(ctx, "Hello!")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Print responses
	for _, msg := range responses {
		if msg.Role == core.RoleAgent {
			fmt.Printf("[%s] %s\n", msg.AgentName, msg.Content)
		}
	}

	// Complete
	mgr.Complete()

	// Wait for async webhooks to finish
	fmt.Println("\nWaiting for webhooks to complete...")
	time.Sleep(2 * time.Second)

	// Print stats
	stats := handler.GetStats()
	fmt.Printf("\n=== Webhook Stats ===\n")
	fmt.Printf("Total Sent: %d\n", stats.TotalSent)
	fmt.Printf("Successful: %d\n", stats.Successful)
	fmt.Printf("Failed: %d\n", stats.Failed)
	fmt.Printf("Pending: %d\n", stats.Pending)
}

// runMockWebhookExample demonstrates webhook integration without API calls.
func runMockWebhookExample(webhookURL string) {
	fmt.Println("Demonstrating webhook event format...")

	// Create webhook handler
	handler := NewWebhookHandler(webhookURL, WebhookConfig{
		MaxRetries: 1,
		Timeout:    5 * time.Second,
		Async:      false, // Sync for demo
	})
	defer handler.Close()

	// Create sample events
	sampleEvents := []core.Event{
		core.NewConversationStartedEvent("conv-123", []core.Agent{
			{ID: "agent-1", Name: "Claude", Model: "claude-3-haiku"},
		}),
		core.NewAgentTypingEvent("agent-1", "Claude"),
		core.NewMessageCreatedEvent(core.Message{
			ID:        "msg-1",
			Role:      core.RoleAgent,
			AgentID:   "agent-1",
			AgentName: "Claude",
			Content:   "Hello! How can I help?",
			Timestamp: time.Now(),
			Metrics: &core.Metrics{
				Duration:     time.Second,
				TotalTokens:  15,
				InputTokens:  5,
				OutputTokens: 10,
				Cost:         0.0001,
				Model:        "claude-3-haiku",
			},
		}),
		core.NewConversationCompletedEvent("conv-123", core.ConversationSummary{
			ID:           "conv-123",
			Status:       core.ConversationStatusCompleted,
			MessageCount: 2,
			AgentCount:   1,
			TotalTokens:  30,
			TotalCost:    0.0002,
			Duration:     5 * time.Second,
		}),
	}

	// Send each event
	fmt.Println("Sending sample events to webhook...")
	for _, event := range sampleEvents {
		fmt.Printf("\n  Sending: %s\n", event.Type)
		if err := handler.Handle(event); err != nil {
			fmt.Printf("    Error: %v\n", err)
		} else {
			fmt.Printf("    Sent successfully\n")
		}
	}

	// Print stats
	stats := handler.GetStats()
	fmt.Printf("\n=== Webhook Stats ===\n")
	fmt.Printf("Total Sent: %d\n", stats.TotalSent)
	fmt.Printf("Successful: %d\n", stats.Successful)
	fmt.Printf("Failed: %d\n", stats.Failed)

	fmt.Println("\nTo receive webhooks, run 'go run server.go' in another terminal.")
}
