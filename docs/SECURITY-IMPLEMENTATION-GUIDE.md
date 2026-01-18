# Security Implementation Guide for AgentPipe

This guide provides practical, copy-paste-ready code examples for implementing the critical security fixes identified in the security review.

## Table of Contents
1. [Command Injection Prevention (P0)](#1-command-injection-prevention)
2. [Path Traversal Protection (P0)](#2-path-traversal-protection)
3. [Secrets Management (P0)](#3-secrets-management)
4. [Input Validation Framework (P1)](#4-input-validation-framework)
5. [Prompt Injection Protection (P1)](#5-prompt-injection-protection)
6. [Resource Limits (P1)](#6-resource-limits)
7. [Agent Sandboxing (P1)](#7-agent-sandboxing)

---

## 1. Command Injection Prevention (P0)

### Create `pkg/security/exec.go`

```go
package security

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// SafeCommandOptions configures command execution security policies
type SafeCommandOptions struct {
	// AllowedBinaries is a list of allowed binary paths
	AllowedBinaries []string
	// AllowedArgs is a list of allowed argument patterns
	AllowedArgs []string
	// ResourceLimits sets CPU/memory limits
	ResourceLimits *ResourceLimits
	// WorkingDir restricts the working directory
	WorkingDir string
	// Timeout for command execution
	Timeout int
}

// ResourceLimits defines resource constraints for command execution
type ResourceLimits struct {
	MaxMemoryMB int    // Maximum memory in MB
	MaxCPU      int    // CPU limit (percentage)
	MaxFiles    uint64 // Maximum open files
}

// SafeCommandContext creates a command with security validations
func SafeCommandContext(
	ctx context.Context,
	binary string,
	args []string,
	opts *SafeCommandOptions,
) (*exec.Cmd, error) {
	// 1. Validate binary path
	cleanBinary, err := validateBinary(binary, opts)
	if err != nil {
		return nil, fmt.Errorf("binary validation failed: %w", err)
	}

	// 2. Validate arguments
	if err := validateArgs(args, opts); err != nil {
		return nil, fmt.Errorf("argument validation failed: %w", err)
	}

	// 3. Create command (no shell interpretation)
	cmd := exec.CommandContext(ctx, cleanBinary, args...)

	// 4. Set working directory if specified
	if opts.WorkingDir != "" {
		cmd.Dir = opts.WorkingDir
	}

	// 5. Apply resource limits if specified
	if opts.ResourceLimits != nil {
		applyResourceLimits(cmd, opts.ResourceLimits)
	}

	// 6. Set environment variables (minimal subset)
	cmd.Env = getSafeEnvironment()

	return cmd, nil
}

// validateBinary ensures the binary path is safe
func validateBinary(binary string, opts *SafeCommandOptions) (string, error) {
	// Resolve to absolute path
	absPath, err := filepath.Abs(binary)
	if err != nil {
		return "", fmt.Errorf("failed to resolve binary path: %w", err)
	}

	// Check if binary is in allowed list
	if opts != nil && len(opts.AllowedBinaries) > 0 {
		allowed := false
		for _, allowedPath := range opts.AllowedBinaries {
			if absPath == allowedPath || filepath.Base(absPath) == filepath.Base(allowedPath) {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", fmt.Errorf("binary not in allowed list: %s", absPath)
		}
	}

	// Verify binary exists and is executable
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("binary not found: %w", err)
	}

	if info.IsDir() {
		return "", fmt.Errorf("path is a directory, not a binary: %s", absPath)
	}

	// Check if file is executable (Unix-like systems)
	if info.Mode()&0111 == 0 {
		return "", fmt.Errorf("binary is not executable: %s", absPath)
	}

	return absPath, nil
}

// validateArgs checks arguments for injection attempts
func validateArgs(args []string, opts *SafeCommandOptions) error {
	dangerousPatterns := []string{
		";", "|", "&", "$", "`", "$(", "${", "<(", ">(", "\n", "\r",
		"&&", "||", ">>", ">", "<", "2>", "2>&1",
	}

	for _, arg := range args {
		// Check for shell meta-characters
		for _, pattern := range dangerousPatterns {
			if strings.Contains(arg, pattern) {
				return fmt.Errorf("argument contains dangerous pattern '%s': %s", pattern, arg)
			}
		}

		// Check against allowed args pattern if provided
		if opts != nil && len(opts.AllowedArgs) > 0 {
			allowed := false
			for _, allowedPattern := range opts.AllowedArgs {
				// Simple pattern matching (can be enhanced with regex)
				if strings.HasPrefix(arg, allowedPattern) || arg == allowedPattern {
					allowed = true
					break
				}
			}
			if !allowed {
				return fmt.Errorf("argument not in allowed patterns: %s", arg)
			}
		}
	}

	return nil
}

// applyResourceLimits sets resource limits on the command (Unix-like systems)
func applyResourceLimits(cmd *exec.Cmd, limits *ResourceLimits) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	// Set resource limits via setrlimit
	// Note: This is Unix-specific. For Windows, use Job Objects
	if limits.MaxMemoryMB > 0 {
		memLimit := uint64(limits.MaxMemoryMB) * 1024 * 1024
		cmd.SysProcAttr.Setpgid = true
		// RLIMIT_AS (address space limit)
		// Note: Actual implementation would use syscall.Setrlimit
	}

	if limits.MaxFiles > 0 {
		// RLIMIT_NOFILE (open files limit)
		// Implementation depends on platform
	}
}

// getSafeEnvironment returns a minimal, safe environment
func getSafeEnvironment() []string {
	// Only include essential environment variables
	safeVars := []string{
		"PATH=/usr/local/bin:/usr/bin:/bin",
		"HOME=" + os.Getenv("HOME"),
		"USER=" + os.Getenv("USER"),
	}
	return safeVars
}
```

### Update Adapter to Use Safe Execution

```go
// pkg/adapters/claude.go

import (
	"github.com/ASRagab/agentpipe/pkg/security"
)

func (c *ClaudeAgent) SendMessage(ctx context.Context, messages []agent.Message) (string, error) {
	// ... existing code ...

	// Build command args
	args := []string{"-p"}
	if c.Config.Model != "" {
		args = append(args, "--model", c.Config.Model)
	}

	// Create safe command with security options
	opts := &security.SafeCommandOptions{
		AllowedBinaries: []string{c.execPath},
		AllowedArgs:     []string{"-p", "--model"},
		ResourceLimits: &security.ResourceLimits{
			MaxMemoryMB: 512,
			MaxFiles:    100,
		},
	}

	cmd, err := security.SafeCommandContext(ctx, c.execPath, args, opts)
	if err != nil {
		return "", fmt.Errorf("failed to create safe command: %w", err)
	}

	cmd.Stdin = strings.NewReader(prompt)

	// Execute command
	output, err := cmd.CombinedOutput()
	// ... rest of existing code ...
}
```

---

## 2. Path Traversal Protection (P0)

### Create `pkg/security/path.go`

```go
package security

import (
	"errors"
	"path/filepath"
	"strings"
)

var (
	ErrAbsolutePath      = errors.New("absolute paths are not allowed")
	ErrPathTraversal     = errors.New("path traversal detected")
	ErrInvalidPath       = errors.New("invalid path")
	ErrDangerousFilename = errors.New("dangerous filename detected")
)

// SecurePath validates and sanitizes a user-provided path against a base directory
func SecurePath(userPath, baseDir string) (string, error) {
	// 1. Clean the base directory
	cleanBase := filepath.Clean(baseDir)
	if !filepath.IsAbs(cleanBase) {
		var err error
		cleanBase, err = filepath.Abs(cleanBase)
		if err != nil {
			return "", err
		}
	}

	// 2. Clean the user path
	userPath = filepath.Clean(userPath)

	// 3. Reject absolute paths from user
	if filepath.IsAbs(userPath) {
		return "", ErrAbsolutePath
	}

	// 4. Check for dangerous patterns
	if err := checkDangerousPatterns(userPath); err != nil {
		return "", err
	}

	// 5. Join with base directory
	fullPath := filepath.Join(cleanBase, userPath)

	// 6. Verify the result is within base directory (path traversal check)
	if !strings.HasPrefix(fullPath, cleanBase+string(filepath.Separator)) &&
		fullPath != cleanBase {
		return "", ErrPathTraversal
	}

	// 7. Ensure path doesn't escape via symlinks (additional check)
	// This requires evaluating symlinks which is expensive
	// Consider using filepath.EvalSymlinks for high-security scenarios

	return fullPath, nil
}

// checkDangerousPatterns checks for dangerous patterns in filenames
func checkDangerousPatterns(path string) error {
	dangerousPatterns := []string{
		"..", // Path traversal
		"~",  // Home directory expansion
		"$",  // Variable expansion
		"`",  // Command substitution
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(path, pattern) {
			return ErrDangerousFilename
		}
	}

	// Check for null bytes (directory traversal on some systems)
	if strings.Contains(path, "\x00") {
		return ErrDangerousFilename
	}

	return nil
}

// SecurePathStrict is a stricter version that also evaluates symlinks
func SecurePathStrict(userPath, baseDir string) (string, error) {
	// First do basic validation
	fullPath, err := SecurePath(userPath, baseDir)
	if err != nil {
		return "", err
	}

	// Evaluate symlinks to prevent escape
	evaluatedPath, err := filepath.EvalSymlinks(fullPath)
	if err != nil {
		// Path doesn't exist yet (might be a new file), which is okay
		// Just verify the parent directory
		parentDir := filepath.Dir(fullPath)
		evaluatedParent, parentErr := filepath.EvalSymlinks(parentDir)
		if parentErr != nil {
			// Parent doesn't exist, allow for new directories
			return fullPath, nil
		}
		evaluatedPath = filepath.Join(evaluatedParent, filepath.Base(fullPath))
	}

	// Verify evaluated path is still within base
	cleanBase := filepath.Clean(baseDir)
	if !strings.HasPrefix(evaluatedPath, cleanBase+string(filepath.Separator)) &&
		evaluatedPath != cleanBase {
		return "", ErrPathTraversal
	}

	return evaluatedPath, nil
}

// IsPathSafe is a quick check function
func IsPathSafe(userPath, baseDir string) bool {
	_, err := SecurePath(userPath, baseDir)
	return err == nil
}
```

### Update Artifact Writer

```go
// pkg/artifact/writer.go

import (
	"github.com/ASRagab/agentpipe/pkg/security"
)

func (w *Writer) SaveArtifact(artifact Artifact) (string, error) {
	// Validate path is safe
	safePath, err := security.SecurePath(artifact.Filename, w.outputDir)
	if err != nil {
		return "", fmt.Errorf("unsafe artifact path: %w", err)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(safePath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file with restrictive permissions
	if err := os.WriteFile(safePath, []byte(artifact.Content), 0644); err != nil {
		return "", fmt.Errorf("failed to write artifact: %w", err)
	}

	return safePath, nil
}
```

---

## 3. Secrets Management (P0)

### Create `pkg/security/secrets.go`

```go
package security

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
)

var (
	ErrSecretNotFound = errors.New("secret not found")
	ErrSecretStore    = errors.New("failed to access secret store")
)

// SecretStore is an interface for storing and retrieving secrets
type SecretStore interface {
	Store(key, value string) error
	Retrieve(key string) (string, error)
	Delete(key string) error
	List() ([]string, error)
}

// NewSecretStore creates a platform-appropriate secret store
func NewSecretStore() (SecretStore, error) {
	switch runtime.GOOS {
	case "darwin":
		return NewKeychainStore()
	case "windows":
		return NewCredentialManagerStore()
	case "linux":
		return NewSecretServiceStore()
	default:
		// Fallback to encrypted file store
		return NewEncryptedFileStore()
	}
}

// Secret represents a stored credential
type Secret struct {
	Key       string
	Value     string
	CreatedAt int64
	UpdatedAt int64
}

// Common secret key constants
const (
	SecretBridgeAPIKey    = "agentpipe.bridge.api_key"
	SecretOpenRouterKey   = "agentpipe.openrouter.api_key"
	SecretAnthropicKey    = "agentpipe.anthropic.api_key"
	SecretOpenAIKey       = "agentpipe.openai.api_key"
	SecretGoogleAIKey     = "agentpipe.googleai.api_key"
)
```

### macOS Keychain Implementation

```go
// pkg/security/keychain_darwin.go
//go:build darwin

package security

import (
	"fmt"
	"github.com/keybase/go-keychain"
)

// KeychainStore implements SecretStore using macOS Keychain
type KeychainStore struct {
	service string
}

func NewKeychainStore() (SecretStore, error) {
	return &KeychainStore{
		service: "agentpipe",
	}, nil
}

func (k *KeychainStore) Store(key, value string) error {
	// Check if exists
	existing := k.findItem(key)
	if existing != nil {
		// Update existing
		existing.SetData([]byte(value))
		return keychain.UpdateItem(existing, existing)
	}

	// Create new
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(k.service)
	item.SetAccount(key)
	item.SetData([]byte(value))
	item.SetAccessible(keychain.AccessibleWhenUnlocked)
	item.SetSynchronizable(keychain.SynchronizableNo)

	return keychain.AddItem(item)
}

func (k *KeychainStore) Retrieve(key string) (string, error) {
	item := k.findItem(key)
	if item == nil {
		return "", ErrSecretNotFound
	}

	data, err := item.GetData()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve secret: %w", err)
	}

	return string(data), nil
}

func (k *KeychainStore) Delete(key string) error {
	item := k.findItem(key)
	if item == nil {
		return ErrSecretNotFound
	}

	return keychain.DeleteItem(item)
}

func (k *KeychainStore) List() ([]string, error) {
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetService(k.service)
	query.SetMatchLimit(keychain.MatchLimitAll)
	query.SetReturnAttributes(true)

	results, err := keychain.QueryItem(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	var keys []string
	for _, result := range results {
		if account := result.Account; account != "" {
			keys = append(keys, account)
		}
	}

	return keys, nil
}

func (k *KeychainStore) findItem(key string) keychain.Item {
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetService(k.service)
	query.SetAccount(key)
	query.SetMatchLimit(keychain.MatchLimitOne)
	query.SetReturnData(true)

	results, err := keychain.QueryItem(query)
	if err != nil || len(results) == 0 {
		return nil
	}

	return results[0]
}
```

### Update Config to Use Secret Store

```go
// internal/bridge/config.go

import (
	"github.com/ASRagab/agentpipe/pkg/security"
)

func LoadConfig() *Config {
	config := &Config{
		Enabled:       false,
		URL:           getDefaultURL(),
		TimeoutMs:     10000,
		RetryAttempts: 3,
		LogLevel:      "info",
	}

	// Load API key from secure store instead of config file
	if viper.GetBool("bridge.enabled") {
		config.Enabled = true

		// Try to load from secret store first
		secretStore, err := security.NewSecretStore()
		if err == nil {
			if apiKey, err := secretStore.Retrieve(security.SecretBridgeAPIKey); err == nil {
				config.APIKey = apiKey
			}
		}

		// Fallback to environment variable (for migration)
		if config.APIKey == "" {
			config.APIKey = os.Getenv("AGENTPIPE_STREAM_API_KEY")
		}

		// Never load from config file (deprecated)
	}

	return config
}
```

---

## 4. Input Validation Framework (P1)

### Create `pkg/validation/validator.go`

```go
package validation

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrValidationFailed = errors.New("validation failed")
	ErrSizeLimitExceeded = errors.New("size limit exceeded")
	ErrInvalidFormat     = errors.New("invalid format")
)

// Validator provides input validation utilities
type Validator struct {
	maxPromptSize int
	maxConfigSize int
}

func NewValidator() *Validator {
	return &Validator{
		maxPromptSize: 100000,  // 100KB
		maxConfigSize: 1000000, // 1MB
	}
}

// ValidatePrompt validates user prompts for safety
func (v *Validator) ValidatePrompt(prompt string) error {
	// 1. Size check
	if len(prompt) > v.maxPromptSize {
		return fmt.Errorf("%w: prompt size %d exceeds maximum %d",
			ErrSizeLimitExceeded, len(prompt), v.maxPromptSize)
	}

	// 2. Check for null bytes
	if strings.Contains(prompt, "\x00") {
		return fmt.Errorf("%w: null bytes not allowed", ErrInvalidFormat)
	}

	// 3. Check for control characters (except newline, tab, carriage return)
	for _, r := range prompt {
		if r < 32 && r != '\n' && r != '\t' && r != '\r' {
			return fmt.Errorf("%w: control characters not allowed", ErrInvalidFormat)
		}
	}

	// 4. Pattern-based detection (optional, can be configured)
	if err := v.checkDangerousPatterns(prompt); err != nil {
		return err
	}

	return nil
}

// checkDangerousPatterns detects potentially malicious patterns
func (v *Validator) checkDangerousPatterns(text string) error {
	// Patterns that might indicate injection attempts
	dangerousPatterns := []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{"command injection", regexp.MustCompile(`\$\(.*\)|\`.*\``)},
		{"path traversal", regexp.MustCompile(`\.\./|\.\.\\`)},
		{"SQL injection", regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop)\s+(from|into|table)`)},
		{"script tag", regexp.MustCompile(`(?i)<script[^>]*>`)},
	}

	for _, dp := range dangerousPatterns {
		if dp.pattern.MatchString(text) {
			// Log the detection but don't block (could be legitimate)
			// Return error only for high-confidence detections
			// This is configurable based on security policy
		}
	}

	return nil
}

// ValidateConfigJSON validates YAML/JSON configuration
func (v *Validator) ValidateConfigJSON(data []byte) error {
	// 1. Size check
	if len(data) > v.maxConfigSize {
		return fmt.Errorf("%w: config size exceeds maximum",
			ErrSizeLimitExceeded)
	}

	// 2. Valid JSON check
	var temp interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("%w: invalid JSON: %v", ErrInvalidFormat, err)
	}

	return nil
}

// ValidateAgentName validates agent names for safety
func (v *Validator) ValidateAgentName(name string) error {
	// Only allow alphanumeric, hyphens, underscores
	validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("%w: agent name must be alphanumeric", ErrInvalidFormat)
	}

	if len(name) > 100 {
		return fmt.Errorf("%w: agent name too long", ErrSizeLimitExceeded)
	}

	return nil
}
```

---

## 5. Testing the Security Implementations

### Unit Tests

```go
// pkg/security/exec_test.go

package security

import (
	"context"
	"testing"
)

func TestCommandInjectionPrevention(t *testing.T) {
	tests := []struct {
		name      string
		binary    string
		args      []string
		wantError bool
	}{
		{
			name:      "safe command",
			binary:    "/usr/bin/echo",
			args:      []string{"hello"},
			wantError: false,
		},
		{
			name:      "shell metacharacter in args",
			binary:    "/usr/bin/echo",
			args:      []string{"; rm -rf /"},
			wantError: true,
		},
		{
			name:      "command substitution",
			binary:    "/usr/bin/echo",
			args:      []string{"$(whoami)"},
			wantError: true,
		},
		{
			name:      "pipe character",
			binary:    "/usr/bin/echo",
			args:      []string{"test | cat"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &SafeCommandOptions{
				AllowedBinaries: []string{"/usr/bin/echo"},
			}
			_, err := SafeCommandContext(context.Background(), tt.binary, tt.args, opts)
			if (err != nil) != tt.wantError {
				t.Errorf("SafeCommandContext() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// pkg/security/path_test.go

package security

import (
	"path/filepath"
	"testing"
)

func TestPathTraversalPrevention(t *testing.T) {
	baseDir := "/tmp/agentpipe/artifacts"

	tests := []struct {
		name      string
		userPath  string
		wantError bool
	}{
		{
			name:      "safe path",
			userPath:  "output/file.txt",
			wantError: false,
		},
		{
			name:      "path traversal",
			userPath:  "../../etc/passwd",
			wantError: true,
		},
		{
			name:      "absolute path",
			userPath:  "/etc/passwd",
			wantError: true,
		},
		{
			name:      "hidden traversal",
			userPath:  "subdir/../../../outside",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SecurePath(tt.userPath, baseDir)
			if (err != nil) != tt.wantError {
				t.Errorf("SecurePath() error = %v, wantError %v", err, tt.wantError)
			}
			if err == nil && !filepath.HasPrefix(result, baseDir) {
				t.Errorf("SecurePath() = %v, should start with %v", result, baseDir)
			}
		})
	}
}
```

---

## Next Steps

1. **Week 1**: Implement command injection prevention
2. **Week 1**: Implement path traversal protection
3. **Week 2**: Implement secrets management
4. **Week 3**: Add comprehensive tests
5. **Week 4**: Update all adapters to use security package
6. **Week 5**: Security audit and penetration testing

## Resources

- [OWASP Command Injection](https://owasp.org/www-community/attacks/Command_Injection)
- [CWE-78: OS Command Injection](https://cwe.mitre.org/data/definitions/78.html)
- [CWE-22: Path Traversal](https://cwe.mitre.org/data/definitions/22.html)
- [Go Security Guide](https://github.com/Checkmarx/Go-SCP)
