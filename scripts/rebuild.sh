#!/bin/bash
# Rebuild agentpipe and install to asdf Go bin

set -e

cd "$(dirname "$0")/.."

echo "Building agentpipe..."
go build -o ~/.asdf/installs/golang/1.24.12/bin/agentpipe .

echo "✓ Build complete!"
echo "Binary location: ~/.asdf/installs/golang/1.24.12/bin/agentpipe"
agentpipe version
