package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// The log narrowed to one agent offers to take back what it did; pressed,
// every change of that agent that can be undone is, newest first, and what
// the person made stays; the log not narrowed to an agent offers nothing.
func TestThePersonTakesBackWhatAnAgentDid(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	asAgent := func(path string, body map[string]any) {
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Sameway-Agent", "Trial Bot")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code >= 300 {
			t.Fatalf("%s: %d %s", path, rec.Code, rec.Body)
		}
	}
	a.Store.Create("note", map[string]any{"title": "Mine to keep"}) // the person's own
	asAgent("/api/note", map[string]any{"title": "Test Note"})
	asAgent("/api/note", map[string]any{"title": "Status Test Note"})
	asAgent("/api/block", map[string]any{"component": "heading", "props": map[string]any{"text": "Trying things"}})

	if page := get(t, h, "/activity").Body.String(); strings.Contains(page, "Take back") {
		t.Error("the whole log offers nothing to take back")
	}
	who := regexp.MustCompile(`<option value="(a-[0-9a-f]+)"[^>]*>Trial Bot`).FindStringSubmatch(get(t, h, "/activity").Body.String())
	if who == nil {
		t.Fatal("the log can be narrowed to Trial Bot")
	}
	page := get(t, h, "/activity?who="+who[1]).Body.String()
	if !strings.Contains(page, "Take back 3 changes by Trial Bot") {
		i := strings.Index(page, "</fieldset>")
		t.Fatalf("narrowed to the agent, the log offers to take back its 3 changes: %q\n%.1500s", regexp.MustCompile(`Take back [^<]*`).FindString(page), page[i:])
	}
	res := postForm(t, h, "/activity/take-back", url.Values{"who": {who[1]}})
	if res.Code >= 400 {
		t.Fatalf("take back: %d %s", res.Code, res.Body)
	}
	notes, _ := a.Store.List("note", store.ListOptions{})
	if len(notes) != 1 || notes[0].Fields["title"] != "Mine to keep" {
		t.Errorf("the agent's notes go, the person's stays: %d", len(notes))
	}
	blocks, _ := a.Store.List("block", store.ListOptions{})
	for _, b := range blocks {
		if p, _ := b.Fields["props"].(map[string]any); p["text"] == "Trying things" {
			t.Error("the agent's block goes")
		}
	}
}
