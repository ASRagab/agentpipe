//go:build e2e

package e2e

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ASRagab/agentpipe/pkg/config"
	"github.com/ASRagab/agentpipe/pkg/core"
	"github.com/ASRagab/agentpipe/pkg/persistence"
)

// Platform-specific notes:
// - Windows: Timer resolution is ~15.6ms (vs ~1ms on Unix)
// - Windows: Path separators are backslashes
// - macOS/Linux: Standard Unix path handling
// - All platforms: Should support UTF-8 content

// TestPlatformInfo logs current platform information.
func TestPlatformInfo(t *testing.T) {
	t.Logf("Platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	t.Logf("Go Version: %s", runtime.Version())
	t.Logf("NumCPU: %d", runtime.NumCPU())

	// Log working directory
	wd, _ := os.Getwd()
	t.Logf("Working Directory: %s", wd)

	// Log temp directory
	t.Logf("Temp Directory: %s", os.TempDir())
}

// TestFilePathHandling verifies that file paths work correctly on the current platform.
func TestFilePathHandling(t *testing.T) {
	TimeoutWrapper(t, 10*time.Second, func(t *testing.T) {
		tempDir := t.TempDir()

		// Test nested directory creation
		nestedDir := filepath.Join(tempDir, "level1", "level2", "level3")
		err := os.MkdirAll(nestedDir, 0750)
		if err != nil {
			t.Fatalf("failed to create nested directory: %v", err)
		}

		// Test file creation with special characters (platform-safe subset)
		testFile := filepath.Join(nestedDir, "test-file_123.json")
		err = os.WriteFile(testFile, []byte(`{"test": true}`), 0600)
		if err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		// Test file read
		data, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("failed to read test file: %v", err)
		}

		if string(data) != `{"test": true}` {
			t.Error("file content mismatch")
		}

		// Test filepath.Join produces valid paths
		path := filepath.Join("parent", "child", "file.txt")
		t.Logf("filepath.Join result: %s", path)

		// Platform-specific separator
		t.Logf("Path separator: %q", string(filepath.Separator))
	})
}

// TestConversationOnPlatform verifies that conversations work on the current platform.
func TestConversationOnPlatform(t *testing.T) {
	TimeoutWrapper(t, 15*time.Second, func(t *testing.T) {
		agents := FixtureSimpleConversation()
		harness := NewTestHarness(t, agents)
		defer harness.Cleanup()

		harness.SetMockResponse("alice", "Platform test from Alice")
		harness.SetMockResponse("bob", "Platform test from Bob")

		harness.Manager.Start()
		ctx := context.Background()

		responses, err := SendTestMessage(ctx, harness, "Platform test message")
		if err != nil {
			t.Fatalf("SendTestMessage failed: %v", err)
		}

		if len(responses) != 2 {
			t.Errorf("expected 2 responses, got %d", len(responses))
		}

		t.Logf("Successfully received %d responses on %s/%s", len(responses), runtime.GOOS, runtime.GOARCH)
	})
}

// TestTimerResolution checks timer behavior on the platform.
func TestTimerResolution(t *testing.T) {
	TimeoutWrapper(t, 10*time.Second, func(t *testing.T) {
		// On Windows, timer resolution is typically ~15.6ms
		// On Unix, it's typically ~1ms

		iterations := 10
		var minDuration, maxDuration time.Duration

		for i := 0; i < iterations; i++ {
			start := time.Now()
			time.Sleep(1 * time.Millisecond)
			elapsed := time.Since(start)

			if minDuration == 0 || elapsed < minDuration {
				minDuration = elapsed
			}
			if elapsed > maxDuration {
				maxDuration = elapsed
			}
		}

		t.Logf("Timer resolution test: min=%v, max=%v", minDuration, maxDuration)

		// On Windows, minimum measurable duration is higher
		if runtime.GOOS == "windows" {
			if minDuration < 10*time.Millisecond {
				t.Log("Note: Windows timer resolution lower than expected")
			}
		} else {
			if minDuration > 5*time.Millisecond {
				t.Log("Note: Unix timer resolution higher than expected")
			}
		}
	})
}

// TestPersistenceOnPlatform verifies that persistence works on the current platform.
func TestPersistenceOnPlatform(t *testing.T) {
	TimeoutWrapper(t, 15*time.Second, func(t *testing.T) {
		tempDir := t.TempDir()

		agents := FixtureSimpleConversation()
		harness := NewTestHarness(t, agents, WithPersistence(tempDir, 0))
		defer harness.Cleanup()

		harness.SetMockResponse("alice", "Platform persistence test")
		harness.SetMockResponse("bob", "Platform persistence test")

		harness.Manager.Start()
		ctx := context.Background()

		// Send messages
		_, _ = SendTestMessage(ctx, harness, "Persistence test")

		// Save
		savePath, err := harness.Manager.Save()
		if err != nil {
			t.Fatalf("Save failed: %v", err)
		}

		// Verify file exists and is readable
		info, err := os.Stat(savePath)
		if err != nil {
			t.Fatalf("failed to stat save file: %v", err)
		}

		t.Logf("Saved file: %s (size: %d bytes)", savePath, info.Size())

		// Load and verify
		conv, err := persistence.LoadConversation(savePath)
		if err != nil {
			t.Fatalf("LoadConversation failed: %v", err)
		}

		if len(conv.Messages) == 0 {
			t.Error("loaded conversation has no messages")
		}
	})
}

