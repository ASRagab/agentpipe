# Artifact Collection Implementation Plan

**Branch:** `feature/artifact-collection`
**Date:** 2026-01-17

## Implementation Tasks

### Task 1: Create pkg/artifact/types.go

```go
package artifact

import "time"

// Artifact represents a file extracted from agent conversation
type Artifact struct {
    AgentID   string    `json:"agent_id"`
    AgentName string    `json:"agent_name"`
    Language  string    `json:"language"`   // From code fence (yaml, go, sql, etc.)
    Filename  string    `json:"filename"`   // Original path from code block
    Content   string    `json:"content"`
    Timestamp time.Time `json:"timestamp"`
    Version   int       `json:"version"`    // For conflict resolution
    SavedPath string    `json:"saved_path"` // Actual path where saved
}

// ExtractResult contains parsed artifacts and cleaned message
type ExtractResult struct {
    Artifacts      []Artifact
    CleanedContent string // Original content with artifact blocks removed (optional)
}

// Config holds artifact extraction configuration
type Config struct {
    Enabled        bool   `yaml:"enabled" json:"enabled"`
    OutputDir      string `yaml:"output_dir" json:"output_dir"`
    InstructAgents bool   `yaml:"instruct_agents" json:"instruct_agents"`
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
    return Config{
        Enabled:        true,
        OutputDir:      "./agentpipe-artifacts",
        InstructAgents: true,
    }
}
```

### Task 2: Create pkg/artifact/parser.go

```go
package artifact

import (
    "regexp"
    "strings"
    "time"
)

// Pattern: ```language:path/to/file.ext
// Captures: language, filename, content
var artifactPattern = regexp.MustCompile("(?s)```(\\w+):([^\\n]+)\\n(.*?)```")

// Parse extracts artifacts from message content
func Parse(content, agentID, agentName string) ExtractResult {
    var artifacts []Artifact

    matches := artifactPattern.FindAllStringSubmatch(content, -1)

    for _, match := range matches {
        if len(match) == 4 {
            artifacts = append(artifacts, Artifact{
                AgentID:   agentID,
                AgentName: agentName,
                Language:  match[1],
                Filename:  strings.TrimSpace(match[2]),
                Content:   match[3],
                Timestamp: time.Now(),
                Version:   1,
            })
        }
    }

    // Create cleaned content (artifact blocks replaced with placeholder)
    cleaned := artifactPattern.ReplaceAllString(content, "[Artifact: $2]")

    return ExtractResult{
        Artifacts:      artifacts,
        CleanedContent: cleaned,
    }
}
```

### Task 3: Create pkg/artifact/writer.go

```go
package artifact

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
)

// Writer handles saving artifacts to filesystem
type Writer struct {
    baseDir  string
    versions map[string]int // Track versions per agent/filename combo
}

// NewWriter creates a new artifact writer
func NewWriter(baseDir string) *Writer {
    return &Writer{
        baseDir:  baseDir,
        versions: make(map[string]int),
    }
}

// Write saves an artifact to the filesystem
// Returns the actual saved path
func (w *Writer) Write(a Artifact) (string, error) {
    // Sanitize agent name for directory (replace unsafe chars)
    safeAgentName := sanitizePath(a.AgentName)

    // Build path: baseDir/agentName/filename
    dir := filepath.Join(w.baseDir, safeAgentName, filepath.Dir(a.Filename))

    // Create directory if needed
    if err := os.MkdirAll(dir, 0755); err != nil {
        return "", fmt.Errorf("failed to create artifact directory: %w", err)
    }

    // Determine filename with version
    baseName := filepath.Base(a.Filename)
    versionKey := fmt.Sprintf("%s/%s", safeAgentName, a.Filename)

    var savePath string
    if version, exists := w.versions[versionKey]; exists {
        // Increment version
        w.versions[versionKey] = version + 1
        savePath = filepath.Join(dir, fmt.Sprintf("%s.%d", baseName, version+1))
    } else {
        // First version
        w.versions[versionKey] = 1
        savePath = filepath.Join(dir, baseName)
    }

    // Write file
    if err := os.WriteFile(savePath, []byte(a.Content), 0644); err != nil {
        return "", fmt.Errorf("failed to write artifact: %w", err)
    }

    return savePath, nil
}

