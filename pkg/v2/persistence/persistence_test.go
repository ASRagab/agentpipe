package persistence_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ASRagab/agentpipe/pkg/v2/core"
	"github.com/ASRagab/agentpipe/pkg/v2/persistence"
)

// createTestConversation creates a conversation for testing.
func createTestConversation() *core.Conversation {
	agents := []core.Agent{
		core.NewAgent("agent-1", "openrouter", "Claude", "claude-3-opus", "openrouter"),
		core.NewAgent("agent-2", "openrouter", "GPT-4", "gpt-4-turbo", "openrouter"),
	}
	conv := core.NewConversation(agents)

	// Add some messages
	conv.AddMessage(core.NewUserMessage("Hello, agents!"))
	conv.AddMessage(core.NewAgentMessage("agent-1", "Claude", "Hello! I'm Claude.", &core.Metrics{
		Duration:     500 * time.Millisecond,
		InputTokens:  10,
		OutputTokens: 20,
		TotalTokens:  30,
		Model:        "claude-3-opus",
		Cost:         0.001,
	}))
	conv.AddMessage(core.NewAgentMessage("agent-2", "GPT-4", "Hi there! I'm GPT-4.", &core.Metrics{
		Duration:     600 * time.Millisecond,
		InputTokens:  15,
		OutputTokens: 25,
		TotalTokens:  40,
		Model:        "gpt-4-turbo",
		Cost:         0.002,
	}))

	return conv
}

func TestSaveConversation(t *testing.T) {
	tmpDir := t.TempDir()
	conv := createTestConversation()

	// Save conversation
	filePath, err := persistence.SaveConversation(conv, tmpDir)
	if err != nil {
		t.Fatalf("SaveConversation failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("saved file does not exist: %s", filePath)
	}

	// Verify file content is valid JSON
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("saved file is empty")
	}

	// Verify filename format
	filename := filepath.Base(filePath)
	if !strings.HasPrefix(filename, "conversation_") {
		t.Errorf("filename should start with 'conversation_', got: %s", filename)
	}
	if !strings.HasSuffix(filename, ".json") {
		t.Errorf("filename should end with '.json', got: %s", filename)
	}
}

func TestSaveConversationNil(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := persistence.SaveConversation(nil, tmpDir)
	if err == nil {
		t.Error("expected error for nil conversation")
	}
}

func TestSaveConversationDefaultDir(t *testing.T) {
	conv := createTestConversation()

	// Save with empty dir (uses default)
	filePath, err := persistence.SaveConversation(conv, "")
	if err != nil {
		t.Fatalf("SaveConversation failed: %v", err)
	}

	// Clean up
	defer os.Remove(filePath)

	// Verify file was created in default directory
	if !strings.Contains(filePath, ".agentpipe") {
		t.Errorf("expected path to contain '.agentpipe', got: %s", filePath)
	}
}

func TestLoadConversation(t *testing.T) {
	tmpDir := t.TempDir()
	conv := createTestConversation()

	// Save conversation
	filePath, err := persistence.SaveConversation(conv, tmpDir)
	if err != nil {
		t.Fatalf("SaveConversation failed: %v", err)
	}

	// Load conversation
	loaded, err := persistence.LoadConversation(filePath)
	if err != nil {
		t.Fatalf("LoadConversation failed: %v", err)
	}

	// Verify all fields match
	if loaded.ID != conv.ID {
		t.Errorf("ID mismatch: got %s, want %s", loaded.ID, conv.ID)
	}
	if len(loaded.Messages) != len(conv.Messages) {
		t.Errorf("message count mismatch: got %d, want %d", len(loaded.Messages), len(conv.Messages))
	}
	if len(loaded.Agents) != len(conv.Agents) {
		t.Errorf("agent count mismatch: got %d, want %d", len(loaded.Agents), len(conv.Agents))
	}
	if loaded.Status != conv.Status {
		t.Errorf("status mismatch: got %s, want %s", loaded.Status, conv.Status)
	}

	// Verify message content
	for i, msg := range loaded.Messages {
		if msg.Content != conv.Messages[i].Content {
			t.Errorf("message %d content mismatch: got %s, want %s", i, msg.Content, conv.Messages[i].Content)
		}
		if msg.Role != conv.Messages[i].Role {
			t.Errorf("message %d role mismatch: got %s, want %s", i, msg.Role, conv.Messages[i].Role)
		}
	}

	// Verify agents
	for i, agent := range loaded.Agents {
		if agent.Name != conv.Agents[i].Name {
			t.Errorf("agent %d name mismatch: got %s, want %s", i, agent.Name, conv.Agents[i].Name)
		}
		if agent.Model != conv.Agents[i].Model {
			t.Errorf("agent %d model mismatch: got %s, want %s", i, agent.Model, conv.Agents[i].Model)
		}
	}
}

func TestLoadConversationMissingFile(t *testing.T) {
	_, err := persistence.LoadConversation("/nonexistent/path/conversation.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestLoadConversationInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "conversation_invalid_2024-01-01_00-00-00.json")

	// Write invalid JSON
	err := os.WriteFile(filePath, []byte("invalid json {{{"), 0600)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err = persistence.LoadConversation(filePath)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "parse") && !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("error should mention parsing, got: %v", err)
	}
}

