// Package errors provides comprehensive error types and handling for the v2 AgentPipe architecture.
// It includes AgentError wrapping, error classification, retry logic support, and user-friendly messaging.
package errors

import (
	"errors"
	"fmt"
	"time"
)

// ErrorType categorizes different kinds of errors for handling and retry logic.
type ErrorType int

const (
	// ErrTypeUnknown indicates an unknown or uncategorized error.
	ErrTypeUnknown ErrorType = iota
	// ErrTypeNetwork indicates a network/connection error (connection refused, DNS, etc.).
	ErrTypeNetwork
	// ErrTypeRateLimit indicates a rate limit was hit (HTTP 429).
	ErrTypeRateLimit
	// ErrTypeTimeout indicates an operation timed out.
	ErrTypeTimeout
	// ErrTypeAuth indicates an authentication/authorization failure (HTTP 401, 403).
	ErrTypeAuth
	// ErrTypeInvalidResponse indicates the API returned an invalid or malformed response.
	ErrTypeInvalidResponse
)

// String returns the string representation of an ErrorType.
func (e ErrorType) String() string {
	switch e {
	case ErrTypeNetwork:
		return "network"
	case ErrTypeRateLimit:
		return "rate_limit"
	case ErrTypeTimeout:
		return "timeout"
	case ErrTypeAuth:
		return "auth"
	case ErrTypeInvalidResponse:
		return "invalid_response"
	default:
		return "unknown"
	}
}

// AgentError wraps an error with agent context and error classification.
// It implements the error interface and supports error chain inspection via Unwrap().
type AgentError struct {
	// AgentID is the unique identifier of the agent that encountered the error.
	AgentID string
	// AgentName is the human-readable name of the agent.
	AgentName string
	// ErrorType categorizes the error for retry logic and display.
	ErrorType ErrorType
	// Message provides additional context about the error.
	Message string
	// Err is the underlying wrapped error.
	Err error
	// Timestamp is when the error occurred.
	Timestamp time.Time
	// RetryCount indicates how many retry attempts have been made.
	RetryCount int
	// RetryAfter is the suggested wait time before retrying (for rate limits).
	RetryAfter time.Duration
	// HTTPStatusCode is the HTTP status code if applicable.
	HTTPStatusCode int
}

// Error implements the error interface.
func (e *AgentError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %s: %v", e.ErrorType, e.AgentName, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s: %s", e.ErrorType, e.AgentName, e.Message)
}

// Unwrap returns the underlying error for error chain inspection.
func (e *AgentError) Unwrap() error {
	return e.Err
}

// IsRetryable returns true if the error type is one that can be retried.
// Network errors, rate limits, and timeouts are retryable.
// Auth errors and invalid responses are not retryable.
func (e *AgentError) IsRetryable() bool {
	switch e.ErrorType {
	case ErrTypeNetwork, ErrTypeRateLimit, ErrTypeTimeout:
		return true
	case ErrTypeAuth, ErrTypeInvalidResponse, ErrTypeUnknown:
		return false
	default:
		return false
	}
}

// UserFriendlyMessage returns an actionable error message for display to users.
func (e *AgentError) UserFriendlyMessage() string {
	switch e.ErrorType {
	case ErrTypeNetwork:
		return fmt.Sprintf("%s is currently unreachable. Check your internet connection and try again.", e.AgentName)
	case ErrTypeRateLimit:
		if e.RetryAfter > 0 {
			return fmt.Sprintf("%s hit a rate limit. Retrying in %s...", e.AgentName, e.RetryAfter.Round(time.Second))
		}
		return fmt.Sprintf("%s hit a rate limit. Please wait a moment and try again.", e.AgentName)
	case ErrTypeTimeout:
		return fmt.Sprintf("%s took too long to respond. The request will be retried.", e.AgentName)
	case ErrTypeAuth:
		return fmt.Sprintf("%s authentication failed. Please check your API key configuration.", e.AgentName)
	case ErrTypeInvalidResponse:
		return fmt.Sprintf("%s returned an invalid response. This may be a temporary issue.", e.AgentName)
	default:
		if e.Message != "" {
			return fmt.Sprintf("%s encountered an error: %s", e.AgentName, e.Message)
		}
		return fmt.Sprintf("%s encountered an unexpected error.", e.AgentName)
	}
}

// WithRetryCount returns a copy of the error with an updated retry count.
func (e *AgentError) WithRetryCount(count int) *AgentError {
	newErr := *e
	newErr.RetryCount = count
	return &newErr
}

// WithRetryAfter returns a copy of the error with a retry-after duration.
func (e *AgentError) WithRetryAfter(d time.Duration) *AgentError {
	newErr := *e
	newErr.RetryAfter = d
	return &newErr
}

