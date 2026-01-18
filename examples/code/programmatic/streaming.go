// Package main demonstrates streaming responses with AgentPipe v2.
//
// This file shows how to:
// - Use adapters directly for streaming
// - Handle streaming events
// - Display real-time output
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	// Import to register adapters
	_ "github.com/ASRagab/agentpipe/pkg/v2/adapters/api"

	"github.com/ASRagab/agentpipe/pkg/v2/adapters"
	"github.com/ASRagab/agentpipe/pkg/v2/core"
)

// RunStreamingExample demonstrates streaming responses from an adapter.
func RunStreamingExample() {
	fmt.Println("=== AgentPipe v2 Streaming Example ===\n")

	if os.Getenv("OPENROUTER_API_KEY") == "" {
		fmt.Println("OPENROUTER_API_KEY not set. Skipping streaming example.")
		return
	}

	// Create an agent
	agent := core.Agent{
		ID:          "streaming-agent",
		Type:        "openrouter",
		Name:        "Claude Streaming",
		Model:       "anthropic/claude-3-haiku",
		AdapterName: "openrouter",
		Config: core.AgentAdapterConfig{
			SystemPrompt: "You are a helpful assistant. Respond in 2-3 sentences.",
			Temperature:  0.7,
			MaxTokens:    256,
		},
	}

	// Get and initialize the adapter
	adapter, err := adapters.Get("openrouter")
	if err != nil {
		fmt.Printf("Error getting adapter: %v\n", err)
		return
	}

	if err := adapter.Initialize(agent); err != nil {
		fmt.Printf("Error initializing adapter: %v\n", err)
		return
	}

	// Check availability
	if !adapter.IsAvailable() {
		fmt.Println("Adapter is not available (check API key)")
		return
	}

	// Create messages
	messages := []core.Message{
		core.NewUserMessage("Tell me a short joke about programming."),
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// === Method 1: Stream directly to stdout ===
	fmt.Println("[Streaming to stdout]")
	fmt.Print("\nAssistant: ")

	metrics, err := adapter.StreamMessage(ctx, messages, os.Stdout)
	if err != nil {
		fmt.Printf("\nError: %v\n", err)
		return
	}

	fmt.Printf("\n\nMetrics: %d tokens, %v, $%.6f\n",
		metrics.TotalTokens, metrics.Duration, metrics.Cost)

	// === Method 2: Stream to a buffer for processing ===
	fmt.Println("\n\n[Streaming to buffer]")

	var buffer bytes.Buffer
	messages2 := []core.Message{
		core.NewUserMessage("What's 2 + 2? Just give the number."),
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel2()

	metrics2, err := adapter.StreamMessage(ctx2, messages2, &buffer)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	response := buffer.String()
	fmt.Printf("Buffered response: %s\n", response)
	fmt.Printf("Metrics: %d tokens, %v, $%.6f\n",
		metrics2.TotalTokens, metrics2.Duration, metrics2.Cost)
}

// StreamWriter is a custom writer that processes streaming chunks.
type StreamWriter struct {
	onChunk func(chunk string)
}

// Write implements io.Writer for custom chunk handling.
func (w *StreamWriter) Write(p []byte) (n int, err error) {
	chunk := string(p)
	if w.onChunk != nil {
		w.onChunk(chunk)
	}
	return len(p), nil
}

// RunCustomStreamingExample demonstrates using a custom writer for streaming.
func RunCustomStreamingExample() {
	fmt.Println("=== Custom Streaming Handler Example ===\n")

	if os.Getenv("OPENROUTER_API_KEY") == "" {
		fmt.Println("OPENROUTER_API_KEY not set. Skipping.")
		return
	}

	agent := core.Agent{
		ID:          "custom-stream-agent",
		Type:        "openrouter",
		Name:        "Custom Streamer",
		Model:       "anthropic/claude-3-haiku",
		AdapterName: "openrouter",
		Config: core.AgentAdapterConfig{
			SystemPrompt: "Respond briefly.",
			Temperature:  0.7,
			MaxTokens:    100,
		},
	}

	adapter, _ := adapters.Get("openrouter")
	if err := adapter.Initialize(agent); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	messages := []core.Message{
		core.NewUserMessage("Count from 1 to 5."),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a custom writer that processes each chunk
	chunkCount := 0
	writer := &StreamWriter{
		onChunk: func(chunk string) {
			chunkCount++
			// You could do custom processing here:
			// - Update a UI element
			// - Send to a WebSocket
			// - Accumulate for analysis
			fmt.Print(chunk) // Just print for this example
		},
	}

	metrics, err := adapter.StreamMessage(ctx, messages, writer)
	if err != nil {
		fmt.Printf("\nError: %v\n", err)
		return
	}

	fmt.Printf("\n\nReceived %d chunks\n", chunkCount)
	fmt.Printf("Metrics: %d tokens, %v\n", metrics.TotalTokens, metrics.Duration)
}
