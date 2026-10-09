package records_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// old writes an entry the way the log kept it before it kept ops: a
// before in the shape of its kind, and no ops.
func old(t *testing.T, b *records.Book, c records.Change) *store.Record {
	t.Helper()
	id := records.Record(b.Store, "human", c)
	e, err := b.Store.Get(records.ActivityType, id)
	if err != nil {
		t.Fatal(err)
	}
	if ops, _ := e.Fields["ops"].([]any); len(ops) > 0 {
		t.Fatal("an old entry keeps no ops")
	}
	return e
}

func undo(t *testing.T, b *records.Book, e *store.Record, action string) records.Change {
	t.Helper()
	_, c, err := b.Undo(e.ID)
	if err != nil {
		t.Fatalf("%s: %v", e.Fields["action"], err)
	}
	if c.Action != action {
		t.Errorf("undoing %s is said as %q, not %q", e.Fields["action"], c.Action, action)
	}
	records.Record(b.Store, "human", c)
	return c
}

func there(b *records.Book, typ, id string) bool {
	_, err := b.Store.Get(typ, id)
	return err == nil
}

// Every kind of entry the log wrote before it kept ops is still undone as
// it was: a thing added, made, removed, deleted, updated, dismissed or put
// off, a tab removed with its blocks, a setting set, an amount logged, a
// canvas cleared and put back, a batch, and a chat deleted.
func TestOldEntriesAreStillUndone(t *testing.T) {
	b := newBook(t)
	st := b.Store

	blk, _ := st.Create(records.BlockType, map[string]any{"component": "heading", "props": map[string]any{"text": "Hi"}})
	undo(t, b, old(t, b, records.Change{Action: "added", Component: "heading", ID: blk.ID, Detail: "Hi"}), "removed")
	if there(b, records.BlockType, blk.ID) {
		t.Error("added: the block is taken away")
	}

	note, _ := st.Create("note", map[string]any{"title": "Made"})
	undo(t, b, old(t, b, records.Change{Action: "created", Component: "note", ID: note.ID, Detail: "Made"}), "deleted")
	if there(b, "note", note.ID) {
		t.Error("created: the note is deleted")
	}
	undo(t, b, old(t, b, records.Change{Action: "deleted", Component: "note", ID: note.ID, Detail: "Made", Before: note.Fields}), "created")
	if title(b, "note", note.ID) != "Made" {
		t.Error("deleted: the note is back whole")
	}

	tab, _ := st.Create(records.CanvasType, map[string]any{"name": "Garden"})
	on, _ := st.Create(records.BlockType, map[string]any{"component": "text", "canvas": tab.ID, "props": map[string]any{"content": "Seeds"}})
	st.Delete(records.BlockType, on.ID)
	st.Delete(records.CanvasType, tab.ID)
	before := map[string]any{"name": "Garden", "blocks": records.Keep([]*store.Record{on})}
	undo(t, b, old(t, b, records.Change{Action: "removed", Component: records.CanvasType, ID: tab.ID, Detail: "Garden", Before: before}), "added")
	if !there(b, records.CanvasType, tab.ID) || !there(b, records.BlockType, on.ID) {
		t.Error("removed: the tab is back with its block")
	}

	for _, verb := range []string{"updated", "done", "snoozed"} {
		was := map[string]any{}
		for k, v := range note.Fields {
			was[k] = v
		}
		st.Update("note", note.ID, map[string]any{"title": "Changed"})
		undo(t, b, old(t, b, records.Change{Action: verb, Component: "note", ID: note.ID, Detail: "Changed", Before: was}), "updated")
		if title(b, "note", note.ID) != "Made" {
			t.Errorf("%s: the note is as it was, got %v", verb, title(b, "note", note.ID))
		}
	}

	b.SetSetting("ui.pace", "calm")
	undo(t, b, old(t, b, records.Change{Action: "set", Component: "ui.pace", Detail: "calm", Before: map[string]any{"value": "brisk"}}), "set")
	if b.Setting("ui.pace") != "brisk" {
		t.Errorf("set: the setting is as it was, got %q", b.Setting("ui.pace"))
	}

	habit, _ := st.Create("habit", map[string]any{"name": "Water"})
	entry, _ := st.Create(records.EntryType, map[string]any{"habit": habit.ID, "at": "2026-10-08T09:00:00Z", "amount": 1})
	undo(t, b, old(t, b, records.Change{Action: "logged", Component: "habit", ID: habit.ID, Detail: "Water: 1", Before: map[string]any{"entry": entry.ID}}), "deleted")
	if there(b, records.EntryType, entry.ID) {
		t.Error("logged: the amount is taken away")
	}

	k, _ := st.Create(records.BlockType, map[string]any{"component": "text", "props": map[string]any{"content": "Kept"}})
	st.Delete(records.BlockType, k.ID)
	undo(t, b, old(t, b, records.Change{Action: "cleared", Detail: "1 blocks", Before: map[string]any{"blocks": records.Keep([]*store.Record{k})}}), "restored")
	if !there(b, records.BlockType, k.ID) {
		t.Error("cleared: the block is back")
	}
	undo(t, b, b.Recent(1)[0], "cleared")
	if there(b, records.BlockType, k.ID) {
		t.Error("restored: the block goes again")
	}

	for _, verb := range []string{"imported", "synced", "arranged", "wrote up", "organised", "suggested", "rescheduled"} {
		made, _ := st.Create("note", map[string]any{"title": verb})
		was := map[string]any{}
		for k, v := range note.Fields {
			was[k] = v
		}
		st.Update("note", note.ID, map[string]any{"title": "By " + verb})
		batch := records.Batch([]records.BatchItem{{Type: "note", ID: made.ID}, {Type: "note", ID: note.ID, Before: was}})
		undo(t, b, old(t, b, records.Change{Action: verb, Component: "note", Detail: verb, Before: batch}), "synced")
		if there(b, "note", made.ID) || title(b, "note", note.ID) != "Made" {
			t.Errorf("%s: the batch is put back", verb)
		}
	}
}
