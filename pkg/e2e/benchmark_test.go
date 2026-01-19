//go:build e2e

package e2e

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/ASRagab/agentpipe/pkg/core"
	"github.com/ASRagab/agentpipe/pkg/events"
	"github.com/ASRagab/agentpipe/pkg/manager"
	"github.com/ASRagab/agentpipe/pkg/pool"
)

// Performance targets from MVP plan
const (
	// Target: Single message round-trip < 100ms with mock adapter
	targetSingleMessageLatency = 100 * time.Millisecond

	// Target: Parallel execution overhead < 20% of sequential
	targetParallelOverheadPercent = 20.0

	// Target: Event bus throughput > 10,000 events/second
	targetEventThroughput = 10000

	// Target: TUI render < 16ms for 60fps
	targetTUIRenderTime = 16 * time.Millisecond

	// Target: Memory per message < 1KB average
	targetMemoryPerMessage = 1024 // bytes
)

// BenchmarkSingleMessage measures time for single message/response cycle.
func BenchmarkSingleMessage(b *testing.B) {
	agents := []core.Agent{
		core.NewAgent("bench-agent", "mock", "Benchmark Agent", "model", "mock"),
	}

	eventBus := events.NewBus()
	defer eventBus.Close()

	mgr, err := manager.NewConversationManager(manager.DefaultConfig(), agents, eventBus)
	if err != nil {
		b.Fatalf("failed to create manager: %v", err)
	}
	defer mgr.Close()

	mgr.Start()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mgr.SendUserMessage(ctx, "Benchmark message")
		if err != nil {
			b.Fatalf("SendUserMessage failed: %v", err)
		}
	}
	b.StopTimer()

	avgLatency := b.Elapsed() / time.Duration(b.N)
	b.ReportMetric(float64(avgLatency.Microseconds()), "μs/op")

	// Compare against target
	if avgLatency > targetSingleMessageLatency {
		b.Logf("WARNING: Average latency %v exceeds target %v", avgLatency, targetSingleMessageLatency)
	}
}

// BenchmarkParallelExecution measures overhead of parallel agent execution.
func BenchmarkParallelExecution(b *testing.B) {
	eventBus := events.NewBus()
	defer eventBus.Close()

	// Create 5 agents with consistent delay
	agents := make([]core.Agent, 5)
	for i := 0; i < 5; i++ {
		agents[i] = core.NewAgent(
			AgentID(i),
			"mock",
			AgentName(i),
			"model",
			"mock",
		)
	}

	mgr, err := manager.NewConversationManager(manager.DefaultConfig(), agents, eventBus)
	if err != nil {
		b.Fatalf("failed to create manager: %v", err)
	}
	defer mgr.Close()

	mgr.Start()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mgr.SendUserMessage(ctx, "Parallel benchmark")
		if err != nil {
			b.Fatalf("SendUserMessage failed: %v", err)
		}
	}
	b.StopTimer()

	avgLatency := b.Elapsed() / time.Duration(b.N)
	b.ReportMetric(float64(avgLatency.Microseconds()), "μs/op")
	b.ReportMetric(float64(len(agents)), "agents")
}

// BenchmarkEventBus measures event publishing throughput.
func BenchmarkEventBus(b *testing.B) {
	eventBus := events.NewBus()
	defer eventBus.Close()

	// Subscribe to all events
	eventBus.SubscribeAll(func(event core.Event) {
		// Minimal processing
		_ = event.Type
	})

	// Create test events
	testMessage := core.NewUserMessage("Benchmark message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eventBus.Publish(core.NewMessageCreatedEvent(testMessage))
	}
	b.StopTimer()

	// Calculate throughput
	eventsPerSecond := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(eventsPerSecond, "events/s")

	if eventsPerSecond < float64(targetEventThroughput) {
		b.Logf("WARNING: Throughput %.0f events/s below target %d", eventsPerSecond, targetEventThroughput)
	}
}

// BenchmarkEventBusWithHandlers measures event bus with multiple handlers.
func BenchmarkEventBusWithHandlers(b *testing.B) {
	eventBus := events.NewBus()
	defer eventBus.Close()

	// Add multiple handlers
	handlerCount := 10
	for i := 0; i < handlerCount; i++ {
		eventBus.SubscribeAll(func(event core.Event) {
			// Simulate minimal processing
			_ = event.Type
		})
	}

	testMessage := core.NewUserMessage("Benchmark")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eventBus.Publish(core.NewMessageCreatedEvent(testMessage))
	}
	b.StopTimer()

	eventsPerSecond := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(eventsPerSecond, "events/s")
	b.ReportMetric(float64(handlerCount), "handlers")
}

