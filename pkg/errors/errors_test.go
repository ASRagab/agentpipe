package errors

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestErrorTypeString(t *testing.T) {
	tests := []struct {
		errType  ErrorType
		expected string
	}{
		{ErrTypeUnknown, "unknown"},
		{ErrTypeNetwork, "network"},
		{ErrTypeRateLimit, "rate_limit"},
		{ErrTypeTimeout, "timeout"},
		{ErrTypeAuth, "auth"},
		{ErrTypeInvalidResponse, "invalid_response"},
		{ErrorType(99), "unknown"}, // Unknown value
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.errType.String(); got != tt.expected {
				t.Errorf("ErrorType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAgentError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *AgentError
		contains []string
	}{
		{
			name: "with underlying error",
			err: &AgentError{
				AgentID:   "agent-1",
				AgentName: "Claude",
				ErrorType: ErrTypeNetwork,
				Message:   "connection failed",
				Err:       errors.New("dial tcp: connection refused"),
			},
			contains: []string{"network", "Claude", "connection failed", "dial tcp"},
		},
		{
			name: "without underlying error",
			err: &AgentError{
				AgentID:   "agent-2",
				AgentName: "Gemini",
				ErrorType: ErrTypeRateLimit,
				Message:   "too many requests",
			},
			contains: []string{"rate_limit", "Gemini", "too many requests"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, c := range tt.contains {
				if !containsAny(errStr, c) {
					t.Errorf("Error() = %v, should contain %v", errStr, c)
				}
			}
		})
	}
}

func TestAgentError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	agentErr := &AgentError{
		AgentID:   "agent-1",
		AgentName: "Claude",
		ErrorType: ErrTypeNetwork,
		Message:   "test",
		Err:       underlying,
	}

	if got := agentErr.Unwrap(); got != underlying {
		t.Errorf("Unwrap() = %v, want %v", got, underlying)
	}

	// Test error chain inspection with errors.Is
	if !errors.Is(agentErr, underlying) {
		t.Error("errors.Is should find underlying error")
	}
}

func TestAgentError_IsRetryable(t *testing.T) {
	tests := []struct {
		errType   ErrorType
		retryable bool
	}{
		{ErrTypeNetwork, true},
		{ErrTypeRateLimit, true},
		{ErrTypeTimeout, true},
		{ErrTypeAuth, false},
		{ErrTypeInvalidResponse, false},
		{ErrTypeUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.errType.String(), func(t *testing.T) {
			err := &AgentError{
				AgentID:   "test",
				AgentName: "Test",
				ErrorType: tt.errType,
			}
			if got := err.IsRetryable(); got != tt.retryable {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.retryable)
			}
		})
	}
}

func TestAgentError_UserFriendlyMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      *AgentError
		contains []string
	}{
		{
			name: "network error",
			err: &AgentError{
				AgentName: "Claude",
				ErrorType: ErrTypeNetwork,
			},
			contains: []string{"Claude", "unreachable", "internet connection"},
		},
		{
			name: "rate limit with retry after",
			err: &AgentError{
				AgentName:  "Gemini",
				ErrorType:  ErrTypeRateLimit,
				RetryAfter: 30 * time.Second,
			},
			contains: []string{"Gemini", "rate limit", "30s"},
		},
		{
			name: "rate limit without retry after",
			err: &AgentError{
				AgentName: "GPT-4",
				ErrorType: ErrTypeRateLimit,
			},
			contains: []string{"GPT-4", "rate limit", "wait"},
		},
		{
			name: "timeout error",
			err: &AgentError{
				AgentName: "Claude",
				ErrorType: ErrTypeTimeout,
			},
			contains: []string{"Claude", "too long", "retried"},
		},
		{
			name: "auth error",
			err: &AgentError{
				AgentName: "OpenRouter",
				ErrorType: ErrTypeAuth,
			},
			contains: []string{"OpenRouter", "authentication", "API key"},
		},
		{
			name: "invalid response",
			err: &AgentError{
				AgentName: "Custom",
				ErrorType: ErrTypeInvalidResponse,
			},
			contains: []string{"Custom", "invalid response"},
		},
		{
			name: "unknown with message",
			err: &AgentError{
				AgentName: "Unknown",
				ErrorType: ErrTypeUnknown,
				Message:   "something went wrong",
			},
			contains: []string{"Unknown", "something went wrong"},
		},
		{
			name: "unknown without message",
			err: &AgentError{
				AgentName: "Unknown",
				ErrorType: ErrTypeUnknown,
			},
			contains: []string{"Unknown", "unexpected error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.UserFriendlyMessage()
			for _, c := range tt.contains {
				if !containsAny(msg, c) {
					t.Errorf("UserFriendlyMessage() = %v, should contain %v", msg, c)
				}
			}
		})
	}
}

