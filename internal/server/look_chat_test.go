package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// The assistant sees the page the way the person gets it. Told the Body
// cannot be reached, it opens the editor as they did and reads where Tab
// goes, instead of guessing; its step says so on the page while it looks.
func TestTheAssistantLooksAtThePageThePersonIsOn(t *testing.T) {
	needBrowser(t)
	a, h := newApp(t)
	page := notePage(t, h)
	model := &scripted{steps: []*llm.Response{
		toolCall("look_at_page", map[string]any{"path": page, "steps": []map[string]any{{"press": "Edit"}}, "only": []string{"controls"}}),
		{Text: "Body is reachable with Tab."},
	}}
	a.Chat.Provider, a.Chat.ProviderErr = model, nil

	var names []string
	for _, tool := range a.Chat.Tools() {
		names = append(names, tool.Name)
	}
	if !strings.Contains(strings.Join(names, " "), "look_at_page") {
		t.Fatalf("the assistant has look_at_page, got %v", names)
	}
	rec := postForm(t, h, "/chat", map[string][]string{"message": {"I cannot reach the body with Tab"}, "from": {"/"}})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("the turn runs, got %d", rec.Code)
	}
	var seen string
	if n := len(model.seen); n > 0 {
		for _, m := range model.seen[n-1].Messages {
			for _, r := range m.ToolResults {
				seen += r.Content
			}
		}
	}
	for _, want := range []string{`"pressed button: Edit block"`, `"textbox: Body"`, `"focus_order"`} {
		if !strings.Contains(seen, want) {
			t.Errorf("the look the model reads carries %s, got\n%.2000s", want, seen)
		}
	}
}
