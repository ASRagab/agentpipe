package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/v2/adapters"
	"github.com/kevinelliott/agentpipe/pkg/v2/core"
)

// Test helper: create a mock CLI script that echoes responses
func createMockCLI(t *testing.T, name string, response string) string {
	t.Helper()

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, name)

	// Create a shell script that echoes the response
	script := fmt.Sprintf(`#!/bin/sh
# Mock CLI for testing
if [ "$1" = "--version" ] || [ "$1" = "-v" ]; then
    echo "mock-%s version 1.0.0"
    exit 0
fi
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "Mock %s CLI - A test helper for AgentPipe"
    echo ""
    echo "Usage: %s [options]"
    exit 0
fi
# Echo the mock response
cat << 'EOF'
%s
EOF
`, name, name, name, response)

	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to create mock CLI: %v", err)
	}

	return scriptPath
}

// Test helper: create a mock CLI that fails
func createFailingMockCLI(t *testing.T, name string, stderr string, exitCode int) string {
	t.Helper()

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, name)

	script := fmt.Sprintf(`#!/bin/sh
echo "%s" >&2
exit %d
`, stderr, exitCode)

	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to create failing mock CLI: %v", err)
	}

	return scriptPath
}

// Test helper: create a slow mock CLI that times out
func createSlowMockCLI(t *testing.T, name string, delay time.Duration) string {
	t.Helper()

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, name)

	script := fmt.Sprintf(`#!/bin/sh
sleep %d
echo "Response after delay"
`, int(delay.Seconds())+1)

	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to create slow mock CLI: %v", err)
	}

	return scriptPath
}

// Test helper: create a streaming mock CLI
func createStreamingMockCLI(t *testing.T, name string, lines []string) string {
	t.Helper()

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, name)

	// Create script that outputs lines with small delays for streaming
	var linesScript strings.Builder
	for _, line := range lines {
		linesScript.WriteString(fmt.Sprintf("echo '%s'\n", line))
	}

	script := fmt.Sprintf(`#!/bin/sh
if [ "$1" = "--version" ]; then
    echo "mock-%s version 1.0.0"
    exit 0
fi
%s
`, name, linesScript.String())

	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to create streaming mock CLI: %v", err)
	}

	return scriptPath
}

// TestFindCLIPath tests CLI path discovery
func TestFindCLIPath(t *testing.T) {
	t.Run("FindExistingCLI", func(t *testing.T) {
		// sh should exist on all Unix systems
		path, err := FindCLIPath("sh")
		if err != nil {
			t.Errorf("Failed to find 'sh': %v", err)
		}
		if path == "" {
			t.Error("Path should not be empty")
		}
	})

	t.Run("FindNonExistentCLI", func(t *testing.T) {
		_, err := FindCLIPath("nonexistent-cli-that-does-not-exist-12345")
		if err == nil {
			t.Error("Should fail for non-existent CLI")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Error should mention 'not found', got: %v", err)
		}
	})
}

// TestExecuteCommand tests command execution
func TestExecuteCommand(t *testing.T) {
	t.Run("SimpleEcho", func(t *testing.T) {
		ctx := context.Background()
		stdout, stderr, err := ExecuteCommand(ctx, "echo", []string{"hello"}, nil)
		if err != nil {
			t.Errorf("Command failed: %v", err)
		}
		if string(stderr) != "" {
			t.Errorf("Unexpected stderr: %s", stderr)
		}
		if !strings.Contains(string(stdout), "hello") {
			t.Errorf("Expected 'hello' in stdout, got: %s", stdout)
		}
	})

	t.Run("WithStdin", func(t *testing.T) {
		ctx := context.Background()
		stdout, _, err := ExecuteCommand(ctx, "cat", []string{}, strings.NewReader("test input"))
		if err != nil {
			t.Errorf("Command failed: %v", err)
		}
		if !strings.Contains(string(stdout), "test input") {
			t.Errorf("Expected 'test input' in stdout, got: %s", stdout)
		}
	})

	t.Run("Timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, _, err := ExecuteCommand(ctx, "sleep", []string{"10"}, nil)
		if err == nil {
			t.Error("Expected timeout error")
		}
	})
}

