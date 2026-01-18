// Package mock provides a mock adapter for testing without real API calls.
package mock

import (
	"context"
	"io"
	"time"

	"github.com/ASRagab/agentpipe/pkg/v2/adapters"
	"github.com/ASRagab/agentpipe/pkg/v2/core"
)

// MockAdapter is a configurable adapter for testing purposes.
// It can simulate responses, delays, errors, and streaming.
type MockAdapter struct {
	// Response is the content to return from SendMessage.
	Response string
	// Delay is how long to wait before responding.
	Delay time.Duration
	// Error is an error to return (if non-nil).
	Error error
	// StreamChunks are the chunks to write during StreamMessage.
	StreamChunks []string
	// StreamDelay is the delay between streaming chunks.
	StreamDelay time.Duration
	// HealthCheckError is the error to return from HealthCheck.
	HealthCheckError error
	// Available controls whether IsAvailable returns true.
	Available bool
	// Model is the model name to return from GetModel.
	Model string

	// agent holds the initialized agent configuration.
	agent core.Agent

	// Metrics to return (optional - if nil, default metrics are created).
	Metrics *core.Metrics

	// SendMessageCalls tracks how many times SendMessage was called.
	SendMessageCalls int
	// StreamMessageCalls tracks how many times StreamMessage was called.
	StreamMessageCalls int
	// LastMessages stores the last messages passed to SendMessage/StreamMessage.
	LastMessages []core.Message

	// OnSendMessage is an optional callback for custom SendMessage behavior.
	// If set, it overrides the default Response/Error behavior.
	OnSendMessage func(ctx context.Context, messages []core.Message) (string, *core.Metrics, error)
	// OnStreamMessage is an optional callback for custom StreamMessage behavior.
	// If set, it overrides the default StreamChunks/Error behavior.
	OnStreamMessage func(ctx context.Context, messages []core.Message, writer io.Writer) (*core.Metrics, error)
}

// NewMockAdapter creates a new mock adapter with sensible defaults.
func NewMockAdapter() *MockAdapter {
	return &MockAdapter{
		Response:  "Mock response",
		Available: true,
		Model:     "mock-model",
	}
}

// Initialize stores the agent configuration.
func (m *MockAdapter) Initialize(agent core.Agent) error {
	m.agent = agent
	return nil
}

// SendMessage simulates sending a message to an AI.
// If OnSendMessage is set, it uses that callback. Otherwise:
// It waits for Delay, returns Error if set, otherwise returns Response.
func (m *MockAdapter) SendMessage(ctx context.Context, messages []core.Message) (string, *core.Metrics, error) {
	m.SendMessageCalls++
	m.LastMessages = messages

	// Use callback if provided
	if m.OnSendMessage != nil {
		return m.OnSendMessage(ctx, messages)
	}

	// Wait for delay if set
	if m.Delay > 0 {
		select {
		case <-ctx.Done():
			return "", nil, ctx.Err()
		case <-time.After(m.Delay):
		}
	}

	// Return error if configured
	if m.Error != nil {
		return "", nil, m.Error
	}

	// Create metrics if not provided
	metrics := m.Metrics
	if metrics == nil {
		metrics = &core.Metrics{
			Duration:     m.Delay,
			InputTokens:  10,
			OutputTokens: 20,
			TotalTokens:  30,
			Model:        m.Model,
			Cost:         0.001,
		}
	}

	return m.Response, metrics, nil
}

// StreamMessage simulates streaming a response.
// If OnStreamMessage is set, it uses that callback. Otherwise:
// It writes StreamChunks to the writer with StreamDelay between each chunk.
func (m *MockAdapter) StreamMessage(ctx context.Context, messages []core.Message, writer io.Writer) (*core.Metrics, error) {
	m.StreamMessageCalls++
	m.LastMessages = messages

	// Use callback if provided
	if m.OnStreamMessage != nil {
		return m.OnStreamMessage(ctx, messages, writer)
	}

	// If no chunks configured, simulate with the Response
	chunks := m.StreamChunks
	if len(chunks) == 0 {
		chunks = []string{m.Response}
	}

	for i, chunk := range chunks {
		// Check for cancellation before each chunk
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Write the chunk
		if _, err := writer.Write([]byte(chunk)); err != nil {
			return nil, err
		}

		// Delay between chunks (not after the last one)
		if m.StreamDelay > 0 && i < len(chunks)-1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(m.StreamDelay):
			}
		}
	}

	// Return error if configured
	if m.Error != nil {
		return nil, m.Error
	}

	// Create metrics if not provided
	metrics := m.Metrics
	if metrics == nil {
		metrics = &core.Metrics{
			Duration:     time.Duration(len(chunks)) * m.StreamDelay,
			InputTokens:  10,
			OutputTokens: 20,
			TotalTokens:  30,
			Model:        m.Model,
			Cost:         0.001,
		}
	}

	return metrics, nil
}

// IsAvailable returns the configured availability.
func (m *MockAdapter) IsAvailable() bool {
	return m.Available
}

// GetModel returns the configured model name.
func (m *MockAdapter) GetModel() string {
	if m.Model != "" {
		return m.Model
	}
	return "mock-model"
}

// HealthCheck returns the configured HealthCheckError.
func (m *MockAdapter) HealthCheck(ctx context.Context) error {
	return m.HealthCheckError
}

// GetAgent returns the initialized agent (for testing).
func (m *MockAdapter) GetAgent() core.Agent {
	return m.agent
}

// Reset clears the call tracking counters.
func (m *MockAdapter) Reset() {
	m.SendMessageCalls = 0
	m.StreamMessageCalls = 0
	m.LastMessages = nil
}

// init registers the mock adapter with the default registry.
func init() {
	adapters.Register("mock", func() adapters.AgentAdapter {
		return NewMockAdapter()
	})
}

// Ensure MockAdapter implements AgentAdapter.
var _ adapters.AgentAdapter = (*MockAdapter)(nil)
