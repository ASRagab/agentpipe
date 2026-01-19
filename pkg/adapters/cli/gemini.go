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

type GeminiCLIAdapter struct {
	BaseCLIAdapter
}

func NewGeminiCLIAdapter() adapters.AgentAdapter {
	return &GeminiCLIAdapter{
		BaseCLIAdapter: BaseCLIAdapter{
			cliName: "gemini",
		},
	}
}

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

func (g *GeminiCLIAdapter) SendMessage(ctx context.Context, messages []core.Message, conversation *core.ConversationContext) (string, *core.Metrics, error) {
	if len(messages) == 0 {
		return "", nil, nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    g.agentName,
		"message_count": len(messages),
	}).Debug("Sending message to Gemini CLI")

	relevantMessages := FilterRelevantMessages(messages, g.agentID, g.agentName)
	prompt := BuildConversationPrompt(g.agentName, g.systemPrompt, relevantMessages, conversation)

	var args []string
	if g.model != "" {
		args = append(args, "--model", g.model)
	}
	args = append(args, g.extraFlags...)

	startTime := time.Now()
	stdout, stderr, err := ExecuteCommand(ctx, g.cliPath, args, strings.NewReader(prompt))
	duration := time.Since(startTime)

	if err != nil {
		if output := strings.TrimSpace(string(stdout)); output != "" {
			metrics := &core.Metrics{
				Duration:     duration,
				InputTokens:  EstimateTokens(prompt),
				OutputTokens: EstimateTokens(output),
				TotalTokens:  EstimateTokens(prompt) + EstimateTokens(output),
				Model:        g.model,
				Cost:         g.estimateCost(EstimateTokens(prompt), EstimateTokens(output)),
			}
			return output, metrics, nil
		}
		return "", nil, HandleCLIError("Gemini", err, stderr)
	}

	content := ParseCLIOutput(string(stdout))
	inputTokens := EstimateTokens(prompt)
	outputTokens := EstimateTokens(content)

	metrics := &core.Metrics{
		Duration:     duration,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		Model:        g.model,
		Cost:         g.estimateCost(inputTokens, outputTokens),
	}

	return content, metrics, nil
}

type geminiFilterWriter struct {
	writer io.Writer
}

func (f *geminiFilterWriter) Write(p []byte) (n int, err error) {
	line := string(p)
	if strings.Contains(line, "Loaded cached credentials") {
		line = strings.ReplaceAll(line, "Loaded cached credentials", "")
	}
	if strings.Contains(line, "To authenticate") {
		line = strings.ReplaceAll(line, "To authenticate", "")
	}

	lines := strings.Split(line, "\n")
	filtered := make([]string, 0, len(lines))
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "Gemini CLI") {
			continue
		}
		filtered = append(filtered, l)
	}

	output := strings.TrimLeft(strings.Join(filtered, "\n"), "\n")
	if strings.TrimSpace(output) == "" {
		return len(p), nil
	}
	return f.writer.Write([]byte(output))
}

func (g *GeminiCLIAdapter) StreamMessage(ctx context.Context, messages []core.Message, writer io.Writer, conversation *core.ConversationContext) (*core.Metrics, error) {
	if len(messages) == 0 {
		return nil, nil
	}

	relevantMessages := FilterRelevantMessages(messages, g.agentID, g.agentName)
	prompt := BuildConversationPrompt(g.agentName, g.systemPrompt, relevantMessages, conversation)

	var args []string
	if g.model != "" {
		args = append(args, "--model", g.model)
	}
	args = append(args, g.extraFlags...)

	filterWriter := &geminiFilterWriter{writer: writer}

	startTime := time.Now()
	err := StreamCommand(ctx, g.cliPath, args, strings.NewReader(prompt), filterWriter)
	duration := time.Since(startTime)

	if err != nil {
		return nil, HandleCLIError("Gemini", err, nil)
	}

	inputTokens := EstimateTokens(prompt)
	outputTokens := 100

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
