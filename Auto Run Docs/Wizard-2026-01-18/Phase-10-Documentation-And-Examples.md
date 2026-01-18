# Phase 10: Documentation and Examples

This phase creates comprehensive documentation and example configurations that enable users to quickly understand and adopt AgentPipe v2. Clear documentation reduces support burden and encourages adoption. Examples demonstrate common use cases and best practices.

## Tasks

- [x] Update main README.md for v2:
  - Add v2 features section highlighting parallel execution, streaming, new TUI
  - Update installation instructions
  - Add quick start section for v2
  - Update configuration example to v2 format
  - Add performance comparison vs v1
  - Update screenshots showing new TUI
  - Note v1 deprecation timeline

  **Completed 2026-01-18**: Added comprehensive v2 documentation to README.md:
  - New "What's New in v2" section with feature comparison table (v1 vs v2)
  - v2 Quick Start section with CLI examples (--v2 flag, --resume, --export, --migrate-config)
  - Performance comparison table (2.25x faster, 3x TUI render, 40% less memory)
  - v1 deprecation notice with timeline (maintenance mode, v2 recommended, migration path)
  - Updated "What's New" section to highlight v0.7.0 with v2 engine
  - Added "After Installation" section recommending v2 and AGENTPIPE_V2 env var
  - Configuration section already had v1/v2 migration documentation

- [x] Create v2-specific documentation in `docs/v2/`:
  - `quickstart.md`: 5-minute getting started guide
  - `configuration.md`: Full config reference with all options
  - `adapters.md`: Guide to available adapters and how to configure
  - `tui.md`: TUI usage guide with keyboard shortcuts
  - `architecture.md`: Technical overview for contributors
  - `migration.md`: Guide for migrating from v1

  **Completed 2026-01-18**: Created comprehensive v2 documentation in `docs/v2/` folder:
  - `README.md`: Documentation index with navigation and document map
  - `quickstart.md`: 5-minute getting started guide with installation, config setup, and first conversation
  - `configuration.md`: Full config reference with all options, types, defaults, and examples
  - `adapters.md`: Complete adapter documentation for openrouter, claude-api, claude-cli, gemini-cli
  - `tui.md`: TUI usage guide with layout overview, keyboard shortcuts, and features
  - `architecture.md`: Technical overview for contributors with package structure and data flow
  - `migration.md`: Comprehensive v1 to v2 migration guide with field mappings and examples
  All documents include YAML frontmatter with wiki-link cross-references for graph exploration.

- [x] Create configuration reference in `docs/v2/configuration.md`:
  - Document every config field with type, default, description
  - Group by section (conversation, agents, tui, logging, persistence)
  - Include example values for each field
  - Document environment variable overrides
  - Document v1 compatibility and migration

  **Completed 2026-01-18**: Included in the comprehensive docs/v2/configuration.md file with:
  - All config sections documented (conversation, agents, tui, logging, persistence)
  - Every field with type, default, and description in tables
  - Complete example configuration with all options
  - Environment variable overrides section
  - CLI flags section
  - Validation and troubleshooting tips

- [x] Create adapter documentation in `docs/v2/adapters.md`:
  - Document each adapter: openrouter, claude-api, claude-cli, gemini-cli
  - Required config fields for each
  - Environment variables for API keys
  - Supported models for each adapter
  - Performance characteristics
  - Known limitations

  **Completed 2026-01-18**: Included in the comprehensive docs/v2/adapters.md file with:
  - All four adapters documented (openrouter, claude-api, claude, gemini)
  - Required and optional config fields for each
  - Environment variables with setup instructions
  - Popular models with pricing tables
  - Performance characteristics (latency, streaming, retry logic)
  - Error handling and retry behavior
  - Custom adapter creation guide

- [x] Create troubleshooting guide in `docs/v2/troubleshooting.md`:
  - Common error messages and solutions
  - API key configuration issues
  - Network and timeout issues
  - TUI rendering issues
  - Performance troubleshooting
  - Debug mode and logging

  **Completed 2026-01-18**: Created comprehensive troubleshooting guide in `docs/v2/troubleshooting.md`:
  - Quick diagnostics section with doctor command usage
  - API key configuration issues (not found, auth failed, 401/403 errors)
  - Network and timeout issues (connection refused, timeouts, rate limits)
  - Configuration issues (missing agents, unknown adapters, invalid durations)
  - TUI rendering issues (terminal size, Unicode, colors, input problems)
  - Performance troubleshooting (slow responses, high CPU, memory issues)
  - Debug mode and logging configuration
  - CLI-specific issues (Claude CLI not found, health check failures)
  - Persistence issues (save/load problems)
  - Migration issues from v1 to v2
  - Reporting bugs section with required information
  Updated docs/v2/README.md to include troubleshooting in User Guides table and Document Map.

- [x] Create example configurations in `examples/v2/`:
  - `minimal.yaml`: Simplest working config with one agent
  - `two-agents.yaml`: Two agents for comparison
  - `multi-model.yaml`: Multiple models from same provider
  - `code-review.yaml`: Config optimized for code discussion
  - `brainstorm.yaml`: Config for creative brainstorming
  - `research.yaml`: Config for research/fact-finding
  - Each with comments explaining the choices

  **Completed 2026-01-18**: Created 6 comprehensive v2 example configurations:
  - `minimal.yaml`: Single Claude 3 Haiku agent via OpenRouter - simplest working config
  - `two-agents.yaml`: Claude Sonnet + GPT-4 Turbo for side-by-side comparison
  - `multi-model.yaml`: 4 agents (Haiku, Sonnet, GPT-4o-mini, Gemini Flash) across price tiers
  - `code-review.yaml`: 3 specialized reviewers (security, performance, maintainability) with detailed system prompts
  - `brainstorm.yaml`: 4 creative personas (Innovator, Strategist, Skeptic, Synthesizer) with high temperatures
  - `research.yaml`: 3 research-focused agents (Investigator, Fact-Checker, Summarizer) with low temperatures for accuracy
  All configs include detailed comments explaining model choices, temperature settings, and use cases.
  Updated docs/v2/README.md with new Example Configurations section.

- [ ] Create code examples in `examples/v2/code/`:
  - `custom-adapter/`: Example of implementing a custom adapter
  - `programmatic/`: Using v2 as a library in Go code
  - `webhook/`: Emitting events to webhook for external integration
  - Each with README explaining the example

- [ ] Write API documentation (godoc):
  - Document all exported types in pkg/v2/core
  - Document all interfaces with usage examples
  - Document all public functions
  - Include package-level documentation
  - Verify godoc renders correctly

- [ ] Create changelog entry for v2.0.0:
  - List all new features
  - List breaking changes from v1
  - List deprecated features
  - List known issues
  - Include migration notes

- [ ] Create video/GIF demos:
  - Record terminal session showing basic conversation
  - Record parallel agent responses with streaming
  - Record TUI features (status, metrics, scrolling)
  - Save as GIF for README
  - Host video on project page or YouTube

- [ ] Update project metadata:
  - Update package description
  - Update keywords for discoverability
  - Update badges in README
  - Ensure license file is current
  - Update CONTRIBUTING.md for v2 development
