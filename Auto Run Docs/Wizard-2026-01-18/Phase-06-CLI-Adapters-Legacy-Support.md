# Phase 06: CLI Adapters and Legacy Support

This phase implements CLI-based adapters for agents that operate through command-line tools (Claude CLI, Gemini CLI) rather than direct API calls. This provides backwards compatibility with existing v1 configurations and supports users who prefer CLI-based interactions.

## Tasks

- [x] Implement CLI adapter base in `pkg/v2/adapters/cli/base.go`:
  - CLIAdapter interface extending AgentAdapter with CLIPath(), CLIArgs() methods
  - BaseCLIAdapter struct with cliPath, model, extraFlags fields
  - FindCLIPath() that checks PATH, common locations (/usr/local/bin, ~/.local/bin)
  - ExecuteCommand() wrapper around exec.Command with context timeout
  - ParseCLIOutput() for extracting response from CLI output
  - HandleCLIError() for consistent error messages
  - **Completed 2026-01-18**: Also includes BuildConversationPrompt(), FilterRelevantMessages(), EstimateTokens() helpers

- [x] Implement Claude CLI adapter in `pkg/v2/adapters/cli/claude.go`:
  - ClaudeCLIAdapter struct embedding BaseCLIAdapter
  - Initialize() that finds claude binary, validates it exists and is executable
  - SendMessage() that:
    - Formats messages as conversation history
    - Executes: `claude -p "message" --no-stream` with stdin for history
    - Parses response text from stdout
    - Returns error with stderr on failure
  - StreamMessage() that:
    - Executes: `claude -p "message"` (streaming mode)
    - Reads stdout line by line, writes to writer
    - Handles partial line buffering
  - IsAvailable() checking claude binary exists
  - HealthCheck() running `claude --version`
  - Register as "claude-cli"
  - **Completed 2026-01-18**: Full implementation with cost estimation (Haiku/Sonnet/Opus pricing)

- [x] Implement Gemini CLI adapter in `pkg/v2/adapters/cli/gemini.go`:
  - GeminiCLIAdapter struct embedding BaseCLIAdapter
  - Initialize() that finds gemini binary
  - SendMessage() that:
    - Executes: `gemini chat "message"`
    - Parses response from stdout
  - StreamMessage() that handles Gemini's output format
  - IsAvailable(), HealthCheck() implementations
  - Register as "gemini-cli"
  - **Completed 2026-01-18**: Full implementation with output cleaning (removes credentials/trace noise), cost estimation (Pro/Flash/Nano pricing)

- [x] Implement generic CLI adapter in `pkg/v2/adapters/cli/generic.go`:
  - GenericCLIAdapter for any CLI tool following a configurable pattern
  - Config fields: cli_path, prompt_flag, history_flag, stream_flag, output_parser
  - Built-in output parsers: plain, json, markdown
  - Allow custom command templates
  - Register as "cli-generic"
  - **Completed 2026-01-18**: Full implementation with OutputParser enum (plain/json/markdown), JSON path extraction, configurable stdin/args modes

- [x] Add CLI adapter tests in `pkg/v2/adapters/cli/cli_test.go`:
  - Create mock CLI scripts for testing (shell scripts that echo responses)
  - TestClaudeCLIAdapter: Mock claude CLI, verify command formatting
  - TestGeminiCLIAdapter: Mock gemini CLI, verify command formatting
  - TestCLINotFound: Verify clear error when CLI not installed
  - TestCLITimeout: Verify context timeout cancels subprocess
  - TestCLIStreamOutput: Verify streaming writes chunks correctly
  - Skip tests if actual CLIs not installed (use build tags)
  - **Completed 2026-01-18**: 30+ test cases covering all adapters, streaming, timeouts, error handling, cost estimation, adapter registration

- [x] Implement v1 configuration compatibility in `pkg/v2/config/migrate.go`:
  - DetectV1Config() checking for v1-style fields (no "adapter" field)
  - MigrateV1Config() that:
    - Maps `type: claude` to `adapter: claude-api` (if ANTHROPIC_API_KEY set) or `adapter: claude-cli`
    - Maps `type: gemini` to `adapter: gemini-cli`
    - Maps `type: openrouter` to `adapter: openrouter`
    - Preserves all other fields
    - Logs migration warnings
  - AutoMigrateOnLoad option in config loader
  - SaveMigratedConfig() to write updated config with backup
  - **Completed 2026-01-18**: Full implementation with V1Config struct, orchestrator migration, automatic adapter detection with API key preference

- [x] Update configuration loader to handle both formats:
  - LoadConfig() calls DetectV1Config() first
  - If v1 detected, call MigrateV1Config()
  - Log deprecation warning about v1 format
  - Continue with v2 initialization
  - **Completed 2026-01-18**: LoadConfigWithOptions() added with AutoMigrateV1 and SaveMigratedConfig options, seamless v1->v2 migration

- [x] Write migration tests in `pkg/v2/config/migrate_test.go`:
  - TestDetectV1Config: Correctly identifies v1 configs
  - TestMigrateV1Config: All agent types migrate correctly
  - TestMigratePreservesFields: Non-migrated fields preserved
  - TestMigrateWithAPIKey: Prefers API adapter when key available
  - TestLoadV1Config: End-to-end v1 config loading
  - **Completed 2026-01-18**: 20+ test cases covering detection, migration, field preservation, API key preference, end-to-end loading, save/backup functionality

- [x] Create example v1-compatible config in `examples/v2/legacy-config.yaml`:
  - Use v1 config format with type: claude, type: gemini
  - Document that this format is deprecated but supported
  - Include comment pointing to v2 config format
  - **Completed 2026-01-18**: Created comprehensive example with 4 agents (claude, gemini, qwen), full deprecation notice, migration behavior documentation

- [ ] Test CLI adapters with real CLIs (manual):
  - Test with Claude CLI if installed
  - Test with Gemini CLI if installed
  - Verify streaming works correctly
  - Verify error handling for network failures
  - Document any CLI-specific quirks discovered
