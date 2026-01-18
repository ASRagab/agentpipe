// Package adapters provides the interface and implementations for AI agent adapters.
// Each adapter connects to a different AI provider (OpenRouter, Claude API, etc.)
// and handles the specifics of API communication, authentication, and message formatting.
package adapters

import (
	"context"
	"io"

	"github.com/kevinelliott/agentpipe/pkg/v2/core"
)

// AgentAdapter defines the interface that all AI provider adapters must implement.
type AgentAdapter interface {
	// Initialize configures the adapter with the given agent configuration.
	Initialize(agent core.Agent) error

	// SendMessage sends messages to the AI and returns the response.
	// This is the synchronous, non-streaming version.
	SendMessage(ctx context.Context, messages []core.Message) (string, *core.Metrics, error)

	// StreamMessage sends messages and streams the response to the writer.
	// Returns the final metrics after streaming completes.
	StreamMessage(ctx context.Context, messages []core.Message, writer io.Writer) (*core.Metrics, error)

	// IsAvailable checks if the adapter is properly configured and can be used.
	// For API adapters, this typically checks if the API key is set.
	IsAvailable() bool

	// GetModel returns the model name this adapter is configured to use.
	GetModel() string

	// HealthCheck performs a minimal request to verify the API is accessible.
	HealthCheck(ctx context.Context) error
}

// AdapterFactory is a function that creates a new adapter instance.
type AdapterFactory func() AgentAdapter
