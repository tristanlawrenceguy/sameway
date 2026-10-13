package agenda

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// Letting go of what has been late a long time. A long Today is mostly old:
// errands from a fortnight ago, each still saying it is overdue, the most
// overdue among the three to start with. Moving them all to today puts the
// same pile on tomorrow; going through them one by one is the curating a
// person gets worse at the more overwhelmed they are (the person this page
// was made for). So when two or more are long late, one press, above the
// three, keeps them with no date: still tasks, found in their list, no
// longer late, all back with one Undo; and the three are then worked out
// from what is left. What the person marked as mattering is never among
// them.

// longLate is how long late a task is before it is offered to let go.
const longLateDays = 14

// longLate is the tasks among those given late longer than longLateDays
// and not marked as mattering.
func (s *Service) longLate(rest []todayItem) []todayItem {
	cut := s.Now().AddDate(0, 0, -longLateDays)
	var out []todayItem
	for _, it := range rest {
		if it.Type == "task" && !it.Soon && !it.At.IsZero() && it.At.Before(cut) && !s.important(it) {
			out = append(out, it)
		}
	}
	return out
}

// letGoOffer is the line that offers it, or "".
func (s *Service) letGoOffer(rest []todayItem) string {
	old := s.longLate(rest)
	if len(old) < 2 {
		return ""
	}
	words := fmt.Sprintf("%d of them have been late more than two weeks.", len(old))
	return `<div class="sw-cluster sw-quiet"><span>` + template.HTMLEscapeString(words) + `</span>` +
		string(s.Form(ui.Form{Action: "/today/let-go", From: "/today", Button: &ui.Button{Label: "Keep them, with no date", Variant: ui.Secondary}})) + `</div>`
}

// todayLetGo keeps every long-late task with no date, as one change.
func (s *Service) todayLetGo(w http.ResponseWriter, r *http.Request) {
	first, rest := s.focus(s.today(s.Now()))
	var ops []records.Op
	for _, it := range s.longLate(append(first, rest...)) {
		ops = append(ops, records.Op{Type: "task", ID: it.ID, After: map[string]any{"due": nil}})
	}
	if len(ops) == 0 {
		s.TellAt(w, r, web.Outcome{Title: "Nothing to keep", Text: "No task has been late that long."}, "/today")
		return
	}
	detail := fmt.Sprintf("%d long-late tasks kept with no date", len(ops))
	act, _, err := s.Apply(r, records.Change{Action: "rescheduled", Component: "task", Detail: detail}, ops...)
	if err != nil {
		s.Failed(w, r, "Not changed", err, "/today")
		return
	}
	s.TellAt(w, r, web.Outcome{Title: "Kept, with no date", Text: fmt.Sprintf("%d tasks are no longer late. They are in your tasks, under No date.", len(ops)), Undo: act, Of: detail}, "/today")
}
