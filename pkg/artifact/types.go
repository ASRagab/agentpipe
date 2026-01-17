// Package artifact provides functionality for extracting and saving file artifacts
// from agent conversation messages. Agents can create files during conversation
// using fenced code blocks with filenames in the format ```language:path/to/file.ext
package artifact

import (
	"time"
)

// Artifact represents a file artifact extracted from an agent's message.
// Artifacts are created when agents use fenced code blocks with filename syntax.
type Artifact struct {
	// AgentID is the unique identifier of the agent that created the artifact
	AgentID string
	// AgentName is the display name of the agent that created the artifact
	AgentName string
	// Type is the artifact type derived from the language (e.g., "go", "python")
	Type string
	// Identifier is the unique identifier for this artifact (typically the filename)
	Identifier string
	// Language is the programming language or file type from the code fence
	Language string
	// Filename is the path/filename specified after the language
	Filename string
	// Content is the actual file content from within the code block
	Content string
	// Timestamp is when the artifact was extracted
	Timestamp time.Time
	// Version is the version number if multiple artifacts have the same filename
	Version int
	// SavedPath is the absolute path where the artifact was saved (set after writing)
	SavedPath string
}

// ExtractResult contains the results of parsing a message for artifacts.
type ExtractResult struct {
	// Artifacts is the list of artifacts found in the message
	Artifacts []Artifact
	// CleanedContent is the message content with artifact blocks removed
	CleanedContent string
}

// Config defines the configuration for artifact collection.
type Config struct {
	// Enabled determines whether artifact collection is active
	Enabled bool `yaml:"enabled"`
	// OutputDir is the base directory for saving artifacts
	OutputDir string `yaml:"output_dir"`
	// InstructAgents determines whether to add artifact instructions to agent prompts
	InstructAgents bool `yaml:"instruct_agents"`
	// AllowedTypes is an optional list of artifact types to accept (empty means all)
	AllowedTypes []string `yaml:"allowed_types"`
	// MaxSizeBytes is the maximum artifact size in bytes (0 means no limit)
	MaxSizeBytes int64 `yaml:"max_size_bytes"`
}

// DefaultConfig returns the default artifact configuration.
// By default, artifacts are disabled and would be saved to ./artifacts/
func DefaultConfig() Config {
	return Config{
		Enabled:        false,
		OutputDir:      "./artifacts",
		InstructAgents: false,
	}
}