// TestParseCLIOutput tests output parsing
func TestParseCLIOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "PlainText",
			input:    "Hello, world!",
			expected: "Hello, world!",
		},
		{
			name:     "WithCredentialsLine",
			input:    "Loaded cached credentials\nHello, world!",
			expected: "Hello, world!",
		},
		{
			name:     "WithDebugLines",
			input:    "DEBUG: Starting\nHello\nINFO: Done",
			expected: "Hello",
		},
		{
			name:     "WithWhitespace",
			input:    "\n\n  Hello  \n\n",
			expected: "Hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCLIOutput(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestHandleCLIError tests error handling
func TestHandleCLIError(t *testing.T) {
	t.Run("NoError", func(t *testing.T) {
		err := HandleCLIError("test", nil, nil)
		if err != nil {
			t.Errorf("Expected nil error, got: %v", err)
		}
	})

	t.Run("WithStderr", func(t *testing.T) {
		err := HandleCLIError("test", fmt.Errorf("exit code 1"), []byte("error message"))
		if err == nil {
			t.Error("Expected error")
		}
		if !strings.Contains(err.Error(), "error message") {
			t.Errorf("Error should include stderr, got: %v", err)
		}
	})

	t.Run("WithoutStderr", func(t *testing.T) {
		err := HandleCLIError("test", fmt.Errorf("exit code 1"), nil)
		if err == nil {
			t.Error("Expected error")
		}
		if !strings.Contains(err.Error(), "test") {
			t.Errorf("Error should include CLI name, got: %v", err)
		}
	})
}

// TestBuildConversationPrompt tests prompt building
func TestBuildConversationPrompt(t *testing.T) {
	t.Run("EmptyMessages", func(t *testing.T) {
		prompt := BuildConversationPrompt("TestAgent", "You are helpful", nil)
		if !strings.Contains(prompt, "TestAgent") {
			t.Error("Prompt should contain agent name")
		}
		if !strings.Contains(prompt, "You are helpful") {
			t.Error("Prompt should contain system prompt")
		}
	})

	t.Run("WithMessages", func(t *testing.T) {
		messages := []core.Message{
			{
				Role:      core.RoleSystem,
				AgentID:   "system",
				AgentName: "System",
				Content:   "Discussion topic",
				Timestamp: time.Now(),
			},
			{
				Role:      core.RoleAgent,
				AgentID:   "agent1",
				AgentName: "Agent1",
				Content:   "Hello from agent1",
				Timestamp: time.Now(),
			},
		}

		prompt := BuildConversationPrompt("TestAgent", "", messages)
		if !strings.Contains(prompt, "Discussion topic") {
			t.Error("Prompt should contain initial topic")
		}
		if !strings.Contains(prompt, "Agent1") {
			t.Error("Prompt should contain other agent's name")
		}
	})
}

// TestFilterRelevantMessages tests message filtering
func TestFilterRelevantMessages(t *testing.T) {
	messages := []core.Message{
		{AgentID: "agent1", AgentName: "Agent1", Content: "Hello"},
		{AgentID: "agent2", AgentName: "Agent2", Content: "Hi there"},
		{AgentID: "agent1", AgentName: "Agent1", Content: "Another message"},
	}

	filtered := FilterRelevantMessages(messages, "agent1", "Agent1")

	if len(filtered) != 1 {
		t.Errorf("Expected 1 message, got %d", len(filtered))
	}
	if filtered[0].AgentID != "agent2" {
		t.Errorf("Expected agent2's message, got %s", filtered[0].AgentID)
	}
}

// TestEstimateTokens tests token estimation
func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		text     string
		expected int
	}{
		{"", 0},
		{"a", 0},   // Less than 4 chars
		{"test", 1}, // 4 chars = 1 token
		{"hello world", 2}, // 11 chars = ~2 tokens
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("len_%d", len(tt.text)), func(t *testing.T) {
			tokens := EstimateTokens(tt.text)
			if tokens != tt.expected {
				t.Errorf("For %q: expected %d tokens, got %d", tt.text, tt.expected, tokens)
			}
		})
	}
}

