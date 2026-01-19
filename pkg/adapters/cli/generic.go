package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/ASRagab/agentpipe/pkg/adapters"
	"github.com/ASRagab/agentpipe/pkg/core"
	"github.com/ASRagab/agentpipe/pkg/log"
)

type OutputParser string

const (
	OutputParserPlain    OutputParser = "plain"
	OutputParserJSON     OutputParser = "json"
	OutputParserMarkdown OutputParser = "markdown"
)

type GenericCLIConfig struct {
	CLIPath         string       `json:"cli_path" yaml:"cli_path"`
	CLIName         string       `json:"cli_name" yaml:"cli_name"`
	PromptFlag      string       `json:"prompt_flag" yaml:"prompt_flag"`
	HistoryFlag     string       `json:"history_flag" yaml:"history_flag"`
	StreamFlag      string       `json:"stream_flag" yaml:"stream_flag"`
	ModelFlag       string       `json:"model_flag" yaml:"model_flag"`
	JSONPath        string       `json:"json_path" yaml:"json_path"`
	CommandTemplate string       `json:"command_template" yaml:"command_template"`
	UseStdin        bool         `json:"use_stdin" yaml:"use_stdin"`
	Args            []string     `json:"args" yaml:"args"`
	ExtraArgs       []string     `json:"extra_args" yaml:"extra_args"`
	OutputParser    OutputParser `json:"output_parser" yaml:"output_parser"`
	ContentPath     string       `json:"content_path" yaml:"content_path"`
}

// GenericCLIAdapter implements the AgentAdapter interface for any CLI tool.
type GenericCLIAdapter struct {
	BaseCLIAdapter
	config GenericCLIConfig
}

func NewGenericCLIAdapter() adapters.AgentAdapter {
	return &GenericCLIAdapter{}
}

func (g *GenericCLIAdapter) Initialize(agent core.Agent) error {
	// Parse config from agent.Config.Extra
	g.config = GenericCLIConfig{
		OutputParser: OutputParserPlain,
		UseStdin:     true,
	}

	if extra := agent.Config.Extra; extra != nil {
		if cliPath, ok := extra["cli_path"].(string); ok {
			g.config.CLIPath = cliPath
		}
		if cliName, ok := extra["cli_name"].(string); ok {
			g.config.CLIName = cliName
		}
		if promptFlag, ok := extra["prompt_flag"].(string); ok {
			g.config.PromptFlag = promptFlag
		}
		if historyFlag, ok := extra["history_flag"].(string); ok {
			g.config.HistoryFlag = historyFlag
		}
		if streamFlag, ok := extra["stream_flag"].(string); ok {
			g.config.StreamFlag = streamFlag
		}
		if modelFlag, ok := extra["model_flag"].(string); ok {
			g.config.ModelFlag = modelFlag
		}
		if parser, ok := extra["output_parser"].(string); ok {
			g.config.OutputParser = OutputParser(parser)
		}
		if jsonPath, ok := extra["json_path"].(string); ok {
			g.config.JSONPath = jsonPath
		}
		if template, ok := extra["command_template"].(string); ok {
			g.config.CommandTemplate = template
		}
		if useStdin, ok := extra["use_stdin"].(bool); ok {
			g.config.UseStdin = useStdin
		}
		if extraArgs, ok := extra["extra_args"].([]interface{}); ok {
			for _, arg := range extraArgs {
				if s, ok := arg.(string); ok {
					g.config.ExtraArgs = append(g.config.ExtraArgs, s)
				}
			}
		}
	}

	// Find CLI path
	var cliPath string
	var err error

	if g.config.CLIPath != "" {
		cliPath = g.config.CLIPath
	} else if g.config.CLIName != "" {
		cliPath, err = FindCLIPath(g.config.CLIName)
		if err != nil {
			log.WithFields(map[string]interface{}{
				"agent_id":   agent.ID,
				"agent_name": agent.Name,
				"cli_name":   g.config.CLIName,
			}).WithError(err).Error("Generic CLI not found")
			return fmt.Errorf("CLI '%s' not found: %w", g.config.CLIName, err)
		}
	} else {
		return fmt.Errorf("either cli_path or cli_name must be specified for generic CLI adapter")
	}

	g.cliPath = cliPath
	g.cliName = g.config.CLIName
	g.model = agent.Model
	g.systemPrompt = agent.Config.SystemPrompt
	g.agentName = agent.Name
	g.agentID = agent.ID
	g.extraFlags = g.config.ExtraArgs

	log.WithFields(map[string]interface{}{
		"agent_id":   agent.ID,
		"agent_name": agent.Name,
		"exec_path":  cliPath,
		"model":      g.model,
	}).Info("Generic CLI adapter initialized successfully")

	return nil
}

