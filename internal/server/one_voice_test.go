package server_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A change in the log is said the same way wherever it is read, however
// it was stored: an entry written before settings had names, and the undo
// of it, read on the log, on the entry's own page and over the API. The
// stored words had a setting's description in them ("Pace — how changes
// arrive"), which the log cleaned and the entry's page and the API did
// not (crew findings 0566, 0571, 0572, 0574, 0582, 0583).
func TestAChangeIsSaidOneWayEverywhere(t *testing.T) {
	a, h := newApp(t)
	old, err := a.Store.Create("activity", map[string]any{
		"actor": "assistant", "action": "set", "target": "ui.pace", "detail": "calm",
		"summary": "Assistant changed Pace — how changes arrive to Calmly",
	})
	if err != nil {
		t.Fatal(err)
	}
	undo, _ := a.Store.Create("activity", map[string]any{
		"actor": "human", "action": "set", "target": "ui.pace", "detail": "quick", "undoes": old.ID,
		"summary": "You undid: Assistant changed Pace — how changes arrive to Calmly",
	})
	want := "Assistant changed pace to Calm"
	// Pages work the words out, so they say them before anything else runs.
	surfaces := map[string]string{
		"the log":          get(t, h, "/activity").Body.String(),
		"the entry's page": get(t, h, "/t/activity/"+old.ID).Body.String(),
		"the undo's page":  get(t, h, "/t/activity/"+undo.ID).Body.String(),
	}
	// The stored words are brought up to date when a workspace opens, so
	// what hands out the fields as they are (the API, MCP) agrees too.
	if n := chat.Resay(a.Store); n != 2 {
		t.Errorf("both old entries are said again, got %d", n)
	}
	surfaces["the API's record"] = get(t, h, "/api/activity/"+undo.ID).Body.String()
	surfaces["the API's list"] = get(t, h, "/api/activity").Body.String()
	for where, body := range surfaces {
		if strings.Contains(body, "how changes arrive") || strings.Contains(body, "Calmly") {
			t.Errorf("%s still says the stored words", where)
		}
	}
	for _, where := range []string{"the entry's page", "the API's record"} {
		body := surfaces[where]
		if where == "the API's record" {
			var rec map[string]any
			json.Unmarshal([]byte(body), &rec)
			body, _ = rec["title"].(string)
			want = "You undid: " + want
		}
		if !strings.Contains(body, want) {
			t.Errorf("%s says %q:\n%.600s", where, want, body)
		}
	}
}

// The crew's findings 0586, 0587, 0591 and 0594, on the entries that
// raised them: a type named as stored (test_type), entries alike in one
// second, and an update's page that read out its stored fields.
func TestTheLogSaysEverythingInWords(t *testing.T) {
	a, h := newApp(t)
	a.Store.Create("activity", map[string]any{"actor": "system", "action": "added", "target": "type", "detail": "test_type", "summary": "System added type test_type"})
	a.Store.Create("activity", map[string]any{"actor": "system", "action": "added", "target": "field", "detail": "test_field on test_type", "summary": "System added field test_field on test_type"})
	for range 3 {
		chat.Record(a.Store, "system", chat.Change{Action: "failed", Detail: "claude: exit status 1"})
	}
	var made map[string]any
	decode(t, postJSON(t, h, "POST", "/api/note", map[string]any{"title": "Plan"}), &made)
	postJSON(t, h, "PATCH", "/api/note/"+made["id"].(string), map[string]any{"title": "Plan B"})

	list := get(t, h, "/t/activity").Body.String()
	for _, raw := range []string{"test_type", "test_field", "(id "} {
		if strings.Contains(list, raw) {
			t.Errorf("the log's list says %q", raw)
		}
	}
	if !strings.Contains(list, "added type Test Type") || !strings.Contains(list, "first of 3") {
		t.Errorf("types in words, and alike entries by which came first")
	}
	entries, _ := a.Store.List("activity", store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 1})
	page := get(t, h, "/t/activity/"+entries[0].ID).Body.String()
	for _, raw := range []string{"<dt>Action</dt>", "<dt>Target id</dt>", "{", "<dt>Before</dt>"} {
		if strings.Contains(page[strings.Index(page, "<main"):], raw) {
			t.Errorf("an update's page shows %q", raw)
		}
	}
	if !strings.Contains(page, "Title was") || !strings.Contains(page, "Plan B") {
		t.Errorf("it says what the note is and what it was")
	}
}
