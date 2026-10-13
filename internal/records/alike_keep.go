package records

import (
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// KeepOne is what makes records alike into one, as ops for one change
// that one Undo takes back: kept stays, what it leaves empty is filled from
// the others in their order, and whatever pointed at the others (a task's
// project, a habit's entries, a reminder's page) points at it; then the
// others go. A list keeps the newest of its Twins; the assistant keeps the
// one the person means.
func KeepOne(st *store.Store, t *schema.Type, kept *store.Record, others []*store.Record) []Op {
	if len(others) == 0 {
		return nil
	}
	gone := map[string]bool{}
	for _, r := range others {
		gone[r.ID] = true
	}
	var ops []Op
	if fill := filled(t, kept, others); len(fill) > 0 {
		ops = append(ops, Op{Type: t.Name, ID: kept.ID, After: fill})
	}
	ops = append(ops, repointed(st, t.Name, kept.ID, gone)...)
	for _, r := range others {
		ops = append(ops, Op{Type: t.Name, ID: r.ID})
	}
	return ops
}

// filled is the fields kept leaves empty that one of the others has.
func filled(t *schema.Type, kept *store.Record, others []*store.Record) map[string]any {
	out := map[string]any{}
	for _, f := range t.Fields {
		if f.ReadOnly || !empty(kept.Fields[f.Name]) {
			continue
		}
		for _, r := range others {
			if v := r.Fields[f.Name]; !empty(v) {
				out[f.Name] = v
				break
			}
		}
	}
	return out
}

func empty(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return x == ""
	case []any:
		return len(x) == 0
	}
	return false
}

// repointed is every record that points at one of gone, pointing at keep
// instead: by a ref field, a list of them, or a reminder's page.
func repointed(st *store.Store, typeName, keep string, gone map[string]bool) []Op {
	var ops []Op
	for _, u := range st.Types().Types {
		var fields []schema.Field
		for _, f := range u.Fields {
			if f.RefTo() == typeName || u.Name == ReminderType && f.Name == "about" {
				fields = append(fields, f)
			}
		}
		if len(fields) == 0 {
			continue
		}
		recs, _ := st.List(u.Name, store.ListOptions{})
		for _, r := range recs {
			if after := pointAt(r, fields, typeName, keep, gone); len(after) > 0 {
				ops = append(ops, Op{Type: u.Name, ID: r.ID, After: after})
			}
		}
	}
	return ops
}

// pointAt is what in r changes to point at keep rather than one of gone.
func pointAt(r *store.Record, fields []schema.Field, typeName, keep string, gone map[string]bool) map[string]any {
	after := map[string]any{}
	for _, f := range fields {
		switch v := r.Fields[f.Name].(type) {
		case string:
			if f.Name == "about" && f.RefTo() == "" {
				if id, ok := pageID(v, typeName); ok && gone[id] {
					after[f.Name] = "/t/" + typeName + "/" + keep
				}
			} else if gone[v] {
				after[f.Name] = keep
			}
		case []any:
			list, changed, seen := make([]any, 0, len(v)), false, map[string]bool{}
			for _, x := range v {
				id, _ := x.(string)
				if gone[id] {
					id, changed = keep, true
				}
				if !seen[id] {
					seen[id] = true
					list = append(list, id)
				}
			}
			if changed {
				after[f.Name] = list
			}
		}
	}
	return after
}

// pageID is the id in a record's page address of the given type.
func pageID(path, typeName string) (string, bool) {
	prefix := "/t/" + typeName + "/"
	if len(path) <= len(prefix) || path[:len(prefix)] != prefix {
		return "", false
	}
	return path[len(prefix):], true
}
