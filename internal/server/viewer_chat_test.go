package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Someone who may only look talks with the assistant as anyone else does,
// and it works for them with their permissions: it reads and answers, and
// a change they ask for is not made but put to the owner, on Today, in
// Sameway's words; the owner's yes makes exactly that, and nothing before.
func TestSomeoneWhoMayOnlyLookAsksAndTheOwnerSaysYes(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	a.Store.Create("note", map[string]any{"title": "Bins go out on Tuesday"})
	vi, _ := a.Store.Create("person", map[string]any{"name": "Vi", "email": "vi@example.com", "access": "view"})
	viewer := records.Visitor{Person: vi.ID, Name: "Vi", Login: "vi@example.com", Access: records.View}
	model := &scripted{steps: []*llm.Response{
		toolCall("find_records", map[string]any{"type": "note"}),
		toolCall("create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "Milk for Saturday"}}),
		{Text: "The bins go out on Tuesday. I have asked the owner to add the note."},
	}}
	a.Chat.Provider = model

	if page := as(t, h, viewer, http.MethodGet, "/chat", "", "").Body.String(); !strings.Contains(page, "What you may not change yourself is asked of the owner, and you hear their yes or no here.") {
		t.Errorf("they are told before they ask that a change waits for a yes\n%s", truncate(page))
	}
	if strings.Contains(get(t, h, "/chat").Body.String(), "asked of the owner") {
		t.Error("the owner is not told about asking themselves")
	}
	form := url.Values{"message": {"When do the bins go out? And add a note: milk for Saturday."}}.Encode()
	if r := as(t, h, viewer, http.MethodPost, "/chat", form, "application/x-www-form-urlencoded"); r.Code >= 400 {
		t.Fatalf("someone who may only look may talk: %d %s", r.Code, r.Body.String())
	}
	if n, _ := a.Store.Count("note"); n != 1 {
		t.Fatalf("nothing is made before the owner says yes: %d notes", n)
	}
	if len(model.seen) < 3 || !strings.Contains(model.seen[2].Messages[len(model.seen[2].Messages)-1].ToolResults[0].Content, "Nothing has changed") {
		t.Errorf("the assistant is told nothing changed and why: %+v", model.seen)
	}
	page := as(t, h, viewer, http.MethodGet, "/chat", "", "").Body.String()
	if !strings.Contains(page, "The bins go out on Tuesday.") {
		t.Errorf("they read its answer in their own conversation\n%s", truncate(page))
	}

	today := get(t, h, "/today").Body.String()
	want := `Vi (vi@example.com) asks you to make a note: &#34;Milk for Saturday&#34;. Make it?`
	if !strings.Contains(today, want) {
		t.Fatalf("the request waits on the owner's Today, in Sameway's words\n%s", truncate(today))
	}
	asks := a.Chat.Suggestions()
	if len(asks) != 1 {
		t.Fatalf("one request: %d", len(asks))
	}
	wantStatus(t, postForm(t, h, "/suggested/yes", url.Values{"id": {asks[0].ID}}), http.StatusSeeOther)
	notes, _ := a.Store.List("note", store.ListOptions{})
	made := false
	for _, n := range notes {
		made = made || n.Fields["title"] == "Milk for Saturday"
	}
	if !made {
		t.Error("the owner's yes makes exactly what was asked")
	}
	if r := as(t, h, viewer, http.MethodPost, "/api/note", `{"title":"x"}`, "application/json"); r.Code != http.StatusForbidden {
		t.Errorf("and they still change nothing themselves anywhere else: %d", r.Code)
	}
}
