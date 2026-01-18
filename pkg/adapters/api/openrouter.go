// Package api provides API-based adapters for AI providers.
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

	"github.com/ASRagab/agentpipe/pkg/adapters"
	"github.com/ASRagab/agentpipe/pkg/core"
	"github.com/ASRagab/agentpipe/pkg/errors"
	"github.com/ASRagab/agentpipe/pkg/log"
)

const (
	openRouterBaseURL = "https://openrouter.ai/api/v1"
	defaultTimeout    = 120 * time.Second
	maxRetries        = 3
)

// OpenRouterAdapter implements the AgentAdapter interface for OpenRouter's API.
type OpenRouterAdapter struct {
	apiKey          string
	model           string
	temperature     float64
	maxTokens       int
	httpClient      *http.Client
	systemPrompt    string
	agentID         string
	agentName       string
	errorClassifier *HTTPErrorClassifier
}

// NewOpenRouterAdapter creates a new OpenRouter adapter instance.
func NewOpenRouterAdapter() adapters.AgentAdapter {
	return &OpenRouterAdapter{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// Initialize configures the adapter with the agent configuration.
func (o *OpenRouterAdapter) Initialize(agent core.Agent) error {
	// Store agent info for error classification
	o.agentID = agent.ID
	o.agentName = agent.Name
	o.errorClassifier = NewHTTPErrorClassifier(agent.ID, agent.Name)

	// Get API key from environment variable
	envVar := agent.Config.APIKeyEnvVar
	if envVar == "" {
		envVar = "OPENROUTER_API_KEY"
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
	o.apiKey = apiKey

	if agent.Model == "" {
		return fmt.Errorf("model must be specified for OpenRouter adapter")
	}
	o.model = agent.Model

	o.temperature = agent.Config.Temperature
	o.maxTokens = agent.Config.MaxTokens
	o.systemPrompt = agent.Config.SystemPrompt

	log.WithFields(map[string]interface{}{
		"agent_id":   agent.ID,
		"agent_name": agent.Name,
		"model":      o.model,
	}).Info("OpenRouter adapter initialized successfully")

	return nil
}

// IsAvailable checks if the API key is configured.
func (o *OpenRouterAdapter) IsAvailable() bool {
	return o.apiKey != ""
}

// GetModel returns the configured model name.
func (o *OpenRouterAdapter) GetModel() string {
	return o.model
}

// HealthCheck performs a minimal test request.
func (o *OpenRouterAdapter) HealthCheck(ctx context.Context) error {
	if o.apiKey == "" {
		return fmt.Errorf("OpenRouter adapter not initialized")
	}

	maxTokens := 1
	req := chatCompletionRequest{
		Model: o.model,
		Messages: []chatMessage{
			{Role: "user", Content: "test"},
		},
		MaxTokens: &maxTokens,
	}

	_, _, err := o.doRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// SendMessage sends messages and returns the response.
func (o *OpenRouterAdapter) SendMessage(ctx context.Context, messages []core.Message) (string, *core.Metrics, error) {
	if len(messages) == 0 {
		return "", nil, nil
	}

	apiMessages := o.buildAPIMessages(messages)
	req := chatCompletionRequest{
		Model:    o.model,
		Messages: apiMessages,
	}

	if o.temperature > 0 {
		req.Temperature = &o.temperature
	}
	if o.maxTokens > 0 {
		req.MaxTokens = &o.maxTokens
	}

	startTime := time.Now()
	resp, duration, err := o.doRequestWithRetry(ctx, req)
	if err != nil {
		return "", nil, err
	}

	if len(resp.Choices) == 0 {
		return "", nil, fmt.Errorf("no response from OpenRouter")
	}

	content := strings.TrimSpace(resp.Choices[0].Message.Content)

	var metrics *core.Metrics
	if resp.Usage != nil {
		metrics = &core.Metrics{
			Duration:     duration,
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
			TotalTokens:  resp.Usage.TotalTokens,
			Model:        resp.Model,
			Cost:         o.estimateCost(resp.Usage.PromptTokens, resp.Usage.CompletionTokens),
		}
	} else {
		metrics = &core.Metrics{
			Duration: time.Since(startTime),
			Model:    o.model,
		}
	}

	return content, metrics, nil
}

// StreamMessage sends messages and streams the response.
func (o *OpenRouterAdapter) StreamMessage(ctx context.Context, messages []core.Message, writer io.Writer) (*core.Metrics, error) {
	if len(messages) == 0 {
		return nil, nil
	}

	apiMessages := o.buildAPIMessages(messages)
	req := chatCompletionRequest{
		Model:    o.model,
		Messages: apiMessages,
		Stream:   true,
	}

	if o.temperature > 0 {
		req.Temperature = &o.temperature
	}
	if o.maxTokens > 0 {
		req.MaxTokens = &o.maxTokens
	}

	startTime := time.Now()
	usage, err := o.doStreamRequest(ctx, req, writer)
	duration := time.Since(startTime)

	if err != nil {
		return nil, err
	}

	var metrics *core.Metrics
	if usage != nil {
		metrics = &core.Metrics{
			Duration:     duration,
			InputTokens:  usage.PromptTokens,
			OutputTokens: usage.CompletionTokens,
			TotalTokens:  usage.TotalTokens,
			Model:        o.model,
			Cost:         o.estimateCost(usage.PromptTokens, usage.CompletionTokens),
		}
	} else {
		metrics = &core.Metrics{
			Duration: duration,
			Model:    o.model,
		}
	}

	return metrics, nil
}

// buildAPIMessages converts core.Message to the API format.
func (o *OpenRouterAdapter) buildAPIMessages(messages []core.Message) []chatMessage {
	apiMessages := make([]chatMessage, 0, len(messages)+1)

	// Add system prompt if configured
	if o.systemPrompt != "" {
		apiMessages = append(apiMessages, chatMessage{
			Role:    "system",
			Content: o.systemPrompt,
		})
	}

	for _, msg := range messages {
		var role, content string

		switch msg.Role {
		case core.RoleSystem:
			role = "system"
			content = msg.Content
		case core.RoleUser:
			role = "user"
			content = msg.Content
		case core.RoleAgent:
			role = "assistant"
			content = msg.Content
		default:
			continue
		}

		apiMessages = append(apiMessages, chatMessage{
			Role:    role,
			Content: content,
		})
	}

	return apiMessages
}

// doRequestWithRetry performs the request with retry logic.
// Uses the v2 errors package for proper error classification and retry decisions.
func (o *OpenRouterAdapter) doRequestWithRetry(ctx context.Context, req chatCompletionRequest) (*chatCompletionResponse, time.Duration, error) {
	var lastErr error
	startTime := time.Now()

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Check for rate limit retry-after first
			var backoff time.Duration
			if retryAfter := GetRetryAfter(lastErr); retryAfter > 0 {
				backoff = retryAfter
				log.WithFields(map[string]interface{}{
					"agent_id":    o.agentID,
					"agent_name":  o.agentName,
					"attempt":     attempt + 1,
					"max":         maxRetries + 1,
					"retry_after": backoff.String(),
				}).Info("Waiting for rate limit retry-after before retry")
			} else {
				// Exponential backoff: 1s, 2s, 4s
				shift := min(attempt-1, 30)
				//nolint:gosec // shift is bounded, safe from overflow
				backoff = time.Duration(1<<uint(shift)) * time.Second
			}

			log.WithFields(map[string]interface{}{
				"agent_id":   o.agentID,
				"agent_name": o.agentName,
				"attempt":    attempt + 1,
				"max":        maxRetries + 1,
				"delay":      backoff.String(),
			}).Info("Retrying OpenRouter request")

			select {
			case <-ctx.Done():
				return nil, 0, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, _, err := o.doRequest(ctx, req)
		duration := time.Since(startTime)
		if err != nil {
			lastErr = err

			// Log the error with attempt info
			log.WithFields(map[string]interface{}{
				"agent_id":   o.agentID,
				"agent_name": o.agentName,
				"attempt":    attempt + 1,
				"max":        maxRetries + 1,
				"error":      err.Error(),
				"retryable":  errors.IsRetryable(err),
			}).Warn("OpenRouter request failed")

			// Only retry if error is retryable
			if !errors.IsRetryable(err) {
				return nil, duration, err
			}
			continue
		}

		if attempt > 0 {
			log.WithFields(map[string]interface{}{
				"agent_id":   o.agentID,
				"agent_name": o.agentName,
				"attempt":    attempt + 1,
			}).Info("OpenRouter request succeeded after retry")
		}

		return resp, duration, nil
	}

	// All retries exhausted - wrap with retry count
	finalErr := errors.WrapError(o.agentID, o.agentName, lastErr)
	if finalErr != nil {
		finalErr = finalErr.WithRetryCount(maxRetries + 1)
	}
	return nil, time.Since(startTime), finalErr
}

// doRequest performs a single HTTP request.
func (o *OpenRouterAdapter) doRequest(ctx context.Context, req chatCompletionRequest) (*chatCompletionResponse, time.Duration, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", openRouterBaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	o.setHeaders(httpReq)

	startTime := time.Now()
	resp, err := o.httpClient.Do(httpReq)
	duration := time.Since(startTime)

	if err != nil {
		// Classify connection-level errors (timeout, DNS, connection refused)
		if o.errorClassifier != nil {
			return nil, duration, o.errorClassifier.ClassifyConnectionError(err)
		}
		return nil, duration, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, duration, o.handleErrorResponse(resp)
	}

	var result chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, duration, errors.NewInvalidResponseError(o.agentID, o.agentName, "failed to decode response", err)
	}

	if result.Error != nil {
		return nil, duration, errors.NewAgentError(o.agentID, o.agentName, errors.ErrTypeUnknown, result.Error.Message, nil)
	}

	return &result, duration, nil
}

// doStreamRequest performs a streaming HTTP request.
func (o *OpenRouterAdapter) doStreamRequest(ctx context.Context, req chatCompletionRequest, writer io.Writer) (*chatUsage, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", openRouterBaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	o.setHeaders(httpReq)

	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		// Classify connection-level errors
		if o.errorClassifier != nil {
			return nil, o.errorClassifier.ClassifyConnectionError(err)
		}
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, o.handleErrorResponse(resp)
	}

	return o.processStream(resp.Body, writer)
}

// processStream reads and processes the SSE stream.
func (o *OpenRouterAdapter) processStream(body io.Reader, writer io.Writer) (*chatUsage, error) {
	scanner := bufio.NewScanner(body)
	var usage *chatUsage

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk chatStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			log.WithError(err).Warn("failed to parse stream chunk")
			continue
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			if _, writeErr := writer.Write([]byte(chunk.Choices[0].Delta.Content)); writeErr != nil {
				return usage, fmt.Errorf("failed to write stream content: %w", writeErr)
			}
		}

		if chunk.Usage != nil {
			usage = chunk.Usage
		}
	}

	if err := scanner.Err(); err != nil {
		return usage, fmt.Errorf("error reading stream: %w", err)
	}

	return usage, nil
}

