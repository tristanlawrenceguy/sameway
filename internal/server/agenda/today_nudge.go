package agenda

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/web"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A late task stayed late: dealing with it meant opening it, finding its
// day, choosing another, saving. Today now puts the usual answers beside
// each task (Done, Tomorrow) and, when several are late, one press that
// moves them all to today, taken back by one Undo. Nothing is asked of a
// model; the person decides, in a press.

// taskPresses are a task's presses on Today.
func (s *Service) taskPresses(it todayItem, late bool) string {
	if it.Type != "task" {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<div class="sw-cluster">`)
	type press struct {
		action, label, to string
		variant           ui.Variant
	}
	presses := []press{{"/today/done", "Done", "", ui.Secondary}, {"/today/move", "Tomorrow", "tomorrow", ui.Quiet}}
	if late {
		presses = append(presses, press{"/today/move", "Today", "today", ui.Quiet})
	}
	for _, p := range presses {
		hidden := ui.Hidden("id", it.ID)
		if p.to != "" {
			hidden = append(hidden, ui.Field{Name: "to", Value: p.to})
		}
		b.WriteString(string(s.Form(ui.Form{Action: p.action, Hidden: hidden, Button: &ui.Button{Label: p.label, Context: it.Title, Variant: p.variant}})))
	}
	b.WriteString(`</div>`)
	return b.String()
}

// lateNudge is the press that moves every late task to today, when there
// are a few.
func (s *Service) lateNudge(late []todayItem, now time.Time) string {
	n := 0
	for _, it := range late {
		if it.Type == "task" {
			n++
		}
	}
	if n < 2 {
		return ""
	}
	oldest := when.Day(late[0].At, now)
	return string(s.Form(ui.Form{Action: "/today/late", Class: "sw-stack",
		Body:   template.HTML(`<p>` + fmt.Sprintf("%d tasks are late, the oldest from %s.", n, oldest) + `</p>`),
		Button: &ui.Button{Label: fmt.Sprintf("Move all %d to today", n), Variant: ui.Secondary}}))
}

// dayFor is a task's due moved to day: the same time on that day, or the
// day alone when it had no time.
func dayFor(at time.Time, allDay bool, day time.Time) string {
	if allDay {
		return day.Format("2006-01-02")
	}
	return time.Date(day.Year(), day.Month(), day.Day(), at.Hour(), at.Minute(), 0, 0, day.Location()).Format(time.RFC3339)
}

// dueOf is a task's due, read.
func (s *Service) dueOf(id string, now time.Time) (title string, at time.Time, allDay bool, err error) {
	rec, err := s.app.Store.Get("task", id)
	if err != nil {
		return "", at, false, errors.New("that task is gone")
	}
	title, _ = rec.Fields["title"].(string)
	v, _ := rec.Fields["due"].(string)
	at, allDay, _ = when.Stored(v, now)
	return title, at, allDay, nil
}

func (s *Service) todayDone(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := r.PostForm.Get("id")
	title, _, _, err := s.dueOf(id, s.Now())
	if err == nil {
		var act string
		_, act, err = records.WriteAs(s.app.Store, s.Who(r), "updated", "task", id, map[string]any{"done": true})
		if err == nil {
			s.TellAt(w, r, web.Outcome{Title: "Done", Text: title + " is done.", Undo: act, Of: title}, "/today")
			return
		}
	}
	s.Failed(w, r, "Not changed", err, "/today")
}

func (s *Service) todayMove(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	now := s.Now()
	id := r.PostForm.Get("id")
	day, word := now, "today"
	if r.PostForm.Get("to") == "tomorrow" {
		day, word = now.AddDate(0, 0, 1), "tomorrow"
	}
	title, at, allDay, err := s.dueOf(id, now)
	if err == nil {
		var act string
		_, act, err = records.WriteAs(s.app.Store, s.Who(r), "updated", "task", id, map[string]any{"due": dayFor(at, allDay, day)})
		if err == nil {
			s.TellAt(w, r, web.Outcome{Title: "Moved", Text: title + " is due " + word + ", " + day.Format("Monday 2 January") + ".", Undo: act, Of: title}, "/today")
			return
		}
	}
	s.Failed(w, r, "Not moved", err, "/today")
}

// todayLate moves every late task to today, as one change.
func (s *Service) todayLate(w http.ResponseWriter, r *http.Request) {
	now := s.Now()
	var ops []records.Op
	for _, it := range s.today(now).Late {
		if _, err := s.app.Store.Get("task", it.ID); err == nil && it.Type == "task" {
			ops = append(ops, records.Op{Type: "task", ID: it.ID, After: map[string]any{"due": dayFor(it.At, it.AllDay, now)}})
		}
	}
	if len(ops) == 0 {
		s.TellAt(w, r, web.Outcome{Title: "Nothing to move", Text: "No task is late."}, "/today")
		return
	}
	// All or none: one that cannot move leaves them all where they were.
	detail := fmt.Sprintf("%d late tasks to today", len(ops))
	act, _, err := s.Apply(r, records.Change{Action: "rescheduled", Component: "task", Detail: detail}, ops...)
	if err != nil {
		s.Failed(w, r, "Not moved", err, "/today")
		return
	}
	s.TellAt(w, r, web.Outcome{Title: "Moved", Text: fmt.Sprintf("%d tasks are due today.", len(ops)), Undo: act, Of: detail}, "/today")
}
