package cli

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/ASRagab/agentpipe/pkg/log"
	"github.com/ASRagab/agentpipe/pkg/v2/adapters"
	"github.com/ASRagab/agentpipe/pkg/v2/core"
)

// GeminiCLIAdapter implements the AgentAdapter interface for Gemini CLI.
type GeminiCLIAdapter struct {
	BaseCLIAdapter
}

// NewGeminiCLIAdapter creates a new Gemini CLI adapter instance.
func NewGeminiCLIAdapter() adapters.AgentAdapter {
	return &GeminiCLIAdapter{
		BaseCLIAdapter: BaseCLIAdapter{
			cliName: "gemini",
		},
	}
}

// Initialize configures the adapter with the agent configuration.
func (g *GeminiCLIAdapter) Initialize(agent core.Agent) error {
	path, err := FindCLIPath("gemini")
	if err != nil {
		log.WithFields(map[string]interface{}{
			"agent_id":   agent.ID,
			"agent_name": agent.Name,
		}).WithError(err).Error("Gemini CLI not found")
		return fmt.Errorf("Gemini CLI not found: %w", err)
	}

	g.cliPath = path
	g.model = agent.Model
	g.systemPrompt = agent.Config.SystemPrompt
	g.agentName = agent.Name
	g.agentID = agent.ID

	// Parse extra flags from config
	if extraFlags, ok := agent.Config.Extra["flags"].([]interface{}); ok {
		for _, flag := range extraFlags {
			if s, ok := flag.(string); ok {
				g.extraFlags = append(g.extraFlags, s)
			}
		}
	}

	log.WithFields(map[string]interface{}{
		"agent_id":   agent.ID,
		"agent_name": agent.Name,
		"exec_path":  path,
		"model":      g.model,
	}).Info("Gemini CLI adapter initialized successfully")

	return nil
}

// SendMessage sends messages to Gemini CLI and returns the response.
func (g *GeminiCLIAdapter) SendMessage(ctx context.Context, messages []core.Message) (string, *core.Metrics, error) {
	if len(messages) == 0 {
		return "", nil, nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    g.agentName,
		"message_count": len(messages),
	}).Debug("Sending message to Gemini CLI")

	// Filter out this agent's own messages
	relevantMessages := FilterRelevantMessages(messages, g.agentID, g.agentName)

	// Build the prompt
	prompt := BuildConversationPrompt(g.agentName, g.systemPrompt, relevantMessages)

	// Build command args
	var args []string

	// Add model flag if specified
	if g.model != "" {
		args = append(args, "--model", g.model)
	}

	// Add any extra flags
	args = append(args, g.extraFlags...)

	startTime := time.Now()
	stdout, stderr, err := ExecuteCommand(ctx, g.cliPath, args, strings.NewReader(prompt))
	duration := time.Since(startTime)

	outputStr := string(stdout)

	// Check for API errors in output
	if strings.Contains(outputStr, "404") || strings.Contains(outputStr, "NOT_FOUND") {
		log.WithFields(map[string]interface{}{
			"agent_name": g.agentName,
			"model":      g.model,
			"duration":   duration.String(),
		}).Error("Gemini model not found")
		return "", nil, fmt.Errorf("Gemini model not found - check model name in config: %s", g.model)
	}
	if strings.Contains(outputStr, "401") || strings.Contains(outputStr, "UNAUTHENTICATED") {
		log.WithFields(map[string]interface{}{
			"agent_name": g.agentName,
			"duration":   duration.String(),
		}).Error("Gemini authentication failed")
		return "", nil, fmt.Errorf("Gemini authentication failed - check API keys")
	}

	// Gemini CLI sometimes exits non-zero but produces valid output
	hasValidOutput := len(outputStr) > 0 && !strings.Contains(outputStr, "error")

	if err != nil {
		if hasValidOutput {
			log.WithFields(map[string]interface{}{
				"agent_name": g.agentName,
				"duration":   duration.String(),
				"exit_error": err.Error(),
			}).Debug("Gemini had exit error but produced valid output, accepting response")
		} else {
			return "", nil, HandleCLIError("Gemini", err, stderr)
		}
	}

	response := g.cleanOutput(outputStr)

	// Estimate metrics (Gemini CLI doesn't provide token counts)
	inputTokens := EstimateTokens(prompt)
	outputTokens := EstimateTokens(response)

	metrics := &core.Metrics{
		Duration:     duration,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		Model:        g.model,
		Cost:         g.estimateCost(inputTokens, outputTokens),
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    g.agentName,
		"duration":      duration.String(),
		"response_size": len(response),
	}).Info("Gemini CLI message sent successfully")

	return response, metrics, nil
}

// cleanOutput removes Gemini-specific noise from the output.
func (g *GeminiCLIAdapter) cleanOutput(output string) string {
	lines := strings.Split(output, "\n")
	cleanedLines := make([]string, 0, len(lines))
	inErrorTrace := false

	for _, line := range lines {
		// Skip common Gemini status messages
		if strings.Contains(line, "Loaded cached credentials") ||
			strings.Contains(line, "To authenticate") ||
			strings.HasPrefix(line, "Gemini CLI") {
			continue
		}

		// Detect start of error trace
		if strings.Contains(line, "Attempt") && strings.Contains(line, "failed with status") {
			inErrorTrace = true
			continue
		}
		if strings.Contains(line, "GaxiosError:") || strings.Contains(line, "at Gaxios._request") {
			inErrorTrace = true
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "at ") || strings.HasPrefix(strings.TrimSpace(line), "at async") {
			inErrorTrace = true
			continue
		}

		// Skip lines that are part of error traces
		if inErrorTrace {
			if strings.TrimSpace(line) == "" {
				continue
			}
			// Check if this looks like actual content
			if !strings.HasPrefix(strings.TrimSpace(line), "{") &&
				!strings.HasPrefix(strings.TrimSpace(line), "[") &&
				!strings.HasPrefix(strings.TrimSpace(line), "}") &&
				!strings.HasPrefix(strings.TrimSpace(line), "]") &&
				!strings.Contains(line, "config:") &&
				!strings.Contains(line, "response:") &&
				!strings.Contains(line, "Symbol(") &&
				len(strings.TrimSpace(line)) > 20 {
				inErrorTrace = false
			} else {
				continue
			}
		}

		cleanedLines = append(cleanedLines, line)
	}

	return strings.TrimSpace(strings.Join(cleanedLines, "\n"))
}

