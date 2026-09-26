package store_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

func twoCopies(t *testing.T) (*store.Store, *store.Store) {
	t.Helper()
	types, err := schema.Load(filepath.Join("..", "..", "examples", "workspaces", "starter", "schema"))
	if err != nil {
		t.Fatal(err)
	}
	open := func() *store.Store {
		s, err := store.Open(":memory:", types)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { s.Close() })
		return s
	}
	return open(), open()
}

// sync takes from b what a has not seen, and the other way.
func sync(t *testing.T, a, b *store.Store) {
	t.Helper()
	for _, pair := range [][2]*store.Store{{a, b}, {b, a}} {
		seen, err := pair[0].Seen()
		if err != nil {
			t.Fatal(err)
		}
		stamps, err := pair[1].Since(seen)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pair[0].Apply(stamps); err != nil {
			t.Fatal(err)
		}
	}
}

func fields(t *testing.T, s *store.Store, typ, id string) map[string]any {
	t.Helper()
	r, err := s.Get(typ, id)
	if err != nil {
		return nil
	}
	return r.Fields
}

// A note made on one computer is on the other after a sync, under the same
// id, with the same words.
func TestARecordMadeOnOneHostReachesTheOther(t *testing.T) {
	a, b := twoCopies(t)
	n, err := a.Create("note", map[string]any{"title": "Shopping", "body": "milk"})
	if err != nil {
		t.Fatal(err)
	}
	sync(t, a, b)
	got := fields(t, b, "note", n.ID)
	if got == nil || got["title"] != "Shopping" || got["body"] != "milk" {
		t.Fatalf("the note should be on the other host: %v", got)
	}
	if again, _ := b.Since(map[string]string{}); len(again) == 0 {
		t.Error("what b took is b's to pass on to a third copy")
	}
}

// Two people changing different fields of the same note at the same time
// both keep their change; the same field goes to the later write, and
// both copies agree which that is.
func TestConcurrentEditsMergeByField(t *testing.T) {
	a, b := twoCopies(t)
	n, _ := a.Create("note", map[string]any{"title": "Plan", "body": "draft"})
	sync(t, a, b)

	a.Update("note", n.ID, map[string]any{"title": "Plan for Monday"})
	b.Update("note", n.ID, map[string]any{"body": "call the plumber"})
	sync(t, a, b)
	for name, s := range map[string]*store.Store{"a": a, "b": b} {
		got := fields(t, s, "note", n.ID)
		if got["title"] != "Plan for Monday" || got["body"] != "call the plumber" {
			t.Errorf("%s should hold both changes: %v", name, got)
		}
	}

	a.Update("note", n.ID, map[string]any{"title": "From a"})
	b.Update("note", n.ID, map[string]any{"title": "From b"})
	sync(t, a, b)
	if fa, fb := fields(t, a, "note", n.ID), fields(t, b, "note", n.ID); !reflect.DeepEqual(fa, fb) {
		t.Errorf("both copies must agree on the same field: a %v, b %v", fa, fb)
	}
}

// A deletion reaches the other copy, and an undo that puts the record back
// afterwards wins over it everywhere.
func TestADeletionAndItsUndoReachEveryCopy(t *testing.T) {
	a, b := twoCopies(t)
	n, _ := a.Create("note", map[string]any{"title": "Old"})
	sync(t, a, b)
	if err := b.Delete("note", n.ID); err != nil {
		t.Fatal(err)
	}
	sync(t, a, b)
	if fields(t, a, "note", n.ID) != nil {
		t.Fatal("a deletion on b removes it on a")
	}
	if _, err := a.Restore("note", n.ID, map[string]any{"title": "Old"}); err != nil {
		t.Fatal(err)
	}
	sync(t, a, b)
	if got := fields(t, b, "note", n.ID); got == nil || got["title"] != "Old" {
		t.Errorf("the undo puts it back on b too: %v", got)
	}
}

