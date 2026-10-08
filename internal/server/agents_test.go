package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// agentPost posts to a page's form address as an agent does.
func agentPost(t *testing.T, h http.Handler, path string, body any) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("POST %s answered an agent with something other than JSON (%s): %.300s", path, rec.Header().Get("Content-Type"), rec.Body.String())
	}
	return rec, out
}

// No action a person can take from a page is out of an agent's reach: each
// POST route a form uses answers an agent in JSON through the same outcome
// a person reads, never with a page to pick apart. A route added as a page
// alone fails here, not weeks later when an agent goes looking.
func TestEveryPageActionAnswersAnAgent(t *testing.T) {
	// Answered otherwise, for a reason: a stream of words as they come
	// (the same turn is POST /api/chat), and files sent as multipart (an
	// agent sends them to POST /api/file/upload).
	elsewhere := map[string]bool{"/chat/stream": true, "/t/file/upload": true}
	var routes []string
	for _, r := range pageActions() { // tools_parity_test.go
		if p := strings.TrimPrefix(r.Pattern, "POST "); !elsewhere[p] {
			routes = append(routes, p)
		}
	}
	if len(routes) < 30 {
		t.Fatalf("found only %d page actions in the route table", len(routes))
	}
	_, h := newApp(t)
	for _, route := range routes {
		path := regexp.MustCompile(`\{[^}]*\}`).ReplaceAllString(route, "nothing")
		_, out := agentPost(t, h, path, map[string]any{})
		if out["answered_with_page"] == true {
			t.Errorf("POST %s answers an agent with a page (%v: %v); end it in tell or failed, which say it to both", route, out["title"], out["text"])
		}
	}
}

// Taking a change back, as an agent: the Undo button's own address, a
// JSON body, and an answer saying what happened and where.
func TestAnAgentUsesTheSameFormsAsAPerson(t *testing.T) {
	a, h := newApp(t)
	rec, out := agentPost(t, h, "/t/note/add", map[string]any{})
	loc, _ := out["location"].(string)
	if rec.Code != http.StatusOK || out["ok"] != true || !strings.HasPrefix(loc, "/t/note/") {
		t.Fatalf("adding a note from its list's button: %d %v", rec.Code, out)
	}
	id := strings.TrimPrefix(strings.SplitN(loc, "?", 2)[0], "/t/note/")
	rec, out = agentPost(t, h, "/t/note/"+id+"/props", map[string]any{"prop-title": "Seeds to order"})
	if rec.Code != http.StatusOK || out["ok"] != true {
		t.Fatalf("naming it through the editor's form: %d %v", rec.Code, out)
	}
	notes, _ := a.Store.List("note", store.ListOptions{})
	if len(notes) != 1 || notes[0].Fields["title"] != "Seeds to order" {
		t.Fatalf("the JSON body is the form: %+v", notes[0].Fields)
	}
	entries, _ := a.Store.List("activity", store.ListOptions{})
	var added string
	for _, e := range entries {
		if e.Fields["target_id"] == notes[0].ID && e.Fields["action"] == "created" {
			added = e.ID
		}
	}
	rec, out = agentPost(t, h, "/activity/"+added+"/undo", map[string]any{})
	if rec.Code != http.StatusOK || out["ok"] != true || out["title"] != "Undone" {
		t.Fatalf("undoing it from the Undo button's address: %d %v", rec.Code, out)
	}
	if _, err := a.Store.Get("note", notes[0].ID); err == nil {
		t.Error("the note is gone again")
	}
	rec, out = agentPost(t, h, "/activity/"+added+"/undo", map[string]any{})
	if rec.Code != http.StatusBadRequest || out["ok"] != false || out["text"] == "" {
		t.Errorf("an entry that cannot be undone now says why: %d %v", rec.Code, out)
	}
}

// A chat turn from an agent can carry a file it added, as a person's can,
// and what chat does not take is refused with the field that is wrong.
func TestAnAgentsChatTurnCarriesAFile(t *testing.T) {
	a, h := newApp(t)
	model := &sees{}
	a.Chat.Provider = model
	up := postJSON(t, h, http.MethodPost, "/api/file/upload", map[string]any{"filename": "list.txt", "content": "U2VlZHM6IGthbGUsIGNoYXJk"})
	var file map[string]any
	decode(t, up, &file)
	id, _ := file["id"].(string)

	res := postJSON(t, h, http.MethodPost, "/api/chat", map[string]any{"message": "What is on it?", "file": id})
	if res.Code != http.StatusOK {
		t.Fatalf("a turn with a file: %d %s", res.Code, res.Body.String())
	}
	last := model.seen[len(model.seen)-1].Messages
	if got := last[len(last)-1].Content; !strings.Contains(got, "kale") {
		t.Errorf("the file's text goes with the message: %q", got)
	}
	for body, field := range map[string]map[string]any{
		"file":    {"message": "hi", "file": "nothing"},
		"attach":  {"message": "hi", "attach": id},
		"message": {"message": 3},
	} {
		res := postJSON(t, h, http.MethodPost, "/api/chat", field)
		if res.Code != http.StatusUnprocessableEntity || !strings.Contains(res.Body.String(), `"`+body+`"`) {
			t.Errorf("%v is refused naming %s: %d %s", field, body, res.Code, res.Body.String())
		}
	}
}

// A change in the log is taken back from its own page too.
func TestAnActivityEntrysPageOffersUndo(t *testing.T) {
	a, h := newApp(t)
	postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Plan"})
	entries, _ := a.Store.List("activity", store.ListOptions{})
	if len(entries) == 0 {
		t.Fatal("adding a note is logged")
	}
	page := get(t, h, "/t/activity/"+entries[0].ID).Body.String()
	if !strings.Contains(page, `action="/activity/`+entries[0].ID+`/undo"`) {
		t.Errorf("the entry's page has its Undo:\n%.1500s", page)
	}
}
