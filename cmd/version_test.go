package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/kevinelliott/agentpipe/internal/version"
)

func TestPrintRunningModeIndicator(t *testing.T) {
	// Save and restore the environment variable
	original := os.Getenv("AGENTPIPE_V2")
	defer func() {
		if original != "" {
			os.Setenv("AGENTPIPE_V2", original)
		} else {
			os.Unsetenv("AGENTPIPE_V2")
		}
	}()

	tests := []struct {
		name     string
		envValue string
		expected string
	}{
		{
			name:     "v1 mode when AGENTPIPE_V2 not set",
			envValue: "",
			expected: "v1",
		},
		{
			name:     "v1 mode when AGENTPIPE_V2 is false",
			envValue: "false",
			expected: "v1",
		},
		{
			name:     "v2 mode when AGENTPIPE_V2 is true",
			envValue: "true",
			expected: "v2",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envValue != "" {
				os.Setenv("AGENTPIPE_V2", tc.envValue)
			} else {
				os.Unsetenv("AGENTPIPE_V2")
			}

			// Capture output
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printRunningModeIndicator()

			w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if !strings.Contains(output, tc.expected) {
				t.Errorf("Expected output to contain %q, got: %s", tc.expected, output)
			}
		})
	}
}

func TestGetV2EngineInfo(t *testing.T) {
	v2Version, features := version.GetV2EngineInfo()

	// Test version is set
	if v2Version == "" {
		t.Error("v2 engine version should not be empty")
	}

	// Test features list is not empty
	if len(features) == 0 {
		t.Error("v2 engine features should not be empty")
	}

	// Test that expected features are present
	expectedFeatures := []string{
		"parallel-execution",
		"graceful-degradation",
		"circuit-breaker",
	}

	for _, expected := range expectedFeatures {
		found := false
		for _, feature := range features {
			if feature == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected feature %q not found in features list", expected)
		}
	}
}

func TestGetV2VersionString(t *testing.T) {
	versionStr := version.GetV2VersionString()

	if versionStr == "" {
		t.Error("v2 version string should not be empty")
	}

	if !strings.Contains(versionStr, "v2 Engine") {
		t.Errorf("v2 version string should contain 'v2 Engine', got: %s", versionStr)
	}
}

func TestGetAdapterType(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"claude-cli", "CLI"},
		{"gemini-cli", "CLI"},
		{"openrouter", "API"},
		{"claude-api", "API"},
		{"something-else", "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := getAdapterType(tc.name)
			if result != tc.expected {
				t.Errorf("getAdapterType(%q) = %q, want %q", tc.name, result, tc.expected)
			}
		})
	}
}

func TestVersionCommandFlags(t *testing.T) {
	// Test that all expected flags are registered
	expectedFlags := []string{
		"check-update",
		"adapters",
		"v2",
		"all",
	}

	for _, flag := range expectedFlags {
		if versionCmd.Flags().Lookup(flag) == nil {
			t.Errorf("Expected flag %q not found in version command", flag)
		}
	}
}

func TestVersionCommandDescription(t *testing.T) {
	if versionCmd.Short == "" {
		t.Error("Version command should have a short description")
	}

	if versionCmd.Long == "" {
		t.Error("Version command should have a long description")
	}

	// Check that long description mentions v2 engine
	if !strings.Contains(versionCmd.Long, "v2") {
		t.Error("Version command long description should mention v2 engine")
	}
}
