package records_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// changeTitles logs one change that retitles each note, as an agent would.
func changeTitles(t *testing.T, b *records.Book, to map[string]string) string {
	t.Helper()
	var ops []records.Op
	for id, title := range to {
		ops = append(ops, records.Op{Type: "note", ID: id, After: map[string]any{"title": title}})
	}
	done, err := b.Apply(ops...)
	if err != nil {
		t.Fatal(err)
	}
	return records.Record(b.Store, "assistant", records.Change{Action: "updated", Component: "note", Detail: "2 notes", Ops: done})
}

// Undo takes back the change it names and nothing after it: a field the
// person changed since is kept, and said; one they did not touch goes back.
func TestUndoKeepsWhatWasChangedSince(t *testing.T) {
	t.Parallel()
	b := newBook(t)
	a, _ := b.Store.Create("note", map[string]any{"title": "Seeds"})
	c, _ := b.Store.Create("note", map[string]any{"title": "Beds"})
	entry := changeTitles(t, b, map[string]string{a.ID: "Seeds to buy", c.ID: "Beds to dig"})
	// Since then the person retitled one and wrote in the other.
	b.Store.Update("note", a.ID, map[string]any{"title": "Seeds, by Friday"})
	b.Store.Update("note", c.ID, map[string]any{"body": "The two by the shed"})

	said, rev, err := b.Undo(entry)
	if err != nil {
		t.Fatal(err)
	}
	if got := title(b, "note", a.ID); got != "Seeds, by Friday" {
		t.Errorf("the title typed since is kept: %v", got)
	}
	if got := title(b, "note", c.ID); got != "Beds" {
		t.Errorf("the title not touched since goes back: %v", got)
	}
	if rec, _ := b.Store.Get("note", c.ID); rec.Fields["body"] != "The two by the shed" {
		t.Errorf("what was written since, which the change did not touch, stays: %v", rec.Fields["body"])
	}
	if !strings.Contains(said, "Kept as changed since: title of Seeds, by Friday.") || len(rev.Kept) != 1 {
		t.Errorf("what was kept is said: %q %v", said, rev.Kept)
	}
}

// A change whose every field was changed again since has nothing left to
// take back: it is not offered, and undoing it says why.
func TestAChangeOverwrittenSinceIsNotUndone(t *testing.T) {
	t.Parallel()
	b := newBook(t)
	a, _ := b.Store.Create("note", map[string]any{"title": "Seeds"})
	done, _ := b.Apply(records.Op{Type: "note", ID: a.ID, After: map[string]any{"title": "Seeds to buy"}})
	entry := records.Record(b.Store, "assistant", records.Change{Action: "updated", Component: "note", ID: a.ID, Detail: "Seeds to buy", Ops: done})
	b.Store.Update("note", a.ID, map[string]any{"title": "Seeds, by Friday"})

	e, _ := b.Store.Get(records.ActivityType, entry)
	if b.Undoable(e) {
		t.Error("not offered: the title it set has been changed since")
	}
	if _, _, err := b.Undo(entry); err == nil || !strings.Contains(err.Error(), "changed since: title of Seeds, by Friday") {
		t.Errorf("says why: %v", err)
	}
	if got := title(b, "note", a.ID); got != "Seeds, by Friday" {
		t.Errorf("nothing changed: %v", got)
	}
}

// Several undone at once are all or none: when one cannot be, those undone
// before it are put back, and the error names it.
func TestUndoEachIsAllOrNone(t *testing.T) {
	t.Parallel()
	b := newBook(t)
	a, _ := b.Store.Create("note", map[string]any{"title": "Seeds"})
	first := changeTitles(t, b, map[string]string{a.ID: "Seeds to buy"})
	made, _ := b.Apply(records.Op{Type: "note", After: map[string]any{"title": "Tools"}})
	second := records.Record(b.Store, "assistant", records.Change{Action: "created", Component: "note", ID: made[0].ID, Ops: made})
	b.Store.Delete("note", made[0].ID) // gone since: its making cannot be undone

	if _, _, err := b.UndoEach([]string{first, second}); err == nil || !strings.Contains(err.Error(), "nothing was undone: the second of 2") {
		t.Fatalf("refused, naming the one: %v", err)
	}
	if got := title(b, "note", a.ID); got != "Seeds to buy" {
		t.Errorf("the first is put back as it was: %v", got)
	}
	said, changes, err := b.UndoEach([]string{first})
	if err != nil || len(changes) != 1 || !strings.HasPrefix(said[0], "undone:") || title(b, "note", a.ID) != "Seeds" {
		t.Errorf("one that can be is undone: %v %v", said, err)
	}
}