// buildArgs constructs the command arguments.
func (g *GenericCLIAdapter) buildArgs(prompt string) []string {
	var args []string

	// Add model flag if specified
	if g.model != "" && g.config.ModelFlag != "" {
		args = append(args, g.config.ModelFlag, g.model)
	}

	// Add prompt flag if not using stdin
	if !g.config.UseStdin && g.config.PromptFlag != "" {
		args = append(args, g.config.PromptFlag, prompt)
	}

	// Add extra args
	args = append(args, g.config.ExtraArgs...)

	return args
}

// parseOutput parses the CLI output according to the configured parser.
func (g *GenericCLIAdapter) parseOutput(output string) (string, error) {
	switch g.config.OutputParser {
	case OutputParserJSON:
		return g.parseJSONOutput(output)
	case OutputParserMarkdown:
		return g.parseMarkdownOutput(output)
	case OutputParserPlain:
		fallthrough
	default:
		return ParseCLIOutput(output), nil
	}
}

// parseJSONOutput extracts content from JSON output.
func (g *GenericCLIAdapter) parseJSONOutput(output string) (string, error) {
	// Try to find JSON in the output
	jsonStart := strings.Index(output, "{")
	jsonEnd := strings.LastIndex(output, "}")
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd < jsonStart {
		return output, nil // No JSON found, return as-is
	}

	jsonStr := output[jsonStart : jsonEnd+1]
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return output, nil // Invalid JSON, return as-is
	}

	// Extract content using JSON path if specified
	if g.config.JSONPath != "" {
		path := strings.Split(g.config.JSONPath, ".")
		current := data
		for i, key := range path {
			if i == len(path)-1 {
				if val, ok := current[key]; ok {
					if str, ok := val.(string); ok {
						return str, nil
					}
					// Convert to string if not already
					jsonBytes, _ := json.Marshal(val)
					return string(jsonBytes), nil
				}
			} else {
				if nested, ok := current[key].(map[string]interface{}); ok {
					current = nested
				} else {
					break
				}
			}
		}
	}

	// Default: look for common content fields
	for _, key := range []string{"content", "text", "response", "message", "output"} {
		if val, ok := data[key]; ok {
			if str, ok := val.(string); ok {
				return str, nil
			}
		}
	}

	return output, nil
}

// parseMarkdownOutput extracts content from markdown code blocks.
func (g *GenericCLIAdapter) parseMarkdownOutput(output string) (string, error) {
	// Pattern to match code blocks
	codeBlockPattern := regexp.MustCompile("```(?:[a-z]*\n)?([\\s\\S]*?)```")
	matches := codeBlockPattern.FindAllStringSubmatch(output, -1)

	if len(matches) == 0 {
		return ParseCLIOutput(output), nil
	}

	// Extract content from all code blocks
	var contents []string
	for _, match := range matches {
		if len(match) > 1 {
			contents = append(contents, strings.TrimSpace(match[1]))
		}
	}

	return strings.Join(contents, "\n\n"), nil
}

