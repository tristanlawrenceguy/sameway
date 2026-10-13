package records_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

func made(t *testing.T, b *records.Book, typ string, fields map[string]any) *store.Record {
	t.Helper()
	rec, err := b.Store.Create(typ, fields)
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

func ids(recs []*store.Record) map[string]bool {
	out := map[string]bool{}
	for _, r := range recs {
		out[r.ID] = true
	}
	return out
}

// Alike is every open record of a type called the same, in any case and
// spacing, whatever its day; a done one, another name and another type are
// not. Twins are only those nothing tells apart: no day, or the same day.
func TestRecordsAlikeAndTwins(t *testing.T) {
	t.Parallel()
	b := newBook(t)
	task, _ := b.Store.Types().Get("task")
	one := made(t, b, "task", map[string]any{"title": "Buy groceries", "due": "2026-10-16T00:00:00Z"})
	two := made(t, b, "task", map[string]any{"title": "buy  Groceries", "due": "2026-10-16T09:00:00Z"})
	other := made(t, b, "task", map[string]any{"title": "Buy groceries", "due": "2026-10-20T00:00:00Z"})
	made(t, b, "task", map[string]any{"title": "Buy groceries", "done": true})
	made(t, b, "task", map[string]any{"title": "Buy bread"})
	made(t, b, "note", map[string]any{"title": "Buy groceries"})

	if got := ids(records.Alike(b.Store, task, one)); len(got) != 2 || !got[two.ID] || !got[other.ID] {
		t.Errorf("alike: the open tasks called the same on any day, not itself, the done one, another name or a note: %v", got)
	}
	twins := records.Twins(b.Store, task)
	if len(twins) != 1 || len(twins[0]) != 2 || !ids(twins[0])[one.ID] || !ids(twins[0])[two.ID] {
		t.Errorf("twins: only the two due the same day: %v", twins)
	}
	if got := ids(records.TwinsOf(b.Store, task, one)); len(got) != 1 || !got[two.ID] {
		t.Errorf("twins of one is the other: %v", got)
	}
	if records.Twins(b.Store, mustType(t, b, "email")) != nil || records.Twins(b.Store, mustType(t, b, "entry")) != nil {
		t.Error("an email or an entry is never one thing twice")
	}
}

// KeepOne makes them one: the kept one takes what it leaves empty from the
// others, what pointed at the others points at it, the others go, and the
// change, logged, is undone in one go.
func TestKeepOneMergesAndUndoes(t *testing.T) {
	t.Parallel()
	b := newBook(t)
	project, _ := b.Store.Types().Get("project")
	keep := made(t, b, "project", map[string]any{"title": "Website redesign"})
	dup := made(t, b, "project", map[string]any{"title": "Website redesign", "notes": "Start with the home page."})
	task := made(t, b, "task", map[string]any{"title": "Draft the menu", "project": dup.ID})
	alarm := made(t, b, "reminder", map[string]any{"title": "Look at it", "at": "2026-10-20T09:00:00Z", "about": "/t/project/" + dup.ID})

	done, err := b.Apply(records.KeepOne(b.Store, project, keep, []*store.Record{dup})...)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.Store.Get("project", dup.ID); err == nil {
		t.Error("the other is gone")
	}
	if k, _ := b.Store.Get("project", keep.ID); k.Fields["notes"] != "Start with the home page." {
		t.Errorf("the kept one took what it lacked: %v", k.Fields["notes"])
	}
	if tk, _ := b.Store.Get("task", task.ID); tk.Fields["project"] != keep.ID {
		t.Errorf("the task's project is the kept one: %v", tk.Fields["project"])
	}
	if al, _ := b.Store.Get("reminder", alarm.ID); al.Fields["about"] != "/t/project/"+keep.ID {
		t.Errorf("the reminder is about the kept one: %v", al.Fields["about"])
	}

	records.Record(b.Store, "human", records.Change{Action: "merged", Component: "project", ID: keep.ID, Detail: "Website redesign, 2 into one", Ops: done})
	if _, _, err := b.Undo(newest(t, b).ID); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Store.Get("project", dup.ID); err != nil {
		t.Error("undone, the other is back")
	}
	if tk, _ := b.Store.Get("task", task.ID); tk.Fields["project"] != dup.ID {
		t.Errorf("undone, the task is in its project again: %v", tk.Fields["project"])
	}
}

func mustType(t *testing.T, b *records.Book, name string) *schema.Type {
	t.Helper()
	ty, ok := b.Store.Types().Get(name)
	if !ok {
		t.Fatalf("no type %s", name)
	}
	return ty
}
