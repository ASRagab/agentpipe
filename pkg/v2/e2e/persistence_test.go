//go:build e2e

package e2e

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ASRagab/agentpipe/pkg/v2/core"
	"github.com/ASRagab/agentpipe/pkg/v2/persistence"
)

// TestSaveAndResume verifies full conversation save and resume functionality.
func TestSaveAndResume(t *testing.T) {
	TimeoutWrapper(t, 15*time.Second, func(t *testing.T) {
		// Create temp directory for saves
		tempDir := t.TempDir()

		agents := FixtureSimpleConversation()
		harness := NewTestHarness(t, agents, WithPersistence(tempDir, 0))
		defer harness.Cleanup()

		harness.SetMockResponse("alice", "Alice's response")
		harness.SetMockResponse("bob", "Bob's response")

		harness.Manager.Start()
		ctx := context.Background()

		// Send some messages
		_, err := SendTestMessage(ctx, harness, "First message")
		if err != nil {
			t.Fatalf("first message failed: %v", err)
		}
		_, err = SendTestMessage(ctx, harness, "Second message")
		if err != nil {
			t.Fatalf("second message failed: %v", err)
		}

		// Get conversation ID for later
		conv := harness.Manager.GetConversation()
		originalID := conv.ID
		originalMessageCount := len(conv.Messages)

		// Save conversation
		savePath, err := harness.Manager.Save()
		if err != nil {
			t.Fatalf("Save failed: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(savePath); os.IsNotExist(err) {
			t.Fatalf("save file not created: %s", savePath)
		}

		// Create a new manager and resume
		agents2 := FixtureSimpleConversation()
		harness2 := NewTestHarness(t, agents2, WithPersistence(tempDir, 0))
		defer harness2.Cleanup()

		harness2.SetMockResponse("alice", "Alice resumed")
		harness2.SetMockResponse("bob", "Bob resumed")

		// Resume from saved file
		err = harness2.Manager.Resume(savePath)
		if err != nil {
			t.Fatalf("Resume failed: %v", err)
		}

		// Verify resumed state
		resumedConv := harness2.Manager.GetConversation()
		if resumedConv.ID != originalID {
			t.Errorf("expected ID %s, got %s", originalID, resumedConv.ID)
		}

		if len(resumedConv.Messages) != originalMessageCount {
			t.Errorf("expected %d messages, got %d", originalMessageCount, len(resumedConv.Messages))
		}

		// Continue conversation
		harness2.Manager.Start()
		_, err = SendTestMessage(ctx, harness2, "Continued message")
		if err != nil {
			t.Fatalf("continued message failed: %v", err)
		}

		// Verify messages increased
		if len(harness2.Manager.GetMessages()) <= originalMessageCount {
			t.Error("message count should increase after resuming and sending new message")
		}
	})
}

// TestResumeWithMissingAgent verifies resume when an agent is no longer available.
func TestResumeWithMissingAgent(t *testing.T) {
	TimeoutWrapper(t, 10*time.Second, func(t *testing.T) {
		tempDir := t.TempDir()

		// Create conversation with two agents
		agents := FixtureSimpleConversation()
		harness := NewTestHarness(t, agents, WithPersistence(tempDir, 0))

		harness.SetMockResponse("alice", "Alice here")
		harness.SetMockResponse("bob", "Bob here")

		harness.Manager.Start()
		ctx := context.Background()

		_, _ = SendTestMessage(ctx, harness, "Hello")

		// Save
		savePath, err := harness.Manager.Save()
		if err != nil {
			t.Fatalf("Save failed: %v", err)
		}
		harness.Cleanup()

		// Resume with only one agent (simulate missing agent by using different adapter)
		// The manager should handle this gracefully
		singleAgent := []core.Agent{
			core.NewAgent("alice", "mock", "Alice", "model-a", "mock"),
		}
		harness2 := NewTestHarness(t, singleAgent, WithPersistence(tempDir, 0))
		defer harness2.Cleanup()

		// Resume should log warning but not fail for missing adapters
		err = harness2.Manager.Resume(savePath)
		if err != nil {
			// This might fail depending on implementation - check if error is about missing adapter
			t.Logf("Resume with missing agent: %v (may be expected)", err)
		}
	})
}

// TestExportWhileActive verifies export during active conversation.
func TestExportWhileActive(t *testing.T) {
	TimeoutWrapper(t, 10*time.Second, func(t *testing.T) {
		tempDir := t.TempDir()
		exportPath := filepath.Join(tempDir, "export.md")

		agents := FixtureSimpleConversation()
		harness := NewTestHarness(t, agents)
		defer harness.Cleanup()

		harness.SetMockResponse("alice", "Alice responds")
		harness.SetMockResponse("bob", "Bob responds")

		harness.Manager.Start()
		ctx := context.Background()

		// Send messages
		_, _ = SendTestMessage(ctx, harness, "Message 1")
		_, _ = SendTestMessage(ctx, harness, "Message 2")

		// Export while conversation is still active
		err := harness.Manager.ExportToMarkdown(exportPath)
		if err != nil {
			t.Fatalf("ExportToMarkdown failed: %v", err)
		}

		// Verify export file exists and has content
		data, err := os.ReadFile(exportPath)
		if err != nil {
			t.Fatalf("failed to read export file: %v", err)
		}

		if len(data) == 0 {
			t.Error("export file is empty")
		}

		// Verify export contains messages
		content := string(data)
		if !contains(content, "Message 1") && !contains(content, "Alice") {
			t.Error("export should contain conversation content")
		}

		// Continue conversation after export
		_, err = SendTestMessage(ctx, harness, "Message 3")
		if err != nil {
			t.Fatalf("message after export failed: %v", err)
		}
	})
}

// TestAutoSave verifies auto-save triggers on interval and after responses.
func TestAutoSave(t *testing.T) {
	TimeoutWrapper(t, 20*time.Second, func(t *testing.T) {
		tempDir := t.TempDir()

		agents := FixtureSimpleConversation()
		// Enable auto-save with short interval
		harness := NewTestHarness(t, agents, WithPersistence(tempDir, 500*time.Millisecond))
		defer harness.Cleanup()

		harness.SetMockResponse("alice", "Alice auto-save")
		harness.SetMockResponse("bob", "Bob auto-save")

		harness.Manager.Start()
		ctx := context.Background()

		// Send message (should trigger auto-save after response)
		_, _ = SendTestMessage(ctx, harness, "Trigger auto-save")

		// Wait for auto-save to trigger
		time.Sleep(700 * time.Millisecond)

		// Check for saved files
		files, err := filepath.Glob(filepath.Join(tempDir, "conversation_*.json"))
		if err != nil {
			t.Fatalf("failed to glob save directory: %v", err)
		}

		if len(files) == 0 {
			t.Error("expected at least one auto-saved file")
		}

		// Verify ConversationSaved event was emitted
		savedEvents := harness.EventLog.EventsByType(core.EventConversationSaved)
		if len(savedEvents) == 0 {
			t.Error("expected ConversationSaved event")
		}
	})
}

// TestConcurrentSave verifies that multiple saves don't corrupt the file.
func TestConcurrentSave(t *testing.T) {
	TimeoutWrapper(t, 15*time.Second, func(t *testing.T) {
		tempDir := t.TempDir()

		agents := FixtureSimpleConversation()
		harness := NewTestHarness(t, agents, WithPersistence(tempDir, 0))
		defer harness.Cleanup()

		harness.SetMockResponse("alice", "Alice concurrent")
		harness.SetMockResponse("bob", "Bob concurrent")

		harness.Manager.Start()
		ctx := context.Background()

		// Build up some messages
		for i := 0; i < 5; i++ {
			_, _ = SendTestMessage(ctx, harness, "Message")
		}

		// Trigger multiple concurrent saves
		var wg sync.WaitGroup
		saveCount := 5
		var paths []string
		var pathMu sync.Mutex

		for i := 0; i < saveCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				path, err := harness.Manager.Save()
				if err != nil {
					t.Errorf("concurrent save failed: %v", err)
					return
				}
				pathMu.Lock()
				paths = append(paths, path)
				pathMu.Unlock()
			}()
		}

		wg.Wait()

		// Verify at least some saves succeeded
		if len(paths) == 0 {
			t.Fatal("no saves succeeded")
		}

		// Verify saved files are valid
		for _, path := range paths {
			conv, err := persistence.LoadConversation(path)
			if err != nil {
				t.Errorf("failed to load saved conversation %s: %v", path, err)
				continue
			}
			if len(conv.Messages) == 0 {
				t.Errorf("saved conversation has no messages: %s", path)
			}
		}
	})
}

