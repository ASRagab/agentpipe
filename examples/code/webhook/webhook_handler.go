// Package main provides a webhook handler for forwarding AgentPipe events.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ASRagab/agentpipe/pkg/core"
)

// WebhookConfig contains configuration for the webhook handler.
type WebhookConfig struct {
	// MaxRetries is the maximum number of retry attempts for failed deliveries.
	MaxRetries int
	// Timeout is the HTTP request timeout.
	Timeout time.Duration
	// Async enables asynchronous delivery (non-blocking).
	Async bool
	// FilterTypes limits which event types are sent.
	// If nil, all events are sent.
	FilterTypes []core.EventType
	// AuthHeader is an optional authorization header value.
	AuthHeader string
}

// WebhookStats tracks delivery statistics.
type WebhookStats struct {
	TotalSent  int64
	Successful int64
	Failed     int64
	Pending    int64
}

// WebhookHandler sends events to an HTTP webhook endpoint.
type WebhookHandler struct {
	url        string
	config     WebhookConfig
	httpClient *http.Client
	stats      WebhookStats
	filterSet  map[core.EventType]bool
	queue      chan core.Event
	wg         sync.WaitGroup
	closed     bool
	mu         sync.RWMutex
}

// NewWebhookHandler creates a new webhook handler.
func NewWebhookHandler(url string, config WebhookConfig) *WebhookHandler {
	// Set defaults
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	// Build filter set
	var filterSet map[core.EventType]bool
	if len(config.FilterTypes) > 0 {
		filterSet = make(map[core.EventType]bool)
		for _, t := range config.FilterTypes {
			filterSet[t] = true
		}
	}

	h := &WebhookHandler{
		url:    url,
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		filterSet: filterSet,
	}

	// Set up async processing if enabled
	if config.Async {
		h.queue = make(chan core.Event, 100)
		h.startWorkers(3)
	}

	return h
}

// startWorkers starts async delivery workers.
func (h *WebhookHandler) startWorkers(count int) {
	for i := 0; i < count; i++ {
		h.wg.Add(1)
		go h.worker()
	}
}

// worker processes events from the queue.
func (h *WebhookHandler) worker() {
	defer h.wg.Done()
	for event := range h.queue {
		atomic.AddInt64(&h.stats.Pending, -1)
		h.sendWithRetry(event)
	}
}

// Handle processes an event and sends it to the webhook.
func (h *WebhookHandler) Handle(event core.Event) error {
	// Check if event type is filtered
	if h.filterSet != nil && !h.filterSet[event.Type] {
		return nil
	}

	h.mu.RLock()
	closed := h.closed
	h.mu.RUnlock()

	if closed {
		return fmt.Errorf("webhook handler is closed")
	}

	atomic.AddInt64(&h.stats.TotalSent, 1)

	if h.config.Async {
		atomic.AddInt64(&h.stats.Pending, 1)
		h.queue <- event
		return nil
	}

	return h.sendWithRetry(event)
}

// sendWithRetry sends an event with retry logic.
func (h *WebhookHandler) sendWithRetry(event core.Event) error {
	var lastErr error

	for attempt := 0; attempt <= h.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			time.Sleep(backoff)
		}

		err := h.send(event)
		if err == nil {
			atomic.AddInt64(&h.stats.Successful, 1)
			return nil
		}

		lastErr = err
	}

	atomic.AddInt64(&h.stats.Failed, 1)
	return fmt.Errorf("webhook delivery failed after %d attempts: %w",
		h.config.MaxRetries+1, lastErr)
}

// send performs a single HTTP request.
func (h *WebhookHandler) send(event core.Event) error {
	// Serialize event
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", h.url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-AgentPipe-Event", string(event.Type))
	req.Header.Set("X-AgentPipe-Event-ID", event.ID)

	if h.config.AuthHeader != "" {
		req.Header.Set("Authorization", h.config.AuthHeader)
	}

	// Send request
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// GetStats returns current delivery statistics.
func (h *WebhookHandler) GetStats() WebhookStats {
	return WebhookStats{
		TotalSent:  atomic.LoadInt64(&h.stats.TotalSent),
		Successful: atomic.LoadInt64(&h.stats.Successful),
		Failed:     atomic.LoadInt64(&h.stats.Failed),
		Pending:    atomic.LoadInt64(&h.stats.Pending),
	}
}

// Close shuts down the webhook handler and waits for pending deliveries.
func (h *WebhookHandler) Close() {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.closed = true
	h.mu.Unlock()

	if h.queue != nil {
		close(h.queue)
		h.wg.Wait()
	}
}

// WebhookPayload represents the JSON structure sent to webhooks.
// This documents the expected format for webhook receivers.
type WebhookPayload struct {
	// ID is the unique event identifier.
	ID string `json:"id"`
	// Type is the event type (e.g., "message.created").
	Type string `json:"type"`
	// Timestamp is when the event occurred.
	Timestamp time.Time `json:"timestamp"`
	// Data contains event-specific data.
	Data interface{} `json:"data"`
}

// MessageCreatedPayload is the data for message.created events.
type MessageCreatedPayload struct {
	ID        string   `json:"id"`
	Role      string   `json:"role"`
	AgentID   string   `json:"agent_id,omitempty"`
	AgentName string   `json:"agent_name,omitempty"`
	Content   string   `json:"content"`
	Timestamp string   `json:"timestamp"`
	Metrics   *Metrics `json:"metrics,omitempty"`
}

// Metrics contains response metrics.
type Metrics struct {
	DurationNs   int64   `json:"duration_ns"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalTokens  int     `json:"total_tokens"`
	Model        string  `json:"model"`
	Cost         float64 `json:"cost"`
}

// ConversationStartedPayload is the data for conversation.started events.
type ConversationStartedPayload struct {
	ConversationID string         `json:"conversation_id"`
	Agents         []AgentPayload `json:"agents"`
}

// AgentPayload represents an agent in event payloads.
type AgentPayload struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Model string `json:"model"`
}

// ConversationCompletedPayload is the data for conversation.completed events.
type ConversationCompletedPayload struct {
	ConversationID string         `json:"conversation_id"`
	Summary        SummaryPayload `json:"summary"`
}

// SummaryPayload contains conversation summary data.
type SummaryPayload struct {
	Status       string  `json:"status"`
	MessageCount int     `json:"message_count"`
	AgentCount   int     `json:"agent_count"`
	TotalTokens  int     `json:"total_tokens"`
	TotalCost    float64 `json:"total_cost"`
	DurationNs   int64   `json:"duration_ns"`
}
