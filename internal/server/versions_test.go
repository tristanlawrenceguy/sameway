package server_test

import (
	"encoding/json"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// A person saves from a page opened before an agent changed the record.
// The editor sends every field as the page showed it: what the person
// left alone keeps the agent's change; what they both changed is the
// person's, and they are told.
func TestASaveFromAnOldPageKeepsWhatOthersChanged(t *testing.T) {
	a, h := newApp(t)
	var made map[string]any
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Seeds", "body": "Kale", "status": "draft"}), &made)
	id := made["id"].(string)
	page := get(t, h, "/t/note/"+id).Body.String()
	attr := func(name string) string {
		m := regexp.MustCompile(name + `="([^"]*)"`).FindStringSubmatch(page)
		if m == nil {
			t.Fatalf("the page says %s", name)
		}
		return html.UnescapeString(m[1])
	}
	version, was := attr("data-version"), attr("data-was")

	// Meanwhile an agent changes the body and the status.
	postJSON(t, h, http.MethodPatch, "/api/note/"+id, map[string]any{"body": "Kale and chard", "status": "published"})

	// The person, on the page as it was, changes the title and the status.
	res := postForm(t, h, "/t/note/"+id+"/props", url.Values{
		"prop-title": {"Seeds to order"}, "prop-body": {"Kale"}, "prop-status": {"draft"},
		"version": {version}, "was": {was},
	})
	rec, _ := a.Store.Get("note", id)
	if rec.Fields["title"] != "Seeds to order" || rec.Fields["body"] != "Kale and chard" {
		t.Errorf("the person's title, and the agent's body left alone: %v", rec.Fields)
	}
	_ = res
	// Status both changed? The person sent it as it was, so it is the agent's.
	if rec.Fields["status"] != "published" {
		t.Errorf("a field the person did not touch keeps the agent's change: %v", rec.Fields["status"])
	}

	// Both change the title: the person's is saved, and they are told.
	page = get(t, h, "/t/note/"+id).Body.String()
	version, was = attr("data-version"), attr("data-was")
	postJSON(t, h, http.MethodPatch, "/api/note/"+id, map[string]any{"title": "Seeds (agent)"})
	// Read as JSON, the message the page would open with (agents.go).
	_, said := agentPost(t, h, "/t/note/"+id+"/props", map[string]any{"prop-title": "Seeds (person)", "version": version, "was": was})
	told, _ := said["text"].(string)
	if rec, _ := a.Store.Get("note", id); rec.Fields["title"] != "Seeds (person)" {
		t.Errorf("the last word is the person's: %v", rec.Fields["title"])
	}
	if !strings.Contains(told, "had been changed since you opened this") {
		t.Errorf("the person is told the title had changed")
	}
}

// An agent that says which version it read is refused, with the record
// as it is, when that version is out of date: over the API with If-Match,
// and through the assistant's update_record.
func TestAnAgentsChangeFromAnOldVersionIsRefused(t *testing.T) {
	a, h := newApp(t)
	var made map[string]any
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Plan"}), &made)
	id := made["id"].(string)
	got := get(t, h, "/api/note/"+id)
	etag := got.Header().Get("ETag")
	if etag == "" {
		t.Fatal("a record comes with its version")
	}
	postJSON(t, h, http.MethodPatch, "/api/note/"+id, map[string]any{"title": "Plan B"})

	req := httptest.NewRequest(http.MethodPatch, "/api/note/"+id, strings.NewReader(`{"title":"Plan C"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", etag)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	var out map[string]any
	json.Unmarshal(res.Body.Bytes(), &out)
	if res.Code != http.StatusPreconditionFailed || out["current"] == nil {
		t.Fatalf("an old version is refused with the record as it is: %d %s", res.Code, res.Body.String())
	}
	if rec, _ := a.Store.Get("note", id); rec.Fields["title"] != "Plan B" {
		t.Errorf("nothing was written: %v", rec.Fields["title"])
	}
	req = httptest.NewRequest(http.MethodPatch, "/api/note/"+id, strings.NewReader(`{"title":"Plan C"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", get(t, h, "/api/note/"+id).Header().Get("ETag"))
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Errorf("the current version is taken: %d %s", res.Code, res.Body.String())
	}

	rec, _ := a.Store.Get("note", id)
	old := records.Version(rec)
	postJSON(t, h, http.MethodPatch, "/api/note/"+id, map[string]any{"title": "Plan D"})
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("update_record", map[string]any{"type": "note", "id": id, "fields": map[string]any{"title": "Plan E"}, "version": old}),
		{Text: "Tried."},
	}}, nil
	postForm(t, h, "/chat", url.Values{"message": {"rename it"}, "from": {"/chat"}})
	if rec, _ := a.Store.Get("note", id); rec.Fields["title"] != "Plan D" {
		t.Errorf("the assistant's change from an old version is not written: %v", rec.Fields["title"])
	}
}

// A record on the canvas is the record drawn again: it tells the editor
// what its own page does, where a save goes, what it showed and its
// language, so an edit made there loses nobody's work either.
func TestARecordOnTheCanvasEditsAsItsPageDoes(t *testing.T) {
	_, h := newApp(t)
	var made map[string]any
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Einkauf", "language": "de"}), &made)
	id := made["id"].(string)
	postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "record", "props": map[string]any{"type": "note", "record": id}})
	editing := regexp.MustCompile(`data-version="[^"]*" data-was="[^"]*" data-edit-action="/t/note/` + id + `/props" lang="de"`)
	page, canvas := get(t, h, "/t/note/"+id).Body.String(), get(t, h, "/").Body.String()
	onPage, onCanvas := editing.FindString(page), editing.FindString(canvas)
	if onPage == "" || onPage != onCanvas {
		t.Errorf("the canvas says what the page says:\npage:   %q\ncanvas: %q", onPage, onCanvas)
	}
}
