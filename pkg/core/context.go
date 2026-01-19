package core

type ConversationParticipant struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

type ConversationContext struct {
	ConversationID string `json:"conversation_id"`
	CurrentTurn    int    `json:"current_turn"`
	MaxTurns       int    `json:"max_turns,omitempty"`
	Mode           string `json:"mode"`

	Participants []ConversationParticipant `json:"participants"`

	ActiveAgentID   string `json:"active_agent_id,omitempty"`
	ActiveAgentName string `json:"active_agent_name,omitempty"`
	LastSpeaker     string `json:"last_speaker,omitempty"`

	InitialPrompt string `json:"initial_prompt,omitempty"`
	TotalMessages int    `json:"total_messages"`
}
