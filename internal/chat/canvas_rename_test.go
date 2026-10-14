package chat_test

import (
	"strings"
	"testing"
)

// A tab can be given another name, which the tabs say from then on; one
// Undo gives it back its old name, Home keeps its own, and two tabs are
// never called the same.
func TestATabCanBeRenamed(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	run(t, svc, "create_canvas", map[string]any{"name": "Test"})
	run(t, svc, "create_canvas", map[string]any{"name": "Work"})
	id := svc.Canvases()[1].ID
	text := run(t, svc, "rename_canvas", map[string]any{"id": id, "name": "Garden"})
	if !strings.Contains(text, `renamed tab "Test" to "Garden"`) || svc.Canvases()[1].Name != "Garden" {
		t.Fatalf("the tab is called Garden now: %s %+v", text, svc.Canvases())
	}
	if _, isErr := svc.Call("rename_canvas", []byte(`{"id":"`+id+`","name":"work"}`)); !isErr {
		t.Error("two tabs are never called the same")
	}
	if _, isErr := svc.Call("rename_canvas", []byte(`{"id":"","name":"Start"}`)); !isErr {
		t.Error("Home keeps its name")
	}
	made := entries(t, svc, "updated")
	if len(made) != 1 {
		t.Fatalf("one entry in the log, got %d", len(made))
	}
	if _, _, err := svc.Undo(made[0].ID); err != nil {
		t.Fatal(err)
	}
	if got := svc.Canvases()[1].Name; got != "Test" {
		t.Errorf("one Undo gives it its old name: %q", got)
	}
}