// TestConfigOnPlatform verifies that config loading works on the current platform.
func TestConfigOnPlatform(t *testing.T) {
	TimeoutWrapper(t, 10*time.Second, func(t *testing.T) {
		tempDir := t.TempDir()

		configContent := `
conversation:
  timeout: 30s
agents:
  - id: platform-agent
    name: Platform Agent
    type: mock
    model: mock-model
`
		configPath := filepath.Join(tempDir, "config.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0600)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if len(cfg.Agents) != 1 {
			t.Errorf("expected 1 agent, got %d", len(cfg.Agents))
		}

		t.Logf("Config loaded successfully on %s/%s", runtime.GOOS, runtime.GOARCH)
	})
}

// TestUTF8Content verifies that UTF-8 content is handled correctly.
func TestUTF8Content(t *testing.T) {
	TimeoutWrapper(t, 10*time.Second, func(t *testing.T) {
		agents := []core.Agent{
			core.NewAgent("utf8-agent", "mock", "UTF-8 Agent", "model", "mock"),
		}
		harness := NewTestHarness(t, agents)
		defer harness.Cleanup()

		// Test various Unicode content
		testCases := []string{
			"Hello, World!",                // ASCII
			"Привет, мир!",                 // Cyrillic
			"你好，世界！",                       // Chinese
			"مرحبا بالعالم",                // Arabic
			"🚀 Emoji content 🎉",            // Emoji
			"Mixed: Hello 你好 🌍",            // Mixed
			"Special: café, résumé, naïve", // Accented Latin
			"Symbols: ∀x∈ℝ, √2 ≈ 1.414",    // Math symbols
		}

		harness.Manager.Start()
		ctx := context.Background()

		for _, tc := range testCases {
			harness.SetMockResponse("utf8-agent", "Echo: "+tc)

			responses, err := SendTestMessage(ctx, harness, tc)
			if err != nil {
				t.Errorf("UTF-8 test failed for %q: %v", tc, err)
				continue
			}

			if len(responses) == 0 {
				t.Errorf("no response for UTF-8 message: %q", tc)
			}
		}

		t.Logf("UTF-8 content tests passed on %s/%s", runtime.GOOS, runtime.GOARCH)
	})
}

// TestConcurrencyOnPlatform verifies that concurrent operations work correctly.
func TestConcurrencyOnPlatform(t *testing.T) {
	TimeoutWrapper(t, 20*time.Second, func(t *testing.T) {
		agents := FixtureSimpleConversation()
		harness := NewTestHarness(t, agents)
		defer harness.Cleanup()

		harness.SetMockDelay("alice", 30*time.Millisecond)
		harness.SetMockDelay("bob", 30*time.Millisecond)
		harness.SetMockResponse("alice", "Concurrent Alice")
		harness.SetMockResponse("bob", "Concurrent Bob")

		harness.Manager.Start()
		ctx := context.Background()

		// Run multiple iterations to stress test
		iterations := 10
		for i := 0; i < iterations; i++ {
			responses, err := SendTestMessage(ctx, harness, "Concurrent test")
			if err != nil {
				t.Errorf("iteration %d failed: %v", i, err)
				continue
			}
			if len(responses) != 2 {
				t.Errorf("iteration %d: expected 2 responses, got %d", i, len(responses))
			}
		}

		t.Logf("Concurrency tests passed on %s/%s", runtime.GOOS, runtime.GOARCH)
	})
}

// TestEventBusOnPlatform verifies that event bus works correctly on the platform.
func TestEventBusOnPlatform(t *testing.T) {
	TimeoutWrapper(t, 10*time.Second, func(t *testing.T) {
		agents := FixtureSimpleConversation()
		harness := NewTestHarness(t, agents)
		defer harness.Cleanup()

		harness.SetMockResponse("alice", "Event bus test Alice")
		harness.SetMockResponse("bob", "Event bus test Bob")

		harness.Manager.Start()
		ctx := context.Background()

		_, err := SendTestMessage(ctx, harness, "Event bus test")
		if err != nil {
			t.Fatalf("SendTestMessage failed: %v", err)
		}

		// Wait for all events
		time.Sleep(200 * time.Millisecond)

		// Verify event sequence
		events := harness.EventLog.Events()
		if len(events) == 0 {
			t.Error("no events recorded")
		}

		// Count event types
		eventCounts := make(map[core.EventType]int)
		for _, e := range events {
			eventCounts[e.Type]++
		}

		t.Logf("Event counts on %s/%s:", runtime.GOOS, runtime.GOARCH)
		for etype, count := range eventCounts {
			t.Logf("  %s: %d", etype, count)
		}
	})
}

// Platform-specific known issues documentation
func TestDocumentPlatformIssues(t *testing.T) {
	issues := map[string][]string{
		"windows": {
			"Timer resolution is ~15.6ms (use >= 20ms delays in tests)",
			"Path separators are backslashes",
			"Some terminal escape sequences may not work",
		},
		"darwin": {
			"No known platform-specific issues",
		},
		"linux": {
			"No known platform-specific issues",
		},
	}

	if knownIssues, ok := issues[runtime.GOOS]; ok {
		t.Logf("Known issues for %s:", runtime.GOOS)
		for _, issue := range knownIssues {
			t.Logf("  - %s", issue)
		}
	} else {
		t.Logf("No documented issues for %s", runtime.GOOS)
	}
}
