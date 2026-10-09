package server_test

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Asked to start over, the assistant clears the conversation as the
// Clear button does: the messages go, the log keeps them, and undo puts
// them back.
func TestTheAssistantClearsTheConversation(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		{Text: "Noted."},
		toolCall("clear_conversation", map[string]any{}),
		{Text: "Cleared. What next?"},
	}}, nil
	postForm(t, h, "/chat", url.Values{"message": {"remember the seeds"}, "from": {"/chat"}})
	postForm(t, h, "/chat", url.Values{"message": {"start over"}, "from": {"/chat"}})
	msgs, _ := a.Chat.Messages()
	if len(msgs) != 1 || msgs[0].Fields["content"] != "Cleared. What next?" {
		t.Fatalf("only the reply after clearing is left, got %d messages", len(msgs))
	}
	entries, _ := a.Store.List(records.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true})
	for _, e := range entries {
		if e.Fields["action"] == "cleared" && e.Fields["target"] == "conversation" {
			if e.Fields["actor"] != "assistant" || !a.Records.Undoable(e) {
				t.Errorf("the clearing is the assistant's and can be undone: %v", e.Fields)
			}
			return
		}
	}
	t.Error("the clearing is in the log")
}

// Asked for a workspace, the assistant makes one beside this one, as the
// workspaces page does; someone let in to change this one cannot.
func TestTheAssistantMakesAWorkspace(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("add_workspace", map[string]any{"name": "Garden"}),
		{Text: "Made."},
	}}, nil
	postForm(t, h, "/chat", url.Values{"message": {"a workspace for the garden"}, "from": {"/chat"}})
	if _, err := os.Stat(filepath.Join(filepath.Dir(a.Workspace.Dir), "garden", "workspace.yaml")); err != nil {
		t.Errorf("the workspace is made beside this one: %v", err)
	}
	for _, tool := range a.Chat.For(records.Visitor{Name: "Bob", Access: records.Edit}).Tools() {
		switch tool.Name {
		case "add_workspace", "open_workspace", "restore_workspace":
			t.Errorf("%s is the owner's", tool.Name)
		}
	}
}
