// Package persistence provides conversation save/load functionality for the v2 architecture.
package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/log"
	"github.com/kevinelliott/agentpipe/pkg/v2/core"
)

// DefaultSaveDir returns the default directory for saving conversations.
func DefaultSaveDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".agentpipe/v2/conversations"
	}
	return filepath.Join(home, ".agentpipe", "v2", "conversations")
}

// SaveConversation saves a conversation to a JSON file.
// Returns the path to the saved file.
func SaveConversation(conversation *core.Conversation, saveDir string) (string, error) {
	if conversation == nil {
		return "", fmt.Errorf("conversation cannot be nil")
	}

	// Use default directory if not specified
	if saveDir == "" {
		saveDir = DefaultSaveDir()
	}

	// Ensure directory exists
	if err := os.MkdirAll(saveDir, 0750); err != nil {
		return "", fmt.Errorf("failed to create save directory: %w", err)
	}

	// Generate filename: {id_first8}_{timestamp}.json
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	idPrefix := conversation.ID
	if len(idPrefix) > 8 {
		idPrefix = idPrefix[:8]
	}
	filename := fmt.Sprintf("conversation_%s_%s.json", idPrefix, timestamp)
	savePath := filepath.Join(saveDir, filename)

	// Marshal conversation with indentation for readability
	data, err := json.MarshalIndent(conversation, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal conversation: %w", err)
	}

	// Write file with secure permissions (owner read/write only)
	if err := os.WriteFile(savePath, data, 0600); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	log.WithFields(map[string]interface{}{
		"conversation_id": conversation.ID,
		"file_path":       savePath,
		"message_count":   len(conversation.Messages),
	}).Info("conversation saved")

	return savePath, nil
}

// ValidateConversation checks that a conversation has the required fields.
func ValidateConversation(conversation *core.Conversation) error {
	if conversation == nil {
		return fmt.Errorf("conversation is nil")
	}
	if conversation.ID == "" {
		return fmt.Errorf("conversation ID is required")
	}
	if conversation.Started.IsZero() {
		return fmt.Errorf("conversation Started time is required")
	}
	// Messages can be empty for a new conversation
	return nil
}
