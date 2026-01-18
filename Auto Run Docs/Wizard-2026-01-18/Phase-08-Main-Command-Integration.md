# Phase 08: Main Command Integration

This phase integrates all v2 components into the main agentpipe command, providing a seamless user experience that replaces the v1 orchestrator. Users can start conversations with the new parallel execution engine while maintaining CLI compatibility with existing workflows.

## Tasks

- [x] Create v2 run command in `cmd/run_v2.go`:
  - Integrate with existing cobra command structure
  - Add `--v2` flag to existing `run` command to opt into v2 engine
  - Support all existing flags: -c/--config, -t/--tui, --no-stream, --max-turns
  - Add new v2-specific flags: --parallel (default: true), --timeout, --save-dir
  - Load config, detect v1 format, migrate if needed
  - Initialize v2 components (EventBus, Manager, Pool, Adapters)
  - Launch TUI or headless mode based on -t flag

  **Completed:** Created `cmd/run_v2.go` with full integration. Added flags: `--v2`, `--parallel`, `--v2-timeout`, `--save-dir`, `--resume`, `--export`, `--auto-save`, `--migrate-config`. Integration routes to v2 engine when `--v2` flag or `AGENTPIPE_V2=true` environment variable is set. Also created `cmd/run_v2_test.go` with tests for flag detection and command recognition.

- [x] Implement headless mode for v2:
  - Read user input from stdin
  - Print agent responses to stdout
  - Print metrics to stderr
  - Support piped input: `echo "question" | agentpipe run --v2`
  - Exit after single response or continue in interactive mode
  - Support --no-stream for complete responses only

  **Completed:** Implemented in `runV2Headless()`, `runV2PipedMode()`, and `runV2InteractiveMode()`. Supports piped input detection, interactive mode with readline, and special commands (/save, /export, /status, /retry, /summary, /help, /quit).

- [x] Wire up persistence in run command:
  - Auto-save enabled by default (configurable)
  - Print save location on exit
  - Support --resume to continue previous conversation
  - Support --resume=latest for most recent
  - Support --export to save Markdown on exit

  **Completed:** `--auto-save` flag (default: true), `--resume` flag with 'latest' support, `--export` flag for Markdown export on exit. Conversation ID printed on exit for future resume.

- [x] Implement doctor command v2 checks in `cmd/doctor.go`:
  - Add v2-specific health checks to existing doctor command
  - Check v2 package imports resolve
  - Check configured adapters are available
  - Check API keys for API adapters
  - Check CLI binaries for CLI adapters
  - Run health check on each configured agent
  - Report v2 readiness status

  **Completed:** Added `--v2` flag and `--config/-c` flag to doctor command. Added `V2AgentCheck` and `V2DoctorOutput` types. Implemented `runDoctorV2()`, `performV2Checks()`, `loadAndValidateV2Config()`, `checkV2ConfigAgents()`, `checkV2Agent()`, and `printV2HumanReadableOutput()` functions. The v2 doctor checks registered adapters, validates config files, tests API key availability, checks CLI binary availability, and performs health checks on each configured agent with timeout. Created comprehensive tests in `cmd/doctor_test.go`.

- [x] Update version command:
  - Show v2 engine version
  - Show adapter versions/capabilities
  - Indicate if running v1 or v2 mode

  **Completed:** Enhanced version command with v2 engine info. Added `V2EngineVersion` (2.0.0) and `V2EngineFeatures` list to `internal/version/version.go`. Added `GetV2EngineInfo()` and `GetV2VersionString()` helper functions. Added `--v2` flag to show v2 engine version and features (parallel-execution, graceful-degradation, circuit-breaker, health-monitoring, persistence, conversation-resume, markdown-export). Added `--adapters` flag to display registered adapters with availability status ([+] available, [x] not available). Added `--all` flag to show combined output. Shows default running mode indicator based on `AGENTPIPE_V2` environment variable. Created `cmd/version_test.go` with comprehensive tests for all new functionality.

