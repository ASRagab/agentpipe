// Package adapters provides the interface and implementations for AI agent adapters.
//
// Each adapter connects AgentPipe to a different AI provider (OpenRouter, Claude API,
// CLI tools, etc.) and handles the specifics of API communication, authentication,
// message formatting, and response streaming.
//
// # Architecture
//
// The adapter system uses a registry pattern:
//
//  1. AdapterFactory functions create new adapter instances
//  2. Factories are registered in the global DefaultRegistry
//  3. The ConversationManager retrieves adapters by name
//
// This allows dynamic adapter selection based on configuration.
//
// # The AgentAdapter Interface
//
// All adapters must implement the AgentAdapter interface:
//
//	type AgentAdapter interface {
//		// Initialize configures the adapter with agent settings
//		Initialize(agent core.Agent) error
//
//		// SendMessage sends messages and returns the response (non-streaming)
//		SendMessage(ctx context.Context, messages []core.Message) (string, *core.Metrics, error)
//
//		// StreamMessage sends messages and streams response to writer
//		StreamMessage(ctx context.Context, messages []core.Message, writer io.Writer) (*core.Metrics, error)
//
//		// IsAvailable checks if the adapter is properly configured
//		IsAvailable() bool
//
//		// GetModel returns the configured model name
//		GetModel() string
//
//		// HealthCheck verifies API accessibility
//		HealthCheck(ctx context.Context) error
//	}
//
// # Registering Adapters
//
// Adapters are registered using factory functions:
//
//	func init() {
//		adapters.Register("myapi", func() adapters.AgentAdapter {
//			return &MyAPIAdapter{}
//		})
//	}
//
// This registration typically happens in an init() function so adapters
// are available when the program starts.
//
// # Using the Registry
//
// Retrieve adapters using the global functions:
//
//	// Check if adapter exists
//	if adapters.Has("openrouter") {
//		adapter, err := adapters.Get("openrouter")
//		if err != nil {
//			log.Fatal(err)
//		}
//
//		err = adapter.Initialize(agent)
//		if err != nil {
//			log.Fatal(err)
//		}
//	}
//
//	// List all registered adapters
//	names := adapters.List()
//	for _, name := range names {
//		fmt.Println("Available adapter:", name)
//	}
//
// # Built-in Adapters
//
// The following adapters are provided:
//
// API-based adapters (in adapters/api/):
//   - openrouter: OpenRouter API with 400+ models
//   - claude-api: Direct Anthropic Claude API
//
// CLI-based adapters (in adapters/cli/):
//   - claude: Claude Code CLI
//   - gemini: Google Gemini CLI
//   - generic: Configurable adapter for other CLI tools
//
// Mock adapter (in adapters/mock/):
//   - mock: Testing adapter with configurable responses
//
// # Implementing a Custom Adapter
//
// To implement a custom adapter:
//
//	type MyAdapter struct {
//		agent  core.Agent
//		client *http.Client
//	}
//
//	func (a *MyAdapter) Initialize(agent core.Agent) error {
//		a.agent = agent
//		a.client = &http.Client{Timeout: 30 * time.Second}
//		// Validate API key is available
//		if os.Getenv(agent.Config.APIKeyEnvVar) == "" {
//			return fmt.Errorf("API key not found in %s", agent.Config.APIKeyEnvVar)
//		}
//		return nil
//	}
//
//	func (a *MyAdapter) SendMessage(ctx context.Context, messages []core.Message) (string, *core.Metrics, error) {
//		start := time.Now()
//
//		// Convert messages to API format
//		apiMessages := convertMessages(messages)
//
//		// Make API call
//		resp, err := a.callAPI(ctx, apiMessages)
//		if err != nil {
//			return "", nil, err
//		}
//
//		// Return response with metrics
//		metrics := &core.Metrics{
//			Duration:     time.Since(start),
//			InputTokens:  resp.Usage.PromptTokens,
//			OutputTokens: resp.Usage.CompletionTokens,
//			TotalTokens:  resp.Usage.TotalTokens,
//			Model:        a.agent.Model,
//			Cost:         calculateCost(resp.Usage, a.agent.Model),
//		}
//
//		return resp.Content, metrics, nil
//	}
//
//	func (a *MyAdapter) StreamMessage(ctx context.Context, messages []core.Message, writer io.Writer) (*core.Metrics, error) {
//		// Implement streaming logic with Server-Sent Events
//		// Write chunks to writer as they arrive
//		return nil, nil
//	}
//
//	func (a *MyAdapter) IsAvailable() bool {
//		return os.Getenv(a.agent.Config.APIKeyEnvVar) != ""
//	}
//
//	func (a *MyAdapter) GetModel() string {
//		return a.agent.Model
//	}
//
//	func (a *MyAdapter) HealthCheck(ctx context.Context) error {
//		// Make minimal API call to verify connectivity
//		return nil
//	}
//
// # Error Handling
//
// Adapters should return structured errors that can be classified:
//
//	// For timeout errors
//	if errors.Is(err, context.DeadlineExceeded) {
//		return "", nil, fmt.Errorf("request timed out: %w", err)
//	}
//
//	// For rate limit errors (HTTP 429)
//	if resp.StatusCode == 429 {
//		return "", nil, fmt.Errorf("rate limit exceeded: too many requests")
//	}
//
//	// For auth errors (HTTP 401/403)
//	if resp.StatusCode == 401 || resp.StatusCode == 403 {
//		return "", nil, fmt.Errorf("authentication failed: check API key")
//	}
//
// The core.ClassifyError function can then categorize these for user display.
//
// # Retry Logic
//
// The retry package provides automatic retry with exponential backoff:
//
//	config := retry.DefaultConfig()
//	config.MaxRetries = 3
//
//	result, metrics, err := retry.WithRetry(ctx, config, func() (string, *core.Metrics, error) {
//		return adapter.SendMessage(ctx, messages)
//	})
//
// # Circuit Breaker
//
// The circuit_breaker package prevents cascading failures:
//
//	cb := adapters.NewCircuitBreaker(adapters.DefaultCircuitBreakerConfig())
//
//	if cb.Allow() {
//		result, err := adapter.SendMessage(ctx, messages)
//		cb.Record(err)
//	} else {
//		return "", nil, fmt.Errorf("circuit breaker open")
//	}
//
// See the pool package for integrated circuit breaker management.
package adapters
