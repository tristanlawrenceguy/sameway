package chat

import (
	"errors"
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// The owner's rule is that everything is reversible, and what cannot be
// is said and asked first. These are the entries that used to be logged
// and still could not be taken back: a setting changed, an amount logged
// against a habit, a batch of records made from a file or a folder.

// EntryType is the content type of what is logged against a habit.
const EntryType = "entry"

// inverseMore reverses the kinds of entry inverse does not know.
func (s *Service) inverseMore(a *store.Record) (func() (Change, error), error) {
	action, _ := a.Fields["action"].(string)
	target, _ := a.Fields["target"].(string)
	before, _ := a.Fields["before"].(map[string]any)
	switch action {
	case "set":
		// A setting goes back to what it was, which may be nothing.
		if s.SetSetting == nil || before == nil {
			return nil, errors.New("the entry does not say what it was")
		}
		was, _ := before["value"].(string)
		now := s.setting(target)
		if now == was {
			return nil, errors.New("it is already as it was")
		}
		return func() (Change, error) {
			if err := s.SetSetting(target, was); err != nil {
				return Change{}, err
			}
			return Change{Action: "set", Component: target, Detail: orNone(was), Before: map[string]any{"value": now}}, nil
		}, nil
	case "logged":
		// An amount logged against a habit is the entry it made.
		id, _ := before["entry"].(string)
		if id == "" {
			return nil, errors.New("the entry does not say what was logged")
		}
		if _, err := s.Store.Get(EntryType, id); err != nil {
			return nil, errors.New("it is already gone")
		}
		return func() (Change, error) { return s.take(EntryType, id) }, nil
	case "imported", "synced":
		// A batch: records made, changed and removed by one import, each
		// with what it was before (nothing, for one it made).
		changes := batchIn(before)
		if len(changes) == 0 {
			return nil, errors.New("the entry does not say what it changed")
		}
		return func() (Change, error) { return s.reverseBatch(target, changes) }, nil
	}
	return nil, errors.New("that kind of entry cannot be undone")
}

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
