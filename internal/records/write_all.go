package records

import "github.com/tristanlawrenceguy/sameway/internal/store"

// Several writes, all or none. create_records and update_records each
// wrote one record at a time and, when one was refused, took back those
// before it in a way of their own: one removed what it had made, the other
// wrote back what it had changed. Both are the same thing, an op run
// backwards, and the next tool that writes several would have needed a
// third. Here it is once.

// Writing is one record to write: an id to change one, or none to make one.
type Writing struct {
	ID     string
	Fields map[string]any
}

// WriteAll writes each as Write does, in order. When one is refused, those
// written before it are put back as they were, and its place is returned
// with why; nothing is left half done.
func WriteAll(st *store.Store, action, typ string, all []Writing) ([]*store.Record, []Change, int, error) {
	var recs []*store.Record
	var changes []Change
	for i, w := range all {
		rec, c, err := Write(st, action, typ, w.ID, w.Fields)
		if err != nil {
			(&Book{Store: st}).putBack(changes)
			return nil, nil, i, err
		}
		recs, changes = append(recs, rec), append(changes, c)
	}
	return recs, changes, -1, nil
}

// putBack runs changes not yet logged backwards, newest first: what was
// made goes, what was changed is as it was. WriteAll and UndoEach both
// end here when one of several is refused: an op run backwards is one
// thing, done once. Best effort: each was written a moment ago, and one
// that cannot be put back does not stop the rest.
func (b *Book) putBack(changes []Change) {
	for i := len(changes) - 1; i >= 0; i-- {
		ops := changes[i].Ops
		back := make([]Op, 0, len(ops))
		for j := len(ops) - 1; j >= 0; j-- {
			back = append(back, Op{Type: ops[j].Type, ID: ops[j].ID, After: ops[j].Before})
		}
		b.Apply(back...)
	}
}

// Joined is the ops of several changes as one, for the log: one entry, one
// Undo.
func Joined(changes []Change) []Op {
	var ops []Op
	for _, c := range changes {
		ops = append(ops, c.Ops...)
	}
	return ops
}
