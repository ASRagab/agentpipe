// Package adapters provides the interface and implementations for AI agent adapters.
package adapters

import (
	"context"
	"io"
	"math/rand"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/log"
	"github.com/kevinelliott/agentpipe/pkg/v2/core"
	"github.com/kevinelliott/agentpipe/pkg/v2/errors"
)

// RetryConfig defines parameters for retry behavior with exponential backoff.
type RetryConfig struct {
	// MaxAttempts is the maximum number of attempts (including the first one).
	// A value of 3 means the original request + 2 retries.
	MaxAttempts int

	// InitialDelay is the delay before the first retry.
	InitialDelay time.Duration

	// MaxDelay is the maximum delay between retries.
	MaxDelay time.Duration

	// BackoffMultiplier is the factor by which the delay increases after each retry.
	BackoffMultiplier float64

	// JitterFactor is the maximum jitter to add (as a fraction of the delay).
	// A value of 0.1 means up to 10% jitter.
	JitterFactor float64
}

// DefaultRetryConfig returns the default retry configuration.
// 3 attempts, 1s initial delay, 30s max delay, 2x backoff, 10% jitter.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      1 * time.Second,
		MaxDelay:          30 * time.Second,
		BackoffMultiplier: 2.0,
		JitterFactor:      0.1,
	}
}

// calculateDelay computes the delay for a given attempt number (0-indexed).
func (c RetryConfig) calculateDelay(attempt int) time.Duration {
	if attempt == 0 {
		return 0
	}

	// Calculate base delay with exponential backoff
	delay := float64(c.InitialDelay)
	for i := 1; i < attempt; i++ {
		delay *= c.BackoffMultiplier
		if delay > float64(c.MaxDelay) {
			delay = float64(c.MaxDelay)
			break
		}
	}

	// Apply jitter
	if c.JitterFactor > 0 {
		//nolint:gosec // using math/rand is fine for jitter, no cryptographic need
		jitter := delay * c.JitterFactor * (rand.Float64()*2 - 1) // -jitter to +jitter
		delay += jitter
	}

	// Ensure delay doesn't exceed max
	if delay > float64(c.MaxDelay) {
		delay = float64(c.MaxDelay)
	}

	// Ensure delay is not negative
	if delay < 0 {
		delay = 0
	}

	return time.Duration(delay)
}

// RetryableAdapter wraps an AgentAdapter with retry logic.
type RetryableAdapter struct {
	adapter   AgentAdapter
	config    RetryConfig
	agentID   string
	agentName string
}

// NewRetryableAdapter creates a new RetryableAdapter wrapping the given adapter.
func NewRetryableAdapter(adapter AgentAdapter, config RetryConfig, agentID, agentName string) *RetryableAdapter {
	return &RetryableAdapter{
		adapter:   adapter,
		config:    config,
		agentID:   agentID,
		agentName: agentName,
	}
}

// Initialize delegates to the wrapped adapter.
func (r *RetryableAdapter) Initialize(agent core.Agent) error {
	return r.adapter.Initialize(agent)
}

// IsAvailable delegates to the wrapped adapter.
func (r *RetryableAdapter) IsAvailable() bool {
	return r.adapter.IsAvailable()
}

// GetModel delegates to the wrapped adapter.
func (r *RetryableAdapter) GetModel() string {
	return r.adapter.GetModel()
}

// HealthCheck delegates to the wrapped adapter (no retry for health checks).
func (r *RetryableAdapter) HealthCheck(ctx context.Context) error {
	return r.adapter.HealthCheck(ctx)
}

