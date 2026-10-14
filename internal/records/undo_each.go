package records

import "fmt"

// Taking back several changes at once: a reply's, from the button under it
// or from undo_change with ids. Each is its own reversal in the log, but
// they are taken back all or none, as the changes that made them were: a
// reply half undone is the half-done state those changes were made in one
// call to avoid. When one cannot be undone, those undone before it are put
// back, and nothing has changed.

// UndoEach undoes the entries in the order given, all or none. It returns
// what each undo said and its reversal, ready to be recorded.
func (b *Book) UndoEach(ids []string) (said []string, changes []Change, err error) {
	for i, id := range ids {
		s, c, err := b.Undo(id)
		if err != nil {
			b.redo(changes)
			return nil, nil, fmt.Errorf("nothing was undone: %s of %d could not be, %v", ordinal(i+1), len(ids), err)
		}
		said, changes = append(said, s), append(changes, c)
	}
	return said, changes, nil
}

// redo puts back what reversals not yet recorded took back, newest first.
func (b *Book) redo(changes []Change) {
	for i := len(changes) - 1; i >= 0; i-- {
		ops := changes[i].Ops
		back := make([]Op, 0, len(ops))
		for j := len(ops) - 1; j >= 0; j-- {
			back = append(back, Op{Type: ops[j].Type, ID: ops[j].ID, After: ops[j].Before})
		}
		b.Apply(back...) // best effort: each was written a moment ago
	}
}

// KeptAll is what a run of undos kept, as one sentence, or "".
func KeptAll(changes []Change) string {
	var kept []string
	for _, c := range changes {
		kept = append(kept, c.Kept...)
	}
	return KeptSaid(Change{Kept: kept})
}

func ordinal(n int) string {
	words := []string{"the first", "the second", "the third", "the fourth", "the fifth"}
	if n <= len(words) {
		return words[n-1]
	}
	return fmt.Sprintf("number %d", n)
}
