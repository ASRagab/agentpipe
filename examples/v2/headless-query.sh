#!/bin/bash
# headless-query.sh - Single question, piped output using v2 engine
#
# This script demonstrates the v2 headless mode where a question is piped
# to the agents and the response is output to stdout.
#
# Great for:
#   - Scripting and automation
#   - CI/CD pipelines
#   - Integration with other tools
#   - Quick one-off queries
#
# Prerequisites:
#   - agentpipe installed and in PATH
#   - At least one agent CLI installed (run: agentpipe doctor)
#   - For OpenRouter agents: OPENROUTER_API_KEY environment variable
#
# Usage:
#   ./headless-query.sh "What is the capital of France?"
#   ./headless-query.sh "Explain quicksort" demo-config.yaml
#   echo "Hello" | ./headless-query.sh
#
# Features demonstrated:
#   - v2 headless mode (non-TUI)
#   - Piped input support
#   - Clean stdout output (metrics to stderr)
#   - Auto-save disabled for quick queries

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Default config
CONFIG="${2:-$SCRIPT_DIR/demo-config.yaml}"

# Check if we have piped input or command line argument
if [ -t 0 ]; then
    # No piped input, use command line argument
    if [ -z "$1" ]; then
        echo "Usage: $0 \"Your question here\" [config.yaml]" >&2
        echo "   or: echo \"Your question\" | $0 [config.yaml]" >&2
        exit 1
    fi
    QUERY="$1"
else
    # Piped input
    QUERY=$(cat)
fi

# Check if config exists
if [ ! -f "$CONFIG" ]; then
    echo "Error: Config file not found: $CONFIG" >&2
    exit 1
fi

# Log to stderr so stdout is clean for the response
echo "Query: $QUERY" >&2
echo "Config: $CONFIG" >&2
echo "---" >&2

# Pipe the query to agentpipe v2 in headless mode
# - No TUI (headless)
# - Auto-save disabled for quick queries
# - Response goes to stdout
echo "$QUERY" | agentpipe run --v2 \
    --config "$CONFIG" \
    --auto-save=false \
    --v2-timeout 90

# Note: The exit code reflects whether the command succeeded
# You can check $? after running this script

