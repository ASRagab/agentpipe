// Package main provides a test webhook receiver server.
//
// Run this server to test the webhook integration example.
// Usage: go run server.go
//
//go:build ignore

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

var eventCount int64

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/webhook", handleWebhook)
	http.HandleFunc("/health", handleHealth)

	fmt.Printf("Webhook receiver listening on http://localhost:%s/webhook\n", port)
	fmt.Println("Press Ctrl+C to stop")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse event
	var event struct {
		ID        string          `json:"id"`
		Type      string          `json:"type"`
		Timestamp time.Time       `json:"timestamp"`
		Data      json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Increment counter
	count := atomic.AddInt64(&eventCount, 1)

	// Log the event
	fmt.Printf("\n[%s] Event #%d received:\n", time.Now().Format("15:04:05"), count)
	fmt.Printf("  ID:        %s\n", event.ID)
	fmt.Printf("  Type:      %s\n", event.Type)
	fmt.Printf("  Timestamp: %s\n", event.Timestamp.Format(time.RFC3339))

	// Log headers
	if eventID := r.Header.Get("X-AgentPipe-Event-ID"); eventID != "" {
		fmt.Printf("  Header ID: %s\n", eventID)
	}

	// Pretty print data based on event type
	printEventData(event.Type, event.Data)

	// Respond with success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "received",
		"eventId": event.ID,
	})
}

func printEventData(eventType string, data json.RawMessage) {
	switch eventType {
	case "conversation.started":
		var d struct {
			ConversationID string `json:"conversation_id"`
			Agents         []struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Model string `json:"model"`
			} `json:"agents"`
		}
		if err := json.Unmarshal(data, &d); err == nil {
			fmt.Printf("  Conversation: %s\n", d.ConversationID)
			fmt.Printf("  Agents: %d\n", len(d.Agents))
			for _, a := range d.Agents {
				fmt.Printf("    - %s (%s)\n", a.Name, a.Model)
			}
		}

	case "message.created":
		var d struct {
			ID        string `json:"id"`
			Role      string `json:"role"`
			AgentName string `json:"agent_name"`
			Content   string `json:"content"`
			Metrics   *struct {
				TotalTokens int     `json:"total_tokens"`
				Cost        float64 `json:"cost"`
			} `json:"metrics"`
		}
		if err := json.Unmarshal(data, &d); err == nil {
			role := d.Role
			if d.AgentName != "" {
				role = d.AgentName
			}
			content := d.Content
			if len(content) > 80 {
				content = content[:80] + "..."
			}
			fmt.Printf("  [%s]: %s\n", role, content)
			if d.Metrics != nil {
				fmt.Printf("  Tokens: %d, Cost: $%.6f\n", d.Metrics.TotalTokens, d.Metrics.Cost)
			}
		}

	case "agent.typing":
		var d struct {
			AgentName string `json:"agent_name"`
		}
		if err := json.Unmarshal(data, &d); err == nil {
			fmt.Printf("  %s is typing...\n", d.AgentName)
		}

	case "agent.done":
		var d struct {
			AgentName string `json:"agent_name"`
			Message   struct {
				Content string `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal(data, &d); err == nil {
			content := d.Message.Content
			if len(content) > 60 {
				content = content[:60] + "..."
			}
			fmt.Printf("  %s finished: %s\n", d.AgentName, content)
		}

	case "agent.error":
		var d struct {
			AgentName string `json:"agent_name"`
			Error     string `json:"error"`
		}
		if err := json.Unmarshal(data, &d); err == nil {
			fmt.Printf("  ERROR from %s: %s\n", d.AgentName, d.Error)
		}

	case "conversation.completed":
		var d struct {
			ConversationID string `json:"conversation_id"`
			Summary        struct {
				Status       string  `json:"status"`
				MessageCount int     `json:"message_count"`
				TotalTokens  int     `json:"total_tokens"`
				TotalCost    float64 `json:"total_cost"`
			} `json:"summary"`
		}
		if err := json.Unmarshal(data, &d); err == nil {
			fmt.Printf("  Status: %s\n", d.Summary.Status)
			fmt.Printf("  Messages: %d\n", d.Summary.MessageCount)
			fmt.Printf("  Tokens: %d\n", d.Summary.TotalTokens)
			fmt.Printf("  Cost: $%.6f\n", d.Summary.TotalCost)
		}

	default:
		// Print raw data for unknown types
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, data, "  ", "  "); err == nil {
			fmt.Printf("  Data:\n  %s\n", pretty.String())
		}
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "healthy",
		"eventsCount": atomic.LoadInt64(&eventCount),
	})
}
