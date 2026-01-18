// Package api provides API-based adapters for AI providers.
package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ASRagab/agentpipe/pkg/errors"
)

// HTTPErrorClassifier provides utilities for classifying HTTP errors
// and creating properly typed AgentErrors from HTTP responses.
type HTTPErrorClassifier struct {
	agentID   string
	agentName string
}

// NewHTTPErrorClassifier creates a new error classifier for the given agent.
func NewHTTPErrorClassifier(agentID, agentName string) *HTTPErrorClassifier {
	return &HTTPErrorClassifier{
		agentID:   agentID,
		agentName: agentName,
	}
}

// ClassifyHTTPError creates an AgentError from an HTTP response.
// It properly detects:
// - Rate limits (HTTP 429) with retry-after parsing
// - Auth errors (HTTP 401, 403) - not retryable
// - Server errors (HTTP 5xx) - retryable as network errors
// - Client errors (HTTP 4xx other than 401, 403, 429) - not retryable
func (c *HTTPErrorClassifier) ClassifyHTTPError(resp *http.Response, bodyMessage string) *errors.AgentError {
	statusCode := resp.StatusCode

	switch {
	case statusCode == http.StatusTooManyRequests:
		return c.handleRateLimitResponse(resp, bodyMessage)

	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return c.handleAuthError(statusCode, bodyMessage)

	case statusCode >= 500:
		return c.handleServerError(statusCode, bodyMessage)

	default:
		// Other client errors (4xx)
		return c.handleClientError(statusCode, bodyMessage)
	}
}

// ClassifyConnectionError creates an AgentError from a connection-level error
// (DNS failure, connection refused, timeout, etc.)
func (c *HTTPErrorClassifier) ClassifyConnectionError(err error) *errors.AgentError {
	if err == nil {
		return nil
	}

	errType := errors.ClassifyError(err)
	agentErr := errors.NewAgentError(c.agentID, c.agentName, errType, err.Error(), err)

	return agentErr
}

// handleRateLimitResponse creates a rate limit error with retry-after parsing.
func (c *HTTPErrorClassifier) handleRateLimitResponse(resp *http.Response, bodyMessage string) *errors.AgentError {
	retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))

	message := "rate limit exceeded"
	if bodyMessage != "" {
		message = bodyMessage
	}

	agentErr := errors.NewAgentError(
		c.agentID,
		c.agentName,
		errors.ErrTypeRateLimit,
		message,
		nil,
	).WithHTTPStatus(resp.StatusCode)

	if retryAfter > 0 {
		agentErr = agentErr.WithRetryAfter(retryAfter)
	}

	return agentErr
}

// handleAuthError creates an authentication error.
func (c *HTTPErrorClassifier) handleAuthError(statusCode int, bodyMessage string) *errors.AgentError {
	message := "authentication failed"
	if bodyMessage != "" {
		message = bodyMessage
	}

	return errors.NewAgentError(
		c.agentID,
		c.agentName,
		errors.ErrTypeAuth,
		message,
		nil,
	).WithHTTPStatus(statusCode)
}

// handleServerError creates a network error for server errors (retryable).
func (c *HTTPErrorClassifier) handleServerError(statusCode int, bodyMessage string) *errors.AgentError {
	message := "server error"
	if bodyMessage != "" {
		message = bodyMessage
	}

	return errors.NewAgentError(
		c.agentID,
		c.agentName,
		errors.ErrTypeNetwork,
		message,
		nil,
	).WithHTTPStatus(statusCode)
}

// handleClientError creates an error for other client errors (not retryable).
func (c *HTTPErrorClassifier) handleClientError(statusCode int, bodyMessage string) *errors.AgentError {
	message := "request error"
	if bodyMessage != "" {
		message = bodyMessage
	}

	// Client errors other than rate limit and auth are treated as invalid response
	return errors.NewAgentError(
		c.agentID,
		c.agentName,
		errors.ErrTypeInvalidResponse,
		message,
		nil,
	).WithHTTPStatus(statusCode)
}

// parseRetryAfter parses the Retry-After header value.
// It supports both seconds (e.g., "120") and HTTP-date formats.
func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}

	// Try to parse as seconds first
	value = strings.TrimSpace(value)
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	// Try to parse as HTTP-date (RFC 1123)
	// Example: "Wed, 21 Oct 2015 07:28:00 GMT"
	if t, err := http.ParseTime(value); err == nil {
		duration := time.Until(t)
		if duration > 0 {
			return duration
		}
	}

	return 0
}

// IsRateLimitError checks if an error is a rate limit error.
func IsRateLimitError(err error) bool {
	if agentErr, ok := errors.AsAgentError(err); ok {
		return agentErr.ErrorType == errors.ErrTypeRateLimit
	}
	return false
}

// IsAuthError checks if an error is an authentication error.
func IsAuthError(err error) bool {
	if agentErr, ok := errors.AsAgentError(err); ok {
		return agentErr.ErrorType == errors.ErrTypeAuth
	}
	return false
}

// IsNetworkError checks if an error is a network error.
func IsNetworkError(err error) bool {
	if agentErr, ok := errors.AsAgentError(err); ok {
		return agentErr.ErrorType == errors.ErrTypeNetwork
	}
	return false
}

// IsTimeoutError checks if an error is a timeout error.
func IsTimeoutError(err error) bool {
	if agentErr, ok := errors.AsAgentError(err); ok {
		return agentErr.ErrorType == errors.ErrTypeTimeout
	}
	return false
}

// GetRetryAfter returns the retry-after duration from an error, if available.
func GetRetryAfter(err error) time.Duration {
	if agentErr, ok := errors.AsAgentError(err); ok {
		return agentErr.RetryAfter
	}
	return 0
}

// GetHTTPStatusCode returns the HTTP status code from an error, if available.
func GetHTTPStatusCode(err error) int {
	if agentErr, ok := errors.AsAgentError(err); ok {
		return agentErr.HTTPStatusCode
	}
	return 0
}
