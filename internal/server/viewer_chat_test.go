package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// Someone who may only look talks with the assistant as anyone else does,
// and it works for them with their permissions: it reads and answers, and
// what it tries to change is refused, so nothing is.
func TestSomeoneWhoMayOnlyLookTalksWithTheAssistant(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	a.Store.Create("note", map[string]any{"title": "Bins go out on Tuesday"})
	vi, _ := a.Store.Create("person", map[string]any{"name": "Vi", "email": "vi@example.com", "access": "view"})
	viewer := records.Visitor{Person: vi.ID, Name: "Vi", Login: "vi@example.com", Access: records.View}
	model := &scripted{steps: []*llm.Response{
		toolCall("find_records", map[string]any{"type": "note"}),
		toolCall("create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "From Vi"}}),
		{Text: "The bins go out on Tuesday."},
	}}
	a.Chat.Provider = model

	form := url.Values{"message": {"When do the bins go out? And note that I asked."}}.Encode()
	if r := as(t, h, viewer, http.MethodPost, "/chat", form, "application/x-www-form-urlencoded"); r.Code >= 400 {
		t.Fatalf("someone who may only look may talk: %d %s", r.Code, r.Body.String())
	}
	if n, _ := a.Store.Count("note"); n != 1 {
		t.Errorf("what the assistant tried to change for them is not changed: %d notes", n)
	}
	if len(model.seen) == 0 {
		t.Fatal("the assistant was asked")
	}
	for _, tl := range model.seen[0].Tools {
		if strings.HasPrefix(tl.Name, "create_") || strings.HasPrefix(tl.Name, "update_") {
			t.Errorf("it is given only what reads, not %s", tl.Name)
		}
	}
	page := as(t, h, viewer, http.MethodGet, "/chat", "", "").Body.String()
	if !strings.Contains(page, "The bins go out on Tuesday.") {
		t.Errorf("they read its answer in their own conversation\n%s", truncate(page))
	}
	if r := as(t, h, viewer, http.MethodPost, "/api/note", `{"title":"x"}`, "application/json"); r.Code != http.StatusForbidden {
		t.Errorf("and still change nothing anywhere else: %d", r.Code)
	}
}
