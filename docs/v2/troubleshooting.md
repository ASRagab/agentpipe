---
type: reference
title: AgentPipe v2 Troubleshooting Guide
created: 2026-01-18
tags:
  - v2
  - troubleshooting
  - debugging
  - errors
related:
  - "[[configuration]]"
  - "[[adapters]]"
  - "[[quickstart]]"
  - "[[tui]]"
---

# AgentPipe v2 Troubleshooting Guide

This guide covers common issues you might encounter when using AgentPipe v2 and how to resolve them.

## Quick Diagnostics

Run the v2 doctor command to check system health:

```bash
# Basic v2 health check
agentpipe doctor --v2

# Full check with config validation
agentpipe doctor --v2 -c ~/.agentpipe/config.yaml

# JSON output for scripting
agentpipe doctor --v2 --json
```

---

## Error Messages and Solutions

### API Key Configuration Issues

#### "API key not found"

The specified environment variable for your API key is not set.

**Solution:**

```bash
# Check if environment variable is set
echo $OPENROUTER_API_KEY
echo $ANTHROPIC_API_KEY
echo $GOOGLE_API_KEY

# Set the environment variable
export OPENROUTER_API_KEY="sk-or-v1-..."
export ANTHROPIC_API_KEY="sk-ant-..."

# Add to your shell profile for persistence
echo 'export OPENROUTER_API_KEY="sk-or-v1-..."' >> ~/.bashrc
# or ~/.zshrc for Zsh users
```

**Verify it's working:**

```bash
# Test that the key is accessible to child processes
env | grep API_KEY
```

#### "authentication failed" (401 Unauthorized)

Your API key is invalid, expired, or doesn't have the required permissions.

**Symptoms:**

- HTTP status code 401 or 403
- Error message mentioning "unauthorized" or "forbidden"

**Solutions:**

1. **Verify the API key is correct:**

   ```bash
   # For OpenRouter
   curl https://openrouter.ai/api/v1/auth/key \
     -H "Authorization: Bearer $OPENROUTER_API_KEY"

   # Should return your key info, not an error
   ```

2. **Check for typos or extra whitespace:**

   ```bash
   # Look for hidden characters
   echo -n "$OPENROUTER_API_KEY" | xxd | head
   ```

