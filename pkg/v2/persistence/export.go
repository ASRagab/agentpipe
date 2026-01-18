package persistence

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kevinelliott/agentpipe/pkg/log"
	"github.com/kevinelliott/agentpipe/pkg/v2/core"
)

// ExportToMarkdown generates a formatted Markdown string from a conversation.
func ExportToMarkdown(conversation *core.Conversation) (string, error) {
	if conversation == nil {
		return "", fmt.Errorf("conversation cannot be nil")
	}

	var sb strings.Builder

	// Write header
	sb.WriteString("# AgentPipe Conversation Export\n\n")

	// Metadata section
	sb.WriteString("## Metadata\n\n")
	sb.WriteString(fmt.Sprintf("- **Conversation ID**: `%s`\n", conversation.ID))
	sb.WriteString(fmt.Sprintf("- **Started**: %s\n", conversation.Started.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("- **Updated**: %s\n", conversation.Updated.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("- **Status**: %s\n", conversation.Status))
	sb.WriteString(fmt.Sprintf("- **Total Messages**: %d\n", len(conversation.Messages)))
	sb.WriteString(fmt.Sprintf("- **Total Agents**: %d\n", len(conversation.Agents)))

	// Agents section
	if len(conversation.Agents) > 0 {
		sb.WriteString("\n### Participants\n\n")
		for _, agent := range conversation.Agents {
			sb.WriteString(fmt.Sprintf("- **%s** (%s) - Model: `%s`\n", agent.Name, agent.Type, agent.Model))
		}
	}

	sb.WriteString("\n---\n\n")

	// Messages section
	sb.WriteString("## Conversation\n\n")

	for i, msg := range conversation.Messages {
		if i > 0 {
			sb.WriteString("\n---\n\n")
		}

		// Message header
		var roleIcon string
		var roleName string
		switch msg.Role {
		case core.RoleUser:
			roleIcon = "👤"
			roleName = "User"
		case core.RoleAgent:
			roleIcon = "🤖"
			roleName = msg.AgentName
			if roleName == "" {
				roleName = "Agent"
			}
		case core.RoleSystem:
			roleIcon = "⚙️"
			roleName = "System"
		}

		sb.WriteString(fmt.Sprintf("### %s %s\n\n", roleIcon, roleName))
		sb.WriteString(fmt.Sprintf("*%s*\n\n", msg.Timestamp.Format(time.RFC3339)))

		// Message content
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			content = "*[No content]*"
		}
		sb.WriteString(content)
		sb.WriteString("\n")

		// Metrics for agent messages
		if msg.Role == core.RoleAgent && msg.Metrics != nil {
			sb.WriteString("\n<details>\n<summary>📊 Metrics</summary>\n\n")
			sb.WriteString(fmt.Sprintf("- Duration: %s\n", msg.Metrics.Duration.Round(time.Millisecond)))
			sb.WriteString(fmt.Sprintf("- Tokens: %d (in: %d, out: %d)\n",
				msg.Metrics.TotalTokens, msg.Metrics.InputTokens, msg.Metrics.OutputTokens))
			if msg.Metrics.Model != "" {
				sb.WriteString(fmt.Sprintf("- Model: `%s`\n", msg.Metrics.Model))
			}
			if msg.Metrics.Cost > 0 {
				sb.WriteString(fmt.Sprintf("- Cost: $%.6f\n", msg.Metrics.Cost))
			}
			sb.WriteString("\n</details>\n")
		}
	}

	// Summary section
	sb.WriteString("\n---\n\n")
	sb.WriteString("## Summary\n\n")

	totalTokens := conversation.TotalTokens()
	totalCost := conversation.TotalCost()
	duration := conversation.Duration()

	sb.WriteString(fmt.Sprintf("- **Total Messages**: %d\n", len(conversation.Messages)))
	sb.WriteString(fmt.Sprintf("- **Total Tokens**: %d\n", totalTokens))
	if totalCost > 0 {
		sb.WriteString(fmt.Sprintf("- **Total Cost**: $%.6f\n", totalCost))
	}
	sb.WriteString(fmt.Sprintf("- **Duration**: %s\n", duration.Round(time.Second)))

	// Footer
	sb.WriteString("\n---\n\n")
	sb.WriteString("*Exported by AgentPipe*\n")

	return sb.String(), nil
}

// SaveAsMarkdown exports a conversation to a Markdown file.
func SaveAsMarkdown(conversation *core.Conversation, outputPath string) error {
	if conversation == nil {
		return fmt.Errorf("conversation cannot be nil")
	}

	content, err := ExportToMarkdown(conversation)
	if err != nil {
		return fmt.Errorf("failed to export to markdown: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(outputPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	log.WithFields(map[string]interface{}{
		"conversation_id": conversation.ID,
		"output_path":     outputPath,
	}).Info("conversation exported to markdown")

	return nil
}

// GenerateMarkdownFilename generates a filename for a Markdown export.
func GenerateMarkdownFilename(conversation *core.Conversation) string {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	idPrefix := conversation.ID
	if len(idPrefix) > 8 {
		idPrefix = idPrefix[:8]
	}
	return fmt.Sprintf("conversation_%s_%s.md", idPrefix, timestamp)
}
