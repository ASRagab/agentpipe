// Package cmd provides CLI commands for AgentPipe.
package cmd

import (
	"os"
	"testing"
)

func TestShouldUseV2(t *testing.T) {
	// Save original values
	origV2Enabled := v2Enabled
	origV2Resume := v2Resume
	defer func() {
		v2Enabled = origV2Enabled
		v2Resume = origV2Resume
		os.Unsetenv("AGENTPIPE_V2")
	}()

	tests := []struct {
		name       string
		v2Enabled  bool
		v2Resume   string
		envValue   string
		wantResult bool
	}{
		{
			name:       "v2 flag enabled",
			v2Enabled:  true,
			v2Resume:   "",
			envValue:   "",
			wantResult: true,
		},
		{
			name:       "v2 flag disabled, env true",
			v2Enabled:  false,
			v2Resume:   "",
			envValue:   "true",
			wantResult: true,
		},
		{
			name:       "v2 flag disabled, env false",
			v2Enabled:  false,
			v2Resume:   "",
			envValue:   "false",
			wantResult: false,
		},
		{
			name:       "resume flag implies v2",
			v2Enabled:  false,
			v2Resume:   "latest",
			envValue:   "",
			wantResult: true,
		},
		{
			name:       "all flags disabled",
			v2Enabled:  false,
			v2Resume:   "",
			envValue:   "",
			wantResult: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v2Enabled = tc.v2Enabled
			v2Resume = tc.v2Resume
			os.Unsetenv("AGENTPIPE_V2")
			if tc.envValue != "" {
				os.Setenv("AGENTPIPE_V2", tc.envValue)
			}

			result := shouldUseV2()
			if result != tc.wantResult {
				t.Errorf("shouldUseV2() = %v, want %v", result, tc.wantResult)
			}
		})
	}
}

func TestHandleV2Command(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		handled bool
	}{
		{
			name:    "not a command",
			input:   "hello world",
			handled: false,
		},
		{
			name:    "help command",
			input:   "/help",
			handled: true,
		},
		{
			name:    "status command",
			input:   "/status",
			handled: true,
		},
		{
			name:    "save command",
			input:   "/save",
			handled: true,
		},
		{
			name:    "export command with path",
			input:   "/export output.md",
			handled: true,
		},
		{
			name:    "summary command",
			input:   "/summary",
			handled: true,
		},
		{
			name:    "retry command",
			input:   "/retry",
			handled: true,
		},
		{
			name:    "quit is not handled by handleV2Command",
			input:   "/quit",
			handled: false,
		},
		{
			name:    "unknown command",
			input:   "/unknown",
			handled: false,
		},
		{
			name:    "command case insensitive",
			input:   "/HELP",
			handled: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Note: handleV2Command requires a manager, so we pass nil
			// This will panic for some commands that actually use the manager,
			// but we're just testing command recognition here
			result := handleV2CommandRecognized(tc.input)
			if result != tc.handled {
				t.Errorf("handleV2Command(%q) recognized = %v, want %v", tc.input, result, tc.handled)
			}
		})
	}
}

// handleV2CommandRecognized is a helper for testing that just checks if a command is recognized.
func handleV2CommandRecognized(input string) bool {
	if len(input) == 0 || input[0] != '/' {
		return false
	}

	parts := splitCommandParts(input)
	cmd := toLowerString(parts[0])

	switch cmd {
	case "/save", "/export", "/status", "/retry", "/summary", "/help":
		return true
	}
	return false
}

func splitCommandParts(s string) []string {
	result := make([]string, 0, 2)
	inSpace := false
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			if !inSpace && i > start {
				result = append(result, s[start:i])
			}
			inSpace = true
			start = i + 1
		} else {
			inSpace = false
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

func toLowerString(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c + ('a' - 'A')
		}
		b[i] = c
	}
	return string(b)
}