// SendMessage sends a message with retry logic.
func (r *RetryableAdapter) SendMessage(ctx context.Context, messages []core.Message) (string, *core.Metrics, error) {
	var lastErr error

	for attempt := 0; attempt < r.config.MaxAttempts; attempt++ {
		// Wait before retry (skip for first attempt)
		if attempt > 0 {
			delay := r.config.calculateDelay(attempt)

			log.WithFields(map[string]interface{}{
				"agent_id":   r.agentID,
				"agent_name": r.agentName,
				"attempt":    attempt + 1,
				"max":        r.config.MaxAttempts,
				"delay":      delay.String(),
			}).Info("Retrying SendMessage")

			select {
			case <-ctx.Done():
				return "", nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		response, metrics, err := r.adapter.SendMessage(ctx, messages)
		if err == nil {
			if attempt > 0 {
				log.WithFields(map[string]interface{}{
					"agent_id":   r.agentID,
					"agent_name": r.agentName,
					"attempt":    attempt + 1,
				}).Info("SendMessage succeeded after retry")
			}
			return response, metrics, nil
		}

		lastErr = err

		// Check if error is retryable
		if !errors.IsRetryable(err) {
			log.WithFields(map[string]interface{}{
				"agent_id":   r.agentID,
				"agent_name": r.agentName,
				"error":      err.Error(),
			}).Debug("Error is not retryable, returning immediately")
			return "", nil, err
		}

		log.WithFields(map[string]interface{}{
			"agent_id":   r.agentID,
			"agent_name": r.agentName,
			"attempt":    attempt + 1,
			"max":        r.config.MaxAttempts,
			"error":      err.Error(),
		}).Warn("SendMessage failed, will retry")
	}

	// All attempts exhausted
	return "", nil, errors.NewAgentError(
		r.agentID,
		r.agentName,
		errors.ErrTypeNetwork,
		"all retry attempts exhausted",
		lastErr,
	).WithRetryCount(r.config.MaxAttempts)
}

// StreamMessage streams a message with retry logic.
func (r *RetryableAdapter) StreamMessage(ctx context.Context, messages []core.Message, writer io.Writer) (*core.Metrics, error) {
	var lastErr error

	for attempt := 0; attempt < r.config.MaxAttempts; attempt++ {
		// Wait before retry (skip for first attempt)
		if attempt > 0 {
			delay := r.config.calculateDelay(attempt)

			log.WithFields(map[string]interface{}{
				"agent_id":   r.agentID,
				"agent_name": r.agentName,
				"attempt":    attempt + 1,
				"max":        r.config.MaxAttempts,
				"delay":      delay.String(),
			}).Info("Retrying StreamMessage")

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		metrics, err := r.adapter.StreamMessage(ctx, messages, writer)
		if err == nil {
			if attempt > 0 {
				log.WithFields(map[string]interface{}{
					"agent_id":   r.agentID,
					"agent_name": r.agentName,
					"attempt":    attempt + 1,
				}).Info("StreamMessage succeeded after retry")
			}
			return metrics, nil
		}

		lastErr = err

		// Check if error is retryable
		if !errors.IsRetryable(err) {
			log.WithFields(map[string]interface{}{
				"agent_id":   r.agentID,
				"agent_name": r.agentName,
				"error":      err.Error(),
			}).Debug("Error is not retryable, returning immediately")
			return nil, err
		}

		log.WithFields(map[string]interface{}{
			"agent_id":   r.agentID,
			"agent_name": r.agentName,
			"attempt":    attempt + 1,
			"max":        r.config.MaxAttempts,
			"error":      err.Error(),
		}).Warn("StreamMessage failed, will retry")
	}

	// All attempts exhausted
	return nil, errors.NewAgentError(
		r.agentID,
		r.agentName,
		errors.ErrTypeNetwork,
		"all retry attempts exhausted",
		lastErr,
	).WithRetryCount(r.config.MaxAttempts)
}

// WithRetry wraps a function call with retry logic.
// The function f should return an error that can be checked with errors.IsRetryable.
func WithRetry[T any](
	ctx context.Context,
	config RetryConfig,
	agentID, agentName string,
	f func() (T, error),
) (T, error) {
	var zero T
	var lastErr error

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Wait before retry (skip for first attempt)
		if attempt > 0 {
			delay := config.calculateDelay(attempt)

			log.WithFields(map[string]interface{}{
				"agent_id":   agentID,
				"agent_name": agentName,
				"attempt":    attempt + 1,
				"max":        config.MaxAttempts,
				"delay":      delay.String(),
			}).Info("Retrying operation")

			select {
			case <-ctx.Done():
				return zero, ctx.Err()
			case <-time.After(delay):
			}
		}

		result, err := f()
		if err == nil {
			if attempt > 0 {
				log.WithFields(map[string]interface{}{
					"agent_id":   agentID,
					"agent_name": agentName,
					"attempt":    attempt + 1,
				}).Info("Operation succeeded after retry")
			}
			return result, nil
		}

		lastErr = err

		// Check if error is retryable
		if !errors.IsRetryable(err) {
			log.WithFields(map[string]interface{}{
				"agent_id":   agentID,
				"agent_name": agentName,
				"error":      err.Error(),
			}).Debug("Error is not retryable, returning immediately")
			return zero, err
		}

		log.WithFields(map[string]interface{}{
			"agent_id":   agentID,
			"agent_name": agentName,
			"attempt":    attempt + 1,
			"max":        config.MaxAttempts,
			"error":      err.Error(),
		}).Warn("Operation failed, will retry")
	}

	// All attempts exhausted
	return zero, errors.NewAgentError(
		agentID,
		agentName,
		errors.ErrTypeNetwork,
		"all retry attempts exhausted",
		lastErr,
	).WithRetryCount(config.MaxAttempts)
}

// RetryOnError executes f with retry logic for simple operations that don't return a value.
func RetryOnError(
	ctx context.Context,
	config RetryConfig,
	agentID, agentName string,
	f func() error,
) error {
	_, err := WithRetry(ctx, config, agentID, agentName, func() (struct{}, error) {
		return struct{}{}, f()
	})
	return err
}
