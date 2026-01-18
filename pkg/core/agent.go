package core

// Agent represents an AI agent participating in a conversation.
// This is the domain model for agents, separate from the adapter implementation.
type Agent struct {
	// ID is the unique identifier for this agent instance.
	ID string `json:"id" yaml:"id"`
	// Type is the agent type (e.g., "openrouter", "claude-api").
	Type string `json:"type" yaml:"type"`
	// Name is the human-readable display name.
	Name string `json:"name" yaml:"name"`
	// Model is the specific AI model to use.
	Model string `json:"model" yaml:"model"`
	// AdapterName is the name of the adapter to use for this agent.
	AdapterName string `json:"adapter_name" yaml:"adapter"`
	// Config contains adapter-specific configuration.
	Config AgentAdapterConfig `json:"config" yaml:"config"`
}

// AgentAdapterConfig contains configuration for an agent's adapter.
type AgentAdapterConfig struct {
	// SystemPrompt is the initial system prompt for the agent.
	SystemPrompt string `json:"system_prompt,omitempty" yaml:"system_prompt"`
	// Temperature controls randomness in responses (0.0 to 2.0).
	Temperature float64 `json:"temperature,omitempty" yaml:"temperature"`
	// MaxTokens limits the response length.
	MaxTokens int `json:"max_tokens,omitempty" yaml:"max_tokens"`
	// APIKeyEnvVar is the environment variable name containing the API key.
	APIKeyEnvVar string `json:"api_key_env,omitempty" yaml:"api_key_env"`
	// Extra contains any additional adapter-specific settings.
	Extra map[string]interface{} `json:"extra,omitempty" yaml:"extra"`
}

// NewAgent creates a new Agent with the given parameters.
func NewAgent(id, agentType, name, model, adapterName string) Agent {
	return Agent{
		ID:          id,
		Type:        agentType,
		Name:        name,
		Model:       model,
		AdapterName: adapterName,
	}
}

// WithConfig sets the adapter configuration and returns the agent.
func (a Agent) WithConfig(config AgentAdapterConfig) Agent {
	a.Config = config
	return a
}

// WithSystemPrompt sets the system prompt and returns the agent.
func (a Agent) WithSystemPrompt(prompt string) Agent {
	a.Config.SystemPrompt = prompt
	return a
}

// WithTemperature sets the temperature and returns the agent.
func (a Agent) WithTemperature(temp float64) Agent {
	a.Config.Temperature = temp
	return a
}

// WithMaxTokens sets the max tokens and returns the agent.
func (a Agent) WithMaxTokens(tokens int) Agent {
	a.Config.MaxTokens = tokens
	return a
}

// AgentStatus represents the current state of an agent.
type AgentStatus string

const (
	// AgentStatusIdle indicates the agent is ready to receive messages.
	AgentStatusIdle AgentStatus = "idle"
	// AgentStatusTyping indicates the agent is generating a response.
	AgentStatusTyping AgentStatus = "typing"
	// AgentStatusError indicates the agent encountered an error.
	AgentStatusError AgentStatus = "error"
	// AgentStatusOffline indicates the agent is not available.
	AgentStatusOffline AgentStatus = "offline"
	// AgentStatusCancelled indicates the agent's request was cancelled.
	AgentStatusCancelled AgentStatus = "cancelled"
)

// AgentState tracks the runtime state of an agent.
type AgentState struct {
	// Agent is the agent configuration.
	Agent Agent `json:"agent"`
	// Status is the current operational status.
	Status AgentStatus `json:"status"`
	// LastError contains the most recent error message (if any).
	LastError string `json:"last_error,omitempty"`
	// MessageCount is the number of messages this agent has sent.
	MessageCount int `json:"message_count"`
	// TotalTokens is the cumulative token usage.
	TotalTokens int `json:"total_tokens"`
	// TotalCost is the cumulative cost in USD.
	TotalCost float64 `json:"total_cost"`
}

// NewAgentState creates a new AgentState for the given agent.
func NewAgentState(agent Agent) AgentState {
	return AgentState{
		Agent:  agent,
		Status: AgentStatusIdle,
	}
}

// SetTyping marks the agent as currently generating a response.
func (s *AgentState) SetTyping() {
	s.Status = AgentStatusTyping
	s.LastError = ""
}

// SetIdle marks the agent as ready for new messages.
func (s *AgentState) SetIdle() {
	s.Status = AgentStatusIdle
}

// SetError marks the agent as having encountered an error.
func (s *AgentState) SetError(err string) {
	s.Status = AgentStatusError
	s.LastError = err
}

// SetCancelled marks the agent as having its request cancelled.
func (s *AgentState) SetCancelled() {
	s.Status = AgentStatusCancelled
	s.LastError = "request cancelled"
}

// RecordMessage updates the agent state after sending a message.
func (s *AgentState) RecordMessage(tokens int, cost float64) {
	s.MessageCount++
	s.TotalTokens += tokens
	s.TotalCost += cost
	s.Status = AgentStatusIdle
}
