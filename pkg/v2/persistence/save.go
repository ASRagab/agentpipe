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

// SaveConversation saves a conversation to a JSON file.
func SaveConversation(conversation *core.Conversation, saveDir string) (string, error) {
	// Ensure directory exists
	if err := os.MkdirAll(saveDir, 0750); err != nil {
		return "", fmt.Errorf("failed to create save directory: %w", err)
	}

	// Generate filename
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("conversation_%s_%s.json", conversation.ID[:8], timestamp)
	filepath := filepath.Join(saveDir, filename)

	// Marshal conversation
	data, err := json.MarshalIndent(conversation, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal conversation: %w", err)
	}

	// Write file
	if err := os.WriteFile(filepath, data, 0600); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	log.WithFields(map[string]interface{}{
		"conversation_id": conversation.ID,
		"file_path":       filepath,
	}).Info("conversation saved")

	return filepath, nil
}

// LoadConversation loads a conversation from a JSON file.
func LoadConversation(filepath string) (*core.Conversation, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var conversation core.Conversation
	if err := json.Unmarshal(data, &conversation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal conversation: %w", err)
	}

	log.WithFields(map[string]interface{}{
		"conversation_id": conversation.ID,
		"file_path":       filepath,
	}).Info("conversation loaded")

	return &conversation, nil
}

// ListConversations lists all saved conversations in a directory.
func ListConversations(saveDir string) ([]string, error) {
	entries, err := os.ReadDir(saveDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			files = append(files, filepath.Join(saveDir, entry.Name()))
		}
	}

	return files, nil
}