3. **Regenerate the API key** at your provider's console:
   - OpenRouter: [openrouter.ai/keys](https://openrouter.ai/keys)
   - Anthropic: [console.anthropic.com/settings/keys](https://console.anthropic.com/settings/keys)

4. **Check account status** - ensure your account is active and has billing set up

---

### Network and Timeout Issues

#### "connection refused" or "dial tcp: connect"

AgentPipe cannot connect to the API server.

**Causes:**

- No internet connection
- Firewall blocking outbound requests
- VPN/proxy issues
- API server outage

**Solutions:**

1. **Check internet connectivity:**

   ```bash
   ping google.com
   curl -I https://api.anthropic.com
   curl -I https://openrouter.ai
   ```

2. **Check firewall/proxy settings:**

   ```bash
   # If using a proxy, set environment variables
   export HTTP_PROXY=http://proxy:port
   export HTTPS_PROXY=http://proxy:port
   ```

3. **Check provider status pages:**
   - OpenRouter: [status.openrouter.ai](https://status.openrouter.ai)
   - Anthropic: Check their status page or Twitter

#### "timeout" or "context deadline exceeded"

The request took too long to complete.

**Causes:**

- Slow network connection
- Large response being generated
- Server under heavy load
- Per-agent timeout too short

**Solutions:**

1. **Increase timeout in config:**

   ```yaml
   conversation:
     timeout: 60s        # Increase from default 30s
     global_timeout: 5m  # For all agents combined

   agents:
     - id: claude
       timeout: 90s      # Per-agent override for slow models
       # ...
   ```

2. **Reduce response length:**

   ```yaml
   agents:
     - id: claude
       config:
         max_tokens: 500  # Limit response length
   ```

3. **Use faster models:**
   - Claude Haiku instead of Opus
   - GPT-3.5 Turbo instead of GPT-4

4. **Check your network latency:**

   ```bash
   # Measure latency to API endpoints
   time curl -o /dev/null -s https://api.anthropic.com/v1/messages
   ```

#### "rate limit exceeded" (429 Too Many Requests)

You've hit the API provider's rate limit.

**Symptoms:**

- Error message containing "rate limit" or "too many requests"
- HTTP status code 429
- `Retry-After` header in response

**Solutions:**

1. **Wait and retry** - AgentPipe automatically retries with exponential backoff (1s, 2s, 4s)

2. **Reduce request frequency:**
   - Send fewer messages per minute
   - Increase `max_tokens` to get more content per request

3. **Upgrade your API tier:**
   - Higher tiers have higher rate limits
   - Contact your provider for limit increases

4. **Use multiple API keys** (if your provider allows):

   ```yaml
   agents:
     - id: claude1
       config:
         api_key_env: ANTHROPIC_API_KEY_1
     - id: claude2
       config:
         api_key_env: ANTHROPIC_API_KEY_2
   ```

---

### Configuration Issues

#### "at least one agent must be configured"

Your configuration file has no agents defined.

**Solution:**

Add at least one agent to your config:

```yaml
agents:
  - id: my-agent
    type: openrouter
    name: My Agent
    model: anthropic/claude-3-haiku
    config:
      api_key_env: OPENROUTER_API_KEY
```

#### "agent X: id is required"

Every agent must have a unique `id` field.

**Incorrect:**

```yaml
agents:
  - type: openrouter      # Missing id!
    name: Claude
```

**Correct:**

```yaml
agents:
  - id: claude            # Required
    type: openrouter
    name: Claude
```

#### "unknown adapter: X"

The adapter type specified doesn't exist.

**Valid adapter names:**

- `openrouter` - OpenRouter API
- `claude-api` - Anthropic Claude API
- `claude` - Claude CLI
- `gemini` - Gemini CLI

**Check your config:**

```yaml
agents:
  - id: my-agent
    type: openrouter      # Must be one of the valid names
```

#### "invalid duration: X"

Duration values must use Go duration format.

**Incorrect:**

```yaml
conversation:
  timeout: 30           # Missing unit!
  timeout: 30 seconds   # Wrong format
```

**Correct:**

```yaml
conversation:
  timeout: 30s          # seconds
  timeout: 2m           # minutes
  timeout: 1h30m        # hours and minutes
```

#### Model not found or invalid

The model identifier is incorrect or not available.

**OpenRouter models:** Use format `provider/model-name`

- `openai/gpt-4-turbo`
- `anthropic/claude-3-opus`
- `google/gemini-pro`

**Claude API models:** Use full model ID

- `claude-sonnet-4-20250514`
- `claude-3-opus-20240229`

**Check available models:**

- OpenRouter: [openrouter.ai/models](https://openrouter.ai/models)
- Anthropic: [docs.anthropic.com/en/docs/models-overview](https://docs.anthropic.com/en/docs/models-overview)

---

### TUI Rendering Issues

#### "Terminal too small"

The terminal window is smaller than the minimum required size (80x24).

**Solution:**

- Resize your terminal window
- Use a terminal emulator with adjustable size
- Try fullscreen mode

#### Garbled or broken characters

Unicode characters aren't displaying correctly.

**Solutions:**

1. **Check terminal Unicode support:**

   ```bash
   echo "Unicode test: ✅ ❌ 🟢 🔴"
   # Should display colored icons
   ```

2. **Use a Unicode-compatible terminal:**
   - macOS: iTerm2, Terminal.app
   - Windows: Windows Terminal, not cmd.exe
   - Linux: GNOME Terminal, Konsole, Kitty

3. **Set proper locale:**

   ```bash
   export LANG=en_US.UTF-8
   export LC_ALL=en_US.UTF-8
   ```

#### Missing or wrong colors

Colors aren't displaying correctly.

**Solutions:**

1. **Check terminal color support:**

   ```bash
   echo $TERM
   # Should be xterm-256color or similar
   ```

2. **Enable 256-color mode:**

   ```bash
   export TERM=xterm-256color
   ```

3. **Use a terminal with true color support:**
   - iTerm2, Windows Terminal, Kitty, Alacritty

#### Can't type in input panel

The input panel doesn't accept keyboard input.

**Solutions:**

1. **Ensure input panel is focused:**
   - Press `Tab` to cycle through panels
   - The input panel should have a visible cursor

2. **Close any overlays:**
   - Press `Esc` to close help overlay
   - Press `Esc` to close error details

3. **Check if conversation is active:**
   - Input is disabled while waiting for all agents to respond

#### Message not sending

Pressing Enter doesn't send the message.

**Solution:**

- Use `Ctrl+Enter` to send, not just `Enter`
- `Enter` creates a new line for multi-line input

---

### Performance Issues

#### Slow agent responses

Responses take too long to appear.

**Solutions:**

1. **Use API adapters instead of CLI:**

   ```yaml
   # Faster (API)
   type: claude-api
   type: openrouter

   # Slower (CLI)
   type: claude
   type: gemini
   ```

2. **Use faster models:**
   - Claude Haiku is 3-5x faster than Opus
   - GPT-3.5 Turbo is faster than GPT-4

3. **Reduce response length:**

   ```yaml
   config:
     max_tokens: 500
   ```

4. **Check network latency:**

   ```bash
   time curl -I https://api.anthropic.com
   ```

#### High CPU usage

AgentPipe using excessive CPU.

**Normal behavior:**

- High CPU during streaming (processing incoming chunks)
- Should idle between messages

**If constantly high:**

1. **Check for runaway processes:**

   ```bash
   ps aux | grep agentpipe
   ```

2. **Update to latest version:**

   ```bash
   go install github.com/ASRagab/agentpipe@latest
   ```

3. **Try headless mode to eliminate TUI overhead:**

   ```bash
   agentpipe run --v2 --no-tui -c config.yaml
   ```

#### Memory issues

AgentPipe using too much memory.

**Solutions:**

1. **Limit conversation history:**

   ```yaml
   conversation:
     max_turns: 50  # Limit conversation length
   ```

2. **Start fresh conversations** instead of continuing long ones

3. **Use streaming mode** (default) which processes chunks incrementally

---

### Debug Mode and Logging

#### Enable debug logging

Get detailed logs for troubleshooting:

```yaml
logging:
  level: debug          # Options: debug, info, warn, error
  format: text          # Or json for structured logs
  file: /tmp/agentpipe.log  # Optional: write to file
```

Or via CLI:

```bash
# Set environment variable
export AGENTPIPE_LOG_LEVEL=debug

# Run with verbose output
agentpipe run --v2 -c config.yaml
```

#### Reading log output

**Common log patterns:**

```
# Successful request
INFO  Sending message to claude (claude-sonnet-4-20250514)
INFO  Received response from claude (1234 tokens, 2.3s)

# Rate limit hit (will retry)
WARN  Rate limit hit for claude, retrying in 2s

# Error
ERROR Failed to send message to claude: connection refused
```

#### Capture logs for bug reports

```bash
# Run with full debug output to file
agentpipe run --v2 -c config.yaml 2>&1 | tee agentpipe-debug.log
```

---

### CLI-Specific Issues

#### Claude CLI not found

```
CLI binary not found: claude
```

**Solutions:**

1. **Install Claude CLI:**

   ```bash
   # macOS
   brew install anthropic/tap/claude

   # Or via npm
   npm install -g @anthropic-ai/cli
   ```

2. **Verify installation:**

   ```bash
   which claude
   claude --version
   ```

3. **Authenticate:**

   ```bash
   claude login
   ```

#### Health check failed for CLI agent

CLI adapters may fail health checks due to authentication issues.

**Solutions:**

1. **Run the CLI directly to test:**

   ```bash
   claude "Hello, world"
   ```

2. **Re-authenticate:**

   ```bash
   claude logout
   claude login
   ```

3. **Increase health check timeout:**

   ```bash
   agentpipe doctor --v2 -c config.yaml
   # Health check uses 10s timeout
   ```

---

### Persistence Issues

#### Can't save conversation

**Check directory permissions:**

```bash
# Default save directory
ls -la ~/.agentpipe/v2/chats/

# Create if missing
mkdir -p ~/.agentpipe/v2/chats
```

**Check config:**

```yaml
persistence:
  save_dir: ~/.agentpipe/v2/chats
  auto_save: true
```

#### Can't load saved conversation

**Check file exists:**

```bash
ls ~/.agentpipe/v2/chats/
```

**Resume conversation:**

```bash
agentpipe run --v2 --resume ~/.agentpipe/v2/chats/conversation-2024-01-15.json
```

---

### Migration Issues

#### v1 config not being migrated

**Manual migration:**

```bash
agentpipe run --v2 --migrate-config -c old-config.yaml
```

**Key differences between v1 and v2:**

| v1 Field | v2 Field |
|----------|----------|
| `orchestrator.mode` | `conversation.mode` |
| `orchestrator.max_turns` | `conversation.max_turns` |
| `orchestrator.turn_timeout` | `conversation.timeout` |

See [[migration]] for complete guide.

---

## Getting More Help

### Before reporting issues

1. **Run doctor:** `agentpipe doctor --v2 -c config.yaml`
2. **Enable debug logging** and capture output
3. **Check version:** `agentpipe --version`
4. **Try minimal config** to isolate the issue

### Reporting bugs

Include in your bug report:

1. AgentPipe version (`agentpipe --version`)
2. Operating system and version
3. Go version if building from source
4. Doctor output (`agentpipe doctor --v2 --json`)
5. Minimal config that reproduces the issue (remove API keys!)
6. Debug logs (remove sensitive information)

### Resources

- **GitHub Issues**: [github.com/ASRagab/agentpipe/issues](https://github.com/ASRagab/agentpipe/issues)
- **Discussions**: [github.com/ASRagab/agentpipe/discussions](https://github.com/ASRagab/agentpipe/discussions)
- **Documentation**: See [[quickstart]], [[configuration]], [[adapters]], [[tui]]
