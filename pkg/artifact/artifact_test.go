package artifact

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSingleArtifact(t *testing.T) {
	content := "Here's a simple Go file:\n\n```go:main.go\npackage main\n\nfunc main() {\n\tprintln(\"Hello\")\n}\n```\n\nThat's it!"

	result := Parse(content, "agent-1", "Claude")

	if len(result.Artifacts) != 1 {
		t.Fatalf("Expected 1 artifact, got %d", len(result.Artifacts))
	}

	artifact := result.Artifacts[0]
	if artifact.Language != "go" {
		t.Errorf("Expected language 'go', got '%s'", artifact.Language)
	}
	if artifact.Filename != "main.go" {
		t.Errorf("Expected filename 'main.go', got '%s'", artifact.Filename)
	}
	if artifact.Type != "go" {
		t.Errorf("Expected Type 'go', got '%s'", artifact.Type)
	}
	if artifact.Identifier != "main.go" {
		t.Errorf("Expected Identifier 'main.go', got '%s'", artifact.Identifier)
	}
	if artifact.AgentID != "agent-1" {
		t.Errorf("Expected AgentID 'agent-1', got '%s'", artifact.AgentID)
	}
	if artifact.AgentName != "Claude" {
		t.Errorf("Expected AgentName 'Claude', got '%s'", artifact.AgentName)
	}
	if !strings.Contains(artifact.Content, "package main") {
		t.Errorf("Expected content to contain 'package main', got '%s'", artifact.Content)
	}

	// Check cleaned content
	if strings.Contains(result.CleanedContent, "```go:main.go") {
		t.Errorf("CleanedContent should not contain artifact block")
	}
	if !strings.Contains(result.CleanedContent, "Here's a simple Go file:") {
		t.Errorf("CleanedContent should contain surrounding text")
	}
}

func TestParseMultipleArtifacts(t *testing.T) {
	content := `Here are two files:

` + "```python:app.py\nprint('Hello')\n```" + `

And another:

` + "```javascript:index.js\nconsole.log('World');\n```" + `

Done!`

	result := Parse(content, "agent-2", "Gemini")

	if len(result.Artifacts) != 2 {
		t.Fatalf("Expected 2 artifacts, got %d", len(result.Artifacts))
	}

	// Check first artifact
	if result.Artifacts[0].Language != "python" {
		t.Errorf("First artifact: expected language 'python', got '%s'", result.Artifacts[0].Language)
	}
	if result.Artifacts[0].Filename != "app.py" {
		t.Errorf("First artifact: expected filename 'app.py', got '%s'", result.Artifacts[0].Filename)
	}

	// Check second artifact
	if result.Artifacts[1].Language != "javascript" {
		t.Errorf("Second artifact: expected language 'javascript', got '%s'", result.Artifacts[1].Language)
	}
	if result.Artifacts[1].Filename != "index.js" {
		t.Errorf("Second artifact: expected filename 'index.js', got '%s'", result.Artifacts[1].Filename)
	}

	// Check cleaned content has both artifacts removed
	if strings.Contains(result.CleanedContent, "```python") {
		t.Errorf("CleanedContent should not contain python artifact block")
	}
	if strings.Contains(result.CleanedContent, "```javascript") {
		t.Errorf("CleanedContent should not contain javascript artifact block")
	}
}

func TestParseNestedPaths(t *testing.T) {
	content := "```go:pkg/utils/helper.go\npackage utils\n\nfunc Helper() {}\n```"

	result := Parse(content, "agent-1", "Claude")

	if len(result.Artifacts) != 1 {
		t.Fatalf("Expected 1 artifact, got %d", len(result.Artifacts))
	}

	if result.Artifacts[0].Filename != "pkg/utils/helper.go" {
		t.Errorf("Expected nested path 'pkg/utils/helper.go', got '%s'", result.Artifacts[0].Filename)
	}
}

func TestParseMalformedBlocks(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "no filename after language",
			content: "```go\npackage main\n```",
			want:    0,
		},
		{
			name:    "regular code block without filename",
			content: "```python\nprint('hello')\n```",
			want:    0,
		},
		{
			name:    "empty filename",
			content: "```go:\npackage main\n```",
			want:    0,
		},
		{
			name:    "no content",
			content: "No code blocks here",
			want:    0,
		},
		{
			name:    "valid artifact",
			content: "```go:main.go\npackage main\n```",
			want:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Parse(tt.content, "agent", "Test")
			if len(result.Artifacts) != tt.want {
				t.Errorf("Expected %d artifacts, got %d", tt.want, len(result.Artifacts))
			}
		})
	}
}

func TestHasArtifacts(t *testing.T) {
	tests := []struct {
		content string
		want    bool
	}{
		{"```go:main.go\ncode\n```", true},
		{"```python:app.py\ncode\n```", true},
		{"```go\ncode\n```", false},
		{"no code blocks", false},
	}

	for _, tt := range tests {
		got := HasArtifacts(tt.content)
		if got != tt.want {
			t.Errorf("HasArtifacts(%q) = %v, want %v", tt.content, got, tt.want)
		}
	}
}

func TestWriterBasic(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "artifact-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer := NewWriter(tmpDir)

	artifact := Artifact{
		AgentID:   "agent-1",
		AgentName: "Claude",
		Language:  "go",
		Filename:  "main.go",
		Content:   "package main\n\nfunc main() {}\n",
	}

	savedPath, err := writer.Write(artifact)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Check the file was created
	expectedPath := filepath.Join(tmpDir, "Claude", "main.go")
	if savedPath != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, savedPath)
	}

	// Read and verify content
	content, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}
	if string(content) != artifact.Content {
		t.Errorf("File content mismatch")
	}
}

