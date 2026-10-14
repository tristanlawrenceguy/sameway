package records

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Undoing a change when the thing was changed again since. An entry keeps
// the whole thing as it was and became, so writing back all of what it was
// would also take back what the person, or another agent, did after it:
// undo "move these tasks to Friday" and a title typed since went too. So
// only what is still as the entry left it goes back, field by field, and
// what was changed since is kept and said. Undo takes back the change it
// names and nothing else.

// sinceFields is what to write to undo op on a record that is now cur:
// each field op changed and that is still as op left it goes back to what
// it was; one changed again since is kept as it is, and named in kept.
func sinceFields(op Op, cur map[string]any) (after map[string]any, kept []string) {
	after = make(map[string]any, len(cur))
	for k, v := range cur {
		after[k] = v
	}
	for _, k := range changedKeys(op.Before, op.After) {
		was, had := op.Before[k]
		switch {
		case !sameValue(cur[k], op.After[k]):
			if !sameValue(cur[k], was) {
				kept = append(kept, k)
			}
		case had:
			after[k] = was
		default:
			delete(after, k)
		}
	}
	return after, kept
}

// changedKeys is the fields that differ between before and after, in
// order, so what is said is the same each time.
func changedKeys(before, after map[string]any) []string {
	var keys []string
	for k := range before {
		if !sameValue(before[k], after[k]) {
			keys = append(keys, k)
		}
	}
	for k := range after {
		if _, in := before[k]; !in && after[k] != nil {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}

func sameValue(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

// keptSaid names the fields kept on one thing: "title of Book the car".
func (b *Book) keptSaid(op Op, cur map[string]any, fields []string) string {
	words := make([]string, len(fields))
	for i, f := range fields {
		words[i] = schema.Words(f)
	}
	name := describe(b.Store, op.Type, &store.Record{ID: op.ID, Type: op.Type, Fields: cur}).Detail
	if name == "" {
		return strings.Join(words, " and ")
	}
	return strings.Join(words, " and ") + " of " + name
}

// KeptSaid is the sentence for what an undo kept, or "" when it kept
// nothing.
func KeptSaid(c Change) string {
	if len(c.Kept) == 0 {
		return ""
	}
	return "Kept as changed since: " + strings.Join(c.Kept, "; ") + "."
}
