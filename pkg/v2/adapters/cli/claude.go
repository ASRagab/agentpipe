package cli

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/log"
	"github.com/kevinelliott/agentpipe/pkg/v2/adapters"
	"github.com/kevinelliott/agentpipe/pkg/v2/core"
)

// ClaudeCLIAdapter implements the AgentAdapter interface for Claude CLI.
type ClaudeCLIAdapter struct {
	BaseCLIAdapter
}

// NewClaudeCLIAdapter creates a new Claude CLI adapter instance.
func NewClaudeCLIAdapter() adapters.AgentAdapter {
	return &ClaudeCLIAdapter{
		BaseCLIAdapter: BaseCLIAdapter{
			cliName: "claude",
		},
	}
}

// Initialize configures the adapter with the agent configuration.
func (c *ClaudeCLIAdapter) Initialize(agent core.Agent) error {
	path, err := FindCLIPath("claude")
	if err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   agent.ID,
			"agent_name": agent.Name,
		}).WithError(err).Error("Claude CLI not found")
		return fmt.Errorf("Claude CLI not found: %w", err)
	}

	c.cliPath = path
	c.model = agent.Model
	c.systemPrompt = agent.Config.SystemPrompt
	c.agentName = agent.Name
	c.agentID = agent.ID

	// Parse extra flags from config
	if extraFlags, ok := agent.Config.Extra["flags"].([]interface{}); ok {
		for _, flag := range extraFlags {
			if s, ok := flag.(string); ok {
				c.extraFlags = append(c.extraFlags, s)
			}
		}
	}

	log.WithFields(map[string]interface{}{
		"agent_id":   agent.ID,
		"agent_name": agent.Name,
		"exec_path":  path,
		"model":      c.model,
	}).Info("Claude CLI adapter initialized successfully")

	return nil
}

// SendMessage sends messages to Claude CLI and returns the response.
func (c *ClaudeCLIAdapter) SendMessage(ctx context.Context, messages []core.Message) (string, *core.Metrics, error) {
	if len(messages) == 0 {
		return "", nil, nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    c.agentName,
		"message_count": len(messages),
	}).Debug("Sending message to Claude CLI")

	// Filter out this agent's own messages
	relevantMessages := FilterRelevantMessages(messages, c.agentID, c.agentName)

	// Build the prompt
	prompt := BuildConversationPrompt(c.agentName, c.systemPrompt, relevantMessages)

	// Build command args - must use -p for non-interactive mode
	args := []string{"-p"}

	// Add model flag if specified
	if c.model != "" {
		args = append(args, "--model", c.model)
	}

	// Add any extra flags
	args = append(args, c.extraFlags...)

	startTime := time.Now()
	stdout, stderr, err := ExecuteCommand(ctx, c.cliPath, args, strings.NewReader(prompt))
	duration := time.Since(startTime)

	if err != nil {
		// Check if we have meaningful output despite an error
		// Claude CLI sometimes exits non-zero but produces valid output
		outputStr := string(stdout)
		if len(outputStr) > 50 && !strings.Contains(outputStr, "error") {
			log.WithFields(map[string]interface{}{
				"agent_name": c.agentName,
				"duration":   duration.String(),
				"exit_error": err.Error(),
			}).Debug("Claude had exit error but produced valid output, accepting response")
		} else {
			return "", nil, HandleCLIError("Claude", err, stderr)
		}
	}

	response := ParseCLIOutput(string(stdout))

	// Estimate metrics (Claude CLI doesn't provide token counts)
	inputTokens := EstimateTokens(prompt)
	outputTokens := EstimateTokens(response)

	metrics := &core.Metrics{
		Duration:     duration,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		Model:        c.model,
		Cost:         c.estimateCost(inputTokens, outputTokens),
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    c.agentName,
		"duration":      duration.String(),
		"response_size": len(response),
	}).Info("Claude CLI message sent successfully")

	return response, metrics, nil
}