// Stamps arriving twice, or out of order, change nothing: the copies end
// the same whichever way round they talk.
func TestSyncingAgainChangesNothing(t *testing.T) {
	a, b := twoCopies(t)
	n, _ := a.Create("note", map[string]any{"title": "One"})
	a.Update("note", n.ID, map[string]any{"title": "Two"})
	all, _ := a.Since(map[string]string{})
	for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
		all[i], all[j] = all[j], all[i]
	}
	b.Apply(all)
	b.Apply(all)
	if got := fields(t, b, "note", n.ID); got["title"] != "Two" {
		t.Errorf("out of order and twice, the latest still wins: %v", got)
	}
	seen, _ := b.Seen()
	if more, _ := a.Since(seen); len(more) != 0 {
		t.Errorf("b has everything a has: %d more", len(more))
	}
}

// What stays on one computer (a chat) is never stamped, so it never
// leaves; records from before hosting was shared are stamped by Seed.
func TestLocalTypesStayAndOldRecordsAreSeeded(t *testing.T) {
	a, b := twoCopies(t)
	a.Local = map[string]bool{"message": true}
	a.Create("message", map[string]any{"role": "user", "content": "private"})
	stamps, _ := a.Since(map[string]string{})
	for _, st := range stamps {
		if st.Type == "message" {
			t.Fatal("a chat message must not be stamped for sync")
		}
	}

	b.Local = map[string]bool{"note": true}
	old, _ := b.Create("note", map[string]any{"title": "From before"})
	b.Local = nil
	if err := b.Seed(); err != nil {
		t.Fatal(err)
	}
	sync(t, a, b)
	if fields(t, a, "note", old.ID) == nil {
		t.Error("a record from before sharing reaches the other copy once seeded")
	}
}

// Single records of a shared type can stay: a log entry saying what was
// said to the assistant is never stamped, while the one saying a note was
// deleted is.
func TestSomeRecordsOfASharedTypeStay(t *testing.T) {
	a, _ := twoCopies(t)
	a.LocalRecord = func(typeName string, f map[string]any) bool { return typeName == "activity" && f["action"] == "said" }
	a.Create("activity", map[string]any{"actor": "human", "action": "said", "detail": "private"})
	a.Create("activity", map[string]any{"actor": "human", "action": "deleted", "target": "note"})
	stamps, _ := a.Since(map[string]string{})
	said, deleted := false, false
	for _, st := range stamps {
		if st.Field == "action" && string(st.Value) == `"said"` {
			said = true
		}
		if st.Field == "action" && string(st.Value) == `"deleted"` {
			deleted = true
		}
	}
	if said || !deleted {
		t.Errorf("said stays (%v), deleted travels (%v)", said, deleted)
	}
}

// Two people rewriting a note's body at once, on two computers, both keep
// their words: the later version stands everywhere, and the other is kept
// as one clash, the same on both. Writing one after the other is no clash.
func TestTextWrittenAtOnceKeepsBothVersions(t *testing.T) {
	a, b := twoCopies(t)
	n, _ := a.Create("note", map[string]any{"title": "Plan", "body": "first draft"})
	sync(t, a, b)

	a.Update("note", n.ID, map[string]any{"body": "draft from a"})
	b.Update("note", n.ID, map[string]any{"body": "draft from b"})
	sync(t, a, b)
	sync(t, a, b)
	fa, fb := fields(t, a, "note", n.ID), fields(t, b, "note", n.ID)
	if fa["body"] != fb["body"] {
		t.Fatalf("both copies agree on the body: %v / %v", fa["body"], fb["body"])
	}
	lost := "draft from a"
	if fa["body"] == lost {
		lost = "draft from b"
	}
	for name, s := range map[string]*store.Store{"a": a, "b": b} {
		clashes, _ := s.List("clash", store.ListOptions{})
		if len(clashes) != 1 || clashes[0].Fields["text"] != lost || clashes[0].Fields["target_id"] != n.ID {
			t.Errorf("%s keeps the replaced version once: %v", name, clashes)
		}
	}

	b.Update("note", n.ID, map[string]any{"body": "b, having read a"})
	sync(t, a, b)
	if clashes, _ := a.List("clash", store.ListOptions{}); len(clashes) != 1 {
		t.Errorf("an edit made over the other is no clash: %d", len(clashes))
	}
}
