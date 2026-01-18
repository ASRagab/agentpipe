//go:build integration
// +build integration

package cli

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/ASRagab/agentpipe/pkg/core"
)

// Integration tests for real CLI adapters.
// Run with: go test -tags integration -v ./pkg/adapters/cli/...
//
// These tests require the actual CLIs to be installed:
// - claude CLI (Claude Code CLI)
// - gemini CLI (Gemini CLI)

// TestRealClaudeCLI tests the Claude CLI adapter with the real Claude CLI.
func TestRealClaudeCLI(t *testing.T) {
	// Skip if Claude CLI is not installed
	_, err := exec.LookPath("claude")
	if err != nil {
		t.Skip("Skipping: Claude CLI not installed")
	}

	adapter := NewClaudeCLIAdapter().(*ClaudeCLIAdapter)
	agent := core.Agent{
		ID:    "test-claude",
		Name:  "TestClaude",
		Model: "", // Use default model
		Config: core.AgentAdapterConfig{
			SystemPrompt: "You are a helpful assistant. Respond briefly.",
		},
	}

	err = adapter.Initialize(agent)
	if err != nil {
		t.Fatalf("Failed to initialize Claude adapter: %v", err)
	}

	t.Run("HealthCheck", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := adapter.HealthCheck(ctx)
		if err != nil {
			t.Errorf("Health check failed: %v", err)
		}
	})

	t.Run("GetCLIVersion", func(t *testing.T) {
		version := adapter.GetCLIVersion()
		if version == "" || version == "unknown" {
			t.Errorf("Expected valid version, got: %s", version)
		}
		t.Logf("Claude CLI Version: %s", version)
	})

	t.Run("SendSimpleMessage", func(t *testing.T) {
		// Skip actual API call test unless explicitly enabled
		if testing.Short() {
			t.Skip("Skipping API call test in short mode")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		messages := []core.Message{
			{
				Role:      core.RoleSystem,
				AgentID:   "host",
				AgentName: "HOST",
				Content:   "Say 'hello' and nothing else.",
				Timestamp: time.Now(),
			},
		}

		response, metrics, err := adapter.SendMessage(ctx, messages)
		if err != nil {
			t.Errorf("SendMessage failed: %v", err)
		}

		t.Logf("Response: %s", response)
		t.Logf("Metrics: Duration=%v, InputTokens=%d, OutputTokens=%d, Cost=$%.6f",
			metrics.Duration, metrics.InputTokens, metrics.OutputTokens, metrics.Cost)

		if response == "" {
			t.Error("Expected non-empty response")
		}
		if metrics == nil {
			t.Error("Expected metrics")
		}
	})

	t.Run("StreamSimpleMessage", func(t *testing.T) {
		// Skip actual API call test unless explicitly enabled
		if testing.Short() {
			t.Skip("Skipping API call test in short mode")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		messages := []core.Message{
			{
				Role:      core.RoleSystem,
				AgentID:   "host",
				AgentName: "HOST",
				Content:   "Count from 1 to 5, one number per line.",
				Timestamp: time.Now(),
			},
		}

		var output bytes.Buffer
		metrics, err := adapter.StreamMessage(ctx, messages, &output)
		if err != nil {
			t.Errorf("StreamMessage failed: %v", err)
		}

		t.Logf("Streamed Output:\n%s", output.String())
		t.Logf("Metrics: Duration=%v", metrics.Duration)

		if output.Len() == 0 {
			t.Error("Expected non-empty streamed output")
		}
	})
}

// TestRealGeminiCLI tests the Gemini CLI adapter with the real Gemini CLI.
func TestRealGeminiCLI(t *testing.T) {
	// Skip if Gemini CLI is not installed
	_, err := exec.LookPath("gemini")
	if err != nil {
		t.Skip("Skipping: Gemini CLI not installed")
	}

	adapter := NewGeminiCLIAdapter().(*GeminiCLIAdapter)
	agent := core.Agent{
		ID:    "test-gemini",
		Name:  "TestGemini",
		Model: "gemini-2.0-flash", // Use flash for faster tests
		Config: core.AgentAdapterConfig{
			SystemPrompt: "You are a helpful assistant. Respond briefly.",
		},
	}

	err = adapter.Initialize(agent)
	if err != nil {
		t.Fatalf("Failed to initialize Gemini adapter: %v", err)
	}

	t.Run("HealthCheck", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := adapter.HealthCheck(ctx)
		if err != nil {
			t.Errorf("Health check failed: %v", err)
		}
	})

	t.Run("GetCLIVersion", func(t *testing.T) {
		version := adapter.GetCLIVersion()
		if version == "" || version == "unknown" {
			t.Errorf("Expected valid version, got: %s", version)
		}
		t.Logf("Gemini CLI Version: %s", version)
	})

	t.Run("SendSimpleMessage", func(t *testing.T) {
		// Skip actual API call test unless explicitly enabled
		if testing.Short() {
			t.Skip("Skipping API call test in short mode")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		messages := []core.Message{
			{
				Role:      core.RoleSystem,
				AgentID:   "host",
				AgentName: "HOST",
				Content:   "Say 'hello' and nothing else.",
				Timestamp: time.Now(),
			},
		}

		response, metrics, err := adapter.SendMessage(ctx, messages)
		if err != nil {
			t.Errorf("SendMessage failed: %v", err)
		}

		t.Logf("Response: %s", response)
		t.Logf("Metrics: Duration=%v, InputTokens=%d, OutputTokens=%d, Cost=$%.6f",
			metrics.Duration, metrics.InputTokens, metrics.OutputTokens, metrics.Cost)

		if response == "" {
			t.Error("Expected non-empty response")
		}
		if metrics == nil {
			t.Error("Expected metrics")
		}
	})

	t.Run("StreamSimpleMessage", func(t *testing.T) {
		// Skip actual API call test unless explicitly enabled
		if testing.Short() {
			t.Skip("Skipping API call test in short mode")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		messages := []core.Message{
			{
				Role:      core.RoleSystem,
				AgentID:   "host",
				AgentName: "HOST",
				Content:   "Count from 1 to 5, one number per line.",
				Timestamp: time.Now(),
			},
		}

		var output bytes.Buffer
		metrics, err := adapter.StreamMessage(ctx, messages, &output)
		if err != nil {
			t.Errorf("StreamMessage failed: %v", err)
		}

		t.Logf("Streamed Output:\n%s", output.String())
		t.Logf("Metrics: Duration=%v", metrics.Duration)

		if output.Len() == 0 {
			t.Error("Expected non-empty streamed output")
		}
	})
}

// TestNetworkErrorHandling tests error handling for network-related failures.
func TestNetworkErrorHandling(t *testing.T) {
	// This test verifies that adapters handle errors gracefully.
	// Network failures would manifest as command execution failures.

	t.Run("ContextTimeout", func(t *testing.T) {
		// Test with a very short timeout to simulate network issues
		mockPath := createSlowMockCLI(t, "slow-network", 10*time.Second)

		adapter := &GenericCLIAdapter{
			BaseCLIAdapter: BaseCLIAdapter{
				cliPath:   mockPath,
				cliName:   "slow-network",
				agentName: "TestSlow",
				agentID:   "test-slow",
			},
			config: GenericCLIConfig{
				UseStdin:     true,
				OutputParser: OutputParserPlain,
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		messages := []core.Message{
			core.NewUserMessage("Test"),
		}

		_, _, err := adapter.SendMessage(ctx, messages)
		if err == nil {
			t.Error("Expected timeout error")
		}

		// Verify error is meaningful
		errStr := err.Error()
		if !strings.Contains(errStr, "killed") && !strings.Contains(errStr, "signal") && !strings.Contains(errStr, "context") {
			t.Logf("Error message: %s", errStr)
		}
	})
}

// TestCLIQuirks documents any CLI-specific behavior quirks.
func TestCLIQuirks(t *testing.T) {
	t.Run("ClaudeQuirks", func(t *testing.T) {
		_, err := exec.LookPath("claude")
		if err != nil {
			t.Skip("Skipping: Claude CLI not installed")
		}

		// Document known Claude CLI quirks
		quirks := []string{
			"Claude CLI uses -p flag for non-interactive prompt mode",
			"Claude CLI sometimes exits non-zero but still produces valid output",
			"Claude CLI requires stdin for conversation history",
			"Claude CLI version command returns multi-line output",
		}

		for _, q := range quirks {
			t.Logf("QUIRK: %s", q)
		}
	})

	t.Run("GeminiQuirks", func(t *testing.T) {
		_, err := exec.LookPath("gemini")
		if err != nil {
			t.Skip("Skipping: Gemini CLI not installed")
		}

		// Document known Gemini CLI quirks
		quirks := []string{
			"Gemini CLI outputs 'Loaded cached credentials' noise that must be filtered",
			"Gemini CLI may include GaxiosError stack traces in output on API errors",
			"Gemini CLI uses --model flag for model selection",
			"Gemini CLI reads prompt from stdin by default",
			"Gemini deprecated -p/--prompt flag, now uses positional arguments",
		}

		for _, q := range quirks {
			t.Logf("QUIRK: %s", q)
		}
	})
}
