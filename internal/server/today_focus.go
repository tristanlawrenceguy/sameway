package server

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/store"
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
func (s *Server) important(it todayItem) bool {
	rec, err := s.app.Store.Get(it.Type, it.ID)
	if err != nil {
		return false
	}
	if p, _ := rec.Fields["priority"].(string); strings.EqualFold(p, "high") || strings.EqualFold(p, "urgent") {
		return true
	}
	return tagged(rec, "important") || tagged(rec, "urgent")
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
// the latest late, then what is due earliest today.
func (s *Server) focus(l todayLists) (first, rest []todayItem) {
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
	matters := map[string]bool{}
	for _, it := range all {
		matters[it.ID] = s.important(it)
	}
	sort.SliceStable(all, func(i, j int) bool {
		if matters[all[i].ID] != matters[all[j].ID] {
			return matters[all[i].ID]
		}
		return all[i].At.Before(all[j].At)
	})
	return all[:focusOn], all[focusOn:]
}

// dumpBox is the box for everything in the person's head.
func (s *Server) dumpBox() string {
	return `<form method="post" action="/today/sort" class="sw-stack">` +
		string(s.component("textarea", map[string]any{"label": "Too much in your head? Put it all here", "name": "words", "rows": 4,
			"hint": "As it comes, in any order: things to do, to remember, to reply to. The assistant makes them tasks with their days and says which three to start with."})) +
		string(s.component("button", map[string]any{"label": "Sort it out for me", "type": "submit", "variant": "secondary"})) + `</form>`
}

// todaySort hands what the person wrote to the assistant, to sort.
func (s *Server) todaySort(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	words := strings.TrimSpace(r.PostForm.Get("words"))
	if words == "" {
		s.failed(w, r, "Nothing to sort", errors.New("write what is on your mind first, as it comes"), "/today")
		return
	}
	ask := fmt.Sprintf("Sort this out for me: make each thing to do a task, with its day when I said one, tag the ones that matter most important, and tell me the three to start with. Ask me only if something cannot be guessed.\n\n%s", words)
	if _, err := s.chatFor(r).SendFile(context.WithoutCancel(r.Context()), "", ask, ""); err != nil {
		s.forgetModel()
		s.failed(w, r, "Not sorted", err, "/today")
		return
	}
	s.tellAt(w, r, outcome{Title: "Sorted", Text: "The assistant has made your tasks; its answer is on Home, and they are on Today."}, "/#chat")
}

// focusSection is Start with these, and the rest folded, or "" when Today
// is short enough to read whole.
func (s *Server) focusSection(l todayLists, list func(string, []todayItem) string) string {
	first, rest := s.focus(l)
	if first == nil {
		return ""
	}
	out := `<p>` + template.HTMLEscapeString(fmt.Sprintf("%d tasks are late or due. Start with these three; the rest wait below.", len(first)+len(rest))) + `</p>` + list("Start with these", first)
	if body, err := s.app.Registry.RenderSlot("disclosure", map[string]any{"label": "The rest", "count": len(rest), "of": "task"}, template.HTML(list("", rest))); err == nil {
		out += string(body)
	}
	return out
}
