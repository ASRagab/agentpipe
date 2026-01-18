package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ASRagab/agentpipe/pkg/log"
	"github.com/ASRagab/agentpipe/pkg/v2/core"
)

// ConversationMetadata contains summary information about a saved conversation.
type ConversationMetadata struct {
	// ID is the conversation's unique identifier.
	ID string `json:"id"`
	// FilePath is the path to the saved file.
	FilePath string `json:"file_path"`
	// Started is when the conversation began.
	Started time.Time `json:"started"`
	// Updated is when the conversation was last modified.
	Updated time.Time `json:"updated"`
	// MessageCount is the number of messages in the conversation.
	MessageCount int `json:"message_count"`
	// AgentCount is the number of participating agents.
	AgentCount int `json:"agent_count"`
	// AgentNames is the list of agent names.
	AgentNames []string `json:"agent_names"`
	// Status is the conversation status.
	Status core.ConversationStatus `json:"status"`
}

// LoadConversation loads a conversation from a JSON file.
// Returns an error for missing file, invalid JSON, or validation failure.
func LoadConversation(filePath string) (*core.Conversation, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("conversation file not found: %s", filePath)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var conversation core.Conversation
	if err := json.Unmarshal(data, &conversation); err != nil {
		return nil, fmt.Errorf("failed to parse conversation JSON: %w", err)
	}

	// Validate required fields
	if err := ValidateConversation(&conversation); err != nil {
		return nil, fmt.Errorf("invalid conversation data: %w", err)
	}

	log.WithFields(map[string]interface{}{
		"conversation_id": conversation.ID,
		"file_path":       filePath,
		"message_count":   len(conversation.Messages),
	}).Info("conversation loaded")

	return &conversation, nil
}

// LoadLatest finds and loads the most recently modified conversation file in the directory.
func LoadLatest(saveDir string) (*core.Conversation, error) {
	if saveDir == "" {
		saveDir = DefaultSaveDir()
	}

	files, err := listConversationFiles(saveDir)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no saved conversations found in %s", saveDir)
	}

	// Files are sorted by modification time (newest first)
	latestFile := files[0]

	log.WithFields(map[string]interface{}{
		"file_path": latestFile,
	}).Debug("loading latest conversation")

	return LoadConversation(latestFile)
}

// ListConversations returns metadata for all saved conversations in the directory.
// Results are sorted by updated time (newest first).
func ListConversations(saveDir string) ([]ConversationMetadata, error) {
	if saveDir == "" {
		saveDir = DefaultSaveDir()
	}

	files, err := listConversationFiles(saveDir)
	if err != nil {
		return nil, err
	}

	metadata := make([]ConversationMetadata, 0, len(files))

	for _, filePath := range files {
		conv, err := LoadConversation(filePath)
		if err != nil {
			log.WithFields(map[string]interface{}{
				"file_path": filePath,
			}).WithError(err).Warn("skipping invalid conversation file")
			continue
		}

		agentNames := make([]string, len(conv.Agents))
		for i, agent := range conv.Agents {
			agentNames[i] = agent.Name
		}

		metadata = append(metadata, ConversationMetadata{
			ID:           conv.ID,
			FilePath:     filePath,
			Started:      conv.Started,
			Updated:      conv.Updated,
			MessageCount: len(conv.Messages),
			AgentCount:   len(conv.Agents),
			AgentNames:   agentNames,
			Status:       conv.Status,
		})
	}

	// Sort by updated time (newest first)
	sort.Slice(metadata, func(i, j int) bool {
		return metadata[i].Updated.After(metadata[j].Updated)
	})

	return metadata, nil
}

// FindConversationByID searches for a conversation file by ID prefix.
// Returns the file path if found.
func FindConversationByID(saveDir, idPrefix string) (string, error) {
	if saveDir == "" {
		saveDir = DefaultSaveDir()
	}

	files, err := listConversationFiles(saveDir)
	if err != nil {
		return "", err
	}

	for _, filePath := range files {
		conv, err := LoadConversation(filePath)
		if err != nil {
			continue
		}

		if strings.HasPrefix(conv.ID, idPrefix) {
			return filePath, nil
		}
	}

	return "", fmt.Errorf("conversation with ID prefix %q not found", idPrefix)
}

// listConversationFiles returns all JSON files in the save directory, sorted by modification time.
func listConversationFiles(saveDir string) ([]string, error) {
	entries, err := os.ReadDir(saveDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	type fileWithTime struct {
		path    string
		modTime time.Time
	}

	var files []fileWithTime
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		if !strings.HasPrefix(entry.Name(), "conversation_") {
			continue
		}

		filePath := filepath.Join(saveDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		files = append(files, fileWithTime{
			path:    filePath,
			modTime: info.ModTime(),
		})
	}

	// Sort by modification time (newest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	result := make([]string, len(files))
	for i, f := range files {
		result[i] = f.path
	}

	return result, nil
}