func TestAgentError_WithMethods(t *testing.T) {
	original := &AgentError{
		AgentID:   "agent-1",
		AgentName: "Claude",
		ErrorType: ErrTypeRateLimit,
		Message:   "rate limited",
	}

	// Test WithRetryCount
	withRetry := original.WithRetryCount(3)
	if withRetry.RetryCount != 3 {
		t.Errorf("WithRetryCount() RetryCount = %v, want 3", withRetry.RetryCount)
	}
	if original.RetryCount != 0 {
		t.Error("WithRetryCount() should not modify original")
	}

	// Test WithRetryAfter
	withAfter := original.WithRetryAfter(30 * time.Second)
	if withAfter.RetryAfter != 30*time.Second {
		t.Errorf("WithRetryAfter() RetryAfter = %v, want 30s", withAfter.RetryAfter)
	}
	if original.RetryAfter != 0 {
		t.Error("WithRetryAfter() should not modify original")
	}

	// Test WithHTTPStatus
	withStatus := original.WithHTTPStatus(429)
	if withStatus.HTTPStatusCode != 429 {
		t.Errorf("WithHTTPStatus() HTTPStatusCode = %v, want 429", withStatus.HTTPStatusCode)
	}
	if original.HTTPStatusCode != 0 {
		t.Error("WithHTTPStatus() should not modify original")
	}
}

