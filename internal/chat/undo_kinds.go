package chat

import (
	"errors"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// The owner's rule is that everything is reversible, and what cannot be
// is said and asked first. These are the entries that used to be logged
// and still could not be taken back: a setting changed, an amount logged
// against a habit, a batch of records made from a file or a folder.

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
	case "imported", "synced", "arranged", "wrote up", "organised", "suggested", "rescheduled":
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
