package chat

import (
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// An event that has happened is a change like any: marked over once it
// ends, logged, so an action can wait for it (when changed, what event,
// over=true) and do what follows a visit, a lesson or a meeting: thank
// them, ask how it went, offer the next time, send the bill. Only events
// that ended lately are marked, so a calendar brought in with years of
// them sets nothing off.

// overWithin is how long after it ends an event is still marked over.
const overWithin = 2 * 24 * time.Hour

// MarkOver marks each event that ended lately as over. It is a background
// job's work.
func (s *Service) MarkOver(now time.Time) error {
	t, ok := s.Store.Types().Get(records.EventType)
	if !ok {
		return nil
	}
	if _, has := t.Field("over"); !has {
		return nil
	}
	recs, err := s.Store.List(records.EventType, store.ListOptions{})
	if err != nil {
		return err
	}
	for _, e := range recs {
		end, ok := s.endOf(e)
		if over, _ := e.Fields["over"].(bool); over || !ok || end.After(now) || now.Sub(end) > overWithin || e.Fields["repeat"] != nil && e.Fields["repeat"] != "" {
			continue
		}
		done, err := s.Apply(records.Op{Type: records.EventType, ID: e.ID, After: map[string]any{"over": true}})
		if err != nil {
			continue
		}
		records.Record(s.Store, "system", records.Change{Action: "over", Component: records.EventType, ID: e.ID, Detail: records.Name(s.Store, t, e), Ops: done})
	}
	return nil
}

// endOf is when an event ends: its end, an hour after a timed start, or
// the end of its day.
func (s *Service) endOf(e *store.Record) (time.Time, bool) {
	for _, f := range []string{"ends", "starts"} {
		v, _ := e.Fields[f].(string)
		at, day, ok := when.Stored(v, s.Now())
		if !ok {
			continue
		}
		switch {
		case day:
			return at.AddDate(0, 0, 1), true
		case f == "starts":
			return at.Add(time.Hour), true
		}
		return at, true
	}
	return time.Time{}, false
}
