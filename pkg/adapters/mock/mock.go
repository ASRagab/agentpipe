// Package mock provides a mock adapter for testing without real API calls.
package mock

import (
	"context"
	"io"
	"time"

	"github.com/ASRagab/agentpipe/pkg/adapters"
	"github.com/ASRagab/agentpipe/pkg/core"
)

type MockAdapter struct {
	Response         string
	Delay            time.Duration
	Error            error
	StreamChunks     []string
	StreamDelay      time.Duration
	HealthCheckError error
	Available        bool
	Model            string

	agent core.Agent

	Metrics *core.Metrics

	SendMessageCalls   int
	StreamMessageCalls int
	LastMessages       []core.Message
	LastConversation   *core.ConversationContext

	OnSendMessage   func(ctx context.Context, messages []core.Message, conversation *core.ConversationContext) (string, *core.Metrics, error)
	OnStreamMessage func(ctx context.Context, messages []core.Message, writer io.Writer, conversation *core.ConversationContext) (*core.Metrics, error)
}

func NewMockAdapter() *MockAdapter {
	return &MockAdapter{
		Response:  "Mock response",
		Available: true,
		Model:     "mock-model",
	}
}

func (m *MockAdapter) Initialize(agent core.Agent) error {
	m.agent = agent
	return nil
}

func (m *MockAdapter) SendMessage(ctx context.Context, messages []core.Message, conversation *core.ConversationContext) (string, *core.Metrics, error) {
	m.SendMessageCalls++
	m.LastMessages = messages
	m.LastConversation = conversation

	if m.OnSendMessage != nil {
		return m.OnSendMessage(ctx, messages, conversation)
	}

	if m.Delay > 0 {
		select {
		case <-ctx.Done():
			return "", nil, ctx.Err()
		case <-time.After(m.Delay):
		}
	}

	if m.Error != nil {
		return "", nil, m.Error
	}

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

func (m *MockAdapter) StreamMessage(ctx context.Context, messages []core.Message, writer io.Writer, conversation *core.ConversationContext) (*core.Metrics, error) {
	m.StreamMessageCalls++
	m.LastMessages = messages
	m.LastConversation = conversation

	if m.OnStreamMessage != nil {
		return m.OnStreamMessage(ctx, messages, writer, conversation)
	}

	chunks := m.StreamChunks
	if len(chunks) == 0 {
		chunks = []string{m.Response}
	}

	if m.Delay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(m.Delay):
		}
	}

	for i, chunk := range chunks {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if _, err := writer.Write([]byte(chunk)); err != nil {
			return nil, err
		}

		if m.StreamDelay > 0 && i < len(chunks)-1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(m.StreamDelay):
			}
		}
	}

	if m.Error != nil {
		return nil, m.Error
	}

	metrics := m.Metrics
	if metrics == nil {
		totalDuration := m.Delay
		if len(chunks) > 1 {
			totalDuration += time.Duration(len(chunks)-1) * m.StreamDelay
		}
		metrics = &core.Metrics{
			Duration:     totalDuration,
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
