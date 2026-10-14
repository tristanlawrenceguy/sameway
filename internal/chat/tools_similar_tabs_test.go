package chat_test

import (
	"strings"
	"testing"
)

// A block added over the same records as one on another tab says which
// tab has it, by its name, and what to do if the person meant that one;
// a list of other records says nothing.
func TestAddingWhatAnotherTabShowsSaysSo(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	run(t, svc, "create_canvas", map[string]any{"name": "Work"})
	work := svc.Canvases()[1].ID
	run(t, svc, "add_component", map[string]any{"component": "collection", "props": map[string]any{"type": "task", "label": "Work tasks"}, "canvas": work})
	text := run(t, svc, "add_component", map[string]any{"component": "collection", "props": map[string]any{"type": "task", "label": "Tasks"}, "canvas": ""})
	if !strings.Contains(text, "Another tab shows the same records as a collection: Work has ") || !strings.Contains(text, `("Work tasks")`) {
		t.Errorf("the add says Work has a list of tasks:\n%s", text)
	}
	if strings.Contains(text, "This tab already had") {
		t.Errorf("nothing like it is on this tab:\n%s", text)
	}
	text = run(t, svc, "add_component", map[string]any{"component": "collection", "props": map[string]any{"type": "note", "label": "Notes"}, "canvas": ""})
	if strings.Contains(text, "Another tab shows") {
		t.Errorf("a list of notes is not like a list of tasks:\n%s", text)
	}
}