// StreamMessage sends messages and streams the response to the writer.
func (c *ClaudeCLIAdapter) StreamMessage(ctx context.Context, messages []core.Message, writer io.Writer) (*core.Metrics, error) {
	if len(messages) == 0 {
		return nil, nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    c.agentName,
		"message_count": len(messages),
	}).Debug("Starting Claude CLI streaming message")

	// Filter out this agent's own messages
	relevantMessages := FilterRelevantMessages(messages, c.agentID, c.agentName)

	// Build the prompt
	prompt := BuildConversationPrompt(c.agentName, c.systemPrompt, relevantMessages)

	// Build command args - use -p for non-interactive mode (streaming by default)
	args := []string{"-p"}

	// Add model flag if specified
	if c.model != "" {
		args = append(args, "--model", c.model)
	}

	// Add any extra flags
	args = append(args, c.extraFlags...)

	startTime := time.Now()
	err := StreamCommand(ctx, c.cliPath, args, strings.NewReader(prompt), writer)
	duration := time.Since(startTime)

	if err != nil {
		return nil, HandleCLIError("Claude", err, nil)
	}

	// Estimate metrics
	inputTokens := EstimateTokens(prompt)
	// Output tokens are hard to estimate for streaming without capturing the full response
	outputTokens := 100 // Rough estimate

	metrics := &core.Metrics{
		Duration:     duration,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		Model:        c.model,
		Cost:         c.estimateCost(inputTokens, outputTokens),
	}

	log.WithFields(map[string]interface{}{
		"agent_name": c.agentName,
		"duration":   duration.String(),
	}).Info("Claude CLI streaming message completed")

	return metrics, nil
}

// HealthCheck verifies the Claude CLI is accessible and working.
func (c *ClaudeCLIAdapter) HealthCheck(ctx context.Context) error {
	if c.cliPath == "" {
		log.WithField("agent_name", c.agentName).Error("Claude health check failed: not initialized")
		return fmt.Errorf("Claude CLI not initialized")
	}

	log.WithField("agent_name", c.agentName).Debug("Starting Claude health check")

	// Try --version first
	cmd := exec.CommandContext(ctx, c.cliPath, "--version")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Try --help if --version fails
		log.WithField("agent_name", c.agentName).Debug("--version check failed, trying --help")
		cmd = exec.CommandContext(ctx, c.cliPath, "--help")
		output, err = cmd.CombinedOutput()

		if err != nil {
			log.WithField("agent_name", c.agentName).WithError(err).Error("Claude health check failed: CLI not responding")
			return fmt.Errorf("Claude CLI not responding to --version or --help: %w", err)
		}
	}

	// Check if output contains something reasonable
	if len(output) < 10 {
		log.WithFields(map[string]interface{}{
			"agent_name":    c.agentName,
			"output_length": len(output),
		}).Error("Claude health check failed: output too short")
		return fmt.Errorf("Claude CLI returned suspiciously short output")
	}

	log.WithField("agent_name", c.agentName).Info("Claude health check passed")
	return nil
}

// GetCLIVersion returns the version of the Claude CLI.
func (c *ClaudeCLIAdapter) GetCLIVersion() string {
	if c.cliPath == "" {
		return "unknown"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.cliPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	version := strings.TrimSpace(string(output))
	// Extract just the version number if present
	if lines := strings.Split(version, "\n"); len(lines) > 0 {
		return lines[0]
	}
	return version
}

// estimateCost calculates an estimated cost based on token usage.
func (c *ClaudeCLIAdapter) estimateCost(inputTokens, outputTokens int) float64 {
	// Default to Sonnet pricing
	inputCostPerMillion := 3.0
	outputCostPerMillion := 15.0

	// Adjust based on model
	if strings.Contains(c.model, "haiku") {
		inputCostPerMillion = 0.25
		outputCostPerMillion = 1.25
	} else if strings.Contains(c.model, "opus") {
		inputCostPerMillion = 15.0
		outputCostPerMillion = 75.0
	}

	return (float64(inputTokens) * inputCostPerMillion / 1_000_000) +
		(float64(outputTokens) * outputCostPerMillion / 1_000_000)
}

func init() {
	adapters.Register("claude-cli", NewClaudeCLIAdapter)
}
