package chat

import (
	"errors"
	"fmt"
)

// A batch is one entry for many records: an import, a sync, a meeting
// written up, writing organised or suggested. It is kept on its entry as
// each record's type, id and what it was, and taken back as one.

// reverseBatch puts every record a batch touched back as it was: one it
// made goes, one it changed or removed comes back. What each was just
// now is kept on the reversal, so undoing that redoes the batch.
func (s *Service) reverseBatch(target string, changes []BatchItem) (Change, error) {
	var back []BatchItem
	n := 0
	for _, c := range changes {
		cur, err := s.Store.Get(c.Type, c.ID)
		var now map[string]any
		if err == nil {
			now = cur.Fields
		}
		switch {
		case c.Before == nil && now == nil:
			continue // made, and already gone
		case c.Before == nil:
			if err := s.Store.Delete(c.Type, c.ID); err != nil {
				return Change{}, err
			}
		case now == nil:
			if _, err := s.Store.Restore(c.Type, c.ID, c.Before); err != nil {
				return Change{}, err
			}
		default:
			if same(now, c.Before) {
				continue
			}
			if _, err := s.Store.Update(c.Type, c.ID, c.Before); err != nil {
				return Change{}, err
			}
		}
		back = append(back, BatchItem{Type: c.Type, ID: c.ID, Before: now})
		n++
	}
	if n == 0 {
		return Change{}, errors.New("every record is already as it was")
	}
	return Change{Action: "synced", Component: target, Detail: fmt.Sprintf("%d records put back", n), Before: Batch(back)}, nil
}
