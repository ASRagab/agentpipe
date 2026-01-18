// Package cli provides CLI-based adapters for AI agents that operate through command-line tools.
// These adapters execute external CLI programs like Claude CLI and Gemini CLI to interact
// with AI models, providing backwards compatibility with existing v1 configurations.
package cli

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ASRagab/agentpipe/pkg/log"
	"github.com/ASRagab/agentpipe/pkg/v2/adapters"
	"github.com/ASRagab/agentpipe/pkg/v2/core"
)

// CLIAdapter extends AgentAdapter with CLI-specific methods.
type CLIAdapter interface {
	adapters.AgentAdapter

	// CLIPath returns the path to the CLI executable.
	CLIPath() string

	// CLIArgs returns the default arguments for the CLI.
	CLIArgs() []string

	// GetCLIVersion returns the version of the CLI tool.
	GetCLIVersion() string
}

// BaseCLIAdapter provides common functionality for CLI-based adapters.
type BaseCLIAdapter struct {
	cliPath      string
	cliName      string
	model        string
	systemPrompt string
	extraFlags   []string
	agentName    string
	agentID      string
}

// CommonCLILocations are common directories to search for CLI binaries.
var CommonCLILocations = []string{
	"/usr/local/bin",
	"/usr/bin",
	"/opt/homebrew/bin",
	filepath.Join(os.Getenv("HOME"), ".local", "bin"),
	filepath.Join(os.Getenv("HOME"), "bin"),
}

// FindCLIPath searches for a CLI binary by name.
// It first checks PATH, then searches common locations.
func FindCLIPath(name string) (string, error) {
	// First try PATH lookup
	path, err := exec.LookPath(name)
	if err == nil {
		return path, nil
	}

	// Check common locations
	for _, dir := range CommonCLILocations {
		candidate := filepath.Join(dir, name)
		if info, statErr := os.Stat(candidate); statErr == nil {
			// Check if executable
			if info.Mode()&0111 != 0 {
				return candidate, nil
			}
		}
	}

	return "", fmt.Errorf("CLI binary '%s' not found in PATH or common locations", name)
}