// setHeaders sets the required HTTP headers.
func (o *OpenRouterAdapter) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)
	req.Header.Set("HTTP-Referer", "https://github.com/ASRagab/agentpipe")
	req.Header.Set("X-Title", "AgentPipe")
}

// handleErrorResponse parses an error response and returns a properly classified AgentError.
func (o *OpenRouterAdapter) handleErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// Can't read body, classify based on status code alone
		if o.errorClassifier != nil {
			return o.errorClassifier.ClassifyHTTPError(resp, fmt.Sprintf("HTTP %d (failed to read error body)", resp.StatusCode))
		}
		return fmt.Errorf("HTTP %d (failed to read error body: %w)", resp.StatusCode, err)
	}

	var errorResp struct {
		Error *chatError `json:"error"`
	}

	var errorMessage string
	if err := json.Unmarshal(body, &errorResp); err != nil {
		errorMessage = string(body)
	} else if errorResp.Error != nil {
		errorMessage = errorResp.Error.Message
	} else {
		errorMessage = string(body)
	}

	// Use the error classifier to create a properly typed error
	if o.errorClassifier != nil {
		return o.errorClassifier.ClassifyHTTPError(resp, errorMessage)
	}

	return fmt.Errorf("HTTP %d: %s", resp.StatusCode, errorMessage)
}

