package server_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// What comes in to sort is put to the model as it comes: what asks
// something waits on Today as a suggested task, kept with a press; what
// asks nothing is filed away; a message shared from a phone is sorted too,
// a page shared to read is not.
func TestWhatComesInIsSortedForThePerson(t *testing.T) {
	a, _ := newApp(t)
	a.Chat.Provider = &scripted{steps: []*llm.Response{
		{Text: "Here it is:\n```json\n{\"task\": true, \"title\": \"Bring the signed form\", \"due\": \"2026-10-16\", \"important\": true, \"for\": \"Ana Silva\", \"why\": \"School needs it Friday\"}\n```"},
		{Text: `{"task": false, "why": "A newsletter"}`},
	}}
	a.Chat.ProviderErr = nil
	srv := server.New(a)
	a.Store.Create("person", map[string]any{"name": "Ana Silva"})
	form, _ := a.Store.Create("note", map[string]any{"title": "Trip on Friday", "body": "From School <office@school.example>, Mon 12 Oct 2026 08:00\n\nPlease bring the signed form by Friday.", "tags": []any{"email", "to sort"}})
	a.Store.Create("note", map[string]any{"title": "October news", "body": "Our autumn range is here.", "tags": []any{"email", "to sort"}})

	if n := srv.TriageWaiting(); n != 2 {
		t.Fatalf("both sorted: %d", n)
	}
	today := get(t, srv, "/today").Body.String()
	for _, want := range []string{"To sort", "Suggested task: Bring the signed form", "important", "for Ana Silva", "School needs it Friday", ">Keep<", ">Change<", ">Not a task<"} {
		if !strings.Contains(today, want) {
			t.Errorf("Today suggests it (%s): %s", want, truncate(today))
		}
	}
	if strings.Contains(today, "October news") {
		t.Error("what asks nothing is filed away")
	}
	if srv.TriageWaiting() != 0 {
		t.Error("each is asked about once")
	}

	rec := postForm(t, srv, "/sort/keep", url.Values{"id": {form.ID}})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("kept: %d %s", rec.Code, truncate(rec.Body.String()))
	}
	tasks, _ := a.Store.List("task", store.ListOptions{})
	var task *store.Record
	for _, r := range tasks {
		if r.Fields["title"] == "Bring the signed form" {
			task = r
		}
	}
	if task == nil || !strings.HasPrefix(task.Fields["due"].(string), "2026-10-16") || !strings.Contains(fmt.Sprint(task.Fields["tags"]), "important") || task.Fields["for"] == nil || task.Fields["for"] == "" {
		t.Fatalf("the task as suggested, for Ana: %v", task)
	}
	if strings.Contains(get(t, srv, "/today").Body.String(), "Trip on Friday") {
		t.Error("kept, it is sorted")
	}

	postForm(t, srv, "/share", url.Values{"text": {"Can you pick up the kids at 4 tomorrow?"}})
	postForm(t, srv, "/share", url.Values{"url": {"http://127.0.0.1:1/recipe"}})
	if page := get(t, srv, "/today").Body.String(); !strings.Contains(page, "Can you pick up the kids") || strings.Contains(page, "127.0.0.1") {
		t.Errorf("a message shared waits to be sorted, a page does not: %s", truncate(page))
	}
}