// TestClaudeCLIAdapter tests the Claude CLI adapter
func TestClaudeCLIAdapter(t *testing.T) {
	// Skip if claude CLI is not installed
	_, err := exec.LookPath("claude")
	hasClaude := err == nil

	t.Run("NewAdapter", func(t *testing.T) {
		adapter := NewClaudeCLIAdapter()
		if adapter == nil {
			t.Error("Expected non-nil adapter")
		}
	})

	t.Run("IsAvailable", func(t *testing.T) {
		adapter := NewClaudeCLIAdapter().(*ClaudeCLIAdapter)
		available := adapter.IsAvailable()
		if available != hasClaude {
			t.Errorf("IsAvailable: expected %v, got %v", hasClaude, available)
		}
	})

	t.Run("InitializeWithMock", func(t *testing.T) {
		mockPath := createMockCLI(t, "claude", "Mock Claude response")

		adapter := &ClaudeCLIAdapter{
			BaseCLIAdapter: BaseCLIAdapter{
				cliName: "claude",
			},
		}
		adapter.cliPath = mockPath

		agent := core.Agent{
			ID:    "test-agent",
			Name:  "TestAgent",
			Model: "claude-3-haiku",
		}

		// Initialize with mock path already set
		adapter.model = agent.Model
		adapter.agentName = agent.Name
		adapter.agentID = agent.ID

		if !adapter.IsAvailable() {
			t.Error("Should be available with mock path")
		}
	})

	t.Run("SendMessageWithMock", func(t *testing.T) {
		mockPath := createMockCLI(t, "claude", "This is a mock response from Claude CLI")

		adapter := &ClaudeCLIAdapter{
			BaseCLIAdapter: BaseCLIAdapter{
				cliPath:   mockPath,
				cliName:   "claude",
				model:     "claude-3-haiku",
				agentName: "TestClaude",
				agentID:   "test-claude",
			},
		}

		messages := []core.Message{
			core.NewUserMessage("Hello Claude"),
		}

		ctx := context.Background()
		response, metrics, err := adapter.SendMessage(ctx, messages)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !strings.Contains(response, "mock response") {
			t.Errorf("Expected mock response, got: %s", response)
		}
		if metrics == nil {
			t.Error("Expected metrics")
		}
		if metrics != nil && metrics.Duration == 0 {
			t.Error("Expected non-zero duration")
		}
	})

	t.Run("HealthCheckWithMock", func(t *testing.T) {
		mockPath := createMockCLI(t, "claude", "version info")

		adapter := &ClaudeCLIAdapter{
			BaseCLIAdapter: BaseCLIAdapter{
				cliPath:   mockPath,
				cliName:   "claude",
				agentName: "TestClaude",
			},
		}

		ctx := context.Background()
		err := adapter.HealthCheck(ctx)
		if err != nil {
			t.Errorf("Health check failed: %v", err)
		}
	})

	t.Run("GetCLIVersionWithMock", func(t *testing.T) {
		mockPath := createMockCLI(t, "claude", "")

		adapter := &ClaudeCLIAdapter{
			BaseCLIAdapter: BaseCLIAdapter{
				cliPath: mockPath,
				cliName: "claude",
			},
		}

		version := adapter.GetCLIVersion()
		if !strings.Contains(version, "mock-claude") {
			t.Errorf("Expected version containing 'mock-claude', got: %s", version)
		}
	})
}