func TestLoadConversationValidationFailure(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "conversation_noID_2024-01-01_00-00-00.json")

	// Write JSON with missing required fields
	err := os.WriteFile(filePath, []byte(`{"messages": []}`), 0600)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err = persistence.LoadConversation(filePath)
	if err == nil {
		t.Error("expected error for validation failure")
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("error should mention 'invalid', got: %v", err)
	}
}

func TestLoadLatest(t *testing.T) {
	tmpDir := t.TempDir()

	// Save multiple conversations with delays
	conv1 := createTestConversation()
	_, err := persistence.SaveConversation(conv1, tmpDir)
	if err != nil {
		t.Fatalf("SaveConversation failed: %v", err)
	}

	// Wait a bit to ensure different timestamps
	time.Sleep(100 * time.Millisecond)

	conv2 := createTestConversation()
	_, err = persistence.SaveConversation(conv2, tmpDir)
	if err != nil {
		t.Fatalf("SaveConversation failed: %v", err)
	}

	// Wait a bit to ensure different timestamps
	time.Sleep(100 * time.Millisecond)

	conv3 := createTestConversation()
	_, err = persistence.SaveConversation(conv3, tmpDir)
	if err != nil {
		t.Fatalf("SaveConversation failed: %v", err)
	}

	// Load latest
	loaded, err := persistence.LoadLatest(tmpDir)
	if err != nil {
		t.Fatalf("LoadLatest failed: %v", err)
	}

	// Should be conv3 (the latest)
	if loaded.ID != conv3.ID {
		t.Errorf("LoadLatest should return most recent conversation, got ID %s, want %s", loaded.ID, conv3.ID)
	}
}

func TestLoadLatestEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := persistence.LoadLatest(tmpDir)
	if err == nil {
		t.Error("expected error for empty directory")
	}
	if !strings.Contains(err.Error(), "no saved conversations") {
		t.Errorf("error should mention 'no saved conversations', got: %v", err)
	}
}

