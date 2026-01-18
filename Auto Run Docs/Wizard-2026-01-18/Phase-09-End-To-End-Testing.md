# Phase 09: End-to-End Testing

This phase implements comprehensive end-to-end testing to ensure all v2 components work together correctly. Tests cover the full conversation flow from user input through agent responses, including TUI rendering, persistence, and error scenarios. This ensures MVP quality and reliability.

## Tasks

- [x] Create E2E test framework in `pkg/v2/e2e/`:
  - Test harness that sets up full v2 stack with mock adapters
  - Helper functions: CreateTestManager(), SendTestMessage(), WaitForResponses()
  - Mock event collector for verifying event sequences
  - Test fixtures for common scenarios
  - Timeout wrappers to fail slow tests

- [x] Write conversation flow E2E tests in `pkg/v2/e2e/conversation_test.go`:
  - TestSingleUserMessage: User sends message, both agents respond
  - TestMultipleTurns: Three-turn conversation with different messages
  - TestLongConversation: 20+ message conversation, verify memory/performance
  - TestAgentContextAwareness: Verify agents receive full message history
  - TestMessageOrdering: Verify messages ordered by start time, not completion

- [x] Write parallel execution E2E tests:
  - TestParallelTiming: Verify N agents complete in O(1) not O(N)
  - TestMixedResponseTimes: Fast and slow agents, verify fast appears first
  - TestAllAgentsFail: All agents fail, verify graceful handling
  - TestPartialFailure: 2/3 agents succeed, verify partial results shown
  - TestAgentTimeout: One agent times out, others succeed

- [x] Write persistence E2E tests:
  - TestSaveAndResume: Full conversation, save, resume, continue
  - TestResumeWithMissingAgent: Resume with agent no longer available
  - TestExportWhileActive: Export during active conversation
  - TestAutoSave: Verify auto-save triggers on interval and after responses
  - TestConcurrentSave: Multiple saves don't corrupt file

- [x] Write TUI E2E tests in `pkg/v2/e2e/tui_test.go`:
  - Use teatest or similar for bubbletea testing
  - TestTUIRender: Verify initial layout renders correctly
  - TestTUIMessageInput: Type message, submit, verify sent
  - TestTUIStreamingDisplay: Verify chunks appear incrementally
  - TestTUIAgentStatus: Verify status updates (typing, done, error)
  - TestTUIKeyboardNav: Verify all shortcuts work
  - TestTUIResize: Verify layout adjusts on terminal resize

- [x] Write error scenario E2E tests:
  - TestNetworkFailureRecovery: Simulate network drop, verify retry
  - TestRateLimitRecovery: Simulate 429, verify backoff and retry
  - TestAuthFailure: Invalid API key, verify clear error message
  - TestCircuitBreakerTrip: Multiple failures, verify circuit opens
  - TestGracefulDegradation: One agent permanently fails, conversation continues

- [x] Write configuration E2E tests:
  - TestV1ConfigMigration: Load v1 config, verify works
  - TestMissingConfig: No config file, verify helpful error
  - TestInvalidConfig: Malformed YAML, verify parse error
  - TestMissingAPIKey: API adapter with no key, verify error message
  - TestUnknownAdapter: Unknown adapter name, verify error

- [x] Create benchmark tests in `pkg/v2/e2e/benchmark_test.go`:
  - BenchmarkSingleMessage: Time for single message/response
  - BenchmarkParallelExecution: Overhead of parallel execution
  - BenchmarkEventBus: Events per second throughput
  - BenchmarkTUIRender: Render time per frame
  - BenchmarkMemoryUsage: Memory for 100-message conversation
  - Compare against performance targets from MVP plan

- [x] Write cross-platform tests:
  - Test on macOS, Linux, Windows (via GitHub Actions)
  - Verify TUI renders correctly on each platform
  - Verify file paths work on each platform
  - Verify CLI adapters work on each platform
  - Document any platform-specific issues

- [x] Create test coverage report:
  - Run all tests with coverage: `go test -coverprofile=coverage.out ./pkg/v2/...`
  - Generate HTML report
  - Verify >80% coverage on all core packages
  - Identify and document any uncovered paths
  - Add tests for uncovered critical paths

- [x] Create E2E test documentation:
  - Document how to run E2E tests
  - Document test fixtures and their purposes
  - Document how to add new E2E tests
  - Document known test flakiness and mitigations
