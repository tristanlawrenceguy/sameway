package chat_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Made again, a task says which open ones are called the same and when
// each is due, so the model can tell a duplicate from the same errand on
// another day; a done one is not counted.
func TestMadeAgainSaysTheOpenOnesAndTheirDays(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	open, _ := svc.Store.Create("task", map[string]any{"title": "Buy groceries", "due": "2026-10-16T00:00:00Z"})
	svc.Store.Create("task", map[string]any{"title": "Buy groceries", "done": true})
	text, _ := svc.Call("create_record", []byte(`{"type":"task","fields":{"title":"buy groceries"}}`))
	if !strings.Contains(text, "1 task called the same was there already, still open: "+open.ID+" (due ") || !strings.Contains(text, "keep_one") {
		t.Errorf("the open one is named with its day, and keep_one offered: %s", text)
	}
}

// keep_one keeps the one given and merges the others into it, as one
// change; with nothing alike it says so and changes nothing.
func TestKeepOneMergesTheOthers(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	keep, _ := svc.Store.Create("habit", map[string]any{"name": "Water", "target": 8.0})
	dup, _ := svc.Store.Create("habit", map[string]any{"name": "water"})
	entry, _ := svc.Store.Create("entry", map[string]any{"habit": dup.ID, "at": "2026-10-12T09:00:00Z"})
	text, isErr := svc.Call("keep_one", []byte(`{"type":"habit","id":"`+keep.ID+`"}`))
	if isErr || !strings.Contains(text, "merged 2 habits") {
		t.Fatalf("merged: %s", text)
	}
	if _, err := svc.Store.Get("habit", dup.ID); err == nil {
		t.Error("the other habit is gone")
	}
	if e, _ := svc.Store.Get("entry", entry.ID); e.Fields["habit"] != keep.ID {
		t.Errorf("its entries count for the kept one: %v", e.Fields["habit"])
	}
	habits, _ := svc.Store.List("habit", store.ListOptions{})
	if text, isErr := svc.Call("keep_one", []byte(`{"type":"habit","id":"`+habits[0].ID+`"}`)); !isErr || !strings.Contains(text, "nothing to merge") {
		t.Errorf("one left, there is nothing to merge: %s", text)
	}
}
