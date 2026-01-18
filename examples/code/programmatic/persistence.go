// Package main demonstrates conversation persistence with AgentPipe v2.
//
// This file shows how to:
// - Enable auto-save
// - Manually save conversations
// - Resume conversations
// - Export to Markdown
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	// Import to register adapters
	_ "github.com/ASRagab/agentpipe/pkg/adapters/api"

	"github.com/ASRagab/agentpipe/pkg/core"
	"github.com/ASRagab/agentpipe/pkg/events"
	"github.com/ASRagab/agentpipe/pkg/manager"
	"github.com/ASRagab/agentpipe/pkg/persistence"
)

// RunPersistenceExample demonstrates save/load functionality.
func RunPersistenceExample() {
	fmt.Println("=== AgentPipe v2 Persistence Example ===")

	// Define a temporary save directory
	tempDir := filepath.Join(os.TempDir(), "agentpipe-example")
	os.MkdirAll(tempDir, 0o755)
	defer os.RemoveAll(tempDir) // Cleanup

	fmt.Printf("Save directory: %s\n\n", tempDir)

	// === Part 1: Create and save a conversation ===
	conversationID := createAndSaveConversation(tempDir)

	// === Part 2: Resume the conversation ===
	resumeConversation(tempDir, conversationID)

	// === Part 3: Export to Markdown ===
	exportToMarkdown(tempDir)
}

// createAndSaveConversation creates a conversation and saves it.
func createAndSaveConversation(saveDir string) string {
	fmt.Println("=== Creating and Saving Conversation ===")

	// Create a simple mock conversation (no API calls needed)
	conv := core.NewConversation([]core.Agent{
		{
			ID:          "agent-1",
			Type:        "mock",
			Name:        "Assistant",
			Model:       "mock-v1",
			AdapterName: "mock",
		},
	})

	// Add some messages
	conv.AddMessage(core.NewUserMessage("Hello!"))
	conv.AddMessage(core.Message{
		ID:        "msg-1",
		Role:      core.RoleAgent,
		AgentID:   "agent-1",
		AgentName: "Assistant",
		Content:   "Hello! How can I help you today?",
		Timestamp: time.Now(),
		Metrics: &core.Metrics{
			Duration:    100 * time.Millisecond,
			TotalTokens: 15,
			Cost:        0.0001,
		},
	})

	conv.AddMessage(core.NewUserMessage("What's the weather like?"))
	conv.AddMessage(core.Message{
		ID:        "msg-2",
		Role:      core.RoleAgent,
		AgentID:   "agent-1",
		AgentName: "Assistant",
		Content:   "I don't have access to real-time weather data, but I'd be happy to help with other questions!",
		Timestamp: time.Now(),
		Metrics: &core.Metrics{
			Duration:    150 * time.Millisecond,
			TotalTokens: 25,
			Cost:        0.0002,
		},
	})

	// Save the conversation
	filePath, err := persistence.SaveConversation(conv, saveDir)
	if err != nil {
		fmt.Printf("Error saving: %v\n", err)
		return ""
	}

	fmt.Printf("Saved to: %s\n", filePath)
	fmt.Printf("Conversation ID: %s\n", conv.ID)
	fmt.Printf("Messages: %d\n\n", len(conv.Messages))

	return conv.ID
}

// resumeConversation demonstrates loading a saved conversation.
func resumeConversation(saveDir, conversationID string) {
	fmt.Println("=== Resuming Conversation ===")

	// Method 1: Load latest conversation
	conv, err := persistence.LoadLatest(saveDir)
	if err != nil {
		fmt.Printf("Error loading latest: %v\n", err)
		return
	}

	fmt.Printf("Loaded conversation: %s\n", conv.ID)
	fmt.Printf("Messages: %d\n", len(conv.Messages))
	fmt.Printf("Duration: %v\n", conv.Duration())
	fmt.Printf("Total Tokens: %d\n", conv.TotalTokens())
	fmt.Printf("Total Cost: $%.6f\n\n", conv.TotalCost())

	// Print messages
	fmt.Println("Messages:")
	for _, msg := range conv.Messages {
		role := string(msg.Role)
		if msg.Role == core.RoleAgent {
			role = msg.AgentName
		}
		fmt.Printf("  [%s]: %s\n", role, truncate(msg.Content, 50))
	}

	// Method 2: Load by ID prefix
	fmt.Printf("\n--- Loading by ID prefix '%s...' ---\n", conversationID[:8])
	filePath, err := persistence.FindConversationByID(saveDir, conversationID[:8])
	if err != nil {
		fmt.Printf("Error finding by ID: %v\n", err)
		return
	}

	conv2, err := persistence.LoadConversation(filePath)
	if err != nil {
		fmt.Printf("Error loading: %v\n", err)
		return
	}

	fmt.Printf("Found: %s\n\n", conv2.ID)
}

