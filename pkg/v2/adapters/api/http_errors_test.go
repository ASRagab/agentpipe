package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/v2/errors"
)

func TestNewHTTPErrorClassifier(t *testing.T) {
	c := NewHTTPErrorClassifier("agent-1", "TestAgent")
	if c == nil {
		t.Fatal("expected non-nil classifier")
	}
	if c.agentID != "agent-1" {
		t.Errorf("expected agentID 'agent-1', got %q", c.agentID)
	}
	if c.agentName != "TestAgent" {
		t.Errorf("expected agentName 'TestAgent', got %q", c.agentName)
	}
}

func TestClassifyHTTPError_RateLimit(t *testing.T) {
	c := NewHTTPErrorClassifier("agent-1", "TestAgent")

	// Create a mock response with 429 and Retry-After header
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     make(http.Header),
	}
	resp.Header.Set("Retry-After", "30")

	err := c.ClassifyHTTPError(resp, "rate limit exceeded")
	if err == nil {
		t.Fatal("expected non-nil error")
	}

	if err.ErrorType != errors.ErrTypeRateLimit {
		t.Errorf("expected ErrTypeRateLimit, got %v", err.ErrorType)
	}

	if err.HTTPStatusCode != 429 {
		t.Errorf("expected HTTP status 429, got %d", err.HTTPStatusCode)
	}

	if err.RetryAfter != 30*time.Second {
		t.Errorf("expected RetryAfter 30s, got %v", err.RetryAfter)
	}

	if !err.IsRetryable() {
		t.Error("rate limit error should be retryable")
	}
}

func TestClassifyHTTPError_Auth401(t *testing.T) {
	c := NewHTTPErrorClassifier("agent-1", "TestAgent")

	resp := &http.Response{
		StatusCode: http.StatusUnauthorized,
		Header:     make(http.Header),
	}

	err := c.ClassifyHTTPError(resp, "invalid API key")
	if err == nil {
		t.Fatal("expected non-nil error")
	}

	if err.ErrorType != errors.ErrTypeAuth {
		t.Errorf("expected ErrTypeAuth, got %v", err.ErrorType)
	}

	if err.HTTPStatusCode != 401 {
		t.Errorf("expected HTTP status 401, got %d", err.HTTPStatusCode)
	}

	if err.IsRetryable() {
		t.Error("auth error should not be retryable")
	}
}

func TestClassifyHTTPError_Auth403(t *testing.T) {
	c := NewHTTPErrorClassifier("agent-1", "TestAgent")

	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Header:     make(http.Header),
	}

	err := c.ClassifyHTTPError(resp, "forbidden")
	if err == nil {
		t.Fatal("expected non-nil error")
	}

	if err.ErrorType != errors.ErrTypeAuth {
		t.Errorf("expected ErrTypeAuth, got %v", err.ErrorType)
	}

	if err.HTTPStatusCode != 403 {
		t.Errorf("expected HTTP status 403, got %d", err.HTTPStatusCode)
	}

	if err.IsRetryable() {
		t.Error("forbidden error should not be retryable")
	}
}

func TestClassifyHTTPError_ServerError(t *testing.T) {
	testCases := []int{500, 502, 503, 504}

	for _, statusCode := range testCases {
		t.Run(fmt.Sprintf("HTTP%d", statusCode), func(t *testing.T) {
			c := NewHTTPErrorClassifier("agent-1", "TestAgent")

			resp := &http.Response{
				StatusCode: statusCode,
				Header:     make(http.Header),
			}

			err := c.ClassifyHTTPError(resp, "server error")
			if err == nil {
				t.Fatal("expected non-nil error")
			}

			if err.ErrorType != errors.ErrTypeNetwork {
				t.Errorf("expected ErrTypeNetwork for HTTP %d, got %v", statusCode, err.ErrorType)
			}

			if err.HTTPStatusCode != statusCode {
				t.Errorf("expected HTTP status %d, got %d", statusCode, err.HTTPStatusCode)
			}

			if !err.IsRetryable() {
				t.Errorf("server error %d should be retryable", statusCode)
			}
		})
	}
}