func TestNewAgentError(t *testing.T) {
	underlying := errors.New("underlying")
	err := NewAgentError("agent-1", "Claude", ErrTypeNetwork, "connection failed", underlying)

	if err.AgentID != "agent-1" {
		t.Errorf("AgentID = %v, want agent-1", err.AgentID)
	}
	if err.AgentName != "Claude" {
		t.Errorf("AgentName = %v, want Claude", err.AgentName)
	}
	if err.ErrorType != ErrTypeNetwork {
		t.Errorf("ErrorType = %v, want ErrTypeNetwork", err.ErrorType)
	}
	if err.Message != "connection failed" {
		t.Errorf("Message = %v, want connection failed", err.Message)
	}
	if err.Err != underlying {
		t.Errorf("Err = %v, want %v", err.Err, underlying)
	}
	if err.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestNewNetworkError(t *testing.T) {
	underlying := errors.New("connection refused")
	err := NewNetworkError("agent-1", "Claude", underlying)

	if err.ErrorType != ErrTypeNetwork {
		t.Errorf("ErrorType = %v, want ErrTypeNetwork", err.ErrorType)
	}
	if !err.IsRetryable() {
		t.Error("Network errors should be retryable")
	}
}

func TestNewRateLimitError(t *testing.T) {
	err := NewRateLimitError("agent-1", "Claude", 30*time.Second)

	if err.ErrorType != ErrTypeRateLimit {
		t.Errorf("ErrorType = %v, want ErrTypeRateLimit", err.ErrorType)
	}
	if err.RetryAfter != 30*time.Second {
		t.Errorf("RetryAfter = %v, want 30s", err.RetryAfter)
	}
	if err.HTTPStatusCode != 429 {
		t.Errorf("HTTPStatusCode = %v, want 429", err.HTTPStatusCode)
	}
	if !err.IsRetryable() {
		t.Error("Rate limit errors should be retryable")
	}
}

func TestNewTimeoutError(t *testing.T) {
	underlying := errors.New("context deadline exceeded")
	err := NewTimeoutError("agent-1", "Claude", underlying)

	if err.ErrorType != ErrTypeTimeout {
		t.Errorf("ErrorType = %v, want ErrTypeTimeout", err.ErrorType)
	}
	if !err.IsRetryable() {
		t.Error("Timeout errors should be retryable")
	}
}

func TestNewAuthError(t *testing.T) {
	underlying := errors.New("invalid API key")
	err := NewAuthError("agent-1", "Claude", 401, underlying)

	if err.ErrorType != ErrTypeAuth {
		t.Errorf("ErrorType = %v, want ErrTypeAuth", err.ErrorType)
	}
	if err.HTTPStatusCode != 401 {
		t.Errorf("HTTPStatusCode = %v, want 401", err.HTTPStatusCode)
	}
	if err.IsRetryable() {
		t.Error("Auth errors should NOT be retryable")
	}
}

func TestNewInvalidResponseError(t *testing.T) {
	underlying := errors.New("unexpected end of JSON")
	err := NewInvalidResponseError("agent-1", "Claude", "failed to parse response", underlying)

	if err.ErrorType != ErrTypeInvalidResponse {
		t.Errorf("ErrorType = %v, want ErrTypeInvalidResponse", err.ErrorType)
	}
	if err.IsRetryable() {
		t.Error("Invalid response errors should NOT be retryable")
	}
}

func TestIsAgentError(t *testing.T) {
	agentErr := NewNetworkError("agent-1", "Claude", errors.New("test"))
	wrappedErr := fmt.Errorf("wrapped: %w", agentErr)
	regularErr := errors.New("regular error")

	if !IsAgentError(agentErr) {
		t.Error("IsAgentError should return true for AgentError")
	}
	if !IsAgentError(wrappedErr) {
		t.Error("IsAgentError should return true for wrapped AgentError")
	}
	if IsAgentError(regularErr) {
		t.Error("IsAgentError should return false for regular error")
	}
	if IsAgentError(nil) {
		t.Error("IsAgentError should return false for nil")
	}
}

func TestAsAgentError(t *testing.T) {
	agentErr := NewNetworkError("agent-1", "Claude", errors.New("test"))
	wrappedErr := fmt.Errorf("wrapped: %w", agentErr)
	regularErr := errors.New("regular error")

	// Direct AgentError
	extracted, ok := AsAgentError(agentErr)
	if !ok || extracted != agentErr {
		t.Error("AsAgentError should extract AgentError")
	}

	// Wrapped AgentError
	extracted, ok = AsAgentError(wrappedErr)
	if !ok || extracted != agentErr {
		t.Error("AsAgentError should extract wrapped AgentError")
	}

	// Regular error
	extracted, ok = AsAgentError(regularErr)
	if ok || extracted != nil {
		t.Error("AsAgentError should return nil for regular error")
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "AgentError network",
			err:       NewNetworkError("agent-1", "Claude", nil),
			retryable: true,
		},
		{
			name:      "AgentError rate limit",
			err:       NewRateLimitError("agent-1", "Claude", 0),
			retryable: true,
		},
		{
			name:      "AgentError timeout",
			err:       NewTimeoutError("agent-1", "Claude", nil),
			retryable: true,
		},
		{
			name:      "AgentError auth",
			err:       NewAuthError("agent-1", "Claude", 401, nil),
			retryable: false,
		},
		{
			name:      "generic timeout error",
			err:       errors.New("context deadline exceeded"),
			retryable: true,
		},
		{
			name:      "generic network error",
			err:       errors.New("dial tcp: connection refused"),
			retryable: true,
		},
		{
			name:      "generic rate limit error",
			err:       errors.New("status 429: too many requests"),
			retryable: true,
		},
		{
			name:      "generic auth error",
			err:       errors.New("401 unauthorized"),
			retryable: false,
		},
		{
			name:      "nil error",
			err:       nil,
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.retryable {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.retryable)
			}
		})
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		errMsg   string
		expected ErrorType
	}{
		// Timeout patterns
		{"context deadline exceeded", ErrTypeTimeout},
		{"request timed out", ErrTypeTimeout},
		{"operation timeout", ErrTypeTimeout},

		// Rate limit patterns
		{"status 429: too many requests", ErrTypeRateLimit},
		{"rate limit exceeded", ErrTypeRateLimit},
		{"request throttled", ErrTypeRateLimit},

		// Network patterns
		{"dial tcp: connection refused", ErrTypeNetwork},
		{"no such host", ErrTypeNetwork},
		{"network unreachable", ErrTypeNetwork},
		{"DNS lookup failed", ErrTypeNetwork},
		{"connection reset by peer", ErrTypeNetwork},
		{"unexpected EOF", ErrTypeNetwork},

		// Auth patterns
		{"401 unauthorized", ErrTypeAuth},
		{"403 forbidden", ErrTypeAuth},
		{"invalid API key", ErrTypeAuth},
		{"authentication failed", ErrTypeAuth},

		// Invalid response patterns
		{"invalid json response", ErrTypeInvalidResponse},
		{"unexpected end of JSON input", ErrTypeInvalidResponse},
		{"malformed response body", ErrTypeInvalidResponse},
		{"failed to unmarshal response", ErrTypeInvalidResponse},

		// Unknown
		{"some random error", ErrTypeUnknown},
		{"", ErrTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.errMsg, func(t *testing.T) {
			err := errors.New(tt.errMsg)
			if got := ClassifyError(err); got != tt.expected {
				t.Errorf("ClassifyError(%q) = %v, want %v", tt.errMsg, got, tt.expected)
			}
		})
	}

	// Test nil error
	if got := ClassifyError(nil); got != ErrTypeUnknown {
		t.Errorf("ClassifyError(nil) = %v, want ErrTypeUnknown", got)
	}
}

