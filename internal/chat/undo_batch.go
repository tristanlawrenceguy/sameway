package chat

import (
	"errors"
	"fmt"
)

// A batch is one entry for many records: an import, a sync, a meeting
// written up, writing organised or suggested. It is kept on its entry as
// each record's type, id and what it was, and taken back as one.

// A batchChange is one record a batch touched and what it was before it.
type batchChange struct {
	typ, id string
	before  map[string]any
}

// Batch is how a batch is kept on its entry: for each record, its type,
// its id, and its fields before, absent for a record the batch made.
func Batch(changes []BatchItem) map[string]any {
	list := make([]any, 0, len(changes))
	for _, c := range changes {
		item := map[string]any{"type": c.Type, "id": c.ID}
		if c.Before != nil {
			item["before"] = c.Before
		}
		list = append(list, item)
	}
	return map[string]any{"changes": list}
}

// BatchItem is one record in a batch, for the code that made the batch.
type BatchItem struct {
	Type, ID string
	Before   map[string]any
}

func batchIn(before map[string]any) []batchChange {
	list, _ := before["changes"].([]any)
	var out []batchChange
	for _, v := range list {
		m, ok := v.(map[string]any)
		if !ok {
			continue
		}
		typ, _ := m["type"].(string)
		id, _ := m["id"].(string)
		was, _ := m["before"].(map[string]any)
		if typ != "" && id != "" {
			out = append(out, batchChange{typ, id, was})
		}
	}
	return out
}

// reverseBatch puts every record a batch touched back as it was: one it
// made goes, one it changed or removed comes back. What each was just
// now is kept on the reversal, so undoing that redoes the batch.
func (s *Service) reverseBatch(target string, changes []batchChange) (Change, error) {
	var back []BatchItem
	n := 0
	for _, c := range changes {
		cur, err := s.Store.Get(c.typ, c.id)
		var now map[string]any
		if err == nil {
			now = cur.Fields
		}
		switch {
		case c.before == nil && now == nil:
			continue // made, and already gone
		case c.before == nil:
			if err := s.Store.Delete(c.typ, c.id); err != nil {
				return Change{}, err
			}
		case now == nil:
			if _, err := s.Store.Restore(c.typ, c.id, c.before); err != nil {
				return Change{}, err
			}
		default:
			if same(now, c.before) {
				continue
			}
			if _, err := s.Store.Update(c.typ, c.id, c.before); err != nil {
				return Change{}, err
			}
		}
		back = append(back, BatchItem{Type: c.typ, ID: c.id, Before: now})
		n++
	}
	if n == 0 {
		return Change{}, errors.New("every record is already as it was")
	}
	return Change{Action: "synced", Component: target, Detail: fmt.Sprintf("%d records put back", n), Before: Batch(back)}, nil
}
