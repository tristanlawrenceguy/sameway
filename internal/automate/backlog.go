package automate

import (
	"context"
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/relate"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// An action that sorts or suggests runs when a record comes, so what was
// in Sameway before it was set up was never sorted: the photos already
// kept when "sort my photos" was asked. Run now, by a press or by the
// assistant, it goes through the records it would have been set off by
// that it has not done yet, oldest first, one at a time in the
// background as if each had just come, each in the activity as it is
// done. What comes meanwhile is sorted first, not left behind three
// hundred photos. Run again, it does only what is left.

// runOverAll runs a sorting or suggesting action over what is already
// here, when nothing set it off.
func (s *Service) runOverAll(ctx context.Context, action *store.Record) Result {
	title, _ := action.Fields["title"].(string)
	what, _ := action.Fields["what"].(string)
	t, ok := s.Store.Types().Get(what)
	if !ok {
		return fail("%s says no kind of record to go through: set what, such as file or note", title)
	}
	recs, err := query.Filter(s.Store, t, records.StringList(action.Fields["only"]), "created_at", 0, s.Now())
	if err != nil {
		return fail("%s: %v", title, err)
	}
	done := s.doneBy(action)
	var todo []*store.Record
	for _, r := range recs {
		if !done[t.Name+"/"+r.ID] {
			todo = append(todo, r)
		}
	}
	ran := records.Change{Action: "ran", Component: records.ActionType, ID: action.ID, Detail: title, Href: relate.Page(records.ActionType, action.ID)}
	if len(todo) == 0 {
		return Result{Text: fmt.Sprintf("%s: every %s it goes through is already done", title, schema.Words(t.Name)), Change: &ran}
	}
	if s.auto == nil { // no worker, as on the command line: one by one here
		for _, r := range todo {
			s.runAutomation(ctx, automationJob{action: action.ID, vars: s.recordVars(t, r), why: "it was run on what was already here", backlog: true})
		}
		return Result{Text: fmt.Sprintf("%s: went through %s", title, schema.Count(len(todo), t.Name)), Change: &ran}
	}
	if n := s.auto.going(action.ID, len(todo)); n > 0 { // pressed twice: each once
		return Result{Text: fmt.Sprintf("%s is already going through what is here: %s left; each is in the activity as it is done", title, schema.Count(n, t.Name))}
	}
	go func() {
		for _, r := range todo {
			s.auto.later <- automationJob{action: action.ID, vars: s.recordVars(t, r), why: "it was run on what was already here", backlog: true}
			s.auto.going(action.ID, -1)
		}
	}()
	ran.Detail = fmt.Sprintf("%s, over %s already here", title, schema.Count(len(todo), t.Name))
	return Result{Text: fmt.Sprintf("%s: going through %s already here, oldest first, one at a time in the background; each is in the activity as it is done", title, schema.Count(len(todo), t.Name)), Change: &ran}
}

// going counts what an action has left to go through. Given n records it
// starts, answering 0, unless it is going already: then it answers with
// what it has left. Given -1, one more has been taken.
func (a *automation) going(action string, n int) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	if n < 0 {
		a.left[action]--
		return 0
	}
	if left := a.left[action]; left > 0 {
		return left
	}
	a.left[action] = n
	return 0
}

// doneBy is the records an action has sorted or suggested from already.
func (s *Service) doneBy(action *store.Record) map[string]bool {
	done := map[string]bool{}
	kind, field, of := records.ClassificationType, "action", "record"
	if action.Fields["kind"] == "suggest" {
		kind, field, of = records.ProposalType, "by_action", "from"
	}
	all, _ := s.Store.List(kind, store.ListOptions{})
	for _, r := range all {
		if r.Fields[field] == action.ID {
			done[fmt.Sprint(r.Fields[of])] = true
		}
	}
	return done
}

// RunOn runs an action on one record when nothing set it off, in the
// background as if the record had just come: a conversation carried on
// since it was brought is looked through again, as when it came.
func (s *Service) RunOn(ctx context.Context, action string, t *schema.Type, rec *store.Record, why string) {
	job := automationJob{action: action, vars: s.recordVars(t, rec), why: why, backlog: true}
	if s.auto == nil {
		s.runAutomation(ctx, job)
		return
	}
	select {
	case s.auto.jobs <- job:
	default:
	}
}

// suggestedFrom is what an action suggested from one record before, made,
// turned down or waiting: a conversation carried on since is read whole
// again, and what its start asked for was put to the person already.
func (s *Service) suggestedFrom(action, from string) []string {
	all, _ := s.Store.List(records.ProposalType, store.ListOptions{})
	var out []string
	for _, p := range all {
		if p.Fields["by_action"] == action && p.Fields["from"] == from {
			out = append(out, fmt.Sprint(p.Fields["summary"]))
		}
	}
	return out
}