func TestWrapError(t *testing.T) {
	// Test wrapping a regular error
	regularErr := errors.New("dial tcp: connection refused")
	wrapped := WrapError("agent-1", "Claude", regularErr)

	if wrapped == nil {
		t.Fatal("WrapError should not return nil for non-nil error")
	}
	if wrapped.ErrorType != ErrTypeNetwork {
		t.Errorf("WrapError should classify error, got %v want ErrTypeNetwork", wrapped.ErrorType)
	}
	if wrapped.AgentID != "agent-1" {
		t.Errorf("AgentID = %v, want agent-1", wrapped.AgentID)
	}

	// Test wrapping nil
	if WrapError("agent-1", "Claude", nil) != nil {
		t.Error("WrapError should return nil for nil error")
	}

	// Test wrapping an existing AgentError
	existingAgentErr := NewRateLimitError("agent-2", "Gemini", 30*time.Second)
	rewrapped := WrapError("agent-1", "Claude", existingAgentErr)
	if rewrapped != existingAgentErr {
		t.Error("WrapError should return existing AgentError as-is")
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		s          string
		substrings []string
		expected   bool
	}{
		{"hello world", []string{"hello"}, true},
		{"HELLO WORLD", []string{"hello"}, true}, // Case insensitive
		{"hello world", []string{"foo", "bar", "world"}, true},
		{"hello world", []string{"foo", "bar"}, false},
		{"", []string{"foo"}, false},
		{"hello", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := containsAny(tt.s, tt.substrings...); got != tt.expected {
				t.Errorf("containsAny(%q, %v) = %v, want %v", tt.s, tt.substrings, got, tt.expected)
			}
		})
	}
}

func TestToLower(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"HELLO", "hello"},
		{"Hello World", "hello world"},
		{"already lowercase", "already lowercase"},
		{"MiXeD CaSe 123", "mixed case 123"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := toLower(tt.input); got != tt.expected {
				t.Errorf("toLower(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello world", "foo", false},
		{"", "foo", false},
		{"hello", "", true},
		{"hello", "hello world", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_in_%s", tt.substr, tt.s), func(t *testing.T) {
			if got := contains(tt.s, tt.substr); got != tt.expected {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.expected)
			}
		})
	}
}

// TestErrorChainInspection verifies that error chains work correctly with standard library functions.
func TestErrorChainInspection(t *testing.T) {
	// Create a chain of errors
	sentinel := errors.New("sentinel error")
	agentErr := NewAgentError("agent-1", "Claude", ErrTypeNetwork, "wrapped sentinel", sentinel)
	outerErr := fmt.Errorf("outer context: %w", agentErr)

	// Test errors.Is through the chain
	if !errors.Is(outerErr, sentinel) {
		t.Error("errors.Is should find sentinel through chain")
	}

	// Test errors.As through the chain
	var extracted *AgentError
	if !errors.As(outerErr, &extracted) {
		t.Error("errors.As should find AgentError through chain")
	}
	if extracted.AgentName != "Claude" {
		t.Errorf("extracted AgentName = %v, want Claude", extracted.AgentName)
	}
}
