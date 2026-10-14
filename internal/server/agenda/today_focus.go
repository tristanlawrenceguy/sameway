package agenda

import (
	"fmt"
	"html/template"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/relate"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// "In those moments my reminder app doesn't help, because I get worse at
// curating the list the more overwhelmed I am" (a person with ADHD, trying
// Sameway). A long Today is one more list to curate. So when there is a lot,
// Today starts with three: what matters most of what is late or due, and
// folds the rest away until those are done. And a box takes everything in
// their head as it comes, for the assistant to sort into tasks with their
// days and say which three to start with. The sorting is the assistant's;
// the person only says it.

// focusFrom is how many late and due tasks make Today start with three.
const (
	focusFrom = 6
	focusOn   = 3
)

// important is a task the person or the assistant marked as mattering: a
// tag important or urgent, or a high priority; or one it said to start with.
func (s *Service) important(it todayItem) bool {
	return s.marked(it) || s.startsWith(it)
}

// marked is a task tagged important or urgent, or of high priority.
func (s *Service) marked(it todayItem) bool {
	rec, err := s.app.Store.Get(it.Type, it.ID)
	if err != nil {
		return false
	}
	if p, _ := rec.Fields["priority"].(string); strings.EqualFold(p, "high") || strings.EqualFold(p, "urgent") {
		return true
	}
	return tagged(rec, "important") || tagged(rec, "urgent")
}

// startTag marks the tasks the assistant said to start with. Asked to sort
// a week out, it told the person to call the dentist first, a short call
// with no day; Today, sorting what matters by day, put three others first,
// and the person had two answers to "what now?". Tagged start, they lead.
const startTag = "start"

// startsWith is a task tagged start.
func (s *Service) startsWith(it todayItem) bool {
	rec, err := s.app.Store.Get(it.Type, it.ID)
	return err == nil && tagged(rec, startTag)
}

func tagged(rec *store.Record, tag string) bool {
	tags, _ := rec.Fields["tags"].([]any)
	for _, t := range tags {
		if s, ok := t.(string); ok && strings.EqualFold(s, tag) {
			return true
		}
	}
	return false
}

// focus splits the late and due tasks into the three to start with and the
// rest, when there are enough to need it: those that matter first, then
// the latest late, then what is due earliest today. What matters includes
// what the person marked so that is not due yet: sorting what was in their
// head, they were told to start with the council tax and the dentist, and
// Today went on starting with two errands from a fortnight ago, because
// neither was due today. Those join the three, never the rest.
//
// What the assistant said to start with leads on any day, a short one too:
// told to start with the dentist, a person with four tasks due found Today
// without it, the same two answers to "what now?". On a short day only
// those tagged start come first; the rest stay as they were, unfolded.
func (s *Service) focus(l todayLists) (first, rest []todayItem) {
	var all []todayItem
	for _, list := range [][]todayItem{l.Late, l.Tasks} {
		for _, it := range list {
			if it.Type == "task" {
				all = append(all, it)
			}
		}
	}
	pool := append(append([]todayItem{}, all...), s.mattersSoon(all)...)
	matters, start, starts := map[string]bool{}, map[string]bool{}, 0
	for i, it := range pool {
		matters[it.ID] = i >= len(all) || s.important(it)
		if start[it.ID] = s.startsWith(it); start[it.ID] {
			starts++
		}
	}
	short := len(all) < focusFrom
	if short && starts == 0 {
		return nil, nil
	}
	sort.SliceStable(pool, func(i, j int) bool {
		if start[pool[i].ID] != start[pool[j].ID] {
			return start[pool[i].ID]
		}
		if matters[pool[i].ID] != matters[pool[j].ID] {
			return matters[pool[i].ID]
		}
		// A task with no day comes after those with one.
		if pool[i].At.IsZero() != pool[j].At.IsZero() {
			return pool[j].At.IsZero()
		}
		return pool[i].At.Before(pool[j].At)
	})
	if short {
		first = pool[:min(starts, focusOn)]
	} else {
		first = pool[:focusOn]
	}
	chosen := map[string]bool{}
	for _, it := range first {
		chosen[it.ID] = true
	}
	for _, it := range all {
		if !chosen[it.ID] {
			rest = append(rest, it)
		}
	}
	return first, rest
}

// soonDays is how far ahead a task that matters is close enough to start.
const soonDays = 7

// mattersSoon is the tasks marked as mattering that are not done, not
// already late or due, and due within soonDays or on no day at all; one
// tagged start on any day, as the assistant said to start with it now.
func (s *Service) mattersSoon(have []todayItem) []todayItem {
	t, ok := s.app.Types.Get("task")
	if !ok {
		return nil
	}
	seen := map[string]bool{}
	for _, it := range have {
		seen[it.ID] = true
	}
	now := s.Now()
	recs, _ := s.app.Store.List("task", store.ListOptions{})
	var out []todayItem
	for _, r := range recs {
		if done, _ := r.Fields["done"].(bool); done || seen[r.ID] {
			continue
		}
		it := todayItem{Title: s.Title(t, r), Href: relate.Page("task", r.ID), Type: "task", ID: r.ID, Soon: true}
		if v, _ := r.Fields["due"].(string); v != "" {
			at, allDay, ok := when.Stored(v, now)
			if !ok || at.After(now.AddDate(0, 0, soonDays)) && !s.startsWith(it) {
				continue
			}
			it.At, it.AllDay, it.When = at, allDay, when.Day(at, now)
		}
		if s.important(it) {
			out = append(out, it)
		}
	}
	return out
}

// focusSection is Start with these, and the rest folded; on a short day
// only what the assistant said to start with, then the rest as usual; or
// "" when Today is short and the assistant named nothing to start with.
func (s *Service) focusSection(l todayLists, list func(string, []todayItem) string) string {
	first, rest := s.focus(l)
	if first == nil {
		return ""
	}
	n := len(rest)
	for _, it := range first {
		if !it.Soon {
			n++
		}
	}
	if n < focusFrom {
		late := map[string]bool{}
		for _, it := range l.Late {
			late[it.ID] = true
		}
		var lateRest, dueRest []todayItem
		for _, it := range rest {
			if late[it.ID] {
				lateRest = append(lateRest, it)
			} else {
				dueRest = append(dueRest, it)
			}
		}
		these := "these"
		if len(first) == 1 {
			these = "this"
		}
		return `<p>The assistant said to start with ` + these + `.</p>` + list("Start with "+these, first) +
			s.lateNudge(lateRest, s.Now()) + list("Overdue", lateRest) + list("Tasks", dueRest) // today_nudge.go
	}
	out := `<p>` + template.HTMLEscapeString(fmt.Sprintf("%d tasks are late or due. Start with these three%s; the rest wait below.", n, s.firstWhy(first))) + `</p>` +
		s.letGoOffer(append(append([]todayItem{}, first...), rest...)) + list("Start with these", first) // today_letgo.go
	if body, err := s.app.Registry.RenderSlot("disclosure", map[string]any{"label": "The rest", "count": len(rest), "of": schema.Noun(len(rest), "task")}, template.HTML(list("", rest))); err == nil {
		out += string(body)
	}
	return out
}

// firstWhy is why the three come in their order, said once for them: a
// starred row in To Do or a red flag in Todoist says it with a mark; here,
// words. What the assistant said comes before what is marked important,
// and the line says so, so a task with no day first does not read as a
// mistake and the two never seem to disagree.
func (s *Service) firstWhy(first []todayItem) string {
	said, marked := 0, false
	for _, it := range first {
		if s.startsWith(it) {
			said++
		} else if s.marked(it) {
			marked = true
		}
	}
	switch {
	case said == len(first):
		return ", as the assistant said"
	case said > 0 && marked:
		return ", what the assistant said first, then what is marked important"
	case said > 0:
		return ", what the assistant said first"
	case marked:
		return ", what is marked important first"
	}
	return ""
}
