package server

import (
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// markOf is the one yes-or-no fact a record offers to change wherever it
// is shown: its first bool field that is meaningful as a toggle, as a
// checkbox. A task has done, a note has pinned, an action has show; a type
// with no such field offers nothing. The checkbox is named with the field
// and, for a screen reader, the record.
func markOf(t *schema.Type, rec *store.Record) (map[string]any, bool) {
	// doneField handles the primary toggle: done/completed/complete/finished.
	if f := doneField(t); f != nil {
		on, _ := rec.Fields[f.Name].(bool)
		return map[string]any{"type": t.Name, "record": rec.ID, "field": f.Name, "label": capitalize(label(f.Name)), "checked": on, "context": titleOf(t, rec)}, true
	}
	// For types without a done field, look for a first toggleable bool like
	// pinned or show — these get checkboxes on detail pages but not in rows.
	for _, f := range t.Shown() {
		if f.Type != "bool" {
			continue
		}
		switch f.Name {
		case "pinned", "show":
			on, _ := rec.Fields[f.Name].(bool)
			return map[string]any{"type": t.Name, "record": rec.ID, "field": f.Name, "label": capitalize(label(f.Name)), "checked": on, "context": titleOf(t, rec)}, true
		}
	}
	return nil, false
}

// doneField is the yes-or-no a thing is finished by, such as a task's done:
// the one worth a box at the front of its row. Pinned or archived is a
// setting, not something finished, so it gets no box that reads as one.
func doneField(t *schema.Type) *schema.Field {
	for _, f := range t.Shown() {
		if f.Type != "bool" {
			continue
		}
		switch f.Name {
		case "done", "completed", "complete", "finished":
			return &f
		}
	}
	return nil
}

// markActions is the mark as the actions list a component's item takes.
// Only primary toggles (done/completed/complete/finished) get checkboxes in
// rows; secondary settings like pinned or show are shown as badges instead.
func markActions(t *schema.Type, rec *store.Record) []any {
	if doneField(t) == nil {
		return nil
	}
	props, ok := markOf(t, rec)
	if !ok {
		return nil
	}
	return []any{map[string]any{"component": "mark", "props": props}}
}
