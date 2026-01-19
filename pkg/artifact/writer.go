package artifact

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// Writer handles saving artifacts to the filesystem.
// It tracks file versions to handle conflicts when multiple artifacts
// have the same filename.
type Writer struct {
	baseDir  string
	versions map[string]int
	mu       sync.Mutex
}

// NewWriter creates a new artifact writer with the specified base directory.
func NewWriter(baseDir string) *Writer {
	return &Writer{
		baseDir:  baseDir,
		versions: make(map[string]int),
	}
}

// Write saves an artifact to the filesystem.
// The artifact is saved to baseDir/agentName/filename.
// If a file with the same name already exists, a version suffix is added
// (e.g., file.go -> file.2.go -> file.3.go).
// Returns the full path where the file was saved.
func (w *Writer) Write(a Artifact) (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Sanitize the agent name and filename for safe filesystem paths
	safeAgentName := sanitizePath(a.AgentName)
	safeFilename := sanitizePath(a.Filename)

	// Build the target directory path
	targetDir := filepath.Join(w.baseDir, safeAgentName, filepath.Dir(safeFilename))

	// Create the directory structure
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create artifact directory: %w", err)
	}

	// Build the full file path
	baseName := filepath.Base(safeFilename)
	fullPath := filepath.Join(targetDir, baseName)

	// Check for version conflicts
	versionKey := filepath.Join(safeAgentName, safeFilename)
	if existingVersion, exists := w.versions[versionKey]; exists {
		// File was already written in this session, increment version
		newVersion := existingVersion + 1
		fullPath = addVersionSuffix(fullPath, newVersion)
		w.versions[versionKey] = newVersion
	} else if _, err := os.Stat(fullPath); err == nil {
		// File exists on disk from a previous run
		// Find the next available version
		version := 2
		for {
			versionedPath := addVersionSuffix(fullPath, version)
			if _, err := os.Stat(versionedPath); os.IsNotExist(err) {
				fullPath = versionedPath
				break
			}
			version++
		}
		w.versions[versionKey] = version
	} else {
		// First time writing this file
		w.versions[versionKey] = 1
	}

	// Write the file
	if err := os.WriteFile(fullPath, []byte(a.Content), 0644); err != nil {
		return "", fmt.Errorf("failed to write artifact file: %w", err)
	}

	return fullPath, nil
}

// GetBaseDir returns the base directory for artifact storage.
func (w *Writer) GetBaseDir() string {
	return w.baseDir
}

// addVersionSuffix adds a version number to a filename before the extension.
// Examples:
//
//	file.go, 2 -> file.2.go
//	path/to/main.py, 3 -> path/to/main.3.py
//	README, 2 -> README.2
func addVersionSuffix(path string, version int) string {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	if ext == "" {
		return filepath.Join(dir, fmt.Sprintf("%s.%d", name, version))
	}
	return filepath.Join(dir, fmt.Sprintf("%s.%d%s", name, version, ext))
}

// unsafePathChars matches characters that are unsafe in file paths.
var unsafePathChars = regexp.MustCompile(`[<>:"|?*\x00-\x1f]`)

// sanitizePath makes a path safe for use on the filesystem.
// It removes or replaces unsafe characters while preserving path separators.
func sanitizePath(path string) string {
	// Normalize path separators to forward slashes first
	path = filepath.ToSlash(path)

	// Remove leading slashes to prevent absolute paths
	path = strings.TrimLeft(path, "/")

	// Remove any parent directory references to prevent path traversal
	parts := strings.Split(path, "/")
	safeParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == ".." || part == "." || part == "" {
			continue
		}
		// Remove unsafe characters from each path component
		safePart := unsafePathChars.ReplaceAllString(part, "_")
		// Trim spaces and dots from edges
		safePart = strings.Trim(safePart, " .")
		if safePart != "" {
			safeParts = append(safeParts, safePart)
		}
	}

	if len(safeParts) == 0 {
		return "unnamed"
	}

	return filepath.Join(safeParts...)
}
