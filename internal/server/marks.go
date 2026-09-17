package server

import (
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// markOf is the one press a record offers wherever it is shown: its first
// yes-or-no field, flipped. A task has done, a note has pinned; a type
// with no such field offers nothing. The button says what it will do and,
// for a screen reader, to which record.
func markOf(t *schema.Type, rec *store.Record) (map[string]any, bool) {
	for _, f := range t.Fields {
		if f.Type != "bool" {
			continue
		}
		on, _ := rec.Fields[f.Name].(bool)
		word := strings.ToLower(label(f.Name))
		props := map[string]any{"type": t.Name, "record": rec.ID, "field": f.Name, "context": titleOf(t, rec)}
		if on {
			props["value"], props["label"] = "false", "Mark as not "+word
		} else {
			props["value"], props["label"] = "true", "Mark as "+word
		}
		return props, true
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