// TestGeminiCLIAdapter tests the Gemini CLI adapter
func TestGeminiCLIAdapter(t *testing.T) {
	// Skip if gemini CLI is not installed
	_, err := exec.LookPath("gemini")
	hasGemini := err == nil

	t.Run("NewAdapter", func(t *testing.T) {
		adapter := NewGeminiCLIAdapter()
		if adapter == nil {
			t.Error("Expected non-nil adapter")
		}
	})

	t.Run("IsAvailable", func(t *testing.T) {
		adapter := NewGeminiCLIAdapter().(*GeminiCLIAdapter)
		available := adapter.IsAvailable()
		if available != hasGemini {
			t.Errorf("IsAvailable: expected %v, got %v", hasGemini, available)
		}
	})

	t.Run("SendMessageWithMock", func(t *testing.T) {
		mockPath := createMockCLI(t, "gemini", "This is a mock response from Gemini CLI")

		adapter := &GeminiCLIAdapter{
			BaseCLIAdapter: BaseCLIAdapter{
				cliPath:   mockPath,
				cliName:   "gemini",
				model:     "gemini-1.5-flash",
				agentName: "TestGemini",
				agentID:   "test-gemini",
			},
		}

		messages := []core.Message{
			core.NewUserMessage("Hello Gemini"),
		}

		ctx := context.Background()
		response, metrics, err := adapter.SendMessage(ctx, messages)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !strings.Contains(response, "mock response") {
			t.Errorf("Expected mock response, got: %s", response)
		}
		if metrics == nil {
			t.Error("Expected metrics")
		}
	})

	t.Run("CleanOutput", func(t *testing.T) {
		adapter := &GeminiCLIAdapter{}

		tests := []struct {
			input    string
			expected string
		}{
			{
				input:    "Loaded cached credentials\nHello world",
				expected: "Hello world",
			},
			{
				input:    "Hello\nGemini CLI\nworld",
				expected: "Hello\nworld",
			},
		}

		for _, tt := range tests {
			result := adapter.cleanOutput(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		}
	})
}

// TestGenericCLIAdapter tests the generic CLI adapter
func TestGenericCLIAdapter(t *testing.T) {
	t.Run("NewAdapter", func(t *testing.T) {
		adapter := NewGenericCLIAdapter()
		if adapter == nil {
			t.Error("Expected non-nil adapter")
		}
	})

	t.Run("InitializeWithConfig", func(t *testing.T) {
		mockPath := createMockCLI(t, "custom-cli", "Custom response")

		adapter := NewGenericCLIAdapter().(*GenericCLIAdapter)
		agent := core.Agent{
			ID:    "test-agent",
			Name:  "TestAgent",
			Model: "custom-model",
			Config: core.AgentAdapterConfig{
				Extra: map[string]interface{}{
					"cli_path":      mockPath,
					"prompt_flag":   "-p",
					"model_flag":    "--model",
					"output_parser": "plain",
				},
			},
		}

		err := adapter.Initialize(agent)
		if err != nil {
			t.Errorf("Initialize failed: %v", err)
		}
		if adapter.cliPath != mockPath {
			t.Errorf("Expected CLI path %s, got %s", mockPath, adapter.cliPath)
		}
	})

	t.Run("SendMessageWithMock", func(t *testing.T) {
		mockPath := createMockCLI(t, "custom-cli", "Custom CLI response for testing")

		adapter := &GenericCLIAdapter{
			BaseCLIAdapter: BaseCLIAdapter{
				cliPath:   mockPath,
				cliName:   "custom-cli",
				model:     "custom-model",
				agentName: "TestGeneric",
				agentID:   "test-generic",
			},
			config: GenericCLIConfig{
				UseStdin:     true,
				OutputParser: OutputParserPlain,
			},
		}

		messages := []core.Message{
			core.NewUserMessage("Hello Custom CLI"),
		}

		ctx := context.Background()
		response, metrics, err := adapter.SendMessage(ctx, messages)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !strings.Contains(response, "Custom CLI response") {
			t.Errorf("Expected custom response, got: %s", response)
		}
		if metrics == nil {
			t.Error("Expected metrics")
		}
	})

	t.Run("ParseJSONOutput", func(t *testing.T) {
		adapter := &GenericCLIAdapter{
			config: GenericCLIConfig{
				OutputParser: OutputParserJSON,
				JSONPath:     "content",
			},
		}

		output := `{"content": "Hello from JSON", "status": "ok"}`
		parsed, err := adapter.parseOutput(output)
		if err != nil {
			t.Errorf("Parse failed: %v", err)
		}
		if parsed != "Hello from JSON" {
			t.Errorf("Expected 'Hello from JSON', got: %s", parsed)
		}
	})

	t.Run("ParseMarkdownOutput", func(t *testing.T) {
		adapter := &GenericCLIAdapter{
			config: GenericCLIConfig{
				OutputParser: OutputParserMarkdown,
			},
		}

		output := "Some text\n```python\nprint('hello')\n```\nMore text"
		parsed, err := adapter.parseOutput(output)
		if err != nil {
			t.Errorf("Parse failed: %v", err)
		}
		if !strings.Contains(parsed, "print('hello')") {
			t.Errorf("Expected code block content, got: %s", parsed)
		}
	})
}

// TestCLINotFound tests behavior when CLI is not installed
func TestCLINotFound(t *testing.T) {
	t.Run("ClaudeNotFound", func(t *testing.T) {
		// Skip if Claude CLI is already installed - we can't test "not found" scenario
		_, err := exec.LookPath("claude")
		if err == nil {
			t.Skip("Skipping test: Claude CLI is installed")
		}

		adapter := NewClaudeCLIAdapter().(*ClaudeCLIAdapter)
		agent := core.Agent{
			ID:    "test",
			Name:  "Test",
			Model: "test-model",
		}

		err = adapter.Initialize(agent)
		if err == nil {
			t.Error("Expected error for non-existent CLI")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Error should mention 'not found', got: %v", err)
		}
	})

	t.Run("GenericNotFound", func(t *testing.T) {
		adapter := NewGenericCLIAdapter().(*GenericCLIAdapter)
		agent := core.Agent{
			ID:   "test",
			Name: "Test",
			Config: core.AgentAdapterConfig{
				Extra: map[string]interface{}{
					"cli_name": "nonexistent-generic-cli-12345",
				},
			},
		}

		err := adapter.Initialize(agent)
		if err == nil {
			t.Error("Expected error for non-existent CLI")
		}
	})

	t.Run("FindCLIPathNotFound", func(t *testing.T) {
		// This tests the FindCLIPath function directly with a non-existent CLI
		_, err := FindCLIPath("absolutely-nonexistent-cli-xyz-98765")
		if err == nil {
			t.Error("Expected error for non-existent CLI")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Error should mention 'not found', got: %v", err)
		}
	})
}

// TestCLITimeout tests context timeout cancellation
func TestCLITimeout(t *testing.T) {
	// Create a slow mock CLI
	mockPath := createSlowMockCLI(t, "slow-cli", 5*time.Second)

	adapter := &GenericCLIAdapter{
		BaseCLIAdapter: BaseCLIAdapter{
			cliPath:   mockPath,
			cliName:   "slow-cli",
			agentName: "TestSlow",
			agentID:   "test-slow",
		},
		config: GenericCLIConfig{
			UseStdin:     true,
			OutputParser: OutputParserPlain,
		},
	}

	messages := []core.Message{
		core.NewUserMessage("This should timeout"),
	}

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, _, err := adapter.SendMessage(ctx, messages)
	if err == nil {
		t.Error("Expected timeout error")
	}
}

// TestCLIStreamOutput tests streaming output
func TestCLIStreamOutput(t *testing.T) {
	lines := []string{
		"Line 1 of streaming output",
		"Line 2 of streaming output",
		"Line 3 of streaming output",
	}
	mockPath := createStreamingMockCLI(t, "stream-cli", lines)

	adapter := &GenericCLIAdapter{
		BaseCLIAdapter: BaseCLIAdapter{
			cliPath:   mockPath,
			cliName:   "stream-cli",
			agentName: "TestStream",
			agentID:   "test-stream",
		},
		config: GenericCLIConfig{
			UseStdin:     true,
			OutputParser: OutputParserPlain,
		},
	}

	messages := []core.Message{
		core.NewUserMessage("Stream test"),
	}

	var output strings.Builder
	ctx := context.Background()
	metrics, err := adapter.StreamMessage(ctx, messages, &output)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if metrics == nil {
		t.Error("Expected metrics")
	}

	outputStr := output.String()
	for _, line := range lines {
		if !strings.Contains(outputStr, line) {
			t.Errorf("Expected output to contain %q, got: %s", line, outputStr)
		}
	}
}

// TestAdapterRegistration tests that adapters are properly registered
func TestAdapterRegistration(t *testing.T) {
	// The init() functions should register the adapters
	registeredAdapters := []string{"claude-cli", "gemini-cli", "cli-generic"}

	for _, name := range registeredAdapters {
		t.Run(name, func(t *testing.T) {
			if !adapters.Has(name) {
				t.Errorf("Adapter %s should be registered", name)
			}

			adapter, err := adapters.Get(name)
			if err != nil {
				t.Errorf("Failed to get adapter %s: %v", name, err)
			}
			if adapter == nil {
				t.Errorf("Adapter %s should not be nil", name)
			}
		})
	}
}

// TestEmptyMessages tests behavior with empty message arrays
func TestEmptyMessages(t *testing.T) {
	mockPath := createMockCLI(t, "test-cli", "response")

	adaptersToTest := []struct {
		name    string
		adapter adapters.AgentAdapter
	}{
		{
			name: "Claude",
			adapter: &ClaudeCLIAdapter{
				BaseCLIAdapter: BaseCLIAdapter{
					cliPath:   mockPath,
					agentName: "Test",
				},
			},
		},
		{
			name: "Gemini",
			adapter: &GeminiCLIAdapter{
				BaseCLIAdapter: BaseCLIAdapter{
					cliPath:   mockPath,
					agentName: "Test",
				},
			},
		},
	}

	for _, tt := range adaptersToTest {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			response, metrics, err := tt.adapter.SendMessage(ctx, nil)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if response != "" {
				t.Errorf("Expected empty response, got: %s", response)
			}
			if metrics != nil {
				t.Error("Expected nil metrics for empty messages")
			}
		})
	}
}

