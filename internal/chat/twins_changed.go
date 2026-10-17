package chat

import (
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// Records called the same were said when one was made (same_title.go), and
// not when one was changed: an agent moving a clean-up day changed both of
// its two events, one after the other, and a parent's four tasks to
// register for clubs stayed four. A change to a record that others are
// called the same as, still open, says so, with how to make them one.

// twinsChanged is a note for each record c changed that has others alike.
func (s *Service) twinsChanged(c records.Change) []string {
	var out []string
	for _, op := range c.Ops {
		if op.Before == nil || op.After == nil || op.Type == records.BlockType || op.Type == records.MessageType || op.Type == records.ActivityType {
			continue
		}
		t, ok := s.Store.Types().Get(op.Type)
		rec, err := s.Store.Get(op.Type, op.ID)
		if !ok || err != nil {
			continue
		}
		if f := t.DoneField(); f != nil {
			if done, _ := rec.Fields[f.Name].(bool); done {
				continue // ticked off, it is not the one to change twice
			}
		}
		alike := records.Alike(s.Store, t, rec)
		if len(alike) == 0 {
			continue
		}
		out = append(out, fmt.Sprintf("%s %q is called the same as %s, still open: %s. If they are one thing twice, keep_one (POST /t/%s/%s/keep-one) makes them one, so a change is made once.",
			op.ID, t.Called(rec.ID, rec.Fields), schema.Count(len(alike), "other"), s.alikeSaid(t, alike), t.Name, op.ID))
	}
	return out
}