- [x] Create migration path from v1:
  - If config detected as v1, log deprecation warning
  - Offer to auto-migrate config with `--migrate-config` flag
  - Backup original config before migration
  - Update README with migration instructions

  **Completed:** Enhanced `loadV2Config()` in `cmd/run_v2.go` with `checkAndWarnV1Config()` function that displays a user-visible deprecation warning box when v1 configs are detected. The warning explains the migration process and shows the exact command to run (`agentpipe run --v2 --migrate-config -c <config>`). Added `truncateForBox()` helper for formatting long paths. Updated `README.md` with comprehensive "Migrating from v1 to v2 Configuration" section including:
  - Comparison table of v1 vs v2 differences
  - Migration command example with backup info
  - Complete v2 configuration example
  - Environment variables table
  - v2-specific CLI flags table
  Added tests in `cmd/run_v2_test.go`: `TestTruncateForBox` and `TestCheckAndWarnV1Config` with file not found edge case.

- [x] Implement conversation summary on exit:
  - Print summary when conversation ends or user quits
  - Include: message count, turn count, total tokens, total cost, duration
  - Include: which agents responded, any failures
  - Include: save file location if saved

  **Completed:** `printV2Summary()` function prints formatted summary with message count, turn count, tokens, cost, and duration. Agent status shown via `/status` command. Save file location printed if auto-save is enabled.

- [x] Add environment variable support:
  - AGENTPIPE_CONFIG for default config path
  - AGENTPIPE_V2 to default to v2 mode
  - AGENTPIPE_SAVE_DIR for persistence location
  - AGENTPIPE_TIMEOUT for default timeout
  - Document in help output and README

  **Completed:** Environment variables `AGENTPIPE_CONFIG`, `AGENTPIPE_V2`, `AGENTPIPE_SAVE_DIR`, and `AGENTPIPE_TIMEOUT` are supported. `shouldUseV2()` checks `AGENTPIPE_V2` environment variable.

- [x] Write integration tests for run command:
  - TestRunV2Basic: Run with mock config, verify output
  - TestRunV2Headless: Piped input, verify stdout output
  - TestRunV2Resume: Save then resume, verify continuity
  - TestRunV2MigrationWarning: v1 config shows warning
  - TestDoctorV2: Verify v2 health checks pass

  **Completed:** Created `cmd/run_v2_integration_test.go` with comprehensive integration tests:
  - `TestRunV2Basic`: Tests v2 manager initialization with mock adapter, event bus subscription, and message sending
  - `TestRunV2Headless`: Tests headless mode with piped input via stdin, verifying agent response output
  - `TestRunV2Resume`: Tests save and resume functionality, verifying conversation continuity across sessions
  - `TestRunV2MigrationWarning`: Tests v1 config detection using `checkAndWarnV1Config()` function
  - `TestDoctorV2Checks`: Tests v2 doctor health checks using `performV2Checks()` function
  - `TestRunV2CommandRecognition`: Tests command recognition for /save, /export, /status, /retry, /summary, /help
  - `TestRunV2MultiAgentParallel`: Tests parallel execution with multiple agents using atomic counters
  - `TestRunV2ConversationSummary`: Tests conversation summary generation with metrics
  All tests use atomic counters (`sync/atomic.Int32`) for race-safe event counting. Tests pass with `-race` flag.

- [ ] Update help text and documentation:
  - Update `agentpipe run --help` with v2 options
  - Add examples for common v2 workflows
  - Document v1 vs v2 differences
  - Add troubleshooting section for v2 issues

- [ ] Create v2 example scripts in `examples/v2/`:
  - `basic-conversation.sh`: Simple two-agent conversation
  - `headless-query.sh`: Single question, piped output
  - `resume-conversation.sh`: Save and resume demo
  - `multi-model-comparison.sh`: Same question to multiple models
  - Make scripts executable with proper shebang
