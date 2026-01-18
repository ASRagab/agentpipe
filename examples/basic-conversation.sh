#!/bin/bash
# basic-conversation.sh - Simple two-agent conversation
#
# This script demonstrates the parallel execution engine with two agents
# having a conversation about a given topic.
#
# Prerequisites:
#   - agentpipe installed and in PATH
#   - At least one agent CLI installed (run: agentpipe doctor)
#   - For OpenRouter agents: OPENROUTER_API_KEY environment variable
#   - For Claude API: ANTHROPIC_API_KEY environment variable
#
# Usage:
#   ./basic-conversation.sh                    # Use default config
#   ./basic-conversation.sh my-config.yaml     # Use custom config
#
# Features demonstrated:
#   - Parallel execution
#   - Auto-save enabled
#   - TUI interface
#   - Metrics display

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG="${1:-$SCRIPT_DIR/demo-config.yaml}"

echo "========================================"
echo "  AgentPipe - Basic Conversation"
echo "========================================"
echo ""
echo "Config: $CONFIG"
echo ""

# Check if config exists
if [ ! -f "$CONFIG" ]; then
    echo "Error: Config file not found: $CONFIG"
    echo ""
    echo "Available configs:"
    ls -1 "$SCRIPT_DIR"/*.yaml 2>/dev/null || echo "  (none found)"
    echo ""
    echo "You can also create one using examples/demo-config.yaml as a template."
    exit 1
fi

# Check for required environment variables based on config
if grep -q "openrouter" "$CONFIG" && [ -z "$OPENROUTER_API_KEY" ]; then
    echo "Warning: OPENROUTER_API_KEY not set (may be needed for OpenRouter agents)"
fi

if grep -q "claude-api" "$CONFIG" && [ -z "$ANTHROPIC_API_KEY" ]; then
    echo "Warning: ANTHROPIC_API_KEY not set (may be needed for Claude API agents)"
fi

echo "Starting conversation with TUI..."
echo "(Press Ctrl+C to exit, use 'q' in TUI)"
echo ""

# Run with TUI enabled, auto-save on
exec agentpipe run \
    --config "$CONFIG" \
    --tui \
    --auto-save \
    --timeout 60
