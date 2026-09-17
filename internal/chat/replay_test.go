package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A reply that used tools is replayed to the model as the tool use it
// was, calls then results then words, so the conversation keeps showing
// tools being used rather than replies that only mention them.
func TestHistoryReplaysTheToolsAReplyUsed(t *testing.T) {
	svc := newFullService(t)
	svc.Provider = &scripted{steps: []*llm.Response{
		call("create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "Plan"}}),
		{Text: "Made the note."},
	}}
	if _, err := svc.Send(context.Background(), "make a note called Plan"); err != nil {
		t.Fatal(err)
	}
	msgs, _ := svc.Store.List(chat.MessageType, store.ListOptions{OrderBy: "created_at"})
	tools, _ := msgs[1].Fields["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("the reply should carry the one tool it used, got %v", msgs[1].Fields["tools"])
	}
	first, _ := tools[0].(map[string]any)
	if first["name"] != "create_record" || !strings.Contains(first["result"].(string), "/t/note/") || first["error"] != false {
		t.Errorf("a used tool keeps its name, args and result: %v", first)
	}

	m := &scripted{}
	svc.Provider = m
	svc.Send(context.Background(), "and now?")
	got := m.seen[0].Messages
	roles := make([]string, len(got))
	for i, msg := range got {
		roles[i] = string(msg.Role)
	}
	want := []string{string(llm.RoleUser), string(llm.RoleAssistant), string(llm.RoleTool), string(llm.RoleAssistant), string(llm.RoleUser)}
	if strings.Join(roles, ",") != strings.Join(want, ",") {
		t.Fatalf("history should be user, the tool turn, the words, then the new ask; got %v", roles)
	}
	if len(got[1].ToolCalls) != 1 || got[1].ToolCalls[0].Name != "create_record" || !strings.Contains(string(got[1].ToolCalls[0].Args), `"Plan"`) {
		t.Errorf("the replayed call carries its name and args: %+v", got[1].ToolCalls)
	}
	if len(got[2].ToolResults) != 1 || got[2].ToolResults[0].CallID != got[1].ToolCalls[0].ID || !strings.Contains(got[2].ToolResults[0].Content, "/t/note/") {
		t.Errorf("the replayed result answers the call: %+v", got[2].ToolResults)
	}
	if got[3].Content != "Made the note." {
		t.Errorf("the words follow the tools: %q", got[3].Content)
	}

	// A reply that used no tools, or an older one without the field, is
	// just its words.
	if len(replayOf(t, svc, "no tools here")) != 0 {
		t.Error("nothing to replay for a plain reply")
	}
}

func replayOf(t *testing.T, svc *chat.Service, content string) []llm.Message {
	t.Helper()
	svc.Store.DeleteAll(chat.MessageType)
	svc.Store.Create(chat.MessageType, map[string]any{"role": "user", "content": "hi"})
	svc.Store.Create(chat.MessageType, map[string]any{"role": "assistant", "content": content})
	m := &scripted{}
	svc.Provider = m
	svc.Send(context.Background(), "again")
	var tools []llm.Message
	for _, msg := range m.seen[0].Messages {
		if len(msg.ToolCalls) > 0 || len(msg.ToolResults) > 0 {
			tools = append(tools, msg)
		}
	}
	return tools
}