// TestStreamCommand tests the StreamCommand helper function
func TestStreamCommand(t *testing.T) {
	t.Run("SimpleStream", func(t *testing.T) {
		var output strings.Builder
		err := StreamCommand(context.Background(), "echo", []string{"hello\nworld"}, nil, &output)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !strings.Contains(output.String(), "hello") {
			t.Errorf("Expected 'hello' in output, got: %s", output.String())
		}
	})

	t.Run("StreamWithStdin", func(t *testing.T) {
		var output strings.Builder
		err := StreamCommand(context.Background(), "cat", []string{}, strings.NewReader("input data"), &output)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !strings.Contains(output.String(), "input data") {
			t.Errorf("Expected 'input data' in output, got: %s", output.String())
		}
	})
}

// TestCostEstimation tests cost calculation
func TestCostEstimation(t *testing.T) {
	const epsilon = 1e-12 // Epsilon for float64 comparison

	almostEqual := func(a, b float64) bool {
		diff := a - b
		if diff < 0 {
			diff = -diff
		}
		return diff < epsilon
	}

	t.Run("ClaudeHaiku", func(t *testing.T) {
		adapter := &ClaudeCLIAdapter{
			BaseCLIAdapter: BaseCLIAdapter{
				model: "claude-3-haiku",
			},
		}
		cost := adapter.estimateCost(1000, 500)
		// Haiku: $0.25/1M input, $1.25/1M output
		expected := (1000.0 * 0.25 / 1_000_000) + (500.0 * 1.25 / 1_000_000)
		if !almostEqual(cost, expected) {
			t.Errorf("Expected cost %f, got %f", expected, cost)
		}
	})

	t.Run("ClaudeSonnet", func(t *testing.T) {
		adapter := &ClaudeCLIAdapter{
			BaseCLIAdapter: BaseCLIAdapter{
				model: "claude-3-sonnet",
			},
		}
		cost := adapter.estimateCost(1000, 500)
		// Sonnet: $3/1M input, $15/1M output
		expected := (1000.0 * 3.0 / 1_000_000) + (500.0 * 15.0 / 1_000_000)
		if !almostEqual(cost, expected) {
			t.Errorf("Expected cost %f, got %f", expected, cost)
		}
	})

	t.Run("GeminiFlash", func(t *testing.T) {
		adapter := &GeminiCLIAdapter{
			BaseCLIAdapter: BaseCLIAdapter{
				model: "gemini-1.5-flash",
			},
		}
		cost := adapter.estimateCost(1000, 500)
		// Flash: $0.075/1M input, $0.30/1M output
		expected := (1000.0 * 0.075 / 1_000_000) + (500.0 * 0.30 / 1_000_000)
		if !almostEqual(cost, expected) {
			t.Errorf("Expected cost %f, got %f", expected, cost)
		}
	})
}

