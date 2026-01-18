package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/log"
	"github.com/kevinelliott/agentpipe/pkg/v2/adapters"
	"github.com/kevinelliott/agentpipe/pkg/v2/core"
)

const (
	claudeBaseURL    = "https://api.anthropic.com/v1"
	claudeAPIVersion = "2023-06-01"
)

// ClaudeAPIAdapter implements the AgentAdapter interface for Anthropic's Claude API.
type ClaudeAPIAdapter struct {
	apiKey       string
	model        string
	maxTokens    int
	temperature  float64
	systemPrompt string
	httpClient   *http.Client
}

// NewClaudeAPIAdapter creates a new Claude API adapter instance.
func NewClaudeAPIAdapter() adapters.AgentAdapter {
	return &ClaudeAPIAdapter{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// Initialize configures the adapter with the agent configuration.
func (c *ClaudeAPIAdapter) Initialize(agent core.Agent) error {
	// Get API key from environment variable
	envVar := agent.Config.APIKeyEnvVar
	if envVar == "" {
		envVar = "ANTHROPIC_API_KEY"
	}

	apiKey := os.Getenv(envVar)
	if apiKey == "" {
		log.WithFields(map[string]interface{}{
			"agent_id":   agent.ID,
			"agent_name": agent.Name,
			"env_var":    envVar,
		}).Error("API key environment variable not set")
		return fmt.Errorf("%s environment variable is required", envVar)
	}
	c.apiKey = apiKey

	if agent.Model == "" {
		return fmt.Errorf("model must be specified for Claude adapter")
	}
	c.model = agent.Model

	// Set defaults if not specified
	c.maxTokens = agent.Config.MaxTokens
	if c.maxTokens == 0 {
		c.maxTokens = 1024 // Claude requires max_tokens
	}

	c.temperature = agent.Config.Temperature
	c.systemPrompt = agent.Config.SystemPrompt

	log.WithFields(map[string]interface{}{
		"agent_id":   agent.ID,
		"agent_name": agent.Name,
		"model":      c.model,
	}).Info("Claude API adapter initialized successfully")

	return nil
}

// IsAvailable checks if the API key is configured.
func (c *ClaudeAPIAdapter) IsAvailable() bool {
	return c.apiKey != ""
}

// GetModel returns the configured model name.
func (c *ClaudeAPIAdapter) GetModel() string {
	return c.model
}

// HealthCheck performs a minimal test request.
func (c *ClaudeAPIAdapter) HealthCheck(ctx context.Context) error {
	if c.apiKey == "" {
		return fmt.Errorf("Claude adapter not initialized")
	}

	req := claudeRequest{
		Model:     c.model,
		MaxTokens: 1,
		Messages: []claudeMessage{
			{Role: "user", Content: "test"},
		},
	}

	_, _, err := c.doRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// SendMessage sends messages and returns the response.
func (c *ClaudeAPIAdapter) SendMessage(ctx context.Context, messages []core.Message) (string, *core.Metrics, error) {
	if len(messages) == 0 {
		return "", nil, nil
	}

	apiMessages := c.buildAPIMessages(messages)
	req := claudeRequest{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		Messages:  apiMessages,
	}

	if c.systemPrompt != "" {
		req.System = c.systemPrompt
	}
	if c.temperature > 0 {
		req.Temperature = &c.temperature
	}

	startTime := time.Now()
	resp, duration, err := c.doRequestWithRetry(ctx, req)
	if err != nil {
		return "", nil, err
	}

	// Extract text content from response
	var content string
	for _, block := range resp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}
	content = strings.TrimSpace(content)

	metrics := &core.Metrics{
		Duration:     duration,
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
		TotalTokens:  resp.Usage.InputTokens + resp.Usage.OutputTokens,
		Model:        resp.Model,
		Cost:         c.estimateCost(resp.Usage.InputTokens, resp.Usage.OutputTokens),
	}

	if metrics.Duration == 0 {
		metrics.Duration = time.Since(startTime)
	}

	return content, metrics, nil
}

// StreamMessage sends messages and streams the response.
func (c *ClaudeAPIAdapter) StreamMessage(ctx context.Context, messages []core.Message, writer io.Writer) (*core.Metrics, error) {
	if len(messages) == 0 {
		return nil, nil
	}

	apiMessages := c.buildAPIMessages(messages)
	req := claudeRequest{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		Messages:  apiMessages,
		Stream:    true,
	}

	if c.systemPrompt != "" {
		req.System = c.systemPrompt
	}
	if c.temperature > 0 {
		req.Temperature = &c.temperature
	}

	startTime := time.Now()
	usage, err := c.doStreamRequest(ctx, req, writer)
	duration := time.Since(startTime)

	if err != nil {
		return nil, err
	}

	metrics := &core.Metrics{
		Duration:     duration,
		InputTokens:  usage.InputTokens,
		OutputTokens: usage.OutputTokens,
		TotalTokens:  usage.InputTokens + usage.OutputTokens,
		Model:        c.model,
		Cost:         c.estimateCost(usage.InputTokens, usage.OutputTokens),
	}

	return metrics, nil
}

// buildAPIMessages converts core.Message to Claude API format.
func (c *ClaudeAPIAdapter) buildAPIMessages(messages []core.Message) []claudeMessage {
	apiMessages := make([]claudeMessage, 0, len(messages))

	for _, msg := range messages {
		var role string

		switch msg.Role {
		case core.RoleUser:
			role = "user"
		case core.RoleAgent:
			role = "assistant"
		case core.RoleSystem:
			// System messages are handled separately in Claude's API
			continue
		default:
			continue
		}

		apiMessages = append(apiMessages, claudeMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	// Claude requires alternating user/assistant messages
	// Add a default user message if empty
	if len(apiMessages) == 0 {
		apiMessages = append(apiMessages, claudeMessage{
			Role:    "user",
			Content: "Hello",
		})
	}

	return apiMessages
}

// doRequestWithRetry performs the request with retry logic.
func (c *ClaudeAPIAdapter) doRequestWithRetry(ctx context.Context, req claudeRequest) (*claudeResponse, time.Duration, error) {
	var lastErr error
	startTime := time.Now()

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			shift := min(attempt-1, 30)
			//nolint:gosec // shift is bounded, safe from overflow
			backoff := time.Duration(1<<uint(shift)) * time.Second

			select {
			case <-ctx.Done():
				return nil, 0, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, _, err := c.doRequest(ctx, req)
		duration := time.Since(startTime)
		if err != nil {
			lastErr = err
			if shouldRetry(err) {
				continue
			}
			return nil, duration, err
		}

		return resp, duration, nil
	}

	return nil, time.Since(startTime), fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// doRequest performs a single HTTP request.
func (c *ClaudeAPIAdapter) doRequest(ctx context.Context, req claudeRequest) (*claudeResponse, time.Duration, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", claudeBaseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq)

	startTime := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	duration := time.Since(startTime)

	if err != nil {
		return nil, duration, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, duration, c.handleErrorResponse(resp)
	}

	var result claudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, duration, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, duration, nil
}

// doStreamRequest performs a streaming HTTP request.
func (c *ClaudeAPIAdapter) doStreamRequest(ctx context.Context, req claudeRequest, writer io.Writer) (*claudeUsage, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", claudeBaseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	return c.processStream(resp.Body, writer)
}

// processStream reads and processes the Claude SSE stream.
func (c *ClaudeAPIAdapter) processStream(body io.Reader, writer io.Writer) (*claudeUsage, error) {
	scanner := bufio.NewScanner(body)
	usage := &claudeUsage{}

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		var event claudeStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			log.WithError(err).Warn("failed to parse stream event")
			continue
		}

		switch event.Type {
		case "content_block_delta":
			if event.Delta != nil && event.Delta.Text != "" {
				if _, writeErr := writer.Write([]byte(event.Delta.Text)); writeErr != nil {
					return usage, fmt.Errorf("failed to write stream content: %w", writeErr)
				}
			}
		case "message_start":
			if event.Message != nil && event.Message.Usage != nil {
				usage.InputTokens = event.Message.Usage.InputTokens
			}
		case "message_delta":
			if event.Usage != nil {
				usage.OutputTokens = event.Usage.OutputTokens
			}
		case "message_stop":
			// End of stream
			return usage, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return usage, fmt.Errorf("error reading stream: %w", err)
	}

	return usage, nil
}

// setHeaders sets the required HTTP headers for Claude API.
func (c *ClaudeAPIAdapter) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", claudeAPIVersion)
}

// handleErrorResponse parses an error response.
func (c *ClaudeAPIAdapter) handleErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP %d (failed to read error body: %w)", resp.StatusCode, err)
	}

	var errorResp claudeErrorResponse
	if err := json.Unmarshal(body, &errorResp); err != nil {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	if errorResp.Error != nil {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, errorResp.Error.Message)
	}

	return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
}

// estimateCost calculates the cost based on token usage.
func (c *ClaudeAPIAdapter) estimateCost(inputTokens, outputTokens int) float64 {
	// Claude pricing (varies by model, this is for claude-3-haiku)
	// Haiku: $0.25 / 1M input, $1.25 / 1M output
	inputCostPerMillion := 0.25
	outputCostPerMillion := 1.25

	// Adjust for other models
	if strings.Contains(c.model, "sonnet") {
		inputCostPerMillion = 3.0
		outputCostPerMillion = 15.0
	} else if strings.Contains(c.model, "opus") {
		inputCostPerMillion = 15.0
		outputCostPerMillion = 75.0
	}

	return (float64(inputTokens) * inputCostPerMillion / 1_000_000) +
		(float64(outputTokens) * outputCostPerMillion / 1_000_000)
}

// Claude API types

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	Messages    []claudeMessage `json:"messages"`
	System      string          `json:"system,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

type claudeResponse struct {
	ID           string               `json:"id"`
	Type         string               `json:"type"`
	Role         string               `json:"role"`
	Model        string               `json:"model"`
	Content      []claudeContentBlock `json:"content"`
	Usage        claudeUsage          `json:"usage"`
	StopReason   string               `json:"stop_reason"`
	StopSequence *string              `json:"stop_sequence"`
}

type claudeContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type claudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type claudeErrorResponse struct {
	Type  string       `json:"type"`
	Error *claudeError `json:"error"`
}

type claudeError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type claudeStreamEvent struct {
	Type    string               `json:"type"`
	Index   int                  `json:"index,omitempty"`
	Delta   *claudeStreamDelta   `json:"delta,omitempty"`
	Message *claudeStreamMessage `json:"message,omitempty"`
	Usage   *claudeUsage         `json:"usage,omitempty"`
}

type claudeStreamDelta struct {
	Type string `json:"type,omitempty"`
	Text string `json:"text,omitempty"`
}

type claudeStreamMessage struct {
	ID    string       `json:"id"`
	Model string       `json:"model"`
	Usage *claudeUsage `json:"usage,omitempty"`
}

func init() {
	adapters.Register("claude-api", NewClaudeAPIAdapter)
}
