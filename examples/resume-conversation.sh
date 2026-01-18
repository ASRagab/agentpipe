#!/bin/bash
# resume-conversation.sh - Save and resume conversation demo
#
# This script demonstrates the v2 persistence features:
#   - Auto-save conversations
#   - Resume previous conversations
#   - Export to Markdown
#
# Prerequisites:
#   - agentpipe installed and in PATH
#   - At least one agent CLI installed (run: agentpipe doctor)
#
# Usage:
#   ./resume-conversation.sh start           # Start new conversation
#   ./resume-conversation.sh resume          # Resume latest conversation
#   ./resume-conversation.sh resume abc123   # Resume specific conversation
#   ./resume-conversation.sh list            # List saved conversations
#   ./resume-conversation.sh export abc123   # Export to Markdown
#
# Features demonstrated:
#   - Auto-save and resume
#   - Conversation ID management
#   - Markdown export

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG="$SCRIPT_DIR/demo-config.yaml"
SAVE_DIR="${AGENTPIPE_SAVE_DIR:-$HOME/.agentpipe/v2/chats}"

show_help() {
    echo "Resume Conversation Demo - v2 Persistence Features"
    echo ""
    echo "Usage: $0 <command> [options]"
    echo ""
    echo "Commands:"
    echo "  start             Start a new conversation (auto-saves)"
    echo "  resume [id]       Resume conversation (default: latest)"
    echo "  list              List all saved conversations"
    echo "  export <id>       Export conversation to Markdown"
    echo "  help              Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 start                    # New conversation with auto-save"
    echo "  $0 resume                   # Resume the most recent conversation"
    echo "  $0 resume a1b2c3d4          # Resume specific conversation"
    echo "  $0 export a1b2c3d4          # Export to conversation_a1b2c3d4.md"
    echo ""
    echo "Environment:"
    echo "  AGENTPIPE_SAVE_DIR   Save directory (default: ~/.agentpipe/v2/chats)"
    echo "  Current: $SAVE_DIR"
}

list_conversations() {
    echo "Saved Conversations"
    echo "==================="
    echo ""

    if [ ! -d "$SAVE_DIR" ]; then
        echo "No conversations found. Save directory does not exist."
        echo "Start a conversation first: $0 start"
        return 0
    fi

    # List conversation files with modification time
    if ls "$SAVE_DIR"/*.json 1>/dev/null 2>&1; then
        echo "ID        | Modified             | File"
        echo "----------|----------------------|---------------------------"
        for f in "$SAVE_DIR"/*.json; do
            if [ -f "$f" ]; then
                basename=$(basename "$f" .json)
                modified=$(stat -f "%Sm" -t "%Y-%m-%d %H:%M" "$f" 2>/dev/null || stat -c "%y" "$f" 2>/dev/null | cut -d. -f1)
                # Show first 8 chars as ID
                id="${basename:0:8}"
                echo "$id  | $modified | $(basename "$f")"
            fi
        done
    else
        echo "No conversations found in $SAVE_DIR"
        echo "Start a conversation first: $0 start"
    fi
}

start_conversation() {
    echo "Starting new v2 conversation..."
    echo "Config: $CONFIG"
    echo "Save Dir: $SAVE_DIR"
    echo ""
    echo "Tip: When you exit (Ctrl+C or /quit), the conversation ID"
    echo "     will be displayed. Use it to resume later:"
    echo "     $0 resume <id>"
    echo ""

    # Start with auto-save enabled
    agentpipe run \
        --config "$CONFIG" \
        --auto-save \
        --save-dir "$SAVE_DIR" \
        --tui
}

resume_conversation() {
    local resume_id="${1:-latest}"

    echo "Resuming conversation: $resume_id"
    echo "Save Dir: $SAVE_DIR"
    echo ""

    agentpipe run \
        --config "$CONFIG" \
        --resume "$resume_id" \
        --save-dir "$SAVE_DIR" \
        --auto-save \
        --tui
}

export_conversation() {
    local conv_id="$1"

    if [ -z "$conv_id" ]; then
        echo "Error: Conversation ID required"
        echo "Usage: $0 export <conversation-id>"
        exit 1
    fi

    local output_file="conversation_${conv_id}.md"

    echo "Exporting conversation $conv_id to $output_file..."

    # Use v2 export feature
    agentpipe run \
        --config "$CONFIG" \
        --resume "$conv_id" \
        --save-dir "$SAVE_DIR" \
        --export "$output_file" \
        --auto-save=false

    if [ -f "$output_file" ]; then
        echo "Exported to: $output_file"
        echo ""
        echo "Preview (first 20 lines):"
        echo "========================="
        head -20 "$output_file"
    fi
}

# Main command dispatch
case "${1:-help}" in
    start)
        start_conversation
        ;;
    resume)
        resume_conversation "$2"
        ;;
    list)
        list_conversations
        ;;
    export)
        export_conversation "$2"
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        echo "Unknown command: $1"
        echo ""
        show_help
        exit 1
        ;;
esac
