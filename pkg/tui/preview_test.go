package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ASRagab/agentpipe/pkg/agent"
)

func TestRenderPreviewTile(t *testing.T) {
	messages := []agent.Message{
		{Content: "First message from agent", Timestamp: time.Now().Unix()},
		{Content: "Second message with more content here", Timestamp: time.Now().Unix()},
		{Content: "Third and final message", Timestamp: time.Now().Unix()},
	}

	tile := renderPreviewTile("Claude", messages, 3, 25, lipgloss.Color("63"), false)

	if tile == "" {
		t.Error("Expected non-empty tile")
	}
	if !strings.Contains(tile, "Claude") {
		t.Error("Expected tile to contain agent name")
	}
}

func TestRenderPreviewTile_Empty(t *testing.T) {
	tile := renderPreviewTile("Claude", []agent.Message{}, 3, 25, lipgloss.Color("63"), false)

	if tile == "" {
		t.Error("Expected non-empty tile even with no messages")
	}
	if !strings.Contains(tile, "Claude") {
		t.Error("Expected tile to contain agent name")
	}
}

func TestRenderPreviewTile_ActiveAgent(t *testing.T) {
	messages := []agent.Message{
		{Content: "Some message", Timestamp: time.Now().Unix()},
	}

	tile := renderPreviewTile("Claude", messages, 3, 25, lipgloss.Color("63"), true)

	if !strings.Contains(tile, "Claude") {
		t.Error("Expected tile to contain agent name")
	}
}

func TestGetLastNLines(t *testing.T) {
	messages := []agent.Message{
		{Content: "Line 1"},
		{Content: "Line 2"},
		{Content: "Line 3"},
		{Content: "Line 4"},
	}

	result := getLastNLines(messages, 2, 50)
	lines := strings.Split(result, "\n")

	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}
}

func TestGetLastNLines_Empty(t *testing.T) {
	result := getLastNLines([]agent.Message{}, 3, 50)
	if result != "" {
		t.Error("Expected empty string for empty messages")
	}
}

func TestGetLastNLines_FewerMessagesThanN(t *testing.T) {
	messages := []agent.Message{
		{Content: "Only one message"},
	}

	result := getLastNLines(messages, 5, 50)
	lines := strings.Split(result, "\n")

	if len(lines) != 1 {
		t.Errorf("Expected 1 line, got %d", len(lines))
	}
	if !strings.Contains(result, "Only one message") {
		t.Error("Expected result to contain the message content")
	}
}

func TestGetLastNLines_LongMessages(t *testing.T) {
	// Test that long messages are properly wrapped
	messages := []agent.Message{
		{Content: "This is a very long message that should be wrapped to multiple lines when the width is small"},
	}

	result := getLastNLines(messages, 3, 20)

	// Should be wrapped since width is 20
	if len(result) == 0 {
		t.Error("Expected non-empty result")
	}
}

func TestRenderPreviewTile_WidthHandling(t *testing.T) {
	messages := []agent.Message{
		{Content: "Short msg"},
	}

	// Test with various widths
	widths := []int{15, 25, 40}
	for _, w := range widths {
		tile := renderPreviewTile("Agent", messages, 3, w, lipgloss.Color("63"), false)
		if tile == "" {
			t.Errorf("Expected non-empty tile for width %d", w)
		}
	}
}

func TestRenderPreviewTile_MultilineContent(t *testing.T) {
	messages := []agent.Message{
		{Content: "Line one\nLine two\nLine three"},
	}

	tile := renderPreviewTile("Agent", messages, 3, 30, lipgloss.Color("63"), false)

	if tile == "" {
		t.Error("Expected non-empty tile")
	}
	if !strings.Contains(tile, "Agent") {
		t.Error("Expected tile to contain agent name")
	}
}

func TestGetLastNLines_PreservesOrder(t *testing.T) {
	messages := []agent.Message{
		{Content: "First"},
		{Content: "Second"},
		{Content: "Third"},
	}

	result := getLastNLines(messages, 2, 50)
	lines := strings.Split(result, "\n")

	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}
	if lines[0] != "Second" {
		t.Errorf("Expected first line to be 'Second', got '%s'", lines[0])
	}
	if lines[1] != "Third" {
		t.Errorf("Expected second line to be 'Third', got '%s'", lines[1])
	}
}
