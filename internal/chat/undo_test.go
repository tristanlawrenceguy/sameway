package chat_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// The activity log runs backwards: an added thing is removed, a removed
// thing comes back with everything it had, an update goes back to what it
// was, and undoing an undo puts it back again. One tool does all of it,
// because every entry carries what the thing was before.
func TestUndoRunsTheLogBackwards(t *testing.T) {
	svc := newFullService(t)
	must := func(name string, args map[string]any) string {
		t.Helper()
		raw, _ := json.Marshal(args)
		text, isErr := svc.Call(name, raw)
		if isErr {
			t.Fatalf("%s: %s", name, text)
		}
		return text
	}
	blocks := func() []*store.Record {
		recs, _ := svc.Store.List(chat.BlockType, store.ListOptions{})
		return recs
	}
	newest := func() string {
		recs, _ := svc.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 1})
		s, _ := recs[0].Fields["summary"].(string)
		return s
	}

	// A block: added, undone, and the undo undone.
	must("add_component", map[string]any{"component": "heading", "props": map[string]any{"text": "Shopping"}})
	id := blocks()[0].ID
	must("undo_change", map[string]any{})
	if len(blocks()) != 0 {
		t.Fatal("undoing an addition should remove the block")
	}
	if got := newest(); got != "Assistant undid: Assistant added heading Shopping" {
		t.Errorf("the undo should say what it undid, got %q", got)
	}
	must("undo_change", map[string]any{})
	if b := blocks(); len(b) != 1 || b[0].ID != id || b[0].Fields["props"].(map[string]any)["text"] != "Shopping" {
		t.Fatalf("undoing the undo should put the same block back, got %v", b)
	}

	// An update goes back to what it was.
	must("update_component", map[string]any{"id": id, "props": map[string]any{"text": "Groceries"}})
	must("undo_change", map[string]any{})
	if b := blocks(); b[0].Fields["props"].(map[string]any)["text"] != "Shopping" {
		t.Errorf("undoing an update should restore the old props, got %v", b[0].Fields["props"])
	}

	// A record: created, undone, back.
	must("create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "Water the plants", "body": "Every Sunday."}})
	notes, _ := svc.Store.List("note", store.ListOptions{})
	must("undo_change", map[string]any{})
	if n, _ := svc.Store.Count("note"); n != 0 {
		t.Fatal("undoing a creation should delete the record")
	}
	must("undo_change", map[string]any{})
	if rec, err := svc.Store.Get("note", notes[0].ID); err != nil || rec.Fields["body"] != "Every Sunday." {
		t.Fatalf("undoing the deletion should bring the record back whole, got %v %v", rec, err)
	}

	// A tab comes back with the blocks that were on it.
	must("create_canvas", map[string]any{"name": "Garden"})
	var garden string
	for _, c := range svc.Canvases() {
		if c.Name == "Garden" {
			garden = c.ID
		}
	}
	must("add_component", map[string]any{"component": "heading", "props": map[string]any{"text": "Beans"}, "canvas": garden})
	must("remove_canvas", map[string]any{"id": garden})
	if len(blocks()) != 1 || svc.HasCanvas(garden) {
		t.Fatal("removing the tab should take its block with it")
	}
	must("undo_change", map[string]any{})
	if !svc.HasCanvas(garden) || len(chat.OnCanvas(blocks(), garden)) != 1 {
		t.Fatal("undoing the removal should put the tab and its block back")
	}

	// Clearing the canvas is undone whole.
	must("clear_canvas", map[string]any{})
	if len(chat.OnCanvas(blocks(), "")) != 0 {
		t.Fatal("clear should empty Home")
	}
	must("undo_change", map[string]any{})
	if len(chat.OnCanvas(blocks(), "")) != 1 {
		t.Fatal("undoing a clear should put the blocks back")
	}

	// An entry that no longer applies says so instead of doing something else.
	entries, _ := svc.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at"})
	var added string
	for _, e := range entries {
		if e.Fields["summary"] == "Assistant added heading Shopping" {
			added = e.ID
		}
	}
	svc.Store.Delete(chat.BlockType, id)
	if text, isErr := svc.Call("undo_change", json.RawMessage(`{"id":"`+added+`"}`)); !isErr || !strings.Contains(text, "already gone") {
		t.Errorf("undoing an addition of something gone should be refused with the reason, got %q", text)
	}

	// A person undoes under their own name.
	if err := svc.UndoAs("human", ""); err != nil {
		t.Fatal(err)
	}
	if got := newest(); !strings.HasPrefix(got, "You undid: ") {
		t.Errorf("a person's undo is theirs, got %q", got)
	}
}
