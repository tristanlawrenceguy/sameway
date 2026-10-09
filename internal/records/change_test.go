package records_test

import (
	"path/filepath"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// newBook is a starter workspace in memory, with settings kept in a map.
func newBook(t *testing.T) *records.Book {
	t.Helper()
	types, err := schema.Load(filepath.Join("..", "..", "examples", "workspaces", "starter", "schema"))
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(":memory:", types)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	settings := map[string]string{}
	return &records.Book{Store: st,
		Setting:    func(k string) string { return settings[k] },
		SetSetting: func(k, v string) error { settings[k] = v; return nil }}
}

func newest(t *testing.T, b *records.Book) *store.Record {
	t.Helper()
	recs := b.Recent(1)
	if len(recs) == 0 {
		t.Fatal("nothing in the log")
	}
	return recs[0]
}

func title(b *records.Book, typ, id string) any {
	rec, err := b.Store.Get(typ, id)
	if err != nil {
		return nil
	}
	return rec.Fields["title"]
}

// A change is written all or none: when one op cannot be, those before it
// are put back, so nothing is left half made.
func TestApplyIsAllOrNone(t *testing.T) {
	t.Parallel()
	b := newBook(t)
	kept, _ := b.Store.Create("note", map[string]any{"title": "Kept"})
	_, err := b.Apply(
		records.Op{Type: "note", After: map[string]any{"title": "Made"}},
		records.Op{Type: "note", ID: kept.ID, After: map[string]any{"title": "Changed"}},
		records.Op{Type: "note", ID: kept.ID},
		records.Op{Type: "no-such-type", After: map[string]any{"title": "x"}},
	)
	if err == nil {
		t.Fatal("an op on a type that is not there fails the change")
	}
	if n, _ := b.Store.Count("note"); n != 1 {
		t.Errorf("the note it made is taken away again: %d notes", n)
	}
	if got := title(b, "note", kept.ID); got != "Kept" {
		t.Errorf("the note it changed and removed is back as it was, got %v", got)
	}
	done, err := b.Apply(records.Op{Type: "note", After: map[string]any{"title": "Made"}})
	if err != nil || len(done) != 1 || done[0].ID == "" || done[0].Before != nil || done[0].After["title"] != "Made" {
		t.Errorf("a change says each op as made, with the new id: %+v %v", done, err)
	}
}

// A batch that keeps its ops is undone as the same ops the other way, and
// the undo undone again, whatever the batch was called.
func TestABatchOfOpsIsUndoneAndRedone(t *testing.T) {
	t.Parallel()
	b := newBook(t)
	changed, _ := b.Store.Create("note", map[string]any{"title": "Before"})
	gone, _ := b.Store.Create("note", map[string]any{"title": "Gone"})
	done, err := b.Apply(
		records.Op{Type: "note", After: map[string]any{"title": "Made"}},
		records.Op{Type: "note", ID: changed.ID, After: map[string]any{"title": "After"}},
		records.Op{Type: "note", ID: gone.ID},
	)
	if err != nil {
		t.Fatal(err)
	}
	made := done[0].ID
	records.Record(b.Store, "human", records.Change{Action: "imported", Component: "note", Detail: "3 notes", Ops: done})
	said := records.Sentence(b.Store, newest(t, b).Fields)
	if !b.Undoable(newest(t, b)) {
		t.Fatal("a batch with ops can be undone")
	}
	if err := b.UndoAs("human", ""); err != nil {
		t.Fatal(err)
	}
	if title(b, "note", made) != nil || title(b, "note", changed.ID) != "Before" || title(b, "note", gone.ID) != "Gone" {
		t.Errorf("undone, each note is as it was: %v %v %v", title(b, "note", made), title(b, "note", changed.ID), title(b, "note", gone.ID))
	}
	if s := records.Sentence(b.Store, newest(t, b).Fields); s != "You undid: "+said {
		t.Errorf("the undo is said as before: %q", s)
	}
	if err := b.UndoAs("human", ""); err != nil {
		t.Fatal(err)
	}
	if title(b, "note", made) != "Made" || title(b, "note", changed.ID) != "After" || title(b, "note", gone.ID) != nil {
		t.Errorf("the undo undone, the batch is back: %v %v %v", title(b, "note", made), title(b, "note", changed.ID), title(b, "note", gone.ID))
	}
}

// An entry for one thing names it, and it must still be as the entry left
// it: an update to a note since deleted is not undone over the deletion.
func TestOneThingMustBeAsTheEntryLeftIt(t *testing.T) {
	t.Parallel()
	b := newBook(t)
	n, _ := b.Store.Create("note", map[string]any{"title": "One"})
	done, _ := b.Apply(records.Op{Type: "note", ID: n.ID, After: map[string]any{"title": "Two"}})
	records.Record(b.Store, "human", records.Change{Action: "updated", Component: "note", ID: n.ID, Detail: "Two", Ops: done})
	entry := newest(t, b)
	b.Store.Delete("note", n.ID)
	if b.Undoable(entry) {
		t.Error("the note is gone: its update cannot be undone")
	}
	b.Store.Restore("note", n.ID, map[string]any{"title": "Two"})
	if _, c, err := b.Undo(entry.ID); err != nil || c.Action != "updated" || title(b, "note", n.ID) != "One" {
		t.Errorf("back, the update is undone as an update: %+v %v", c, err)
	}

	// A setting goes back to what it was, which may be nothing.
	done, err := b.Apply(records.Op{Type: records.SettingOp, ID: "ui.pace", After: map[string]any{"value": "calm"}})
	if err != nil {
		t.Fatal(err)
	}
	records.Record(b.Store, "human", records.Change{Action: "set", Component: "ui.pace", Detail: "calm", Ops: done})
	if _, c, err := b.Undo(""); err != nil || c.Action != "set" || c.Component != "ui.pace" || b.Setting("ui.pace") != "" {
		t.Errorf("a setting is undone back to nothing: %+v %v %q", c, err, b.Setting("ui.pace"))
	}
}

// What a writer logs keeps its ops and no before of its own shape: made,
// changed and deleted through WriteAs, each undone by them.
func TestWritesKeepTheirOps(t *testing.T) {
	t.Parallel()
	b := newBook(t)
	who := records.Who{Actor: "human"}
	rec, _, err := records.WriteAs(b.Store, who, "created", "note", "", map[string]any{"title": "One"})
	if err != nil {
		t.Fatal(err)
	}
	records.WriteAs(b.Store, who, "updated", "note", rec.ID, map[string]any{"title": "Two"})
	records.WriteAs(b.Store, who, "deleted", "note", rec.ID, nil)
	entries := b.Recent(3)
	for _, e := range entries {
		ops := records.EntryOps(e)
		if len(ops) != 1 || ops[0].ID != rec.ID || e.Fields["before"] != nil {
			t.Errorf("%s keeps its op and no before: %+v %v", e.Fields["action"], ops, e.Fields["before"])
		}
	}
	for i, want := range []any{"Two", "One", nil} {
		if err := b.UndoAs("human", entries[i].ID); err != nil {
			t.Fatal(err)
		}
		if got := title(b, "note", rec.ID); got != want {
			t.Errorf("undone back to %v, got %v", want, got)
		}
	}
}

// A change logged is heard once, whole, by each listener, however many
// records it wrote; each record written is heard by the store's own.
func TestListenersHearEachChangeOnce(t *testing.T) {
	t.Parallel()
	b := newBook(t)
	var heard, also []records.Change
	records.Listen(b.Store, func(actor string, c records.Change) { heard = append(heard, c) })
	records.Listen(b.Store, func(actor string, c records.Change) { also = append(also, c) })
	writes := 0
	b.Store.Listen(func(t *schema.Type, was, now *store.Record) {
		if t.Name == "note" {
			writes++
		}
	})
	rec, entry, err := records.WriteAs(b.Store, records.Who{Actor: "human"}, "created", "note", "", map[string]any{"title": "One"})
	if err != nil {
		t.Fatal(err)
	}
	if len(heard) != 1 || heard[0].Activity != entry || len(heard[0].Ops) != 1 || heard[0].Ops[0].ID != rec.ID || len(also) != 1 {
		t.Fatalf("one change, heard once by each, with its entry and ops: %+v %d", heard, len(also))
	}
	done, _ := b.Apply(
		records.Op{Type: "note", After: map[string]any{"title": "Two"}},
		records.Op{Type: "note", After: map[string]any{"title": "Three"}},
		records.Op{Type: "note", ID: rec.ID},
	)
	records.Record(b.Store, "human", records.Change{Action: "imported", Component: "note", Detail: "3 notes", Ops: done})
	if len(heard) != 2 || len(heard[1].Ops) != 3 || len(also) != 2 {
		t.Errorf("a batch of three is one change, heard once: %d", len(heard))
	}
	if writes != 4 {
		t.Errorf("the store's own listener hears each record written: %d", writes)
	}
}
