// Package main provides an example custom adapter implementation for AgentPipe v2.
//
// This example demonstrates how to implement the AgentAdapter interface
// to connect AgentPipe to a custom AI provider or service.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/ASRagab/agentpipe/pkg/log"
	"github.com/ASRagab/agentpipe/pkg/v2/adapters"
	"github.com/ASRagab/agentpipe/pkg/v2/core"
	"github.com/ASRagab/agentpipe/pkg/v2/errors"
)

// EchoAdapter is an example custom adapter that echoes back the last user message.
// This demonstrates the adapter interface without requiring an external API.
// Replace this with your actual API implementation.
type EchoAdapter struct {
	// Configuration fields
	model        string
	temperature  float64
	maxTokens    int
	systemPrompt string

	// State fields
	agentID   string
	agentName string

	// HTTP client for API calls (if your adapter uses HTTP)
	httpClient *http.Client
}

// NewEchoAdapter creates a new instance of the echo adapter.
// This function is registered with the adapter registry.
func NewEchoAdapter() adapters.AgentAdapter {
	return &EchoAdapter{
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Initialize configures the adapter with the agent configuration.
// This is called once when the agent is set up.
func (e *EchoAdapter) Initialize(agent core.Agent) error {
	// Store agent identifiers for logging and error messages
	e.agentID = agent.ID
	e.agentName = agent.Name
	e.model = agent.Model

	// Extract adapter-specific configuration
	e.temperature = agent.Config.Temperature
	e.maxTokens = agent.Config.MaxTokens
	e.systemPrompt = agent.Config.SystemPrompt

	// Validate required configuration
	if e.model == "" {
		return fmt.Errorf("model is required for EchoAdapter")
	}

	// Example: Get API key from environment variable
	// In a real adapter, you would use this key for authentication
	envVar := agent.Config.APIKeyEnvVar
	if envVar == "" {
		envVar = "ECHO_API_KEY" // Default environment variable
	}

	apiKey := os.Getenv(envVar)
	if apiKey == "" {
		// For the echo adapter, we don't require an API key
		// A real adapter would return an error here:
		// return fmt.Errorf("%s environment variable is required", envVar)
		log.WithFields(map[string]interface{}{
			"agent_id":   agent.ID,
			"agent_name": agent.Name,
		}).Debug("No API key configured (echo adapter doesn't require one)")
	}

	log.WithFields(map[string]interface{}{
		"agent_id":   agent.ID,
		"agent_name": agent.Name,
		"model":      e.model,
	}).Info("EchoAdapter initialized successfully")

	return nil
}

// IsAvailable checks if the adapter is properly configured and can be used.
// For API adapters, this typically checks if the API key is set.
func (e *EchoAdapter) IsAvailable() bool {
	// For the echo adapter, we're always available
	// A real adapter would check: return e.apiKey != ""
	return true
}

// GetModel returns the configured model name.
func (e *EchoAdapter) GetModel() string {
	return e.model
}

// HealthCheck performs a minimal test request to verify the API is accessible.
// This is called during preflight checks before starting a conversation.
func (e *EchoAdapter) HealthCheck(ctx context.Context) error {
	// For the echo adapter, we just verify initialization
	if e.model == "" {
		return fmt.Errorf("adapter not initialized")
	}

	// A real adapter would make a minimal API call here:
	// req := minimalRequest{Model: e.model, Messages: []message{{Role: "user", Content: "test"}}}
	// _, err := e.doRequest(ctx, req)
	// return err

	return nil
}

// SendMessage sends messages to the AI and returns the response.
// This is the synchronous, non-streaming version.
func (e *EchoAdapter) SendMessage(ctx context.Context, messages []core.Message) (string, *core.Metrics, error) {
	if len(messages) == 0 {
		return "", nil, nil
	}

	startTime := time.Now()

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return "", nil, errors.NewTimeoutError(e.agentID, e.agentName, ctx.Err())
	default:
	}

	// Find the last user message to echo
	var lastUserMessage string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == core.RoleUser {
			lastUserMessage = messages[i].Content
			break
		}
	}

	if lastUserMessage == "" {
		return "", nil, fmt.Errorf("no user message found")
	}

	// Build the response
	// In a real adapter, this is where you would:
	// 1. Convert messages to your API format
	// 2. Make the HTTP request
	// 3. Parse the response
	// 4. Handle errors with proper classification
	var response string
	if e.systemPrompt != "" {
		response = fmt.Sprintf("[%s] Echo: %s", e.systemPrompt, lastUserMessage)
	} else {
		response = fmt.Sprintf("Echo: %s", lastUserMessage)
	}

	// Simulate some processing time
	time.Sleep(100 * time.Millisecond)

	duration := time.Since(startTime)

	// Create metrics
	// In a real adapter, you would get these from the API response
	metrics := &core.Metrics{
		Duration:     duration,
		InputTokens:  len(lastUserMessage) / 4, // Rough estimate
		OutputTokens: len(response) / 4,        // Rough estimate
		TotalTokens:  (len(lastUserMessage) + len(response)) / 4,
		Model:        e.model,
		Cost:         0.0001, // Example cost
	}

	return response, metrics, nil
}