// BenchmarkMemoryUsage measures memory for a conversation.
func BenchmarkMemoryUsage(b *testing.B) {
	// Force GC before measuring
	runtime.GC()

	var mStart, mEnd runtime.MemStats
	runtime.ReadMemStats(&mStart)

	agents := []core.Agent{
		core.NewAgent("mem-agent", "mock", "Memory Agent", "model", "mock"),
	}

	eventBus := events.NewBus()
	defer eventBus.Close()

	mgr, err := manager.NewConversationManager(manager.DefaultConfig(), agents, eventBus)
	if err != nil {
		b.Fatalf("failed to create manager: %v", err)
	}
	defer mgr.Close()

	mgr.Start()
	ctx := context.Background()

	// Send 100 messages
	messageCount := 100
	for i := 0; i < messageCount; i++ {
		_, _ = mgr.SendUserMessage(ctx, "Memory benchmark message with some content to simulate realistic usage")
	}

	// Force GC and measure
	runtime.GC()
	runtime.ReadMemStats(&mEnd)

	// Calculate memory per message (user + agent = 2 messages per round)
	totalMessages := messageCount * 2
	memoryUsed := mEnd.HeapAlloc - mStart.HeapAlloc
	memoryPerMessage := memoryUsed / uint64(totalMessages)

	b.ReportMetric(float64(memoryPerMessage), "bytes/msg")
	b.ReportMetric(float64(memoryUsed)/1024, "KB-total")

	if memoryPerMessage > targetMemoryPerMessage {
		b.Logf("WARNING: Memory per message %d bytes exceeds target %d", memoryPerMessage, targetMemoryPerMessage)
	}
}

// BenchmarkPoolExecution measures raw pool parallel execution.
func BenchmarkPoolExecution(b *testing.B) {
	eventBus := events.NewBus()
	defer eventBus.Close()

	agentPool := pool.NewPool(eventBus, 10*time.Second)

	// Add agents with mock adapters
	agentCount := 5
	for i := 0; i < agentCount; i++ {
		agent := core.NewAgent(AgentID(i), "mock", AgentName(i), "model", "mock")
		// Note: We'd need access to mock adapters here, using harness would be cleaner
		// For now, just benchmark the pool coordination overhead
		_ = agent
	}

	messages := []core.Message{core.NewUserMessage("Benchmark")}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = agentPool.ExecuteParallel(ctx, messages, nil)
	}
	b.StopTimer()
}

// BenchmarkMessageCreation measures message creation overhead.
func BenchmarkMessageCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = core.NewUserMessage("Benchmark message content")
	}
	b.StopTimer()
}

// BenchmarkConversationGrowth measures performance as conversation grows.
func BenchmarkConversationGrowth(b *testing.B) {
	sizes := []int{10, 50, 100, 500}

	for _, size := range sizes {
		b.Run(string(rune('0'+size/10))+"0msgs", func(b *testing.B) {
			agents := []core.Agent{
				core.NewAgent("growth-agent", "mock", "Growth Agent", "model", "mock"),
			}

			eventBus := events.NewBus()
			defer eventBus.Close()

			mgr, err := manager.NewConversationManager(manager.DefaultConfig(), agents, eventBus)
			if err != nil {
				b.Fatalf("failed to create manager: %v", err)
			}
			defer mgr.Close()

			mgr.Start()
			ctx := context.Background()

			// Pre-populate conversation to target size
			for i := 0; i < size; i++ {
				_, _ = mgr.SendUserMessage(ctx, "Pre-populate message")
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = mgr.SendUserMessage(ctx, "Benchmark after growth")
			}
			b.StopTimer()

			b.ReportMetric(float64(size), "prev_msgs")
		})
	}
}

// BenchmarkEventTypeFiltering measures overhead of event type filtering.
func BenchmarkEventTypeFiltering(b *testing.B) {
	eventBus := events.NewBus()
	defer eventBus.Close()

	// Subscribe to specific event type
	eventBus.Subscribe(core.EventAgentDone, func(event core.Event) {
		_ = event.Type
	})

	testMessage := core.NewUserMessage("Benchmark")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Publish events of different types
		eventBus.Publish(core.NewMessageCreatedEvent(testMessage))
		eventBus.Publish(core.NewAgentTypingEvent("agent", "Agent"))
		eventBus.Publish(core.NewAgentDoneEvent("agent", "Agent", testMessage))
	}
	b.StopTimer()
}

// BenchmarkConcurrentMessages measures concurrent message handling.
func BenchmarkConcurrentMessages(b *testing.B) {
	agents := []core.Agent{
		core.NewAgent("concurrent-agent", "mock", "Concurrent Agent", "model", "mock"),
	}

	eventBus := events.NewBus()
	defer eventBus.Close()

	mgr, err := manager.NewConversationManager(manager.DefaultConfig(), agents, eventBus)
	if err != nil {
		b.Fatalf("failed to create manager: %v", err)
	}
	defer mgr.Close()

	mgr.Start()
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = mgr.SendUserMessage(ctx, "Concurrent message")
		}
	})
}

