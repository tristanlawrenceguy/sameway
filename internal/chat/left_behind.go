package chat

import (
	"fmt"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// An event moved to another day left its tasks on the old one, and an
// agent asked to move the event and what came up at it believed each task
// was a call of its own, and gave up. When a record's day moves, what is
// in it (a project's tasks, an event's) and still on the old day is named,
// with the one change that moves them all: said for every way in.

// leftBehind is what c left on the old day of each record it moved.
func (s *Service) leftBehind(c records.Change) []string {
	var out []string
	now := s.Now()
	for _, op := range c.Ops {
		t, ok := s.Store.Types().Get(op.Type)
		if !ok || op.Before == nil || op.After == nil {
			continue
		}
		day := t.DayField()
		was, _ := op.Before[day].(string)
		is, _ := op.After[day].(string)
		from, _, okFrom := when.Stored(was, now)
		to, _, okTo := when.Stored(is, now)
		if day == "" || !okFrom || !okTo || from.Format("2006-01-02") == to.Format("2006-01-02") {
			continue
		}
		rec, err := s.Store.Get(op.Type, op.ID)
		if err != nil {
			continue
		}
		for _, w := range records.Withins(s.Store, t) {
			if said := s.stillOn(w, rec.ID, from, to, t.Called(rec.ID, rec.Fields)); said != "" {
				out = append(out, said)
			}
		}
	}
	return out
}

// stillOn names what is in the record and still on its old day, not done.
func (s *Service) stillOn(w records.Within, id string, from, to time.Time, called string) string {
	day := w.T.DayField()
	if day == "" {
		return ""
	}
	recs, _ := s.Store.List(w.T.Name, store.ListOptions{})
	var ids, names []string
	done := w.T.DoneField()
	for _, r := range recs {
		v, _ := r.Fields[day].(string)
		at, _, ok := when.Stored(v, s.Now())
		if in, _ := r.Fields[w.Field].(string); in != id || !ok || at.Format("2006-01-02") != from.Format("2006-01-02") {
			continue
		}
		if done != nil {
			if ticked, _ := r.Fields[done.Name].(bool); ticked {
				continue
			}
		}
		ids, names = append(ids, r.ID), append(names, w.T.Called(r.ID, r.Fields))
	}
	if len(ids) == 0 {
		return ""
	}
	return fmt.Sprintf("%s in %s %s still on its old day, %s: %s. To move them with it, update_records (PATCH /api/%s with ids) and %s %s, in one change; ids %s.",
		schema.Count(len(ids), w.T.Name), called, isAre(len(ids)), when.Day(from, s.Now()), schema.Names(names), w.T.Name, day, to.Format("2006-01-02"), strings.Join(ids, ", "))
}

func isAre(n int) string {
	if n == 1 {
		return "is"
	}
	return "are"
}
