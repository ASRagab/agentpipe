// Reference: architecture-design.md
// This sketch outlines the core components for orchestrating a multi-agent workflow in AgentPipe.

package orchestrator

import "time"

// --- Core Types (Ref: Agent Roles, Artifact Management) ---

// Agent represents a participant in the conversation, aligned with roles in the architecture.
type Agent struct {
	Name string // e.g., "Architect", "Coder", "Reviewer"
	Type string // e.g., "claude", "gemini-2-5-pro"
	Role string // The agent's specialized function
	// Connection details to the actual LLM client would go here.
}

// Artifact represents a file created by an agent.
type Artifact struct {
	Path    string // e.g., "docs/design/feature-architecture.md"
	Content []byte
	Owner   string // Name of the agent that created it
}

// ConversationState tracks the entire collaborative session.
type ConversationState struct {
	History   []Turn     // Record of each agent's contribution
	Artifacts []Artifact // All artifacts produced during the session
	StartTime time.Time
}

// Turn represents a single action taken by an agent.
type Turn struct {
	AgentName string
	Input     string // The prompt/context given to the agent
	Output    string // The raw response from the agent
	Timestamp time.Time
}

// --- Orchestration Engine (Ref: Orchestration Flow, Conversation Modes) ---

// Orchestrator manages the multi-agent workflow.
type Orchestrator struct {
	Agents         []Agent
	Mode           string // "sequential", "reactive", "free-form"
	MaxTurns       int
	State          ConversationState
	SuccessChecker func(ConversationState) bool // Logic for "Success Criteria"
}

// NewOrchestrator initializes the engine based on AgentPipe configuration.
func NewOrchestrator(config Config) (*Orchestrator, error) {
	// TODO: Load agents from config file (Ref: "AgentPipe Configuration")
	// TODO: Initialize state
	// TODO: Set up success checker based on "Success Criteria"
	return &Orchestrator{}, nil
}

// Run executes the main orchestration loop.
func (o *Orchestrator) Run() (finalState ConversationState, err error) {
	// Loop until max turns are reached or the feature is complete.
	for i := 0; i < o.MaxTurns; i++ {
		// 1. Determine which agent's turn it is (Ref: "Conversation Modes")
		currentAgent := o.getAgentByTurn(i)

		// 2. Prepare the prompt/context for the agent.
		prompt := o.buildPrompt()

		// 3. Execute the turn (call the LLM).
		rawOutput, err := o.executeTurn(currentAgent, prompt)
		if err != nil {
			// Handle LLM API errors
			return o.State, err
		}

		// 4. Parse artifacts from the agent's output.
		// TODO: Implement logic to find fenced code blocks and create Artifact structs.
		// (Ref: "Artifact Management")
		// The path should align with naming conventions, e.g., "agentpipe-artifacts/Coder/implementation-sketch.go"
		newArtifacts := parseArtifacts(rawOutput)

		// 5. Update conversation state.
		o.updateState(currentAgent, prompt, rawOutput, newArtifacts)

		// 6. Check for completion (Ref: "Success Criteria")
		if o.isFeatureComplete() {
			// e.g., Reviewer approved, all tests pass.
			break
		}
	}
	return o.State, nil
}

// --- Helper Functions (Implementation Details) ---

// getAgentByTurn implements the logic for different conversation modes.
func (o *Orchestrator) getAgentByTurn(turnIndex int) Agent {
	switch o.Mode {
	case "sequential":
		// Default mode: Architect -> Coder -> Reviewer -> ...
		// (Ref: "Sequential (Default)")
		return o.Agents[turnIndex%len(o.Agents)]
	case "reactive":
		// TODO: Implement logic to dynamically select the next agent.
		// This might involve analyzing the last response for keywords or calls to other agents.
		// (Ref: "Reactive")
		fallthrough // Fallback to sequential for now
	case "free-form":
		// TODO: Could allow user to select next agent or have a "discussion" phase.
		// (Ref: "Free-form")
		fallthrough // Fallback to sequential for now
	default:
		return o.Agents[turnIndex%len(o.Agents)]
	}
}

// executeTurn calls the appropriate LLM and handles the response.
func (o *Orchestrator) executeTurn(agent Agent, prompt string) (string, error) {
	// TODO: Interface with the actual LLM client based on agent.Type
	// and get the response.
	return "LLM raw output with 