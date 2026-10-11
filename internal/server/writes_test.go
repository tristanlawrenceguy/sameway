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

// That every way in writes through records at all is checked by
// tools/check (writes.go), across every package.

// Made, changed and deleted from a page, over the API and by the
// assistant, a record leaves the same trail each time: an entry in the
// log, by whoever did it, that can be undone.
func TestEveryWayInLeavesTheSameTrail(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	trail := func(way, id, action, actor string) {
		t.Helper()
		entries, _ := a.Store.List(records.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true})
		for _, e := range entries {
			if e.Fields["target_id"] == id && e.Fields["action"] == action {
				if e.Fields["actor"] != actor {
					t.Errorf("%s: %s is logged as %v's, not %s's", way, action, e.Fields["actor"], actor)
				}
				if !a.Records.Undoable(e) {
					t.Errorf("%s: %s cannot be undone", way, action)
				}
				return
			}
		}
		t.Errorf("%s: %s left nothing in the log", way, action)
	}

	// A page.
	res := postForm(t, h, "/t/note/add", url.Values{})
	id := strings.TrimPrefix(strings.SplitN(res.Header().Get("Location"), "?", 2)[0], "/t/note/")
	trail("a page", id, "created", "human")
	postForm(t, h, "/t/note/"+id+"/props", url.Values{"prop-title": {"From a page"}})
	trail("a page", id, "updated", "human")
	postForm(t, h, "/t/note/"+id+"/delete", url.Values{})
	trail("a page", id, "deleted", "human")

	// The API.
	var made map[string]any
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "From the API"}), &made)
	id, _ = made["id"].(string)
	trail("the API", id, "created", records.ActorAgent)
	postJSON(t, h, http.MethodPatch, "/api/note/"+id, map[string]any{"title": "Changed"})
	trail("the API", id, "updated", records.ActorAgent)
	do(t, h, http.MethodDelete, "/api/note/"+id, nil, "")
	trail("the API", id, "deleted", records.ActorAgent)
	up := postJSON(t, h, http.MethodPost, "/api/file/upload", map[string]any{"filename": "later.pdf"})
	var stub map[string]any
	decode(t, up, &stub)
	if up.Code != http.StatusCreated {
		t.Fatalf("a file without content: %d %v", up.Code, stub)
	}
	trail("the API's file without content", stub["id"].(string), "created", records.ActorAgent)

	// The assistant.
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "From the assistant"}}),
		{Text: "Made."},
	}}, nil
	postForm(t, h, "/chat", url.Values{"message": {"a note"}, "from": {"/chat"}})
	notes, _ := a.Store.List("note", store.ListOptions{})
	for _, n := range notes {
		if n.Fields["title"] == "From the assistant" {
			trail("the assistant", n.ID, "created", "assistant")
		}
	}
}