// WithHTTPStatus returns a copy of the error with an HTTP status code.
func (e *AgentError) WithHTTPStatus(code int) *AgentError {
	newErr := *e
	newErr.HTTPStatusCode = code
	return &newErr
}

// NewAgentError creates a new AgentError with the given parameters.
func NewAgentError(agentID, agentName string, errType ErrorType, message string, err error) *AgentError {
	return &AgentError{
		AgentID:   agentID,
		AgentName: agentName,
		ErrorType: errType,
		Message:   message,
		Err:       err,
		Timestamp: time.Now(),
	}
}

// NewNetworkError creates a new AgentError for network failures.
func NewNetworkError(agentID, agentName string, err error) *AgentError {
	return NewAgentError(agentID, agentName, ErrTypeNetwork, "network error", err)
}

// NewRateLimitError creates a new AgentError for rate limit responses.
func NewRateLimitError(agentID, agentName string, retryAfter time.Duration) *AgentError {
	e := NewAgentError(agentID, agentName, ErrTypeRateLimit, "rate limit exceeded", nil)
	e.RetryAfter = retryAfter
	e.HTTPStatusCode = 429
	return e
}

// NewTimeoutError creates a new AgentError for timeout situations.
func NewTimeoutError(agentID, agentName string, err error) *AgentError {
	return NewAgentError(agentID, agentName, ErrTypeTimeout, "request timed out", err)
}

// NewAuthError creates a new AgentError for authentication failures.
func NewAuthError(agentID, agentName string, statusCode int, err error) *AgentError {
	e := NewAgentError(agentID, agentName, ErrTypeAuth, "authentication failed", err)
	e.HTTPStatusCode = statusCode
	return e
}

// NewInvalidResponseError creates a new AgentError for malformed responses.
func NewInvalidResponseError(agentID, agentName, details string, err error) *AgentError {
	return NewAgentError(agentID, agentName, ErrTypeInvalidResponse, details, err)
}

// IsAgentError checks if an error is an AgentError.
func IsAgentError(err error) bool {
	var agentErr *AgentError
	return errors.As(err, &agentErr)
}

// AsAgentError attempts to extract an AgentError from an error chain.
func AsAgentError(err error) (*AgentError, bool) {
	var agentErr *AgentError
	if errors.As(err, &agentErr) {
		return agentErr, true
	}
	return nil, false
}

// IsRetryable checks if an error is retryable, handling both AgentError and other errors.
func IsRetryable(err error) bool {
	if agentErr, ok := AsAgentError(err); ok {
		return agentErr.IsRetryable()
	}
	// For non-AgentError errors, classify based on error message
	return classifyAndCheckRetryable(err)
}

// classifyAndCheckRetryable attempts to classify a generic error and check if it's retryable.
func classifyAndCheckRetryable(err error) bool {
	if err == nil {
		return false
	}
	errType := ClassifyError(err)
	switch errType {
	case ErrTypeNetwork, ErrTypeRateLimit, ErrTypeTimeout:
		return true
	default:
		return false
	}
}

// ClassifyError attempts to classify a generic error into an ErrorType based on common patterns.
func ClassifyError(err error) ErrorType {
	if err == nil {
		return ErrTypeUnknown
	}

	errStr := err.Error()

	// Check for timeout patterns
	if containsAny(errStr, "timeout", "timed out", "deadline exceeded", "context deadline") {
		return ErrTypeTimeout
	}

	// Check for rate limit patterns
	if containsAny(errStr, "rate limit", "too many requests", "429", "throttl") {
		return ErrTypeRateLimit
	}

	// Check for network patterns
	if containsAny(errStr, "connection refused", "no such host", "network", "dns",
		"dial", "connect:", "connection reset", "eof", "broken pipe") {
		return ErrTypeNetwork
	}

	// Check for auth patterns
	if containsAny(errStr, "unauthorized", "forbidden", "authentication", "api key",
		"invalid key", "401", "403", "auth") {
		return ErrTypeAuth
	}

	// Check for invalid response patterns
	if containsAny(errStr, "invalid json", "unexpected end", "malformed",
		"parse error", "unmarshal", "decode error") {
		return ErrTypeInvalidResponse
	}

	return ErrTypeUnknown
}

// WrapError wraps a generic error as an AgentError with automatic classification.
func WrapError(agentID, agentName string, err error) *AgentError {
	if err == nil {
		return nil
	}

	// If already an AgentError, return as-is
	if agentErr, ok := AsAgentError(err); ok {
		return agentErr
	}

	errType := ClassifyError(err)
	return NewAgentError(agentID, agentName, errType, err.Error(), err)
}

// containsAny checks if s contains any of the substrings (case-insensitive).
func containsAny(s string, substrings ...string) bool {
	lower := toLower(s)
	for _, sub := range substrings {
		if contains(lower, toLower(sub)) {
			return true
		}
	}
	return false
}

// toLower converts a string to lowercase (simple ASCII-only implementation).
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
