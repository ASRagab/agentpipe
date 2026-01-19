// Package core provides error types and helpers for the v2 AgentPipe architecture.
package core

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ErrorType categorizes different kinds of errors for display and handling.
type ErrorType string

const (
	// ErrorTypeTimeout indicates an operation timed out.
	ErrorTypeTimeout ErrorType = "timeout"
	// ErrorTypeRateLimit indicates a rate limit was hit.
	ErrorTypeRateLimit ErrorType = "rate_limit"
	// ErrorTypeNetwork indicates a network/connection error.
	ErrorTypeNetwork ErrorType = "network"
	// ErrorTypeAuthentication indicates an authentication failure.
	ErrorTypeAuthentication ErrorType = "authentication"
	// ErrorTypeAPI indicates an API-level error.
	ErrorTypeAPI ErrorType = "api"
	// ErrorTypeInternal indicates an internal error.
	ErrorTypeInternal ErrorType = "internal"
	// ErrorTypeUnknown indicates an unknown error type.
	ErrorTypeUnknown ErrorType = "unknown"
)

// ErrorInfo contains detailed information about an error for TUI display.
type ErrorInfo struct {
	// Type categorizes the error for styling and behavior.
	Type ErrorType `json:"type"`
	// Message is the human-readable error message.
	Message string `json:"message"`
	// AgentID is the ID of the agent that encountered the error (if applicable).
	AgentID string `json:"agent_id,omitempty"`
	// AgentName is the name of the agent that encountered the error (if applicable).
	AgentName string `json:"agent_name,omitempty"`
	// Timestamp is when the error occurred.
	Timestamp time.Time `json:"timestamp"`
	// Recoverable indicates whether the error can be retried.
	Recoverable bool `json:"recoverable"`
	// RetryHint provides guidance on how to retry (if recoverable).
	RetryHint string `json:"retry_hint,omitempty"`
}

// NewErrorInfo creates a new ErrorInfo with the given parameters.
func NewErrorInfo(errType ErrorType, message string) ErrorInfo {
	return ErrorInfo{
		Type:        errType,
		Message:     message,
		Timestamp:   time.Now(),
		Recoverable: isRecoverable(errType),
		RetryHint:   getRetryHint(errType),
	}
}

// NewAgentErrorInfo creates a new ErrorInfo for an agent-related error.
func NewAgentErrorInfo(errType ErrorType, message, agentID, agentName string) ErrorInfo {
	info := NewErrorInfo(errType, message)
	info.AgentID = agentID
	info.AgentName = agentName
	return info
}

// isRecoverable determines if an error type is recoverable.
func isRecoverable(errType ErrorType) bool {
	switch errType {
	case ErrorTypeTimeout, ErrorTypeRateLimit, ErrorTypeNetwork:
		return true
	default:
		return false
	}
}

// getRetryHint returns a retry hint for the given error type.
func getRetryHint(errType ErrorType) string {
	switch errType {
	case ErrorTypeTimeout:
		return "Press 'r' to retry or wait for automatic retry"
	case ErrorTypeRateLimit:
		return "Rate limit exceeded. Wait a moment and try again"
	case ErrorTypeNetwork:
		return "Check your connection and press 'r' to retry"
	case ErrorTypeAuthentication:
		return "Check your API key configuration"
	default:
		return ""
	}
}

// FormatErrorMessage formats an error as a system message for display.
func FormatErrorMessage(info ErrorInfo) string {
	prefix := "Error"
	if info.AgentName != "" {
		prefix = fmt.Sprintf("%s failed to respond", info.AgentName)
	}

	return fmt.Sprintf("%s: %s", prefix, info.Message)
}

// NewErrorMessage creates a new system message for an error.
func NewErrorMessage(info ErrorInfo) Message {
	content := FormatErrorMessage(info)

	return Message{
		ID:        uuid.New().String(),
		Timestamp: info.Timestamp,
		Role:      RoleSystem,
		AgentID:   info.AgentID,
		AgentName: info.AgentName,
		Content:   content,
		Status:    MessageStatusError,
	}
}

// ClassifyError attempts to classify an error string into an ErrorType.
func ClassifyError(errMsg string) ErrorType {
	// Simple pattern matching for common error types
	switch {
	case containsAny(errMsg, "timeout", "timed out", "deadline exceeded"):
		return ErrorTypeTimeout
	case containsAny(errMsg, "rate limit", "too many requests", "429"):
		return ErrorTypeRateLimit
	case containsAny(errMsg, "connection", "network", "dns", "dial", "refused"):
		return ErrorTypeNetwork
	case containsAny(errMsg, "auth", "unauthorized", "forbidden", "api key", "401", "403"):
		return ErrorTypeAuthentication
	case containsAny(errMsg, "api error", "500", "502", "503", "504"):
		return ErrorTypeAPI
	default:
		return ErrorTypeUnknown
	}
}

// containsAny checks if s contains any of the substrings (case-insensitive).
func containsAny(s string, substrings ...string) bool {
	lower := toLower(s)
	for _, sub := range substrings {
		if containsIgnoreCase(lower, sub) {
			return true
		}
	}
	return false
}

// toLower converts a string to lowercase.
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

// containsIgnoreCase checks if s contains substr (both already lowercased).
func containsIgnoreCase(s, substr string) bool {
	subLen := len(substr)
	if subLen == 0 {
		return true
	}
	if subLen > len(s) {
		return false
	}
	for i := 0; i <= len(s)-subLen; i++ {
		if s[i:i+subLen] == substr {
			return true
		}
	}
	return false
}