// ExecuteCommand runs a CLI command with the given context and returns stdout, stderr, and error.
// The context is used for timeout control.
func ExecuteCommand(ctx context.Context, path string, args []string, stdin io.Reader) (stdout []byte, stderr []byte, err error) {
	cmd := exec.CommandContext(ctx, path, args...)
	if stdin != nil {
		cmd.Stdin = stdin
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	startTime := time.Now()
	err = cmd.Run()
	duration := time.Since(startTime)

	log.WithFields(map[string]interface{}{
		"path":     path,
		"args":     args,
		"duration": duration.String(),
	}).Debug("CLI command executed")

	return stdoutBuf.Bytes(), stderrBuf.Bytes(), err
}

// StreamCommand runs a CLI command and streams its output to the writer.
func StreamCommand(ctx context.Context, path string, args []string, stdin io.Reader, writer io.Writer) error {
	cmd := exec.CommandContext(ctx, path, args...)
	if stdin != nil {
		cmd.Stdin = stdin
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Use a scanner for line-by-line streaming with partial line buffering
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if _, writeErr := fmt.Fprintln(writer, line); writeErr != nil {
			return fmt.Errorf("failed to write to output: %w", writeErr)
		}
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return fmt.Errorf("error reading output: %w", scanErr)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}

// ParseCLIOutput extracts the meaningful response from CLI output.
// It removes common preamble, status messages, and formatting.
func ParseCLIOutput(output string) string {
	lines := strings.Split(output, "\n")
	cleaned := make([]string, 0, len(lines))

	for _, line := range lines {
		// Skip common status/meta lines
		if strings.Contains(line, "Loaded cached credentials") ||
			strings.Contains(line, "To authenticate") ||
			strings.HasPrefix(line, "DEBUG:") ||
			strings.HasPrefix(line, "INFO:") {
			continue
		}
		cleaned = append(cleaned, line)
	}

	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

// HandleCLIError creates a consistent error message from CLI execution failures.
func HandleCLIError(cliName string, err error, stderr []byte) error {
	if err == nil {
		return nil
	}

	// Check for context cancellation
	if ctx, ok := err.(*exec.ExitError); ok {
		if strings.Contains(ctx.Error(), "killed") || strings.Contains(ctx.Error(), "signal") {
			return fmt.Errorf("%s execution was cancelled or timed out", cliName)
		}
	}

	// Include stderr in error message if available
	stderrStr := strings.TrimSpace(string(stderr))
	if stderrStr != "" {
		return fmt.Errorf("%s execution failed: %w\nstderr: %s", cliName, err, stderrStr)
	}

	return fmt.Errorf("%s execution failed: %w", cliName, err)
}

// CLIPath returns the path to the CLI executable.
func (b *BaseCLIAdapter) CLIPath() string {
	return b.cliPath
}

// CLIArgs returns the default arguments for the CLI.
func (b *BaseCLIAdapter) CLIArgs() []string {
	return b.extraFlags
}

// GetModel returns the configured model name.
func (b *BaseCLIAdapter) GetModel() string {
	return b.model
}

// IsAvailable checks if the CLI binary exists.
func (b *BaseCLIAdapter) IsAvailable() bool {
	if b.cliPath != "" {
		_, err := os.Stat(b.cliPath)
		return err == nil
	}
	_, err := FindCLIPath(b.cliName)
	return err == nil
}

// BuildConversationPrompt creates a formatted prompt string from messages.
// This provides a consistent format for multi-agent conversations.
func BuildConversationPrompt(agentName, systemPrompt string, messages []core.Message) string {
	var prompt strings.Builder

	// Part 1: Identity and Role
	prompt.WriteString("AGENT SETUP:\n")
	prompt.WriteString(strings.Repeat("=", 60))
	prompt.WriteString("\n")
	prompt.WriteString(fmt.Sprintf("You are '%s' participating in a multi-agent conversation.\n\n", agentName))

	if systemPrompt != "" {
		prompt.WriteString("YOUR ROLE AND INSTRUCTIONS:\n")
		prompt.WriteString(systemPrompt)
		prompt.WriteString("\n\n")
	}

	prompt.WriteString("ARTIFACT CREATION:\n")
	prompt.WriteString("To create a saveable artifact, use fenced code blocks with a filename:\n")
	prompt.WriteString("  ```language:path/to/filename.ext\n")
	prompt.WriteString("  content here\n")
	prompt.WriteString("  ```\n")
	prompt.WriteString("Artifacts will be saved to the workspace automatically.\n")
	prompt.WriteString(strings.Repeat("=", 60))
	prompt.WriteString("\n\n")

	// Part 2: Conversation context
	if len(messages) > 0 {
		var initialPrompt string
		var otherMessages []core.Message

		// Find the orchestrator's initial prompt
		for _, msg := range messages {
			if msg.Role == core.RoleSystem && (msg.AgentID == "system" || msg.AgentID == "host" || msg.AgentName == "System" || msg.AgentName == "HOST") && initialPrompt == "" {
				initialPrompt = msg.Content
			} else {
				otherMessages = append(otherMessages, msg)
			}
		}

		// Show initial prompt as direct instruction
		if initialPrompt != "" {
			prompt.WriteString("YOUR TASK - PLEASE RESPOND TO THIS:\n")
			prompt.WriteString(strings.Repeat("=", 60))
			prompt.WriteString("\n")
			prompt.WriteString(initialPrompt)
			prompt.WriteString("\n")
			prompt.WriteString(strings.Repeat("=", 60))
			prompt.WriteString("\n\n")
		}

		// Show conversation history
		if len(otherMessages) > 0 {
			prompt.WriteString("CONVERSATION SO FAR:\n")
			prompt.WriteString(strings.Repeat("-", 60))
			prompt.WriteString("\n")
			for _, msg := range otherMessages {
				timestamp := msg.Timestamp.Format("15:04:05")
				if msg.Role == core.RoleSystem {
					prompt.WriteString(fmt.Sprintf("[%s] SYSTEM: %s\n", timestamp, msg.Content))
				} else {
					prompt.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, msg.AgentName, msg.Content))
				}
			}
			prompt.WriteString(strings.Repeat("-", 60))
			prompt.WriteString("\n\n")
		}

		if initialPrompt != "" {
			prompt.WriteString(fmt.Sprintf("Now respond to the task above as %s. Provide a direct, thoughtful answer.", agentName))
		} else {
			prompt.WriteString(fmt.Sprintf("Now, as %s, respond to the conversation.", agentName))
		}
	}

	return prompt.String()
}

// FilterRelevantMessages removes this agent's own messages from the list.
func FilterRelevantMessages(messages []core.Message, agentID, agentName string) []core.Message {
	relevant := make([]core.Message, 0, len(messages))

	for _, msg := range messages {
		// Skip this agent's own messages
		if msg.AgentID == agentID || msg.AgentName == agentName {
			continue
		}
		relevant = append(relevant, msg)
	}

	return relevant
}

// EstimateTokens provides a rough estimate of token count based on character count.
// This is a fallback when actual token counts are not available from CLI output.
func EstimateTokens(text string) int {
	// Rough estimate: ~4 characters per token for English text
	return len(text) / 4
}
