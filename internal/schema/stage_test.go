package schema

import (
	"testing"
	"time"
)

func taskType(t *testing.T) *Type {
	t.Helper()
	typ, err := Parse([]byte(`name: task
fields:
  title:
    type: string
  done:
    type: bool
    default: false
  status:
    type: enum
    values: [todo, doing, done]
    default: todo
  due:
    type: datetime
  repeat:
    type: repeat
`))
	if err != nil {
		t.Fatal(err)
	}
	return typ
}

// A task's tick and status say one thing, whichever was changed.
func TestATasksTickAndStatusKeepInStep(t *testing.T) {
	t.Parallel()
	typ := taskType(t)
	if typ.Stage() != "status" {
		t.Fatalf("a task's stage is its status, got %q", typ.Stage())
	}
	cases := []struct {
		name          string
		before, clean map[string]any
		done          bool
		status        string
	}{
		{"moved to Done", map[string]any{"done": false, "status": "todo"}, map[string]any{"done": false, "status": "done"}, true, "done"},
		{"moved out of Done", map[string]any{"done": true, "status": "done"}, map[string]any{"done": true, "status": "doing"}, false, "doing"},
		{"ticked", map[string]any{"done": false, "status": "doing"}, map[string]any{"done": true, "status": "doing"}, true, "done"},
		{"unticked", map[string]any{"done": true, "status": "done"}, map[string]any{"done": false, "status": "done"}, false, "todo"},
		{"both at once, contradicting: the tick", map[string]any{"done": false, "status": "todo"}, map[string]any{"done": true, "status": "doing"}, true, "done"},
		{"written whole: the tick", nil, map[string]any{"done": true, "status": "todo"}, true, "done"},
		{"in step already", map[string]any{"done": false, "status": "todo"}, map[string]any{"done": false, "status": "doing"}, false, "doing"},
	}
	for _, c := range cases {
		typ.KeepInStep(c.before, c.clean)
		if c.clean["done"] != c.done || c.clean["status"] != c.status {
			t.Errorf("%s: got done %v, status %v", c.name, c.clean["done"], c.clean["status"])
		}
	}
}

// Given only one of the two, a record gets the other from it.
func TestATaskGivenOneGetsTheOther(t *testing.T) {
	t.Parallel()
	typ := taskType(t)
	for _, c := range []struct {
		in     map[string]any
		done   bool
		status string
	}{
		{map[string]any{"title": "a", "done": true}, true, "done"},
		{map[string]any{"title": "b"}, false, "todo"},
		{map[string]any{"title": "c", "status": "done"}, true, "done"},
		{map[string]any{"title": "d", "status": "doing"}, false, "doing"},
	} {
		out, err := typ.Normalize(c.in)
		if err != nil {
			t.Fatal(err)
		}
		if out["done"] != c.done || out["status"] != c.status {
			t.Errorf("%v: got done %v, status %v", c.in["title"], out["done"], out["status"])
		}
	}
	read := map[string]any{"done": true, "status": "todo"}
	typ.Unstored(read, map[string]bool{"status": true})
	if read["status"] != "done" {
		t.Error("a task stored before its status reads Done from its tick")
	}
}

// Finished, a repeating task is due again and To do.
func TestARepeatingTaskFinishedIsToDoAgain(t *testing.T) {
	t.Parallel()
	typ := taskType(t)
	before := map[string]any{"done": false, "status": "doing", "due": "2026-10-06T00:00:00Z", "repeat": "FREQ=WEEKLY"}
	clean := map[string]any{"done": false, "status": "done", "due": "2026-10-06T00:00:00Z", "repeat": "FREQ=WEEKLY"}
	typ.KeepInStep(before, clean)
	if !typ.Advance(before, clean, time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)) {
		t.Fatal("moved to Done, it repeats")
	}
	if clean["done"] != false || clean["status"] != "todo" || clean["due"] == before["due"] {
		t.Errorf("due again and To do, got %v", clean)
	}
	if !typ.Advanced(map[string]any{"status": "done"}, clean) {
		t.Error("a move to Done that moved it on says so")
	}
	if typ.SaysMoreThanTick("todo") || typ.SaysMoreThanTick("done") || !typ.SaysMoreThanTick("doing") {
		t.Error("only Doing says more than the tick")
	}
}
