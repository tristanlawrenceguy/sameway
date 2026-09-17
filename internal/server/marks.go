package server

import (
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// markOf is the one yes-or-no fact a record offers to change wherever it
// is shown: its first bool field, as a checkbox. A task has done, a note
// has pinned; a type with no such field offers nothing. The checkbox is
// named with the field and, for a screen reader, the record.
func markOf(t *schema.Type, rec *store.Record) (map[string]any, bool) {
	for _, f := range t.Fields {
		if f.Type != "bool" {
			continue
		}
		on, _ := rec.Fields[f.Name].(bool)
		return map[string]any{"type": t.Name, "record": rec.ID, "field": f.Name, "label": capitalize(label(f.Name)), "checked": on, "context": titleOf(t, rec)}, true
	}
	return nil, false
}

// markActions is the mark as the actions list a component's item takes.
func markActions(t *schema.Type, rec *store.Record) []any {
	props, ok := markOf(t, rec)
	if !ok {
		return nil
	}
	return []any{map[string]any{"component": "mark", "props": props}}
}
