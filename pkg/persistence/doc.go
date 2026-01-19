// Package persistence provides conversation save/load functionality for the v2 architecture.
//
// This package handles saving conversations to JSON files, loading saved conversations,
// listing available conversations, and exporting to human-readable Markdown format.
//
// # File Format
//
// Conversations are saved as JSON files with the naming pattern:
//
//	conversation_{id_prefix}_{timestamp}.json
//
// Where:
//   - {id_prefix} is the first 8 characters of the conversation UUID
//   - {timestamp} is the save time in format 2006-01-02_15-04-05
//
// Example: conversation_abc12345_2024-01-15_14-30-22.json
//
// # Default Storage Location
//
// Conversations are saved to:
//
//	~/.agentpipe/v2/conversations/
//
// Use DefaultSaveDir() to get this path:
//
//	saveDir := persistence.DefaultSaveDir()
//	fmt.Println(saveDir)  // /Users/name/.agentpipe/v2/conversations
//
// # Saving Conversations
//
// Save a conversation to the default directory:
//
//	filePath, err := persistence.SaveConversation(conversation, "")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Saved to: %s\n", filePath)
//
// Save to a custom directory:
//
//	filePath, err := persistence.SaveConversation(conversation, "/custom/path")
//
// Files are saved with secure permissions (0600 - owner read/write only).
// Directories are created automatically with permissions 0750.
//
// # Loading Conversations
//
// Load a specific conversation by file path:
//
//	conv, err := persistence.LoadConversation("/path/to/conversation.json")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Loaded %d messages\n", len(conv.Messages))
//
// Load the most recently modified conversation:
//
//	conv, err := persistence.LoadLatest("")  // Empty string uses default dir
//	if err != nil {
//		log.Fatal(err)
//	}
//
// Find a conversation by ID prefix:
//
//	filePath, err := persistence.FindConversationByID("", "abc123")
//	if err != nil {
//		log.Fatal(err)
//	}
//	conv, err := persistence.LoadConversation(filePath)
//
// # Listing Conversations
//
// List all saved conversations with metadata:
//
//	metadata, err := persistence.ListConversations("")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	for _, m := range metadata {
//		fmt.Printf("ID: %s, Messages: %d, Agents: %v\n",
//			m.ID[:8], m.MessageCount, m.AgentNames)
//		fmt.Printf("  Updated: %s\n", m.Updated.Format(time.RFC3339))
//	}
//
// The ConversationMetadata type contains:
//
//	type ConversationMetadata struct {
//		ID           string                  // Conversation UUID
//		FilePath     string                  // Path to saved file
//		Started      time.Time               // When conversation began
//		Updated      time.Time               // Last modification time
//		MessageCount int                     // Number of messages
//		AgentCount   int                     // Number of agents
//		AgentNames   []string                // Agent display names
//		Status       core.ConversationStatus // active, completed, etc.
//	}
//
// Results are sorted by updated time (newest first).
//
// # Exporting to Markdown
//
// Export a conversation to a human-readable Markdown file:
//
//	err := persistence.SaveAsMarkdown(conversation, "/path/to/output.md")
//	if err != nil {
//		log.Fatal(err)
//	}
//
// Or get the Markdown content as a string:
//
//	markdown, err := persistence.ExportToMarkdown(conversation)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(markdown)
//
// Generate a suggested filename:
//
//	filename := persistence.GenerateMarkdownFilename(conversation)
//	// Returns: conversation_abc12345_2024-01-15_14-30-22.md
//
// The Markdown export includes:
//   - Conversation metadata (ID, timestamps, status)
//   - List of participating agents with models
//   - All messages with timestamps and role icons
//   - Agent metrics (duration, tokens, cost) in collapsible sections
//   - Summary statistics
//
// # Integration with ConversationManager
//
// The ConversationManager uses this package for persistence:
//
//	// Configure auto-save
//	config := manager.Config{
//		Persistence: manager.PersistenceConfig{
//			Enabled:      true,
//			SaveDir:      persistence.DefaultSaveDir(),
//			SaveInterval: 5 * time.Minute,
//		},
//	}
//
//	mgr, _ := manager.NewConversationManager(config, agents, nil)
//
//	// Manual save
//	filePath, err := mgr.Save()
//
//	// Export to Markdown
//	err = mgr.ExportToMarkdown("/path/to/output.md")
//
//	// Resume a conversation
//	err = mgr.Resume("latest")     // Most recent
//	err = mgr.Resume("abc123")     // By ID prefix
//	err = mgr.Resume("/path.json") // By file path
//
// # Validation
//
// Conversations are validated on load:
//
//	err := persistence.ValidateConversation(conversation)
//	if err != nil {
//		// Errors include:
//		// - "conversation is nil"
//		// - "conversation ID is required"
//		// - "conversation Started time is required"
//		log.Fatal(err)
//	}
//
// # Error Handling
//
// Common error scenarios:
//
//	// File not found
//	conv, err := persistence.LoadConversation("/nonexistent.json")
//	// Error: "conversation file not found: /nonexistent.json"
//
//	// Invalid JSON
//	conv, err := persistence.LoadConversation("/invalid.json")
//	// Error: "failed to parse conversation JSON: ..."
//
//	// No saved conversations
//	conv, err := persistence.LoadLatest("")
//	// Error: "no saved conversations found in /path"
//
//	// ID not found
//	path, err := persistence.FindConversationByID("", "xyz789")
//	// Error: "conversation with ID prefix \"xyz789\" not found"
//
// # Thread Safety
//
// Functions in this package are NOT thread-safe. If multiple goroutines need
// to access the same conversation files, use appropriate synchronization.
// The ConversationManager handles this internally with mutex locking.
//
// # Security
//
// Files are created with restrictive permissions:
//   - Files: 0600 (owner read/write only)
//   - Directories: 0750 (owner full, group read/execute)
//
// Conversation files may contain sensitive information from AI conversations.
// Consider encrypting files at rest if storing confidential data.
package persistence