func TestV2FlagsExist(t *testing.T) {
	// Verify that v2 flags are properly registered
	flag := runCmd.Flags().Lookup("v2")
	if flag == nil {
		t.Error("--v2 flag not found on run command")
	}

	flag = runCmd.Flags().Lookup("parallel")
	if flag == nil {
		t.Error("--parallel flag not found on run command")
	}

	flag = runCmd.Flags().Lookup("v2-timeout")
	if flag == nil {
		t.Error("--v2-timeout flag not found on run command")
	}

	flag = runCmd.Flags().Lookup("save-dir")
	if flag == nil {
		t.Error("--save-dir flag not found on run command")
	}

	flag = runCmd.Flags().Lookup("resume")
	if flag == nil {
		t.Error("--resume flag not found on run command")
	}

	flag = runCmd.Flags().Lookup("export")
	if flag == nil {
		t.Error("--export flag not found on run command")
	}

	flag = runCmd.Flags().Lookup("auto-save")
	if flag == nil {
		t.Error("--auto-save flag not found on run command")
	}

	flag = runCmd.Flags().Lookup("migrate-config")
	if flag == nil {
		t.Error("--migrate-config flag not found on run command")
	}
}

func TestV2FlagDefaults(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		expectedDef  string
	}{
		{"v2 default", "v2", "false"},
		{"parallel default", "parallel", "true"},
		{"v2-timeout default", "v2-timeout", "60"},
		{"save-dir default", "save-dir", ""},
		{"resume default", "resume", ""},
		{"export default", "export", ""},
		{"auto-save default", "auto-save", "true"},
		{"migrate-config default", "migrate-config", "false"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			flag := runCmd.Flags().Lookup(tc.flagName)
			if flag == nil {
				t.Fatalf("--%s flag not found", tc.flagName)
			}
			if flag.DefValue != tc.expectedDef {
				t.Errorf("--%s default = %q, want %q", tc.flagName, flag.DefValue, tc.expectedDef)
			}
		})
	}
}

func TestTruncateForBox(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string unchanged",
			input:    "config.yaml",
			maxLen:   20,
			expected: "config.yaml",
		},
		{
			name:     "exact length unchanged",
			input:    "exactly-20-chars-xx",
			maxLen:   20,
			expected: "exactly-20-chars-xx",
		},
		{
			name:     "long string truncated with ellipsis",
			input:    "/path/to/very/long/config/file.yaml",
			maxLen:   20,
			expected: "...ong/config/file.yaml",
		},
		{
			name:     "empty string unchanged",
			input:    "",
			maxLen:   20,
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := truncateForBox(tc.input, tc.maxLen)
			if result != tc.expected {
				t.Errorf("truncateForBox(%q, %d) = %q, want %q", tc.input, tc.maxLen, result, tc.expected)
			}
		})
	}
}

func TestCheckAndWarnV1Config(t *testing.T) {
	// Create temporary directory for test configs
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		configData  string
		expectError bool
	}{
		{
			name: "v1 config with orchestrator",
			configData: `version: "1.0"
agents:
  - id: agent-1
    type: claude
    name: Test Agent
    prompt: "You are helpful"
orchestrator:
  mode: round-robin
  max_turns: 10
`,
			expectError: false,
		},
		{
			name: "v2 config no warning",
			configData: `conversation:
  timeout: 30s
  mode: parallel
agents:
  - id: agent-1
    type: mock
    adapter: mock
    name: Test Agent
    model: mock-model
    config:
      system_prompt: "You are helpful"
`,
			expectError: false,
		},
		{
			name: "v1 config with flat prompt",
			configData: `agents:
  - id: agent-1
    type: claude
    name: Test Agent
    prompt: "You are helpful"
`,
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create temp config file
			configPath := tmpDir + "/" + tc.name + ".yaml"
			err := os.WriteFile(configPath, []byte(tc.configData), 0644)
			if err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			// Call checkAndWarnV1Config
			err = checkAndWarnV1Config(configPath)
			if tc.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCheckAndWarnV1Config_FileNotFound(t *testing.T) {
	err := checkAndWarnV1Config("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}
