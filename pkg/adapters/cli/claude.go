package cli

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/ASRagab/agentpipe/pkg/adapters"
	"github.com/ASRagab/agentpipe/pkg/core"
	"github.com/ASRagab/agentpipe/pkg/log"
)

type ClaudeCLIAdapter struct {
	BaseCLIAdapter
}

func NewClaudeCLIAdapter() adapters.AgentAdapter {
	return &ClaudeCLIAdapter{
		BaseCLIAdapter: BaseCLIAdapter{
			cliName: "claude",
		},
	}
}

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

func (c *ClaudeCLIAdapter) SendMessage(ctx context.Context, messages []core.Message, conversation *core.ConversationContext) (string, *core.Metrics, error) {
	if len(messages) == 0 {
		return "", nil, nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    c.agentName,
		"message_count": len(messages),
	}).Debug("Sending message to Claude CLI")

	relevantMessages := FilterRelevantMessages(messages, c.agentID, c.agentName)
	prompt := BuildConversationPrompt(c.agentName, c.systemPrompt, relevantMessages, conversation)

	args := []string{"-p"}
	if c.model != "" {
		args = append(args, "--model", c.model)
	}
	args = append(args, c.extraFlags...)

	startTime := time.Now()
	stdout, stderr, err := ExecuteCommand(ctx, c.cliPath, args, strings.NewReader(prompt))
	duration := time.Since(startTime)

	if err != nil {
		if output := strings.TrimSpace(string(stdout)); output != "" {
			metrics := &core.Metrics{
				Duration:     duration,
				InputTokens:  EstimateTokens(prompt),
				OutputTokens: EstimateTokens(output),
				TotalTokens:  EstimateTokens(prompt) + EstimateTokens(output),
				Model:        c.model,
				Cost:         c.estimateCost(EstimateTokens(prompt), EstimateTokens(output)),
			}
			return output, metrics, nil
		}
		return "", nil, HandleCLIError("Claude", err, stderr)
	}

	content := ParseCLIOutput(string(stdout))
	inputTokens := EstimateTokens(prompt)
	outputTokens := EstimateTokens(content)

	metrics := &core.Metrics{
		Duration:     duration,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		Model:        c.model,
		Cost:         c.estimateCost(inputTokens, outputTokens),
	}

	return content, metrics, nil
}

func (c *ClaudeCLIAdapter) StreamMessage(ctx context.Context, messages []core.Message, writer io.Writer, conversation *core.ConversationContext) (*core.Metrics, error) {
	if len(messages) == 0 {
		return nil, nil
	}

	relevantMessages := FilterRelevantMessages(messages, c.agentID, c.agentName)
	prompt := BuildConversationPrompt(c.agentName, c.systemPrompt, relevantMessages, conversation)

	args := []string{"-p"}
	if c.model != "" {
		args = append(args, "--model", c.model)
	}
	args = append(args, c.extraFlags...)

	startTime := time.Now()
	err := StreamCommand(ctx, c.cliPath, args, strings.NewReader(prompt), writer)
	duration := time.Since(startTime)

	if err != nil {
		return nil, HandleCLIError("Claude", err, nil)
	}

	inputTokens := EstimateTokens(prompt)
	outputTokens := 100

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