// StreamMessage sends messages and streams the response to the writer.
func (g *GeminiCLIAdapter) StreamMessage(ctx context.Context, messages []core.Message, writer io.Writer) (*core.Metrics, error) {
	if len(messages) == 0 {
		return nil, nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    g.agentName,
		"message_count": len(messages),
	}).Debug("Starting Gemini CLI streaming message")

	// Filter out this agent's own messages
	relevantMessages := FilterRelevantMessages(messages, g.agentID, g.agentName)

	// Build the prompt
	prompt := BuildConversationPrompt(g.agentName, g.systemPrompt, relevantMessages)

	// Build command args
	var args []string
	if g.model != "" {
		args = append(args, "--model", g.model)
	}
	args = append(args, g.extraFlags...)

	// Create a filtering writer to skip noise lines
	filterWriter := &geminiFilterWriter{writer: writer}

	startTime := time.Now()
	err := StreamCommand(ctx, g.cliPath, args, strings.NewReader(prompt), filterWriter)
	duration := time.Since(startTime)

	if err != nil {
		return nil, HandleCLIError("Gemini", err, nil)
	}

	// Estimate metrics
	inputTokens := EstimateTokens(prompt)
	outputTokens := 100 // Rough estimate for streaming

	metrics := &core.Metrics{
		Duration:     duration,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		Model:        g.model,
		Cost:         g.estimateCost(inputTokens, outputTokens),
	}

	log.WithFields(map[string]interface{}{
		"agent_name": g.agentName,
		"duration":   duration.String(),
	}).Info("Gemini CLI streaming message completed")

	return metrics, nil
}

// geminiFilterWriter filters out Gemini-specific noise during streaming.
type geminiFilterWriter struct {
	writer io.Writer
}

func (f *geminiFilterWriter) Write(p []byte) (n int, err error) {
	line := string(p)
	// Filter out noise lines
	if strings.Contains(line, "Loaded cached credentials") ||
		strings.Contains(line, "To authenticate") ||
		strings.HasPrefix(strings.TrimSpace(line), "Gemini CLI") {
		return len(p), nil // Pretend we wrote it
	}
	return f.writer.Write(p)
}

// HealthCheck verifies the Gemini CLI is accessible and working.
func (g *GeminiCLIAdapter) HealthCheck(ctx context.Context) error {
	if g.cliPath == "" {
		log.WithField("agent_name", g.agentName).Error("Gemini health check failed: not initialized")
		return fmt.Errorf("Gemini CLI not initialized")
	}

	log.WithField("agent_name", g.agentName).Debug("Starting Gemini health check")

	// Try --help first (Gemini CLI behavior)
	cmd := exec.CommandContext(ctx, g.cliPath, "--help")
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Gemini might be interactive, try --version
		log.WithField("agent_name", g.agentName).Debug("--help check failed, trying --version")
		testCmd := exec.Command(g.cliPath, "--version")
		if startErr := testCmd.Start(); startErr != nil {
			log.WithField("agent_name", g.agentName).WithError(startErr).Error("Gemini health check failed: CLI not responding")
			return fmt.Errorf("Gemini CLI cannot be executed: %w", startErr)
		}
		// Kill the process if still running
		if testCmd.Process != nil {
			_ = testCmd.Process.Kill()
			_ = testCmd.Wait()
		}
		log.WithField("agent_name", g.agentName).Info("Gemini health check passed")
		return nil
	}

	// Check if output looks like gemini help
	if len(output) < 50 {
		log.WithField("agent_name", g.agentName).Error("Gemini health check failed: suspiciously short help output")
		return fmt.Errorf("Gemini CLI returned suspiciously short help output")
	}

	log.WithField("agent_name", g.agentName).Info("Gemini health check passed")
	return nil
}

// GetCLIVersion returns the version of the Gemini CLI.
func (g *GeminiCLIAdapter) GetCLIVersion() string {
	if g.cliPath == "" {
		return "unknown"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, g.cliPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	version := strings.TrimSpace(string(output))
	if lines := strings.Split(version, "\n"); len(lines) > 0 {
		return lines[0]
	}
	return version
}

// estimateCost calculates an estimated cost based on token usage.
func (g *GeminiCLIAdapter) estimateCost(inputTokens, outputTokens int) float64 {
	// Gemini pricing (varies by model)
	// Default to Gemini 1.5 Pro pricing
	inputCostPerMillion := 3.50
	outputCostPerMillion := 10.50

	// Adjust based on model
	if strings.Contains(g.model, "flash") {
		inputCostPerMillion = 0.075
		outputCostPerMillion = 0.30
	} else if strings.Contains(g.model, "nano") {
		inputCostPerMillion = 0.0
		outputCostPerMillion = 0.0
	}

	return (float64(inputTokens) * inputCostPerMillion / 1_000_000) +
		(float64(outputTokens) * outputCostPerMillion / 1_000_000)
}

func init() {
	adapters.Register("gemini-cli", NewGeminiCLIAdapter)
}