// estimateCost calculates the cost based on token usage.
// This is a rough estimate; actual pricing varies by model.
func (o *OpenRouterAdapter) estimateCost(inputTokens, outputTokens int) float64 {
	// Default pricing (varies by model)
	inputCostPer1K := 0.0001
	outputCostPer1K := 0.0002

	return (float64(inputTokens) * inputCostPer1K / 1000) + (float64(outputTokens) * outputCostPer1K / 1000)
}

// API types for OpenRouter

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

type chatCompletionResponse struct {
	ID      string       `json:"id"`
	Model   string       `json:"model"`
	Choices []chatChoice `json:"choices"`
	Usage   *chatUsage   `json:"usage,omitempty"`
	Error   *chatError   `json:"error,omitempty"`
}

type chatChoice struct {
	Index        int         `json:"index"`
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type chatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type chatError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

type chatStreamChunk struct {
	ID      string             `json:"id"`
	Model   string             `json:"model"`
	Choices []chatStreamChoice `json:"choices"`
	Usage   *chatUsage         `json:"usage,omitempty"`
}

type chatStreamChoice struct {
	Index        int             `json:"index"`
	Delta        chatStreamDelta `json:"delta"`
	FinishReason *string         `json:"finish_reason"`
}

type chatStreamDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

func init() {
	adapters.Register("openrouter", NewOpenRouterAdapter)
}
