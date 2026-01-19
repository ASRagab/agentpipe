// Package e2e provides comprehensive end-to-end testing for the AgentPipe architecture.
//
// # Overview
//
// This package contains tests that verify the complete stack works together correctly,
// from user input through agent responses, including TUI rendering, persistence, and
// error handling. All tests use the "e2e" build tag and are excluded from normal builds.
//
// # Running E2E Tests
//
// To run all E2E tests:
//
//	go test -tags e2e -v ./pkg/e2e/...
//
// To run a specific test file:
//
//	go test -tags e2e -v ./pkg/e2e/conversation_test.go
//
// To run tests with race detection:
//
//	go test -tags e2e -race -v ./pkg/e2e/...
//
// To run benchmarks:
//
//	go test -tags e2e -bench=. -benchmem ./pkg/e2e/...
//
// # Test Files
//
// The package contains the following test files:
//
//   - framework.go: Test harness, helpers, and fixtures (no build tag)
//   - conversation_test.go: Conversation flow tests (single message, multi-turn, context)
//   - parallel_test.go: Parallel execution tests (timing, mixed speeds, failures)
//   - persistence_test.go: Save/load tests (resume, export, auto-save)
//   - tui_test.go: Terminal UI tests (render, input, navigation, resize)
//   - error_test.go: Error handling tests (network, rate limit, auth, circuit breaker)
//   - config_test.go: Configuration tests (loading, validation, migration)
//   - benchmark_test.go: Performance benchmarks (latency, throughput, memory)
//   - platform_test.go: Cross-platform verification (paths, timing, UTF-8)
//
// # Test Framework
//
// The TestHarness provides a complete test environment:
//
//	harness := NewTestHarness(t, agents,
//	    WithTimeout(30*time.Second),
//	    WithPersistence(tempDir, 0),
//	    WithGracefulDegradation(cfg),
//	)
//	defer harness.Cleanup()
//
// Key components:
//   - TestHarness: Complete stack with mock adapters
//   - EventCollector: Records all events for verification
//   - TimeoutWrapper: Prevents test hangs
//   - Mock configuration: SetMockResponse, SetMockError, SetMockDelay
//
// # Test Fixtures
//
// Common fixtures are provided for consistent test setup:
//
//   - FixtureSimpleConversation(): Two agents (Alice, Bob)
//   - FixtureLargeGroup(): Five agents for group testing
//   - FixtureMixedReliability(): Agents with different error behaviors
//
// # Assertions
//
// Helper functions for common assertions:
//
//   - AssertEventSequence(t, collector, types...): Verify event order
//   - AssertEventCount(t, collector, type, expected): Verify event counts
//   - AssertNoErrors(t, collector): Verify no error events
//   - AssertMessageCount(t, harness, expected): Verify message count
//
// # Performance Targets
//
// The benchmark tests compare against these targets:
//
//   - Single message latency: < 100ms with mock adapter
//   - Event bus throughput: > 10,000 events/second
//   - TUI render time: < 16ms for 60fps
//   - Memory per message: < 1KB average
//
// # Platform-Specific Notes
//
// Windows:
//   - Timer resolution is ~15.6ms (tests use >= 20ms delays)
//   - Use filepath.Join for cross-platform paths
//
// macOS/Linux:
//   - No known platform-specific issues
//
// # Adding New Tests
//
// 1. Add the "//go:build e2e" tag at the top of the file
// 2. Use TimeoutWrapper to prevent hanging tests
// 3. Use TestHarness for consistent setup
// 4. Use fixtures for common agent configurations
// 5. Use assertion helpers for clear failure messages
// 6. Clean up resources with defer harness.Cleanup()
//
// Example test:
//
//	func TestNewFeature(t *testing.T) {
//	    TimeoutWrapper(t, 10*time.Second, func(t *testing.T) {
//	        agents := FixtureSimpleConversation()
//	        harness := NewTestHarness(t, agents)
//	        defer harness.Cleanup()
//
//	        harness.SetMockResponse("alice", "Test response")
//	        harness.Manager.Start()
//
//	        responses, err := SendTestMessage(context.Background(), harness, "Test")
//	        if err != nil {
//	            t.Fatalf("unexpected error: %v", err)
//	        }
//
//	        AssertEventCount(t, harness.EventLog, core.EventAgentDone, 2)
//	    })
//	}
//
// # Test Coverage
//
// Generate coverage report:
//
//	go test -tags e2e -coverprofile=coverage.out ./pkg/...
//	go tool cover -html=coverage.out -o coverage.html
//
// Target coverage: > 80% on all core packages.
//
// # Known Test Flakiness
//
// Some tests may be flaky due to:
//
//   - Timing-dependent assertions (mitigated by generous timeouts)
//   - Event ordering in parallel execution (mitigated by WaitForResponses)
//   - Platform-specific timer resolution (mitigated by platform-aware delays)
//
// If a test is flaky, increase timeouts or add explicit synchronization.
package e2e