// TestPerformanceTargets runs all performance tests and reports against targets.
func TestPerformanceTargets(t *testing.T) {
	TimeoutWrapper(t, 60*time.Second, func(t *testing.T) {
		t.Log("=== Performance Targets Report ===")

		// Test single message latency
		t.Run("SingleMessageLatency", func(t *testing.T) {
			agents := []core.Agent{
				core.NewAgent("perf-agent", "mock", "Perf Agent", "model", "mock"),
			}

			eventBus := events.NewBus()
			defer eventBus.Close()

			mgr, err := manager.NewConversationManager(manager.DefaultConfig(), agents, eventBus)
			if err != nil {
				t.Fatalf("failed to create manager: %v", err)
			}
			defer mgr.Close()

			mgr.Start()
			ctx := context.Background()

			iterations := 10
			var totalDuration time.Duration

			for i := 0; i < iterations; i++ {
				start := time.Now()
				_, err := mgr.SendUserMessage(ctx, "Performance test")
				if err != nil {
					t.Fatalf("SendUserMessage failed: %v", err)
				}
				totalDuration += time.Since(start)
			}

			avgLatency := totalDuration / time.Duration(iterations)
			status := "PASS"
			if avgLatency > targetSingleMessageLatency {
				status = "FAIL"
			}

			t.Logf("Single Message Latency: %v (target: <%v) [%s]",
				avgLatency, targetSingleMessageLatency, status)
		})

		// Test event bus throughput
		t.Run("EventBusThroughput", func(t *testing.T) {
			eventBus := events.NewBus()
			defer eventBus.Close()

			eventBus.SubscribeAll(func(event core.Event) {
				_ = event.Type
			})

			testMessage := core.NewUserMessage("Benchmark")
			iterations := 100000

			start := time.Now()
			for i := 0; i < iterations; i++ {
				eventBus.Publish(core.NewMessageCreatedEvent(testMessage))
			}
			elapsed := time.Since(start)

			eventsPerSecond := float64(iterations) / elapsed.Seconds()
			status := "PASS"
			if eventsPerSecond < float64(targetEventThroughput) {
				status = "FAIL"
			}

			t.Logf("Event Bus Throughput: %.0f events/s (target: >%d) [%s]",
				eventsPerSecond, targetEventThroughput, status)
		})

		// Test parallel execution overhead
		t.Run("ParallelOverhead", func(t *testing.T) {
			// Sequential baseline
			agents1 := []core.Agent{
				core.NewAgent("seq-agent", "mock", "Sequential Agent", "model", "mock"),
			}

			eventBus1 := events.NewBus()
			defer eventBus1.Close()

			mgr1, _ := manager.NewConversationManager(manager.DefaultConfig(), agents1, eventBus1)
			defer mgr1.Close()

			mgr1.Start()
			ctx := context.Background()

			iterations := 5
			var seqTotal time.Duration
			for i := 0; i < iterations; i++ {
				start := time.Now()
				_, _ = mgr1.SendUserMessage(ctx, "Sequential test")
				seqTotal += time.Since(start)
			}
			seqAvg := seqTotal / time.Duration(iterations)

			// Parallel with 5 agents
			agents5 := make([]core.Agent, 5)
			for i := 0; i < 5; i++ {
				agents5[i] = core.NewAgent(AgentID(i), "mock", AgentName(i), "model", "mock")
			}

			eventBus2 := events.NewBus()
			defer eventBus2.Close()

			mgr2, _ := manager.NewConversationManager(manager.DefaultConfig(), agents5, eventBus2)
			defer mgr2.Close()

			mgr2.Start()

			var parTotal time.Duration
			for i := 0; i < iterations; i++ {
				start := time.Now()
				_, _ = mgr2.SendUserMessage(ctx, "Parallel test")
				parTotal += time.Since(start)
			}
			parAvg := parTotal / time.Duration(iterations)

			// Calculate overhead (parallel should be similar to or less than 5x sequential
			// due to parallel execution; we're measuring coordination overhead)
			overhead := float64(parAvg) / float64(seqAvg) * 100

			status := "PASS"
			// For 5 agents, parallel should be faster than 5x sequential
			// If overhead > 120% of single agent time, that's concerning
			if overhead > 120 {
				status = "WARN"
			}

			t.Logf("Parallel Overhead: %.1f%% of single agent (seq=%v, par=%v) [%s]",
				overhead, seqAvg, parAvg, status)
		})
	})
}
