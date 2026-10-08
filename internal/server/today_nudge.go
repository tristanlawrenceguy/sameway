package server

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A late task stayed late: dealing with it meant opening it, finding its
// day, choosing another, saving. Today now puts the usual answers beside
// each task (Done, Tomorrow) and, when several are late, one press that
// moves them all to today, taken back by one Undo. Nothing is asked of a
// model; the person decides, in a press.

// taskPresses are a task's presses on Today.
func (s *Server) taskPresses(it todayItem, late bool) string {
	if it.Type != "task" {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<div class="sw-cluster">`)
	presses := []struct{ action, label, to, variant string }{{"/today/done", "Done", "", "secondary"}, {"/today/move", "Tomorrow", "tomorrow", "quiet"}}
	if late {
		presses = append(presses, struct{ action, label, to, variant string }{"/today/move", "Today", "today", "quiet"})
	}
	for _, p := range presses {
		b.WriteString(`<form method="post" action="` + p.action + `"><input type="hidden" name="id" value="` + it.ID + `">`)
		if p.to != "" {
			b.WriteString(`<input type="hidden" name="to" value="` + p.to + `">`)
		}
		b.WriteString(string(s.component("button", map[string]any{"label": p.label, "context": it.Title, "type": "submit", "variant": p.variant})) + `</form>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

// lateNudge is the press that moves every late task to today, when there
// are a few.
func (s *Server) lateNudge(late []todayItem, now time.Time) string {
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
	return `<form method="post" action="/today/late" class="sw-stack"><p>` + fmt.Sprintf("%d tasks are late, the oldest from %s.", n, oldest) + `</p>` +
		string(s.component("button", map[string]any{"label": fmt.Sprintf("Move all %d to today", n), "type": "submit", "variant": "secondary"})) + `</form>`
}

// dayFor is a task's due moved to day: the same time on that day, or the
// day alone when it had no time.
func dayFor(at time.Time, allDay bool, day time.Time) string {
	if allDay {
		return day.Format("2006-01-02")
	}
	return time.Date(day.Year(), day.Month(), day.Day(), at.Hour(), at.Minute(), 0, 0, time.Local).Format(time.RFC3339)
}

// dueOf is a task's due, read.
func (s *Server) dueOf(id string, now time.Time) (title string, at time.Time, allDay bool, err error) {
	rec, err := s.app.Store.Get("task", id)
	if err != nil {
		return "", at, false, errors.New("that task is gone")
	}
	title, _ = rec.Fields["title"].(string)
	v, _ := rec.Fields["due"].(string)
	at, allDay, _ = when.Parse(v, now)
	return title, at.In(time.Local), allDay, nil
}

func (s *Server) todayDone(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id := r.PostForm.Get("id")
	title, _, _, err := s.dueOf(id, time.Now())
	if err == nil {
		var act string
		_, act, err = records.WriteAs(s.app.Store, s.who(r), "updated", "task", id, map[string]any{"done": true})
		if err == nil {
			s.tellAt(w, r, outcome{Title: "Done", Text: title + " is done.", Undo: act, Of: title}, "/today")
			return
		}
	}
	s.failed(w, r, "Not changed", err, "/today")
}

func (s *Server) todayMove(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	now := time.Now()
	id := r.PostForm.Get("id")
	day, word := now, "today"
	if r.PostForm.Get("to") == "tomorrow" {
		day, word = now.AddDate(0, 0, 1), "tomorrow"
	}
	title, at, allDay, err := s.dueOf(id, now)
	if err == nil {
		var act string
		_, act, err = records.WriteAs(s.app.Store, s.who(r), "updated", "task", id, map[string]any{"due": dayFor(at, allDay, day)})
		if err == nil {
			s.tellAt(w, r, outcome{Title: "Moved", Text: title + " is due " + word + ", " + day.Format("Monday 2 January") + ".", Undo: act, Of: title}, "/today")
			return
		}
	}
	s.failed(w, r, "Not moved", err, "/today")
}

// todayLate moves every late task to today, as one change.
func (s *Server) todayLate(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	var batch []records.BatchItem
	for _, it := range s.today(now).Late {
		if it.Type != "task" {
			continue
		}
		rec, err := s.app.Store.Get("task", it.ID)
		if err != nil {
			continue
		}
		before := rec.Fields
		if _, err := s.app.Store.Update("task", it.ID, map[string]any{"due": dayFor(it.At, it.AllDay, now)}); err != nil {
			s.failed(w, r, "Not all moved", err, "/today")
			return
		}
		batch = append(batch, records.BatchItem{Type: "task", ID: it.ID, Before: before})
	}
	if len(batch) == 0 {
		s.tellAt(w, r, outcome{Title: "Nothing to move", Text: "No task is late."}, "/today")
		return
	}
	detail := fmt.Sprintf("%d late tasks to today", len(batch))
	who := s.who(r)
	act := records.Record(s.app.Store, who.Actor, records.Change{Action: "rescheduled", Component: "task", Detail: detail, Before: records.Batch(batch), By: who.By, Via: who.Via, ByLogin: who.ByLogin})
	s.tellAt(w, r, outcome{Title: "Moved", Text: fmt.Sprintf("%d tasks are due today.", len(batch)), Undo: act, Of: detail}, "/today")
}

func (s *Server) todayNudgeRoutes(m *http.ServeMux) {
	s.todayFocusRoutes(m) // today_focus.go
	m.HandleFunc("POST /today/done", s.todayDone)
	m.HandleFunc("POST /today/move", s.todayMove)
	m.HandleFunc("POST /today/late", s.todayLate)
}