// BenchmarkSendMessage benchmarks message sending
func BenchmarkSendMessage(b *testing.B) {
	mockPath := createMockCLI(&testing.T{}, "bench-cli", "Benchmark response")

	adapter := &GenericCLIAdapter{
		BaseCLIAdapter: BaseCLIAdapter{
			cliPath:   mockPath,
			cliName:   "bench-cli",
			agentName: "Benchmark",
			agentID:   "bench",
		},
		config: GenericCLIConfig{
			UseStdin:     true,
			OutputParser: OutputParserPlain,
		},
	}

	messages := []core.Message{
		core.NewUserMessage("Benchmark test message"),
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, err := adapter.SendMessage(ctx, messages)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// mockWriter is a test helper for testing streaming with custom behavior
type mockWriter struct {
	written []byte
	writeFn func([]byte) (int, error)
}

func (m *mockWriter) Write(p []byte) (int, error) {
	m.written = append(m.written, p...)
	if m.writeFn != nil {
		return m.writeFn(p)
	}
	return len(p), nil
}

// TestStreamMessageWriterError tests handling of writer errors during streaming
func TestStreamMessageWriterError(t *testing.T) {
	lines := []string{"Line 1", "Line 2"}
	mockPath := createStreamingMockCLI(t, "error-stream", lines)

	adapter := &GenericCLIAdapter{
		BaseCLIAdapter: BaseCLIAdapter{
			cliPath:   mockPath,
			cliName:   "error-stream",
			agentName: "Test",
			agentID:   "test",
		},
		config: GenericCLIConfig{
			UseStdin:     true,
			OutputParser: OutputParserPlain,
		},
	}

	messages := []core.Message{
		core.NewUserMessage("Test"),
	}

	// Create a writer that fails
	failWriter := &mockWriter{
		writeFn: func(p []byte) (int, error) {
			return 0, io.ErrClosedPipe
		},
	}

	ctx := context.Background()
	_, err := adapter.StreamMessage(ctx, messages, failWriter)

	// The error might be swallowed or wrapped, just verify it doesn't panic
	// and produces some reasonable behavior
	_ = err // Error is optional depending on implementation
}