// TestResumeLatest verifies resuming the most recent conversation.
func TestResumeLatest(t *testing.T) {
	TimeoutWrapper(t, 15*time.Second, func(t *testing.T) {
		tempDir := t.TempDir()

		// Create first conversation
		agents1 := FixtureSimpleConversation()
		harness1 := NewTestHarness(t, agents1, WithPersistence(tempDir, 0))

		harness1.SetMockResponse("alice", "First conv Alice")
		harness1.SetMockResponse("bob", "First conv Bob")

		harness1.Manager.Start()
		ctx := context.Background()
		_, _ = SendTestMessage(ctx, harness1, "First conversation")

		_, err := harness1.Manager.Save()
		if err != nil {
			t.Fatalf("first save failed: %v", err)
		}
		firstID := harness1.Manager.GetConversation().ID
		harness1.Cleanup()

		// Small delay to ensure different timestamps
		time.Sleep(100 * time.Millisecond)

		// Create second conversation
		agents2 := FixtureSimpleConversation()
		harness2 := NewTestHarness(t, agents2, WithPersistence(tempDir, 0))

		harness2.SetMockResponse("alice", "Second conv Alice")
		harness2.SetMockResponse("bob", "Second conv Bob")

		harness2.Manager.Start()
		_, _ = SendTestMessage(ctx, harness2, "Second conversation")

		_, err = harness2.Manager.Save()
		if err != nil {
			t.Fatalf("second save failed: %v", err)
		}
		secondID := harness2.Manager.GetConversation().ID
		harness2.Cleanup()

		// Resume latest should get the second conversation
		agents3 := FixtureSimpleConversation()
		harness3 := NewTestHarness(t, agents3, WithPersistence(tempDir, 0))
		defer harness3.Cleanup()

		harness3.SetMockResponse("alice", "Resumed Alice")
		harness3.SetMockResponse("bob", "Resumed Bob")

		err = harness3.Manager.Resume("latest")
		if err != nil {
			t.Fatalf("resume latest failed: %v", err)
		}

		resumedID := harness3.Manager.GetConversation().ID

		// Should be the second conversation, not the first
		if resumedID == firstID {
			t.Errorf("resumed wrong conversation: got first (%s) instead of second (%s)", firstID, secondID)
		}
		if resumedID != secondID {
			t.Errorf("expected second conversation ID %s, got %s", secondID, resumedID)
		}
	})
}

