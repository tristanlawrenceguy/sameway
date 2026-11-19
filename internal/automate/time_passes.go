package automate

import (
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A record comes to match an action's conditions when it changes, and also
// when time passes: a visit starts within a day (starts<+1d), a task is
// late (due<today), a payment is still unpaid a week on (paid=false,
// created_at<-7d). Now and then each action that waits for a record to
// come to match is looked at again: one that matches now and did not when
// last looked sets it off, as a change would. So a reminder the day before,
// a nudge for a bill, or a word when something falls due is an action in
// the person's words, not a feature of its own. The first look only notes
// the time, so what already matched sets nothing off.

const (
	lookedKey   = "automate:looked:" // and the action's id: when it was last looked at
	mostPerLook = 50                 // set off by one look, so a clock put right does not flood
)

// TimePassed sets off what time alone has brought to match. It is a
// background job's work.
func (s *Service) TimePassed(now time.Time) error {
	if s.auto == nil {
		return nil
	}
	at, ok := s.Store.Types().Get(records.ActionType)
	if !ok {
		return nil
	}
	if _, has := at.Field("when"); !has {
		return nil
	}
	actions, err := s.Store.List(records.ActionType, store.ListOptions{})
	if err != nil {
		return err
	}
	set := 0
	for _, a := range actions {
		set += s.lookAgain(a, now, mostPerLook-set)
	}
	return nil
}

// lookAgain sets off one action for what has come to match it since it was
// last looked at, up to most, and says how many.
func (s *Service) lookAgain(a *store.Record, now time.Time, most int) int {
	when, _ := a.Fields["when"].(string)
	what, _ := a.Fields["what"].(string)
	only := records.StringList(a.Fields["only"])
	if when != "changed" || len(only) == 0 {
		return 0
	}
	t, ok := s.Store.Types().Get(what)
	if !ok || t.Internal {
		return 0
	}
	conds, err := query.ParseAll(t, only)
	if err != nil {
		return 0
	}
	key := lookedKey + a.ID
	last, err := time.Parse(time.RFC3339, s.Store.Meta(key))
	s.Store.SetMeta(key, now.UTC().Format(time.RFC3339))
	if err != nil || !now.After(last) {
		return 0 // the first look, or the clock turned back
	}
	recs, err := s.Store.List(t.Name, store.ListOptions{})
	if err != nil {
		return 0
	}
	set := 0
	for _, r := range recs {
		if set >= most {
			break
		}
		if !query.Match(t, r, conds, now) || query.Match(t, r, conds, last) || !s.auto.fresh(a.ID, r.ID) {
			continue
		}
		job := automationJob{action: a.ID, vars: s.recordVars(t, r), why: whenWords(when, t, records.Name(s.Store, t, r))}
		select {
		case s.auto.jobs <- job:
			set++
		default:
			return set
		}
	}
	return set
}
