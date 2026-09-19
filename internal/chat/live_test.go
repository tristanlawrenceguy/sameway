package chat

import (
	"encoding/json"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// TestDescribeLabelClearConversation verifies that the describe helper returns
// a readable progress label for clear_conversation rather than the generic
// title-cased form. This keeps the live UI consistent with other tool labels.
func TestDescribeLabelClearConversation(t *testing.T) {
	got := describe(llm.ToolCall{Name: "clear_conversation"})
	want := "Clearing the conversation"
	if got != want {
		t.Errorf("describe(clear_conversation): expected %q, got %q", want, got)
	}
}

// TestDescribeLabelClearCanvas ensures clear_canvas still returns its label.
func TestDescribeLabelClearCanvas(t *testing.T) {
	got := describe(llm.ToolCall{Name: "clear_canvas"})
	want := "Clearing the page"
	if got != want {
		t.Errorf("describe(clear_canvas): expected %q, got %q", want, got)
	}
}

// TestDescribeLabelAddComponent verifies the label includes the component name.
func TestDescribeLabelAddComponent(t *testing.T) {
	args, _ := json.Marshal(map[string]any{"component": "card"})
	got := describe(llm.ToolCall{Name: "add_component", Args: args})
	if got != "Adding a card" {
		t.Errorf("describe(add_component card): expected %q, got %q", "Adding a card", got)
	}
}

// TestDescribeLabelUnknownUsesTitleCase verifies the fallback for unknown tools.
func TestDescribeLabelUnknownUsesTitleCase(t *testing.T) {
	got := describe(llm.ToolCall{Name: "frobnicate"})
	want := "Frobnicate"
	if got != want {
		t.Errorf("describe(frobnicate): expected %q, got %q", want, got)
	}
}