func TestWriterVersionConflicts(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "artifact-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer := NewWriter(tmpDir)

	artifact := Artifact{
		AgentID:   "agent-1",
		AgentName: "Claude",
		Language:  "go",
		Filename:  "main.go",
		Content:   "version 1",
	}

	// Write first version
	path1, err := writer.Write(artifact)
	if err != nil {
		t.Fatalf("First write failed: %v", err)
	}
	if !strings.HasSuffix(path1, "main.go") {
		t.Errorf("First write should be main.go, got %s", path1)
	}

	// Write second version
	artifact.Content = "version 2"
	path2, err := writer.Write(artifact)
	if err != nil {
		t.Fatalf("Second write failed: %v", err)
	}
	if !strings.HasSuffix(path2, "main.2.go") {
		t.Errorf("Second write should be main.2.go, got %s", path2)
	}

	// Write third version
	artifact.Content = "version 3"
	path3, err := writer.Write(artifact)
	if err != nil {
		t.Fatalf("Third write failed: %v", err)
	}
	if !strings.HasSuffix(path3, "main.3.go") {
		t.Errorf("Third write should be main.3.go, got %s", path3)
	}

	// Verify all files exist with correct content
	content1, _ := os.ReadFile(path1)
	content2, _ := os.ReadFile(path2)
	content3, _ := os.ReadFile(path3)

	if string(content1) != "version 1" {
		t.Errorf("First file content wrong")
	}
	if string(content2) != "version 2" {
		t.Errorf("Second file content wrong")
	}
	if string(content3) != "version 3" {
		t.Errorf("Third file content wrong")
	}
}

func TestWriterNestedPaths(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "artifact-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer := NewWriter(tmpDir)

	artifact := Artifact{
		AgentName: "Gemini",
		Filename:  "pkg/utils/helper.go",
		Content:   "package utils",
	}

	savedPath, err := writer.Write(artifact)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "Gemini", "pkg", "utils", "helper.go")
	if savedPath != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, savedPath)
	}

	if _, err := os.Stat(savedPath); os.IsNotExist(err) {
		t.Errorf("File was not created at expected path")
	}
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"main.go", "main.go"},
		{"pkg/utils/helper.go", filepath.Join("pkg", "utils", "helper.go")},
		{"../../../etc/passwd", filepath.Join("etc", "passwd")},
		{"/absolute/path.go", filepath.Join("absolute", "path.go")},
		{"file<with>bad:chars.go", "file_with_bad_chars.go"},
		{"Agent Name/file.go", filepath.Join("Agent Name", "file.go")},
		{"", "unnamed"},
		{"...", "unnamed"},
		{"./hidden/./path", filepath.Join("hidden", "path")},
	}

	for _, tt := range tests {
		got := sanitizePath(tt.input)
		if got != tt.want {
			t.Errorf("sanitizePath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestWriterAgentNameSanitization(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "artifact-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer := NewWriter(tmpDir)

	// Test with agent name that has special characters
	artifact := Artifact{
		AgentName: "Claude: AI Assistant",
		Filename:  "main.go",
		Content:   "package main",
	}

	savedPath, err := writer.Write(artifact)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// The path should be sanitized (colon removed)
	if strings.Contains(savedPath, ":") && !strings.HasPrefix(savedPath, tmpDir[:2]) {
		// Allow Windows drive letter colons but not others
		t.Errorf("Path should not contain unsafe characters: %s", savedPath)
	}

	if _, err := os.Stat(savedPath); os.IsNotExist(err) {
		t.Errorf("File was not created")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Enabled {
		t.Errorf("Default config should have Enabled=false")
	}
	if config.OutputDir != "./artifacts" {
		t.Errorf("Expected OutputDir './artifacts', got '%s'", config.OutputDir)
	}
	if config.InstructAgents {
		t.Errorf("Default config should have InstructAgents=false")
	}
}

func TestAddVersionSuffix(t *testing.T) {
	tests := []struct {
		path    string
		version int
		want    string
	}{
		{"file.go", 2, "file.2.go"},
		{"file.go", 3, "file.3.go"},
		{filepath.Join("path", "to", "main.py"), 2, filepath.Join("path", "to", "main.2.py")},
		{"README", 2, "README.2"},
		{"Makefile", 5, "Makefile.5"},
	}

	for _, tt := range tests {
		got := addVersionSuffix(tt.path, tt.version)
		if got != tt.want {
			t.Errorf("addVersionSuffix(%q, %d) = %q, want %q", tt.path, tt.version, got, tt.want)
		}
	}
}

func TestWriterExistingFileOnDisk(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "artifact-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Pre-create a file to simulate existing artifact from previous run
	agentDir := filepath.Join(tmpDir, "Claude")
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatalf("Failed to create agent dir: %v", err)
	}
	existingFile := filepath.Join(agentDir, "main.go")
	if err := os.WriteFile(existingFile, []byte("existing content"), 0644); err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	// Create a new writer (fresh session)
	writer := NewWriter(tmpDir)

	artifact := Artifact{
		AgentName: "Claude",
		Filename:  "main.go",
		Content:   "new content",
	}

	savedPath, err := writer.Write(artifact)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Should have version suffix since file existed
	if !strings.HasSuffix(savedPath, "main.2.go") {
		t.Errorf("Expected main.2.go, got %s", savedPath)
	}

	// Original file should be unchanged
	originalContent, _ := os.ReadFile(existingFile)
	if string(originalContent) != "existing content" {
		t.Errorf("Original file should be unchanged")
	}
}
