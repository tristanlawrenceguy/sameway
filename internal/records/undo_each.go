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
			b.putBack(changes) // write_all.go
			return nil, nil, fmt.Errorf("nothing was undone: %s of %d could not be, %v", ordinal(i+1), len(ids), err)
		}
		said, changes = append(said, s), append(changes, c)
	}
	return said, changes, nil
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