func (g *GenericCLIAdapter) SendMessage(ctx context.Context, messages []core.Message, conversation *core.ConversationContext) (string, *core.Metrics, error) {
	if len(messages) == 0 {
		return "", nil, nil
	}

	log.WithFields(map[string]interface{}{
		"agent_name":    g.agentName,
		"message_count": len(messages),
	}).Debug("Sending message to generic CLI")

	relevantMessages := FilterRelevantMessages(messages, g.agentID, g.agentName)
	prompt := BuildConversationPrompt(g.agentName, g.systemPrompt, relevantMessages, conversation)

	args := g.buildArgs(prompt)
	if g.config.UseStdin {
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
				}
				return output, metrics, nil
			}
			return "", nil, HandleCLIError(g.cliName, err, stderr)
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
		}

		return content, metrics, nil
	}

	startTime := time.Now()
	stdout, stderr, err := ExecuteCommand(ctx, g.cliPath, args, nil)
	duration := time.Since(startTime)
	if err != nil {
		if output := strings.TrimSpace(string(stdout)); output != "" {
			metrics := &core.Metrics{
				Duration:     duration,
				InputTokens:  EstimateTokens(prompt),
				OutputTokens: EstimateTokens(output),
				TotalTokens:  EstimateTokens(prompt) + EstimateTokens(output),
				Model:        g.model,
			}
			return output, metrics, nil
		}
		return "", nil, HandleCLIError(g.cliName, err, stderr)
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
	}

	return content, metrics, nil
}

func (g *GenericCLIAdapter) StreamMessage(ctx context.Context, messages []core.Message, writer io.Writer, conversation *core.ConversationContext) (*core.Metrics, error) {
	if len(messages) == 0 {
		return nil, nil
	}

	relevantMessages := FilterRelevantMessages(messages, g.agentID, g.agentName)
	prompt := BuildConversationPrompt(g.agentName, g.systemPrompt, relevantMessages, conversation)

	args := g.buildArgs(prompt)
	if g.config.StreamFlag != "" {
		args = append(args, g.config.StreamFlag)
	}

	var stdin io.Reader
	if g.config.UseStdin {
		stdin = strings.NewReader(prompt)
	}

	startTime := time.Now()
	err := StreamCommand(ctx, g.cliPath, args, stdin, writer)
	duration := time.Since(startTime)

	if err != nil {
		return nil, HandleCLIError(g.cliName, err, nil)
	}

	inputTokens := EstimateTokens(prompt)
	outputTokens := 100

	metrics := &core.Metrics{
		Duration:     duration,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		Model:        g.model,
	}

	log.WithFields(map[string]interface{}{
		"agent_name": g.agentName,
		"duration":   duration.String(),
	}).Info("Generic CLI streaming message completed")

	return metrics, nil
}

func (g *GenericCLIAdapter) HealthCheck(ctx context.Context) error {
	if g.cliPath == "" {
		log.WithField("agent_name", g.agentName).Error("Generic CLI health check failed: not initialized")
		return fmt.Errorf("Generic CLI not initialized")
	}

	log.WithField("agent_name", g.agentName).Debug("Starting generic CLI health check")

	// Try --version first, then --help
	for _, flag := range []string{"--version", "--help", "-v", "-h"} {
		cmd := exec.CommandContext(ctx, g.cliPath, flag)
		output, err := cmd.CombinedOutput()
		if err == nil && len(output) > 0 {
			log.WithField("agent_name", g.agentName).Info("Generic CLI health check passed")
			return nil
		}
	}

	// If no flag works, just check if the binary is executable
	cmd := exec.Command(g.cliPath)
	if err := cmd.Start(); err != nil {
		log.WithField("agent_name", g.agentName).WithError(err).Error("Generic CLI health check failed: CLI not responding")
		return fmt.Errorf("CLI '%s' cannot be executed: %w", g.cliName, err)
	}
	// Kill the process
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}

	log.WithField("agent_name", g.agentName).Info("Generic CLI health check passed")
	return nil
}

// GetCLIVersion returns the version of the CLI.
func (g *GenericCLIAdapter) GetCLIVersion() string {
	if g.cliPath == "" {
		return "unknown"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try common version flags
	for _, flag := range []string{"--version", "-v", "version"} {
		cmd := exec.CommandContext(ctx, g.cliPath, flag)
		output, err := cmd.Output()
		if err == nil && len(output) > 0 {
			version := strings.TrimSpace(string(output))
			if lines := strings.Split(version, "\n"); len(lines) > 0 {
				return lines[0]
			}
			return version
		}
	}

	return "unknown"
}

func init() {
	adapters.Register("cli-generic", NewGenericCLIAdapter)
}
