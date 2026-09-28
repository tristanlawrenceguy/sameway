package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// The shape of the content can change while the workspace runs: an agent
// adds a type or a field over the API, the assistant does the same with
// its tools, and the pages and catalogue show it at once.
func TestTheShapeOfContentCanChangeWhileRunning(t *testing.T) {
	a, h := newApp(t)
	made := postJSON(t, h, http.MethodPost, "/api/types", map[string]any{
		"name": "ritual", "description": "Something done often.",
		"fields": []map[string]any{{"name": "name", "type": "string", "required": true}, {"name": "every", "type": "enum", "values": []string{"day", "week"}, "default": "day"}},
	})
	wantStatus(t, made, http.StatusCreated)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/ritual", map[string]any{"name": "Walk"}), http.StatusCreated)
	if page := get(t, h, "/t/ritual").Body.String(); !strings.Contains(page, "Walk") {
		t.Error("the new type has its list page with the record on it")
	}
	if nav := get(t, h, "/").Body.String(); !strings.Contains(nav, `href="/t/ritual"`) {
		t.Error("the new type is in the header with the others")
	}
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/types/ritual/fields", map[string]any{"name": "streak", "type": "int", "default": 0}), http.StatusCreated)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/ritual", map[string]any{"name": "Read", "streak": 3}), http.StatusCreated)
	bad := postJSON(t, h, http.MethodPost, "/api/types/ritual/fields", map[string]any{"name": "streak", "type": "int"})
	wantStatus(t, bad, http.StatusUnprocessableEntity)
	if !strings.Contains(bad.Body.String(), "already has") {
		t.Errorf("a repeated field says so: %s", bad.Body.String())
	}
	if !strings.Contains(get(t, h, "/api/activity").Body.String(), "streak on ritual") {
		t.Error("the change is in the activity log")
	}

	// The assistant adds a property and uses it in the same conversation.
	model := &scripted{steps: []*llm.Response{
		toolCall("add_field", map[string]any{"type": "note", "name": "due", "kind": "datetime", "description": "When it is needed by."}),
		toolCall("create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "Plan", "due": "2026-10-01T00:00:00Z"}}),
		{Text: "Notes have a due date now, and the plan is due on the first."},
	}}
	a.Chat.Provider, a.Chat.ProviderErr = model, nil
	wantStatus(t, postForm(t, h, "/chat", url.Values{"message": {"give notes a due date and make the plan due 1 October"}}), http.StatusSeeOther)
	if len(model.seen) != 3 || !strings.Contains(model.seen[2].System, `"due"`) {
		t.Errorf("the catalogue in the next prompt should carry the new field: %d calls", len(model.seen))
	}
	if added := model.seen[1].Messages[len(model.seen[1].Messages)-1].ToolResults[0].Content; !strings.Contains(added, "added due (datetime) to note") {
		t.Errorf("the tool says what it added: %q", added)
	}
	var notes struct {
		Records []struct{ Fields map[string]any }
	}
	decode(t, get(t, h, "/api/note"), &notes)
	if len(notes.Records) != 1 || notes.Records[0].Fields["due"] != "2026-10-01T00:00:00Z" {
		t.Errorf("the record carries the field the assistant just added: %+v", notes.Records)
	}
	if page := get(t, h, "/t/note").Body.String(); strings.Contains(page, "Bad") {
		t.Error("nothing odd on the page")
	}
}
