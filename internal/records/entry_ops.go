package records

import (
	"encoding/json"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// How the log keeps a change's ops, and reads them back: from the ops a
// new entry keeps, or from the before an older entry kept in a shape of
// its own for its kind (undo_legacy.go).

// opsField is ops as the log's json field keeps them.
func opsField(ops []Op) any {
	raw, _ := json.Marshal(ops)
	var out []any
	json.Unmarshal(raw, &out)
	return out
}

// EntryOps is what an entry in the log changed, as ops: its own, or, for
// an entry from before the log kept ops, the batch its before keeps, each
// record with what it was (its after is not known).
func EntryOps(e *store.Record) []Op {
	if raw, ok := e.Fields["ops"].([]any); ok && len(raw) > 0 {
		b, _ := json.Marshal(raw)
		var ops []Op
		if json.Unmarshal(b, &ops) == nil {
			return ops
		}
	}
	before, _ := e.Fields["before"].(map[string]any)
	var ops []Op
	for _, c := range BatchOf(before) {
		ops = append(ops, Op{Type: c.Type, ID: c.ID, Before: c.Before})
	}
	return ops
}

// hasOps says whether an entry keeps its ops, as every new one does.
func hasOps(e *store.Record) bool {
	raw, _ := e.Fields["ops"].([]any)
	return len(raw) > 0
}

// Was is what the thing a change names was before it, from its ops: the
// fields of the record it changed or removed, nil for one it made.
func (c Change) Was() map[string]any {
	for _, op := range c.Ops {
		if op.ID == c.ID {
			return op.Before
		}
	}
	return c.Before
}

// Touched is how many records of a type a change wrote.
func (c Change) Touched(typ string) int {
	n := 0
	for _, op := range c.Ops {
		if op.Type == typ {
			n++
		}
	}
	return n
}

// EntryBefore is what the thing an entry names was before it: the before
// its op keeps (a setting's as {"value": ...}), or an older entry's own.
func EntryBefore(e *store.Record) map[string]any {
	if ops := EntryOps(e); hasOps(e) {
		if i := primary(e, ops); i >= 0 {
			return ops[i].Before
		}
		return nil
	}
	before, _ := e.Fields["before"].(map[string]any)
	return before
}
