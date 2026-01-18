// Package cmd provides CLI commands for AgentPipe.
package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ASRagab/agentpipe/pkg/v2/config"
	"github.com/ASRagab/agentpipe/pkg/v2/core"
	"github.com/ASRagab/agentpipe/pkg/v2/events"
	"github.com/ASRagab/agentpipe/pkg/v2/manager"
	"github.com/ASRagab/agentpipe/pkg/v2/persistence"

	// Import mock adapter to register it
	_ "github.com/ASRagab/agentpipe/pkg/v2/adapters/mock"
)

// TestRunV2Basic tests running the v2 engine with a mock configuration.
// It verifies that agents respond and output is generated.
func TestRunV2Basic(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a mock v2 configuration file
	configContent := `
conversation:
  timeout: 30s
  mode: parallel

agents:
  - id: test-agent-1
    type: mock
    adapter: mock
    name: "Test Agent 1"
    model: mock-model
    config:
      system_prompt: "You are a helpful test agent"

  - id: test-agent-2
    type: mock
    adapter: mock
    name: "Test Agent 2"
    model: mock-model
    config:
      system_prompt: "You are another helpful test agent"

persistence:
  auto_save: false
`
	configPath := filepath.Join(tmpDir, "test-config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Load the config
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify config loaded correctly
	if len(cfg.Agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(cfg.Agents))
	}

	// Initialize agents
	agents, err := cfg.InitializeAgents()
	if err != nil {
		t.Fatalf("failed to initialize agents: %v", err)
	}

	if len(agents) != 2 {
		t.Errorf("expected 2 agents initialized, got %d", len(agents))
	}

	// Create event bus
	eventBus := events.NewBus()
	defer eventBus.Close()

	// Track events with atomic counter for race-safe counting
	var messageCreatedCount atomic.Int32
	eventBus.Subscribe(core.EventMessageCreated, func(event core.Event) {
		messageCreatedCount.Add(1)
	})

	// Create manager
	managerCfg := manager.Config{
		Timeout: cfg.Conversation.Timeout,
	}

	mgr, err := manager.NewConversationManager(managerCfg, agents, eventBus)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer mgr.Close()

	// Start conversation
	mgr.Start()

	// Send a test message
	ctx := context.Background()
	responses, err := mgr.SendUserMessage(ctx, "Hello, test!")
	if err != nil {
		t.Fatalf("failed to send user message: %v", err)
	}

	// Verify we got responses from both agents
	if len(responses) != 2 {
		t.Errorf("expected 2 responses, got %d", len(responses))
	}

	// Verify responses have content
	for _, msg := range responses {
		if msg.Content == "" {
			t.Errorf("expected non-empty response from %s", msg.AgentName)
		}
		if msg.Role != core.RoleAgent {
			t.Errorf("expected agent role, got %s", msg.Role)
		}
	}

	// Complete conversation
	mgr.Complete()

	// Wait for events to be processed
	time.Sleep(50 * time.Millisecond)

	// Verify events were fired (user message + 2 agent messages = 3)
	if messageCreatedCount.Load() < 3 {
		t.Errorf("expected at least 3 message created events, got %d", messageCreatedCount.Load())
	}

	// Verify conversation state
	conv := mgr.GetConversation()
	if conv.Status != core.ConversationStatusCompleted {
		t.Errorf("expected completed status, got %s", conv.Status)
	}
	if len(conv.Messages) < 3 {
		t.Errorf("expected at least 3 messages (1 user + 2 agent), got %d", len(conv.Messages))
	}
}