func TestClassifyHTTPError_ClientError(t *testing.T) {
	testCases := []int{400, 404, 422}

	for _, statusCode := range testCases {
		t.Run(fmt.Sprintf("HTTP%d", statusCode), func(t *testing.T) {
			c := NewHTTPErrorClassifier("agent-1", "TestAgent")

			resp := &http.Response{
				StatusCode: statusCode,
				Header:     make(http.Header),
			}

			err := c.ClassifyHTTPError(resp, "client error")
			if err == nil {
				t.Fatal("expected non-nil error")
			}

			if err.ErrorType != errors.ErrTypeInvalidResponse {
				t.Errorf("expected ErrTypeInvalidResponse for HTTP %d, got %v", statusCode, err.ErrorType)
			}

			if err.HTTPStatusCode != statusCode {
				t.Errorf("expected HTTP status %d, got %d", statusCode, err.HTTPStatusCode)
			}

			if err.IsRetryable() {
				t.Errorf("client error %d should not be retryable", statusCode)
			}
		})
	}
}

func TestClassifyConnectionError(t *testing.T) {
	c := NewHTTPErrorClassifier("agent-1", "TestAgent")

	testCases := []struct {
		name         string
		err          error
		expectedType errors.ErrorType
		retryable    bool
	}{
		{
			name:         "connection refused",
			err:          fmt.Errorf("dial tcp: connection refused"),
			expectedType: errors.ErrTypeNetwork,
			retryable:    true,
		},
		{
			name:         "timeout",
			err:          fmt.Errorf("context deadline exceeded"),
			expectedType: errors.ErrTypeTimeout,
			retryable:    true,
		},
		{
			name:         "DNS error",
			err:          fmt.Errorf("no such host"),
			expectedType: errors.ErrTypeNetwork,
			retryable:    true,
		},
		{
			name:         "nil error",
			err:          nil,
			expectedType: errors.ErrTypeUnknown,
			retryable:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := c.ClassifyConnectionError(tc.err)

			if tc.err == nil {
				if err != nil {
					t.Errorf("expected nil error for nil input, got %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("expected non-nil error")
			}

			if err.ErrorType != tc.expectedType {
				t.Errorf("expected %v, got %v", tc.expectedType, err.ErrorType)
			}

			if err.IsRetryable() != tc.retryable {
				t.Errorf("expected retryable=%v, got %v", tc.retryable, err.IsRetryable())
			}
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	testCases := []struct {
		name     string
		value    string
		expected time.Duration
	}{
		{
			name:     "seconds",
			value:    "120",
			expected: 120 * time.Second,
		},
		{
			name:     "seconds with spaces",
			value:    " 30 ",
			expected: 30 * time.Second,
		},
		{
			name:     "zero",
			value:    "0",
			expected: 0,
		},
		{
			name:     "empty",
			value:    "",
			expected: 0,
		},
		{
			name:     "invalid",
			value:    "not-a-number",
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseRetryAfter(tc.value)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestIsRateLimitError(t *testing.T) {
	rateLimitErr := errors.NewRateLimitError("agent-1", "TestAgent", 30*time.Second)
	if !IsRateLimitError(rateLimitErr) {
		t.Error("expected IsRateLimitError to return true for rate limit error")
	}

	authErr := errors.NewAuthError("agent-1", "TestAgent", 401, nil)
	if IsRateLimitError(authErr) {
		t.Error("expected IsRateLimitError to return false for auth error")
	}

	regularErr := fmt.Errorf("regular error")
	if IsRateLimitError(regularErr) {
		t.Error("expected IsRateLimitError to return false for regular error")
	}
}

func TestIsAuthError(t *testing.T) {
	authErr := errors.NewAuthError("agent-1", "TestAgent", 401, nil)
	if !IsAuthError(authErr) {
		t.Error("expected IsAuthError to return true for auth error")
	}

	rateLimitErr := errors.NewRateLimitError("agent-1", "TestAgent", 30*time.Second)
	if IsAuthError(rateLimitErr) {
		t.Error("expected IsAuthError to return false for rate limit error")
	}
}

func TestIsNetworkError(t *testing.T) {
	networkErr := errors.NewNetworkError("agent-1", "TestAgent", fmt.Errorf("connection refused"))
	if !IsNetworkError(networkErr) {
		t.Error("expected IsNetworkError to return true for network error")
	}

	authErr := errors.NewAuthError("agent-1", "TestAgent", 401, nil)
	if IsNetworkError(authErr) {
		t.Error("expected IsNetworkError to return false for auth error")
	}
}

func TestIsTimeoutError(t *testing.T) {
	timeoutErr := errors.NewTimeoutError("agent-1", "TestAgent", fmt.Errorf("deadline exceeded"))
	if !IsTimeoutError(timeoutErr) {
		t.Error("expected IsTimeoutError to return true for timeout error")
	}

	networkErr := errors.NewNetworkError("agent-1", "TestAgent", fmt.Errorf("connection refused"))
	if IsTimeoutError(networkErr) {
		t.Error("expected IsTimeoutError to return false for network error")
	}
}

func TestGetRetryAfter(t *testing.T) {
	rateLimitErr := errors.NewRateLimitError("agent-1", "TestAgent", 30*time.Second)
	if GetRetryAfter(rateLimitErr) != 30*time.Second {
		t.Errorf("expected 30s, got %v", GetRetryAfter(rateLimitErr))
	}

	authErr := errors.NewAuthError("agent-1", "TestAgent", 401, nil)
	if GetRetryAfter(authErr) != 0 {
		t.Errorf("expected 0, got %v", GetRetryAfter(authErr))
	}

	regularErr := fmt.Errorf("regular error")
	if GetRetryAfter(regularErr) != 0 {
		t.Errorf("expected 0 for regular error, got %v", GetRetryAfter(regularErr))
	}
}

func TestGetHTTPStatusCode(t *testing.T) {
	authErr := errors.NewAuthError("agent-1", "TestAgent", 401, nil)
	if GetHTTPStatusCode(authErr) != 401 {
		t.Errorf("expected 401, got %d", GetHTTPStatusCode(authErr))
	}

	networkErr := errors.NewNetworkError("agent-1", "TestAgent", fmt.Errorf("connection refused"))
	if GetHTTPStatusCode(networkErr) != 0 {
		t.Errorf("expected 0 for network error without status, got %d", GetHTTPStatusCode(networkErr))
	}

	regularErr := fmt.Errorf("regular error")
	if GetHTTPStatusCode(regularErr) != 0 {
		t.Errorf("expected 0 for regular error, got %d", GetHTTPStatusCode(regularErr))
	}
}

// TestHTTPErrorClassifier_Integration tests the classifier with a real HTTP server
func TestHTTPErrorClassifier_Integration(t *testing.T) {
	testCases := []struct {
		name         string
		statusCode   int
		headers      map[string]string
		body         string
		expectedType errors.ErrorType
		retryable    bool
	}{
		{
			name:         "rate limit with retry-after",
			statusCode:   429,
			headers:      map[string]string{"Retry-After": "60"},
			body:         `{"error": {"message": "Too many requests"}}`,
			expectedType: errors.ErrTypeRateLimit,
			retryable:    true,
		},
		{
			name:         "unauthorized",
			statusCode:   401,
			headers:      nil,
			body:         `{"error": {"message": "Invalid API key"}}`,
			expectedType: errors.ErrTypeAuth,
			retryable:    false,
		},
		{
			name:         "internal server error",
			statusCode:   500,
			headers:      nil,
			body:         "Internal Server Error",
			expectedType: errors.ErrTypeNetwork,
			retryable:    true,
		},
		{
			name:         "bad gateway",
			statusCode:   502,
			headers:      nil,
			body:         "Bad Gateway",
			expectedType: errors.ErrTypeNetwork,
			retryable:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for k, v := range tc.headers {
					w.Header().Set(k, v)
				}
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			// Make request
			resp, err := http.Get(server.URL)
			if err != nil {
				t.Fatalf("failed to make request: %v", err)
			}
			defer resp.Body.Close()

			// Classify error
			c := NewHTTPErrorClassifier("test-agent", "TestAgent")
			agentErr := c.ClassifyHTTPError(resp, tc.body)

			if agentErr.ErrorType != tc.expectedType {
				t.Errorf("expected %v, got %v", tc.expectedType, agentErr.ErrorType)
			}

			if agentErr.IsRetryable() != tc.retryable {
				t.Errorf("expected retryable=%v, got %v", tc.retryable, agentErr.IsRetryable())
			}
		})
	}
}
