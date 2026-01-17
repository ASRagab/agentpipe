package artifact

import (
	"regexp"
	"strings"
	"time"
)

// artifactPattern matches fenced code blocks with language:filename syntax.
// Format: ```language:path/to/file.ext followed by content and closing ```
// Examples:
//   - ```go:cmd/main.go
//   - ```python:src/utils/helper.py
//   - ```javascript:lib/index.js
var artifactPattern = regexp.MustCompile("(?s)```([a-zA-Z0-9_+-]+):([^\n`]+)\n(.*?)```")

// placeholderPattern detects generic placeholder paths that shouldn't be saved.
// These are paths like "path/to/file.ext" or "your/filename.go" that are examples, not real files.
var placeholderPattern = regexp.MustCompile(`^(path|your|example|sample|my)[/-]`)

// isValidArtifactPath checks if a filename is a valid artifact path (not a placeholder).
// Returns false for generic placeholder paths that agents sometimes generate as examples.
func isValidArtifactPath(filename string) bool {
	// Reject empty filenames
	if filename == "" {
		return false
	}

	// Reject placeholder paths like "path/to/file.ext" or "your/filename.go"
	lowerFilename := strings.ToLower(filename)
	if placeholderPattern.MatchString(lowerFilename) {
		return false
	}

	// Reject paths that are clearly template examples
	if strings.Contains(lowerFilename, "<") && strings.Contains(lowerFilename, ">") {
		return false
	}

	// Reject paths containing common placeholder words in the filename itself
	base := strings.ToLower(filename)
	if idx := strings.LastIndex(base, "/"); idx != -1 {
		base = base[idx+1:]
	}
	placeholderWords := []string{"example", "sample", "your-", "my-", "placeholder"}
	for _, word := range placeholderWords {
		if strings.HasPrefix(base, word) {
			return false
		}
	}

	return true
}

// Parse extracts all artifacts from a message content string.
// It looks for fenced code blocks in the format ```language:filename
// and extracts the language, filename, and content from each match.
// Returns an ExtractResult containing the found artifacts and the
// message content with artifact blocks removed.
func Parse(content, agentID, agentName string) ExtractResult {
	result := ExtractResult{
		Artifacts:      []Artifact{},
		CleanedContent: content,
	}

	matches := artifactPattern.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return result
	}

	// Extract artifacts from matches
	for _, match := range matches {
		if len(match) < 8 {
			continue
		}

		// Extract captured groups
		language := content[match[2]:match[3]]
		filename := strings.TrimSpace(content[match[4]:match[5]])
		artifactContent := content[match[6]:match[7]]

		// Skip if filename is empty or is a placeholder path
		if !isValidArtifactPath(filename) {
			continue
		}

		artifact := Artifact{
			AgentID:    agentID,
			AgentName:  agentName,
			Type:       language,
			Identifier: filename,
			Language:   language,
			Filename:   filename,
			Content:    artifactContent,
			Timestamp:  time.Now(),
			Version:    1,
		}

		result.Artifacts = append(result.Artifacts, artifact)
	}

	// Remove artifact blocks from content
	result.CleanedContent = artifactPattern.ReplaceAllString(content, "")
	result.CleanedContent = strings.TrimSpace(result.CleanedContent)

	return result
}

// HasArtifacts checks if the content contains any artifact blocks.
// This is a quick check without fully parsing the content.
func HasArtifacts(content string) bool {
	return artifactPattern.MatchString(content)
}
