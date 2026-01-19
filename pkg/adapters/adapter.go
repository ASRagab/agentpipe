// Package adapters provides the interface and implementations for AI agent adapters.
package adapters

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/ASRagab/agentpipe/pkg/core"
)

type AgentAdapter interface {
	Initialize(agent core.Agent) error

	SendMessage(ctx context.Context, messages []core.Message, conversation *core.ConversationContext) (string, *core.Metrics, error)
	StreamMessage(ctx context.Context, messages []core.Message, writer io.Writer, conversation *core.ConversationContext) (*core.Metrics, error)

	IsAvailable() bool
	GetModel() string
	HealthCheck(ctx context.Context) error
}

func BuildConversationContextPrompt(conversation *core.ConversationContext) string {
	if conversation == nil {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("CONVERSATION CONTEXT:\n")
	builder.WriteString(strings.Repeat("-", 40))
	builder.WriteString("\n")
	if conversation.ConversationID != "" {
		builder.WriteString(fmt.Sprintf("• Session: %s", conversation.ConversationID))
		if conversation.CurrentTurn > 0 {
			builder.WriteString(fmt.Sprintf(" (Turn %d", conversation.CurrentTurn))
			if conversation.MaxTurns > 0 {
				builder.WriteString(fmt.Sprintf(" of %d", conversation.MaxTurns))
			}
			builder.WriteString(")")
		}
		builder.WriteString("\n")
	} else if conversation.CurrentTurn > 0 {
		builder.WriteString(fmt.Sprintf("• Turn: %d", conversation.CurrentTurn))
		if conversation.MaxTurns > 0 {
			builder.WriteString(fmt.Sprintf(" of %d", conversation.MaxTurns))
		}
		builder.WriteString("\n")
	}
	if conversation.Mode != "" {
		builder.WriteString(fmt.Sprintf("• Mode: %s\n", conversation.Mode))
	}
	if len(conversation.Participants) > 0 {
		builder.WriteString("• Participants:\n")
		for _, participant := range conversation.Participants {
			label := participant.Name
			if participant.Type != "" {
				label = fmt.Sprintf("%s (%s)", label, participant.Type)
			}
			builder.WriteString(fmt.Sprintf("  - %s\n", label))
		}
	}
	if conversation.LastSpeaker != "" {
		builder.WriteString(fmt.Sprintf("• Last speaker: %s\n", conversation.LastSpeaker))
	}
	builder.WriteString(strings.Repeat("-", 40))
	builder.WriteString("\n\n")

	builder.WriteString("RESPONSE GUIDELINES:\n")
	builder.WriteString("• You are ONE voice in a GROUP conversation, not a solo assistant.\n")
	builder.WriteString("• Keep responses concise (2–4 short paragraphs max).\n")
	builder.WriteString("• Build on what others said; avoid repeating their points.\n")
	builder.WriteString("• No preambles, no sign-offs, no meta-commentary.\n")
	builder.WriteString("• If you have nothing new to add, say so briefly and yield.\n")
	return builder.String()
}

// AdapterFactory is a function that creates a new adapter instance.
type AdapterFactory func() AgentAdapter
