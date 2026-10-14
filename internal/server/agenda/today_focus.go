package agenda

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/relate"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/web"
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
// tag important or urgent, or a high priority.
func (s *Service) important(it todayItem) bool {
	rec, err := s.app.Store.Get(it.Type, it.ID)
	if err != nil {
		return false
	}
	if p, _ := rec.Fields["priority"].(string); strings.EqualFold(p, "high") || strings.EqualFold(p, "urgent") {
		return true
	}
	return tagged(rec, "important") || tagged(rec, "urgent") || tagged(rec, startTag)
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
func (s *Service) focus(l todayLists) (first, rest []todayItem) {
	var all []todayItem
	for _, list := range [][]todayItem{l.Late, l.Tasks} {
		for _, it := range list {
			if it.Type == "task" {
				all = append(all, it)
			}
		}
	}
	if len(all) < focusFrom {
		return nil, nil
	}
	matters, start := map[string]bool{}, map[string]bool{}
	for _, it := range all {
		matters[it.ID] = s.important(it)
	}
	pool := append(append([]todayItem{}, all...), s.mattersSoon(all)...)
	for _, it := range pool[len(all):] {
		matters[it.ID] = true
	}
	for _, it := range pool {
		start[it.ID] = s.startsWith(it)
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
	first = pool[:focusOn]
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
// already late or due, and due within soonDays or on no day at all.
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
			if !ok || at.After(now.AddDate(0, 0, soonDays)) {
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

// dumpBox is the box for everything in the person's head.
func (s *Service) dumpBox() string {
	return `<h2>Too much in your head?</h2>` + string(s.Form(ui.Form{Action: "/today/sort", Class: "sw-stack",
		Body: s.Component("textarea", map[string]any{"label": "Everything on your mind", "name": "words", "rows": 4,
			"hint": "As it comes, in any order: things to do, to remember, to reply to. The assistant makes them tasks with their days and says which three to start with."}),
		Button: &ui.Button{Label: "Sort it out for me", Variant: ui.Secondary}}))
}

// todaySort hands what the person wrote to the assistant, to sort.
func (s *Service) todaySort(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	words := strings.TrimSpace(r.PostForm.Get("words"))
	if words == "" {
		s.Failed(w, r, "Nothing to sort", errors.New("write what is on your mind first, as it comes"), "/today")
		return
	}
	ask := fmt.Sprintf("Sort this out for me: make each thing to do a task, with its day when I said one, tag the ones that matter most important, and tell me the three to start with, tagged start (and take start off any task that had it before). Ask me only if something cannot be guessed.\n\n%s", words)
	if _, err := s.ChatFor(r).SendFile(context.WithoutCancel(r.Context()), "", ask, ""); err != nil {
		s.ForgetModel()
		s.Failed(w, r, "Not sorted", err, "/today")
		return
	}
	// Back where they were: the page that says what to start with, not the
	// busiest page they have. What it said in full is in the chat.
	s.TellAt(w, r, web.Outcome{Title: "Sorted", Text: "The assistant has made your tasks. Its answer, with the three to start with, is in Chat."}, "/today")
}

// focusSection is Start with these, and the rest folded, or "" when Today
// is short enough to read whole.
func (s *Service) focusSection(l todayLists, list func(string, []todayItem) string) string {
	first, rest := s.focus(l)
	if first == nil {
		return ""
	}
	n, why := len(rest), ""
	for _, it := range first {
		if !it.Soon {
			n++
		}
		// Why one not due is first, said once for the three: a starred row
		// in To Do or a red flag in Todoist says it with a mark; here, words.
		if why == "" && s.important(it) {
			why = ", what is marked important first"
		}
	}
	out := `<p>` + template.HTMLEscapeString(fmt.Sprintf("%d tasks are late or due. Start with these three%s; the rest wait below.", n, why)) + `</p>` +
		s.letGoOffer(append(append([]todayItem{}, first...), rest...)) + list("Start with these", first) // today_letgo.go
	if body, err := s.app.Registry.RenderSlot("disclosure", map[string]any{"label": "The rest", "count": len(rest), "of": schema.Noun(len(rest), "task")}, template.HTML(list("", rest))); err == nil {
		out += string(body)
	}
	return out
}
