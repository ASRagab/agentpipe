#!/bin/bash

# Setup script for creating the Homebrew tap repository
# Run this after creating the repository on GitHub

set -e

echo "🍺 Setting up Homebrew tap for ASRagab/agentpipe"
echo ""

# Check if gh CLI is installed
if ! command -v gh &> /dev/null; then
    echo "⚠️  GitHub CLI (gh) is not installed."
    echo "   Install it with: brew install gh"
    echo "   Then authenticate: gh auth login"
    exit 1
fi

# Create the tap repository on GitHub
echo "📦 Creating homebrew-tap repository on GitHub..."
gh repo create kevinelliott/homebrew-tap --public --description "Homebrew formulae for kevinelliott's projects" --clone || {
    echo "   Repository may already exist, continuing..."
}

# Set up local directory
TAP_DIR="$HOME/homebrew-tap"
if [ ! -d "$TAP_DIR" ]; then
    echo "📂 Cloning tap repository..."
    gh repo clone kevinelliott/homebrew-tap "$TAP_DIR"
fi

cd "$TAP_DIR"

# Create Formula directory
echo "📁 Creating Formula directory..."
mkdir -p Formula

# Copy the formula
echo "📝 Copying agentpipe formula..."
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
AGENTPIPE_DIR="$(dirname "$SCRIPT_DIR")"

if [ -f "$AGENTPIPE_DIR/Formula/agentpipe-multiarch.rb" ]; then
    cp "$AGENTPIPE_DIR/Formula/agentpipe-multiarch.rb" Formula/agentpipe.rb
elif [ -f "$AGENTPIPE_DIR/Formula/agentpipe.rb" ]; then
    cp "$AGENTPIPE_DIR/Formula/agentpipe.rb" Formula/
else
    echo "❌ Could not find agentpipe formula"
    exit 1
fi

# Create README
echo "📄 Creating README..."
cat > README.md << 'EOF'
# kevinelliott Homebrew Tap

This tap contains formulae for kevinelliott's projects.

## Installation

```bash
brew tap kevinelliott/tap
```

## Available Formulae

### AgentPipe

AgentPipe orchestrates conversations between multiple AI CLI agents (Claude, Gemini, Qwen, Codex, Ollama).

```bash
# Install from tap
brew install kevinelliott/tap/agentpipe

# Or tap first, then install
brew tap kevinelliott/tap
brew install agentpipe
```

#### Features
- Multi-agent conversations with various AI CLIs
- Beautiful TUI interface with colored output
- Response metrics tracking (duration, tokens, cost)
- Chat logging to ~/.agentpipe/chats/
- Health checks for all agents
- YAML configuration support

#### Quick Start
```bash
# Check available agents
agentpipe doctor

# Start a conversation
agentpipe run -a claude:Alice -a gemini:Bob -p "Let's discuss AI"

# Use enhanced TUI
agentpipe run -c examples/brainstorm.yaml --enhanced-tui
```

## Development

To install the latest development version:

```bash
brew install --HEAD kevinelliott/tap/agentpipe
```

## Issues

For issues with formulae, please file them at the respective project repositories:
- [AgentPipe Issues](https://github.com/ASRagab/agentpipe/issues)

For tap-specific issues:
- [Tap Issues](https://github.com/kevinelliott/homebrew-tap/issues)
EOF

# Commit and push
echo "💾 Committing changes..."
git add .
git commit -m "Add agentpipe formula" || echo "No changes to commit"

echo "🚀 Pushing to GitHub..."
git push origin main || git push --set-upstream origin main

echo ""
echo "✅ Homebrew tap setup complete!"
echo ""
echo "Next steps:"
echo "1. Create a release in the agentpipe repository:"
echo "   cd $AGENTPIPE_DIR"
echo "   make release-build"
echo "   gh release create v0.1.0 dist/*.tar.gz --title 'AgentPipe v0.1.0'"
echo ""
echo "2. Update the formula with SHA256 hashes:"
echo "   shasum -a 256 dist/*.tar.gz"
echo "   Edit $TAP_DIR/Formula/agentpipe.rb with the hashes"
echo ""
echo "3. Test the tap:"
echo "   brew tap kevinelliott/tap"
echo "   brew install agentpipe"
echo ""
echo "📚 Full instructions: $AGENTPIPE_DIR/HOMEBREW_TAP_SETUP.md"