func TestListConversations(t *testing.T) {
	tmpDir := t.TempDir()

	// Save multiple conversations
	conv1 := createTestConversation()
	_, err := persistence.SaveConversation(conv1, tmpDir)
	if err != nil {
		t.Fatalf("SaveConversation failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	conv2 := createTestConversation()
	_, err = persistence.SaveConversation(conv2, tmpDir)
	if err != nil {
		t.Fatalf("SaveConversation failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	conv3 := createTestConversation()
	_, err = persistence.SaveConversation(conv3, tmpDir)
	if err != nil {
		t.Fatalf("SaveConversation failed: %v", err)
	}

	// List conversations
	metadata, err := persistence.ListConversations(tmpDir)
	if err != nil {
		t.Fatalf("ListConversations failed: %v", err)
	}

	if len(metadata) != 3 {
		t.Errorf("expected 3 conversations, got %d", len(metadata))
	}

	// Check metadata fields
	for _, m := range metadata {
		if m.ID == "" {
			t.Error("metadata ID should not be empty")
		}
		if m.FilePath == "" {
			t.Error("metadata FilePath should not be empty")
		}
		if m.MessageCount != 3 {
			t.Errorf("expected 3 messages, got %d", m.MessageCount)
		}
		if m.AgentCount != 2 {
			t.Errorf("expected 2 agents, got %d", m.AgentCount)
		}
		if len(m.AgentNames) != 2 {
			t.Errorf("expected 2 agent names, got %d", len(m.AgentNames))
		}
	}

	// Should be sorted by updated time (newest first)
	if metadata[0].ID != conv3.ID {
		t.Error("first result should be most recent conversation")
	}
}

func TestListConversationsEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	metadata, err := persistence.ListConversations(tmpDir)
	if err != nil {
		t.Fatalf("ListConversations failed: %v", err)
	}

	if len(metadata) != 0 {
		t.Errorf("expected 0 conversations, got %d", len(metadata))
	}
}

func TestListConversationsNonexistentDir(t *testing.T) {
	metadata, err := persistence.ListConversations("/nonexistent/path")
	if err != nil {
		t.Fatalf("ListConversations should not fail for nonexistent dir: %v", err)
	}

	if len(metadata) != 0 {
		t.Errorf("expected 0 conversations, got %d", len(metadata))
	}
}

func TestExportToMarkdown(t *testing.T) {
	conv := createTestConversation()

	// Export to markdown
	markdown, err := persistence.ExportToMarkdown(conv)
	if err != nil {
		t.Fatalf("ExportToMarkdown failed: %v", err)
	}

	// Verify markdown content
	if !strings.Contains(markdown, "# AgentPipe Conversation Export") {
		t.Error("markdown should contain title")
	}
	if !strings.Contains(markdown, conv.ID) {
		t.Error("markdown should contain conversation ID")
	}
	if !strings.Contains(markdown, "## Metadata") {
		t.Error("markdown should contain metadata section")
	}
	if !strings.Contains(markdown, "## Conversation") {
		t.Error("markdown should contain conversation section")
	}
	if !strings.Contains(markdown, "## Summary") {
		t.Error("markdown should contain summary section")
	}

	// Verify agents are listed
	if !strings.Contains(markdown, "Claude") {
		t.Error("markdown should contain agent name 'Claude'")
	}
	if !strings.Contains(markdown, "GPT-4") {
		t.Error("markdown should contain agent name 'GPT-4'")
	}

	// Verify messages
	if !strings.Contains(markdown, "Hello, agents!") {
		t.Error("markdown should contain user message")
	}
	if !strings.Contains(markdown, "Hello! I'm Claude.") {
		t.Error("markdown should contain Claude's message")
	}
	if !strings.Contains(markdown, "Hi there! I'm GPT-4.") {
		t.Error("markdown should contain GPT-4's message")
	}

	// Verify metrics are included
	if !strings.Contains(markdown, "Metrics") {
		t.Error("markdown should contain metrics")
	}
	if !strings.Contains(markdown, "Tokens") {
		t.Error("markdown should contain token info")
	}
}

func TestExportToMarkdownNil(t *testing.T) {
	_, err := persistence.ExportToMarkdown(nil)
	if err == nil {
		t.Error("expected error for nil conversation")
	}
}

func TestSaveAsMarkdown(t *testing.T) {
	tmpDir := t.TempDir()
	conv := createTestConversation()

	outputPath := filepath.Join(tmpDir, "export.md")
	err := persistence.SaveAsMarkdown(conv, outputPath)
	if err != nil {
		t.Fatalf("SaveAsMarkdown failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("markdown file was not created")
	}

	// Verify content
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read markdown file: %v", err)
	}

	if !strings.Contains(string(data), "# AgentPipe Conversation Export") {
		t.Error("markdown file should contain title")
	}
}

func TestSaveAsMarkdownNil(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "export.md")

	err := persistence.SaveAsMarkdown(nil, outputPath)
	if err == nil {
		t.Error("expected error for nil conversation")
	}
}

func TestValidateConversation(t *testing.T) {
	tests := []struct {
		name        string
		conv        *core.Conversation
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil conversation",
			conv:        nil,
			expectError: true,
			errorMsg:    "nil",
		},
		{
			name: "missing ID",
			conv: &core.Conversation{
				Started: time.Now(),
			},
			expectError: true,
			errorMsg:    "ID",
		},
		{
			name: "missing Started",
			conv: &core.Conversation{
				ID: "test-id",
			},
			expectError: true,
			errorMsg:    "Started",
		},
		{
			name: "valid conversation",
			conv: &core.Conversation{
				ID:      "test-id",
				Started: time.Now(),
			},
			expectError: false,
		},
		{
			name:        "valid with messages",
			conv:        createTestConversation(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := persistence.ValidateConversation(tt.conv)
			if tt.expectError {
				if err == nil {
					t.Error("expected error")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("error should contain %q, got: %v", tt.errorMsg, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestFindConversationByID(t *testing.T) {
	tmpDir := t.TempDir()
	conv := createTestConversation()

	_, err := persistence.SaveConversation(conv, tmpDir)
	if err != nil {
		t.Fatalf("SaveConversation failed: %v", err)
	}

	// Find by full ID
	filePath, err := persistence.FindConversationByID(tmpDir, conv.ID)
	if err != nil {
		t.Fatalf("FindConversationByID failed: %v", err)
	}
	if filePath == "" {
		t.Error("expected file path")
	}

	// Find by ID prefix
	idPrefix := conv.ID[:8]
	filePath, err = persistence.FindConversationByID(tmpDir, idPrefix)
	if err != nil {
		t.Fatalf("FindConversationByID failed with prefix: %v", err)
	}
	if filePath == "" {
		t.Error("expected file path")
	}
}

func TestFindConversationByIDNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := persistence.FindConversationByID(tmpDir, "nonexistent-id")
	if err == nil {
		t.Error("expected error for nonexistent ID")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestGenerateMarkdownFilename(t *testing.T) {
	conv := createTestConversation()

	filename := persistence.GenerateMarkdownFilename(conv)

	if !strings.HasPrefix(filename, "conversation_") {
		t.Errorf("filename should start with 'conversation_', got: %s", filename)
	}
	if !strings.HasSuffix(filename, ".md") {
		t.Errorf("filename should end with '.md', got: %s", filename)
	}
	if !strings.Contains(filename, conv.ID[:8]) {
		t.Errorf("filename should contain ID prefix, got: %s", filename)
	}
}

func TestDefaultSaveDir(t *testing.T) {
	dir := persistence.DefaultSaveDir()

	if !strings.Contains(dir, ".agentpipe") {
		t.Errorf("default dir should contain '.agentpipe', got: %s", dir)
	}
	if !strings.Contains(dir, "v2") {
		t.Errorf("default dir should contain 'v2', got: %s", dir)
	}
	if !strings.Contains(dir, "conversations") {
		t.Errorf("default dir should contain 'conversations', got: %s", dir)
	}
}