// exportToMarkdown demonstrates exporting to Markdown.
func exportToMarkdown(saveDir string) {
	fmt.Println("=== Exporting to Markdown ===")

	// Load the latest conversation
	conv, err := persistence.LoadLatest(saveDir)
	if err != nil {
		fmt.Printf("Error loading: %v\n", err)
		return
	}

	// Export to Markdown
	mdPath := filepath.Join(saveDir, "conversation.md")
	if err := persistence.SaveAsMarkdown(conv, mdPath); err != nil {
		fmt.Printf("Error exporting: %v\n", err)
		return
	}

	fmt.Printf("Exported to: %s\n\n", mdPath)

	// Read and print the Markdown
	content, _ := os.ReadFile(mdPath)
	fmt.Println("--- Markdown Content ---")
	fmt.Println(string(content))
}

// RunAutoSaveExample demonstrates auto-save functionality.
func RunAutoSaveExample() {
	fmt.Println("=== Auto-Save Example ===")

	// This example requires API access
	if os.Getenv("OPENROUTER_API_KEY") == "" {
		fmt.Println("OPENROUTER_API_KEY not set. See persistence.go for manual save/load examples.")
		return
	}

	tempDir := filepath.Join(os.TempDir(), "agentpipe-autosave")
	os.MkdirAll(tempDir, 0o755)
	defer os.RemoveAll(tempDir)

	// Create agents
	agents := []core.Agent{
		{
			ID:          "assistant",
			Type:        "openrouter",
			Name:        "Assistant",
			Model:       "anthropic/claude-3-haiku",
			AdapterName: "openrouter",
			Config: core.AgentAdapterConfig{
				SystemPrompt: "Be brief.",
				MaxTokens:    100,
			},
		},
	}

	// Create event bus
	eventBus := events.NewBus()

	// Subscribe to save events
	eventBus.Subscribe(core.EventConversationSaved, func(event core.Event) {
		if data, ok := event.Data.(core.ConversationSavedData); ok {
			fmt.Printf("[Auto-saved to %s]\n", data.FilePath)
		}
	})

	// Configure with auto-save enabled
	config := manager.Config{
		Timeout: 30 * time.Second,
		Persistence: manager.PersistenceConfig{
			Enabled:      true, // Enable auto-save
			SaveDir:      tempDir,
			SaveInterval: 5 * time.Second, // Also save every 5 seconds
		},
	}

	// Create manager
	mgr, err := manager.NewConversationManager(config, agents, eventBus)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer mgr.Close()

	mgr.Start()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Send a message (triggers auto-save on response)
	fmt.Println("\n[User] Hi!")
	_, err = mgr.SendUserMessage(ctx, "Hi!")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Manually save
	fmt.Println("\n--- Manual Save ---")
	filePath, err := mgr.Save()
	if err != nil {
		fmt.Printf("Error saving: %v\n", err)
	} else {
		fmt.Printf("Saved to: %s\n", filePath)
	}

	// Export to Markdown
	fmt.Println("\n--- Export to Markdown ---")
	mdPath := filepath.Join(tempDir, "chat.md")
	if err := mgr.ExportToMarkdown(mdPath); err != nil {
		fmt.Printf("Error exporting: %v\n", err)
	} else {
		fmt.Printf("Exported to: %s\n", mdPath)
	}

	mgr.Complete()
}

// truncate shortens a string to max length.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