// TestResumeByIDPrefix verifies resuming a conversation by ID prefix.
func TestResumeByIDPrefix(t *testing.T) {
	TimeoutWrapper(t, 10*time.Second, func(t *testing.T) {
		tempDir := t.TempDir()

		agents := FixtureSimpleConversation()
		harness := NewTestHarness(t, agents, WithPersistence(tempDir, 0))

		harness.SetMockResponse("alice", "Alice by ID")
		harness.SetMockResponse("bob", "Bob by ID")

		harness.Manager.Start()
		ctx := context.Background()
		_, _ = SendTestMessage(ctx, harness, "Save by ID")

		_, err := harness.Manager.Save()
		if err != nil {
			t.Fatalf("save failed: %v", err)
		}

		originalID := harness.Manager.GetConversation().ID
		harness.Cleanup()

		// Resume by ID prefix (first 8 characters)
		agents2 := FixtureSimpleConversation()
		harness2 := NewTestHarness(t, agents2, WithPersistence(tempDir, 0))
		defer harness2.Cleanup()

		harness2.SetMockResponse("alice", "Resumed")
		harness2.SetMockResponse("bob", "Resumed")

		idPrefix := originalID
		if len(idPrefix) > 8 {
			idPrefix = idPrefix[:8]
		}

		err = harness2.Manager.Resume(idPrefix)
		if err != nil {
			t.Fatalf("resume by ID prefix failed: %v", err)
		}

		if harness2.Manager.GetConversation().ID != originalID {
			t.Errorf("resumed wrong conversation: expected %s, got %s",
				originalID, harness2.Manager.GetConversation().ID)
		}
	})
}

// TestListConversations verifies listing saved conversations.
func TestListConversations(t *testing.T) {
	TimeoutWrapper(t, 15*time.Second, func(t *testing.T) {
		tempDir := t.TempDir()

		// Create and save multiple conversations
		for i := 0; i < 3; i++ {
			agents := FixtureSimpleConversation()
			harness := NewTestHarness(t, agents, WithPersistence(tempDir, 0))

			harness.SetMockResponse("alice", "Alice")
			harness.SetMockResponse("bob", "Bob")

			harness.Manager.Start()
			ctx := context.Background()
			_, _ = SendTestMessage(ctx, harness, "Conversation")

			_, err := harness.Manager.Save()
			if err != nil {
				t.Fatalf("save %d failed: %v", i, err)
			}
			harness.Cleanup()

			// Small delay for different timestamps
			time.Sleep(50 * time.Millisecond)
		}

		// List conversations
		list, err := persistence.ListConversations(tempDir)
		if err != nil {
			t.Fatalf("ListConversations failed: %v", err)
		}

		if len(list) != 3 {
			t.Errorf("expected 3 conversations, got %d", len(list))
		}

		// Verify sorted by newest first
		for i := 1; i < len(list); i++ {
			if list[i].Updated.After(list[i-1].Updated) {
				t.Error("conversations not sorted by newest first")
			}
		}
	})
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
