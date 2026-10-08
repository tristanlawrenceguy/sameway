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
func (s *Server) markOf(t *schema.Type, rec *store.Record) (map[string]any, bool) {
	// The primary toggle, done or the like (schema DoneField).
	if f := t.DoneField(); f != nil {
		on, _ := rec.Fields[f.Name].(bool)
		return map[string]any{"type": t.Name, "record": rec.ID, "field": f.Name, "label": f.Display(), "checked": on, "context": s.title(t, rec)}, true
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
			return map[string]any{"type": t.Name, "record": rec.ID, "field": f.Name, "label": f.Display(), "checked": on, "context": s.title(t, rec)}, true
		}
	}
	return nil, false
}

// markActions is the mark as the actions list a component's item takes.
// Only primary toggles (done/completed/complete/finished) get checkboxes in
// rows; secondary settings like pinned or show are shown as badges instead.
func (s *Server) markActions(t *schema.Type, rec *store.Record) []any {
	if t.DoneField() == nil {
		return nil
	}
	props, ok := s.markOf(t, rec)
	if !ok {
		return nil
	}
	return []any{map[string]any{"component": "mark", "props": props}}
}