// StreamMessage sends messages and streams the response to the writer.
// Returns the final metrics after streaming completes.
func (e *EchoAdapter) StreamMessage(ctx context.Context, messages []core.Message, writer io.Writer) (*core.Metrics, error) {
	if len(messages) == 0 {
		return nil, nil
	}

	startTime := time.Now()

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, errors.NewTimeoutError(e.agentID, e.agentName, ctx.Err())
	default:
	}

	// Find the last user message
	var lastUserMessage string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == core.RoleUser {
			lastUserMessage = messages[i].Content
			break
		}
	}

	if lastUserMessage == "" {
		return nil, fmt.Errorf("no user message found")
	}

	// Build the response
	var response string
	if e.systemPrompt != "" {
		response = fmt.Sprintf("[%s] Echo: %s", e.systemPrompt, lastUserMessage)
	} else {
		response = fmt.Sprintf("Echo: %s", lastUserMessage)
	}

	// Stream the response word by word
	// In a real adapter, you would parse SSE events from the API
	words := []byte(response)
	for i, char := range words {
		select {
		case <-ctx.Done():
			return nil, errors.NewTimeoutError(e.agentID, e.agentName, ctx.Err())
		default:
		}

		// Write one character at a time (simulating streaming)
		if _, err := writer.Write([]byte{byte(char)}); err != nil {
			return nil, fmt.Errorf("failed to write stream content: %w", err)
		}

		// Small delay to simulate streaming
		if i < len(words)-1 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	duration := time.Since(startTime)

	// Create metrics
	metrics := &core.Metrics{
		Duration:     duration,
		InputTokens:  len(lastUserMessage) / 4,
		OutputTokens: len(response) / 4,
		TotalTokens:  (len(lastUserMessage) + len(response)) / 4,
		Model:        e.model,
		Cost:         0.0001,
	}

	return metrics, nil
}

// === Example helper methods for a real HTTP-based adapter ===

// chatMessage represents a message in the API format.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatRequest represents an API request.
type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

// chatResponse represents an API response.
type chatResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message      chatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// buildAPIMessages converts core.Message to the API format.
// This is a common pattern used by most adapters.
func (e *EchoAdapter) buildAPIMessages(messages []core.Message) []chatMessage {
	apiMessages := make([]chatMessage, 0, len(messages)+1)

	// Add system prompt if configured
	if e.systemPrompt != "" {
		apiMessages = append(apiMessages, chatMessage{
			Role:    "system",
			Content: e.systemPrompt,
		})
	}

	// Convert each message
	for _, msg := range messages {
		var role string
		switch msg.Role {
		case core.RoleSystem:
			role = "system"
		case core.RoleUser:
			role = "user"
		case core.RoleAgent:
			role = "assistant"
		default:
			continue // Skip unknown roles
		}

		apiMessages = append(apiMessages, chatMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	return apiMessages
}

// doRequest performs an HTTP request to the API.
// This is an example of how to make API calls.
func (e *EchoAdapter) doRequest(ctx context.Context, apiURL string, req chatRequest) (*chatResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	// httpReq.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.httpClient.Do(httpReq)
	if err != nil {
		// Classify connection errors
		return nil, errors.NewNetworkError(e.agentID, e.agentName, err)
	}
	defer resp.Body.Close()

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		return nil, e.handleErrorResponse(resp)
	}

	// Parse response
	var result chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.NewInvalidResponseError(e.agentID, e.agentName, "failed to decode response", err)
	}

	return &result, nil
}

// handleErrorResponse handles HTTP error responses.
// This demonstrates proper error classification.
func (e *EchoAdapter) handleErrorResponse(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	message := string(body)

	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return errors.NewAuthError(e.agentID, e.agentName, resp.StatusCode, fmt.Errorf(message))
	case http.StatusTooManyRequests:
		// Try to parse retry-after header
		retryAfter := 60 * time.Second // Default
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if d, err := time.ParseDuration(ra + "s"); err == nil {
				retryAfter = d
			}
		}
		return errors.NewRateLimitError(e.agentID, e.agentName, retryAfter)
	case http.StatusServiceUnavailable, http.StatusBadGateway, http.StatusGatewayTimeout:
		return errors.NewNetworkError(e.agentID, e.agentName, fmt.Errorf(message))
	default:
		return errors.NewAgentError(e.agentID, e.agentName, errors.ErrTypeUnknown, message, nil)
	}
}

// Register the adapter with the default registry.
// This allows the adapter to be used in configuration files.
func init() {
	adapters.Register("echo", NewEchoAdapter)
}