// sanitizePath makes a string safe for use in file paths
func sanitizePath(s string) string {
    // Replace problematic characters
    replacer := strings.NewReplacer(
        "/", "-",
        "\\", "-",
        ":", "-",
        "*", "-",
        "?", "-",
        "\"", "-",
        "<", "-",
        ">", "-",
        "|", "-",
        " ", "_",
    )
    return replacer.Replace(s)
}
```

### Task 4: Create pkg/artifact/artifact_test.go

Test cases:

- Parse single artifact from content
- Parse multiple artifacts from content
- Parse nested path artifacts
- Handle malformed code blocks (no filename)
- Handle empty content
- Version conflict resolution
- Directory creation
- Agent name sanitization

### Task 5: Integrate into Orchestrator

In `pkg/orchestrator/orchestrator.go`:

1. Add artifact writer field:

```go
type Orchestrator struct {
    // ... existing fields
    artifactWriter  *artifact.Writer
    artifactConfig  artifact.Config
}
```

1. Initialize in constructor or config method:

```go
func (o *Orchestrator) SetArtifactConfig(cfg artifact.Config) {
    o.artifactConfig = cfg
    if cfg.Enabled {
        o.artifactWriter = artifact.NewWriter(cfg.OutputDir)
    }
}
```

1. After agent response, extract and save:

```go
// In processAgentResponse or similar
if o.artifactWriter != nil {
    result := artifact.Parse(response, agent.ID, agent.Name)
    for _, a := range result.Artifacts {
        savedPath, err := o.artifactWriter.Write(a)
        if err != nil {
            // Log error but don't fail conversation
            log.Error("failed to save artifact", "error", err)
        } else {
            // Emit artifact saved event for TUI
            o.emitArtifactSaved(a, savedPath)
        }
    }
}
```

### Task 6: Add TUI Notifications

In `pkg/tui/enhanced.go`:

1. Add new message type for artifacts:

```go
type artifactSaved struct {
    agentName string
    filename  string
    savedPath string
}
```

1. Handle in Update():

```go
case artifactSaved:
    // Add system message showing artifact save
    sysMsg := agent.Message{
        AgentID:   "system",
        AgentName: "System",
        Role:      "system",
        Content:   fmt.Sprintf("[Artifact saved: %s/%s]", msg.agentName, msg.filename),
        Timestamp: time.Now().Unix(),
    }
    m.messages = append(m.messages, sysMsg)
```

### Task 7: Update Agent Prompts

In each adapter's `buildPrompt()` method, add:

```go
if o.artifactConfig.InstructAgents {
    prompt.WriteString("\nARTIFACT CREATION:\n")
    prompt.WriteString("To create a saveable artifact, use fenced code blocks with a filename:\n")
    prompt.WriteString("  ```language:path/to/filename.ext\n")
    prompt.WriteString("  content here\n")
    prompt.WriteString("  ```\n")
    prompt.WriteString("Artifacts will be saved to the workspace automatically.\n\n")
}
```

### Task 8: Add CLI Flags

In `cmd/run.go`:

```go
runCmd.Flags().String("output-dir", "./agentpipe-artifacts", "Directory for saving artifacts")
runCmd.Flags().Bool("no-artifacts", false, "Disable artifact extraction")
```

Wire up to orchestrator config.

### Task 9: Add Config File Support

In `pkg/config/config.go`, add:

```go
type Config struct {
    // ... existing fields
    Artifacts artifact.Config `yaml:"artifacts" json:"artifacts"`
}
```

## Execution Order

1. **Phase 1 (Core):** Tasks 1-4 (types, parser, writer, tests)
2. **Phase 2 (Integration):** Task 5 (orchestrator)
3. **Phase 3 (UX):** Tasks 6-8 (TUI, prompts, CLI)
4. **Phase 4 (Config):** Task 9 (config file)

## Testing Checklist

- [ ] Unit tests for parser pass
- [ ] Unit tests for writer pass
- [ ] Integration test with mock agent
- [ ] Manual test with real conversation
- [ ] Verify TUI shows artifact notifications
- [ ] Verify --output-dir flag works
- [ ] Verify --no-artifacts flag works
- [ ] Verify config file artifacts section works
