package server_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

type changesOut struct {
	Changes []map[string]any `json:"changes"`
	Cursor  string           `json:"cursor"`
}

// An agent follows what changes as a page does: from a cursor, oldest
// first, waiting when asked until there is something.
func TestAnAgentFollowsChanges(t *testing.T) {
	a, h := newApp(t)
	read := func(path string) changesOut {
		t.Helper()
		var out changesOut
		if err := json.Unmarshal(get(t, h, path).Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	var made map[string]any
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Seeds"}), &made)
	start := read("/api/changes")
	if len(start.Changes) == 0 || start.Changes[len(start.Changes)-1]["id"] != made["id"] || start.Changes[len(start.Changes)-1]["said"] == nil {
		t.Fatalf("the newest changes, the owner's with their words: %+v", start.Changes)
	}
	postJSON(t, h, http.MethodPatch, "/api/note/"+made["id"].(string), map[string]any{"title": "Seeds to order"})
	next := read("/api/changes?since=" + start.Cursor)
	if len(next.Changes) != 1 || next.Changes[0]["action"] != "updated" || next.Changes[0]["title"] != "Seeds to order" || next.Changes[0]["version"] == nil {
		t.Fatalf("only what changed since the cursor: %+v", next.Changes)
	}
	if again := read("/api/changes?since=" + next.Cursor); len(again.Changes) != 0 {
		t.Errorf("nothing new, nothing said: %+v", again.Changes)
	}

	go func() {
		time.Sleep(400 * time.Millisecond)
		a.Store.Create("note", map[string]any{"title": "Later"})
		records.Record(a.Store, "human", records.Change{Action: "created", Component: "note", Detail: "Later"})
	}()
	began := time.Now()
	waited := read("/api/changes?wait=10&since=" + next.Cursor)
	if len(waited.Changes) != 1 || time.Since(began) > 5*time.Second {
		t.Errorf("a wait ends when something changes: %d in %s", len(waited.Changes), time.Since(began))
	}

	// Someone let in reads the changes to what they may read, not who.
	a.Chat.SetSetting = func(string, string) error { return nil }
	records.Record(a.Store, "human", records.Change{Action: "set", Component: "ui.pace", Detail: "quick"})
	editor := records.Visitor{Name: "Bob", Login: "bob@example.com", Access: records.Edit}
	var theirs changesOut
	json.Unmarshal(as(t, h, editor, http.MethodGet, "/api/changes?since="+start.Cursor, "", "").Body.Bytes(), &theirs)
	for _, c := range theirs.Changes {
		if c["type"] != "note" || c["said"] != nil || c["undo"] != nil {
			t.Errorf("an editor reads changes to notes, without who or undo: %+v", c)
		}
	}
	if len(theirs.Changes) == 0 {
		t.Error("an editor follows the notes too")
	}
}
