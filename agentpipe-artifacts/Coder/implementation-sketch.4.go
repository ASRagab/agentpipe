// Package orchestrator implements the core logic for multi-agent collaboration based on architecture-design.md.
package orchestrator

import (
	"context"
	"errors"
)

// --- Core Types (Ref: "Agent Roles", "Artifact Management") ---

// Agent represents a participant in the conversation, mapping to a specific LLM client.
type Agent struct {
	Name string
	Role string // The prompt/persona for the agent (e.g., "You are a coder...")
	// LLMClient would be an interface for interacting with different models (Claude, Gemini, etc.)
}

// Artifact represents a file created by an agent.
type Artifact struct {
	Path    string // The destination path, e.g., "docs/design/feature-architecture.md"
	Content []byte
	Owner   string // Name of the agent that created it
}

// ConversationState holds the complete history and context of the collaboration.
type ConversationState struct {
	Phase      string      // e.g., "Requirements", "Design", "Implementation", "Review"
	Turn       int         // Current turn number
	History    []string    // Chronological record of messages
	Artifacts  []*Artifact // All artifacts produced so far
	Agents     []*Agent
	Mode       string // "sequential", "reactive", "free-form"
	MaxTurns   int
}

// Orchestrator manages the entire feature development lifecycle.
type Orchestrator struct {
	State *ConversationState
	// Config would hold system-level settings
}

// --- Orchestration Logic (Ref: "Orchestration Flow", "Conversation Modes") ---

// NewOrchestrator initializes the system from a configuration file.
func NewOrchestrator(configPath string) (*Orchestrator, error) {
	// TODO: Load config YAML (agents, mode, max-turns) from configPath.
	// TODO: Initialize ConversationState based on the loaded config.
	// Ref: "AgentPipe Configuration"
	return &Orchestrator{
		State: &ConversationState{
			// ... initialized from config
		},
	}, nil
}

// Run starts and manages the main orchestration loop.
func (o *Orchestrator) Run(ctx context.Context) ([]*Artifact, error) {
	// The loop continues until the feature is complete or max turns are reached.
	// Ref: "Orchestration Flow" & "Success Criteria"
	for o.State.Turn < o.State.MaxTurns {
		if o.isFeatureComplete() {
			// Success criteria met.
			break
		}

		// 1. Determine which agent's turn it is.
		// Ref: "Sequential (Default)"
		currentAgent := o.getAgentByTurn()
		if currentAgent == nil {
			return nil, errors.New("could not determine next agent")
		}
		
		// TODO: Implement Phase 2 Gate: "Design Validation"
		// If currentAgent is Coder and previous phase was Design, first run Reviewer.

		// 2. Build the prompt for the current agent.
		prompt := o.buildPrompt(currentAgent)

		// 3. Execute the turn by calling the agent's LLM.
		response, err := o.executeTurn(ctx, currentAgent, prompt)
		if err != nil {
			// TODO: Add retry logic or handle LLM errors gracefully.
			return nil, err
		}

		// 4. Parse artifacts from the response.
		// Artifacts are expected in fenced code blocks, e.g., 