// TestRunV2Headless tests running v2 in headless mode with piped input.
// Simulates: echo "question" | agentpipe run --v2
func TestRunV2Headless(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create mock config
	configContent := `
conversation:
  timeout: 10s
  mode: parallel

agents:
  - id: headless-agent
    type: mock
    adapter: mock
    name: "Headless Agent"
    model: mock-model
    config:
      system_prompt: "You respond to piped input"

persistence:
  auto_save: false
`
	configPath := filepath.Join(tmpDir, "headless-config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Load config
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Initialize agents
	agents, err := cfg.InitializeAgents()
	if err != nil {
		t.Fatalf("failed to initialize agents: %v", err)
	}

	// Create manager
	mgr, err := manager.NewConversationManager(manager.DefaultConfig(), agents, nil)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer mgr.Close()

	// Simulate piped input
	pipedInput := "What is the meaning of life?"
	inputReader := strings.NewReader(pipedInput)

	// Read the input (simulating what runV2PipedMode does)
	input, err := io.ReadAll(inputReader)
	if err != nil {
		t.Fatalf("failed to read piped input: %v", err)
	}

	content := strings.TrimSpace(string(input))
	if content == "" {
		t.Fatal("expected non-empty input")
	}

	// Send message and get responses
	ctx := context.Background()
	responses, err := mgr.SendUserMessage(ctx, content)
	if err != nil {
		t.Fatalf("failed to send message: %v", err)
	}

	// Verify we got a response
	if len(responses) != 1 {
		t.Errorf("expected 1 response from single agent, got %d", len(responses))
	}

	// Verify response content
	if len(responses) > 0 && responses[0].Content == "" {
		t.Error("expected non-empty response content")
	}

	// Complete and verify
	mgr.Complete()
	summary := mgr.Summary()

	if summary.MessageCount < 2 {
		t.Errorf("expected at least 2 messages (1 user + 1 agent), got %d", summary.MessageCount)
	}
}

// TestRunV2Resume tests saving and resuming a conversation.
func TestRunV2Resume(t *testing.T) {
	// Create temporary directory for saves
	tmpDir := t.TempDir()
	saveDir := filepath.Join(tmpDir, "conversations")

	// Create mock config - auto_save disabled to avoid race condition during test
	configContent := `
conversation:
  timeout: 10s
  mode: parallel

agents:
  - id: resume-agent
    type: mock
    adapter: mock
    name: "Resume Agent"
    model: mock-model
    config:
      system_prompt: "You are a persistent agent"

persistence:
  auto_save: false
`
	configPath := filepath.Join(tmpDir, "resume-config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Load config
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	cfg.Persistence.SaveDir = saveDir

	// Initialize agents
	agents, err := cfg.InitializeAgents()
	if err != nil {
		t.Fatalf("failed to initialize agents: %v", err)
	}

	// Create first conversation manager - Persistence.Enabled=false to avoid auto-save race conditions
	// We'll use manual Save() to test persistence explicitly
	managerCfg := manager.Config{
		Timeout: cfg.Conversation.Timeout,
		SaveDir: saveDir,
		Persistence: manager.PersistenceConfig{
			Enabled: false,
			SaveDir: saveDir,
		},
	}

	mgr1, err := manager.NewConversationManager(managerCfg, agents, nil)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// Start first conversation
	mgr1.Start()
	ctx := context.Background()

	// Send first message
	_, err = mgr1.SendUserMessage(ctx, "Hello, this is message 1")
	if err != nil {
		t.Fatalf("failed to send message 1: %v", err)
	}

	// Send second message
	_, err = mgr1.SendUserMessage(ctx, "This is message 2")
	if err != nil {
		t.Fatalf("failed to send message 2: %v", err)
	}

	// Get the conversation ID before saving
	conv1 := mgr1.GetConversation()
	conv1ID := conv1.ID

	// Save conversation
	savePath, err := mgr1.Save()
	if err != nil {
		t.Fatalf("failed to save conversation: %v", err)
	}

	if savePath == "" {
		t.Error("expected non-empty save path")
	}

	// Verify file exists
	if _, err := os.Stat(savePath); os.IsNotExist(err) {
		t.Errorf("save file does not exist: %s", savePath)
	}

	// Close first manager
	mgr1.Close()

	// Create new manager and resume
	mgr2, err := manager.NewConversationManager(managerCfg, agents, nil)
	if err != nil {
		t.Fatalf("failed to create second manager: %v", err)
	}
	defer mgr2.Close()

	// Load the saved conversation
	loadedConv, err := persistence.LoadConversation(savePath)
	if err != nil {
		t.Fatalf("failed to load conversation: %v", err)
	}

	// Verify conversation ID matches
	if loadedConv.ID != conv1ID {
		t.Errorf("conversation ID mismatch: expected %s, got %s", conv1ID, loadedConv.ID)
	}

	// Verify messages were preserved
	if len(loadedConv.Messages) < 4 {
		t.Errorf("expected at least 4 messages (2 user + 2 agent), got %d", len(loadedConv.Messages))
	}

	// Verify message content
	foundFirstMessage := false
	foundSecondMessage := false
	for _, msg := range loadedConv.Messages {
		if msg.Content == "Hello, this is message 1" {
			foundFirstMessage = true
		}
		if msg.Content == "This is message 2" {
			foundSecondMessage = true
		}
	}

	if !foundFirstMessage {
		t.Error("first user message not found in loaded conversation")
	}
	if !foundSecondMessage {
		t.Error("second user message not found in loaded conversation")
	}
}

// TestRunV2MigrationWarning tests that v1 configurations show deprecation warnings.
func TestRunV2MigrationWarning(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create a v1-style configuration
	v1ConfigContent := `
version: "1.0"

agents:
  - id: agent-1
    type: claude
    name: Claude Assistant
    prompt: "You are a helpful assistant"

orchestrator:
  mode: round-robin
  max_turns: 10
  turn_timeout: 30s
`
	v1ConfigPath := filepath.Join(tmpDir, "v1-config.yaml")
	if err := os.WriteFile(v1ConfigPath, []byte(v1ConfigContent), 0644); err != nil {
		t.Fatalf("failed to write v1 config: %v", err)
	}

	// Read the config file to detect v1 format
	data, err := os.ReadFile(v1ConfigPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	// Test v1 detection
	isV1, err := config.DetectV1Config(data)
	if err != nil {
		t.Fatalf("failed to detect v1 config: %v", err)
	}

	if !isV1 {
		t.Error("expected v1 config to be detected as v1 format")
	}

	// Test checkAndWarnV1Config function captures and returns without error
	// Capture stderr to verify warning is printed
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err = checkAndWarnV1Config(v1ConfigPath)
	if err != nil {
		t.Errorf("checkAndWarnV1Config returned error: %v", err)
	}

	// Restore stderr and read captured output
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// Verify warning message was printed
	if !strings.Contains(output, "DEPRECATION WARNING") {
		t.Error("expected deprecation warning in output")
	}
	if !strings.Contains(output, "v1 configuration format detected") {
		t.Error("expected v1 format detection message")
	}
	if !strings.Contains(output, "--migrate-config") {
		t.Error("expected migration command suggestion in output")
	}
}

// TestDoctorV2Checks tests that v2 doctor health checks work correctly.
func TestDoctorV2Checks(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create a valid v2 config with mock adapter (which should be available)
	configContent := `
conversation:
  timeout: 30s
  mode: parallel

agents:
  - id: mock-agent
    type: mock
    adapter: mock
    name: "Mock Agent"
    model: mock-model
    config:
      system_prompt: "You are a test agent"
`
	configPath := filepath.Join(tmpDir, "doctor-config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Save and restore doctorConfig global
	origConfig := doctorConfig
	defer func() {
		doctorConfig = origConfig
	}()

	doctorConfig = configPath

	// Run v2 checks
	output := performV2Checks()

	// Verify registered adapters exist
	if len(output.RegisteredAdapters) == 0 {
		t.Error("expected registered adapters")
	}

	// Check that mock adapter is registered
	foundMock := false
	for _, adapter := range output.RegisteredAdapters {
		if adapter == "mock" {
			foundMock = true
			break
		}
	}
	if !foundMock {
		t.Errorf("expected mock adapter in registered adapters: %v", output.RegisteredAdapters)
	}

	// Verify config was validated
	if output.ConfigFile != configPath {
		t.Errorf("expected config file %q, got %q", configPath, output.ConfigFile)
	}
	if !output.ConfigValid {
		t.Errorf("expected valid config, got error: %s", output.ConfigError)
	}

	// Verify agent check was performed
	if output.TotalAgents != 1 {
		t.Errorf("expected 1 total agent, got %d", output.TotalAgents)
	}

	// Check that agent check was performed
	if len(output.AgentChecks) != 1 {
		t.Errorf("expected 1 agent check, got %d", len(output.AgentChecks))
	}

	// Verify the mock agent check
	if len(output.AgentChecks) > 0 {
		check := output.AgentChecks[0]
		if check.ID != "mock-agent" {
			t.Errorf("expected agent ID 'mock-agent', got %q", check.ID)
		}
		if check.Type != "mock" {
			t.Errorf("expected agent type 'mock', got %q", check.Type)
		}
		if check.Adapter != "mock" {
			t.Errorf("expected adapter 'mock', got %q", check.Adapter)
		}
		if !check.Available {
			t.Error("expected mock adapter to be available")
		}
	}
}

// TestRunV2CommandRecognition tests that v2 special commands are properly recognized.
func TestRunV2CommandRecognition(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		isCommand bool
		isExitCmd bool
	}{
		{"normal text", "Hello world", false, false},
		{"save command", "/save", true, false},
		{"export command", "/export", true, false},
		{"export with path", "/export output.md", true, false},
		{"status command", "/status", true, false},
		{"retry command", "/retry", true, false},
		{"summary command", "/summary", true, false},
		{"help command", "/help", true, false},
		{"quit command", "/quit", false, true},
		{"exit command", "/exit", false, true},
		{"q shorthand", "/q", false, true},
		{"unknown command", "/unknown", false, false},
		{"uppercase command", "/HELP", true, false},
		{"mixed case", "/Status", true, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Check if it's a recognized command (using the helper)
			isRecognized := handleCommandRecognized(tc.input)
			if isRecognized != tc.isCommand {
				t.Errorf("handleCommandRecognized(%q) = %v, want %v", tc.input, isRecognized, tc.isCommand)
			}

			// Check exit commands separately
			content := strings.TrimSpace(tc.input)
			isExit := content == "/quit" || content == "/exit" || content == "/q"
			if isExit != tc.isExitCmd {
				t.Errorf("exit command check for %q = %v, want %v", tc.input, isExit, tc.isExitCmd)
			}
		})
	}
}

// TestRunV2MultiAgentParallel tests parallel execution with multiple agents.
func TestRunV2MultiAgentParallel(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create config with 3 agents
	configContent := `
conversation:
  timeout: 30s
  mode: parallel

agents:
  - id: agent-a
    type: mock
    adapter: mock
    name: "Agent A"
    model: mock-model
    config:
      system_prompt: "You are Agent A"

  - id: agent-b
    type: mock
    adapter: mock
    name: "Agent B"
    model: mock-model
    config:
      system_prompt: "You are Agent B"

  - id: agent-c
    type: mock
    adapter: mock
    name: "Agent C"
    model: mock-model
    config:
      system_prompt: "You are Agent C"

persistence:
  auto_save: false
`
	configPath := filepath.Join(tmpDir, "parallel-config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Load config
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Initialize agents
	agents, err := cfg.InitializeAgents()
	if err != nil {
		t.Fatalf("failed to initialize agents: %v", err)
	}

	if len(agents) != 3 {
		t.Fatalf("expected 3 agents, got %d", len(agents))
	}

	// Create event bus to track events
	eventBus := events.NewBus()
	defer eventBus.Close()

	// Use atomic counters for race-safe counting
	var agentTypingCount, agentDoneCount atomic.Int32
	eventBus.Subscribe(core.EventAgentTyping, func(event core.Event) {
		agentTypingCount.Add(1)
	})
	eventBus.Subscribe(core.EventAgentDone, func(event core.Event) {
		agentDoneCount.Add(1)
	})

	// Create manager
	mgr, err := manager.NewConversationManager(manager.DefaultConfig(), agents, eventBus)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer mgr.Close()

	// Start and send message
	mgr.Start()
	ctx := context.Background()
	responses, err := mgr.SendUserMessage(ctx, "Hello to all agents!")
	if err != nil {
		t.Fatalf("failed to send message: %v", err)
	}

	// All 3 agents should respond
	if len(responses) != 3 {
		t.Errorf("expected 3 responses from parallel agents, got %d", len(responses))
	}

	// Wait for events
	time.Sleep(100 * time.Millisecond)

	// Verify all agents sent typing and done events
	if agentTypingCount.Load() != 3 {
		t.Errorf("expected 3 AgentTyping events, got %d", agentTypingCount.Load())
	}
	if agentDoneCount.Load() != 3 {
		t.Errorf("expected 3 AgentDone events, got %d", agentDoneCount.Load())
	}

	// Verify different agents responded
	agentNames := make(map[string]bool)
	for _, resp := range responses {
		agentNames[resp.AgentName] = true
	}

	if len(agentNames) != 3 {
		t.Errorf("expected responses from 3 different agents, got %d", len(agentNames))
	}
}

// TestRunV2ConversationSummary tests the summary generation on conversation end.
func TestRunV2ConversationSummary(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create mock config
	configContent := `
conversation:
  timeout: 10s
  mode: parallel

agents:
  - id: summary-agent
    type: mock
    adapter: mock
    name: "Summary Agent"
    model: mock-model
    config:
      system_prompt: "You are a test agent"

persistence:
  auto_save: false
`
	configPath := filepath.Join(tmpDir, "summary-config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Load config
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Initialize agents
	agents, err := cfg.InitializeAgents()
	if err != nil {
		t.Fatalf("failed to initialize agents: %v", err)
	}

	// Create manager
	mgr, err := manager.NewConversationManager(manager.DefaultConfig(), agents, nil)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	defer mgr.Close()

	// Send multiple messages
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		_, err = mgr.SendUserMessage(ctx, "Test message")
		if err != nil {
			t.Fatalf("failed to send message %d: %v", i, err)
		}
	}

	// Complete conversation
	mgr.Complete()

	// Get summary
	summary := mgr.Summary()

	// Verify summary fields
	if summary.MessageCount != 6 { // 3 user + 3 agent
		t.Errorf("expected 6 messages in summary, got %d", summary.MessageCount)
	}

	// Verify duration is positive
	if summary.Duration <= 0 {
		t.Error("expected positive duration in summary")
	}
}
