package server

import (
	"html/template"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/relate"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// What is on today was spread over the task list, the calendar and the
// reminders, and found by looking at each. Today puts it on one page:
// what is late, the tasks due today, the day's events and reminders, each
// a link. A morning brief, at a time the owner sets on Help, sends the
// same as one notification, desktop and phone alike, leading to the page.
// It is read from the records, no model asked, so it is there at once
// and costs nothing.

// todayItem is one thing on today's page.
type todayItem struct {
	Title, Href, When string
	At                time.Time
	Type, ID          string
	AllDay            bool
}

// todayLists are what is late and what is today, by kind.
type todayLists struct {
	Late, Tasks, Events, Reminders []todayItem
}

func (l todayLists) count() int {
	return len(l.Late) + len(l.Tasks) + len(l.Events) + len(l.Reminders)
}

// today reads the records for the day now falls on.
func (s *Server) today(now time.Time) todayLists {
	var out todayLists
	day := now.Format("2006-01-02")
	add := func(typ, field string, keep func(*store.Record, time.Time, bool) *[]todayItem) {
		t, ok := s.app.Types.Get(typ)
		if !ok {
			return
		}
		recs, _ := s.app.Store.List(typ, store.ListOptions{})
		for _, r := range recs {
			v, _ := r.Fields[field].(string)
			// Stored, not Parse: a whole day is kept as midnight UTC, which
			// read as a moment is 2am here, or yesterday west of Greenwich.
			at, allDay, ok := when.Stored(v, now.In(time.Local))
			if v == "" || !ok {
				continue
			}
			at = at.In(time.Local)
			list := keep(r, at, at.Format("2006-01-02") == day)
			if list == nil {
				continue
			}
			it := todayItem{Title: s.title(t, r), Href: relate.Page(typ, r.ID), At: at, Type: typ, ID: r.ID, AllDay: allDay}
			it.When = todayWhen(at, allDay, at.Format("2006-01-02") < day, now, s.h24())
			*list = append(*list, it)
		}
	}
	add("task", "due", func(r *store.Record, at time.Time, isToday bool) *[]todayItem {
		if done, _ := r.Fields["done"].(bool); done {
			return nil
		}
		switch {
		case isToday:
			return &out.Tasks
		case at.Before(now):
			return &out.Late
		}
		return nil
	})
	add("event", "starts", func(_ *store.Record, _ time.Time, isToday bool) *[]todayItem {
		if isToday {
			return &out.Events
		}
		return nil
	})
	add("reminder", "at", func(r *store.Record, _ time.Time, isToday bool) *[]todayItem {
		if st, _ := r.Fields["state"].(string); isToday && st != "dismissed" && st != "done" {
			return &out.Reminders
		}
		return nil
	})
	for _, l := range []*[]todayItem{&out.Late, &out.Tasks, &out.Events, &out.Reminders} {
		sort.SliceStable(*l, func(i, j int) bool { return (*l)[i].At.Before((*l)[j].At) })
	}
	return out
}

func (s *Server) todayPage(w http.ResponseWriter, r *http.Request) {
	now := s.now()
	l := s.today(now)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	list := func(title string, items []todayItem) string {
		if len(items) == 0 {
			return ""
		}
		var b strings.Builder
		if title != "" {
			b.WriteString(`<h2>` + title + `</h2>`)
		}
		b.WriteString(`<ul class="sw-plain sw-rows">`)
		for _, it := range items {
			b.WriteString(`<li class="sw-row"><p class="sw-row__title"><a class="sw-link" href="` + template.HTMLEscapeString(it.Href) + `">` + template.HTMLEscapeString(it.Title) + `</a>`)
			if it.When != "" {
				b.WriteString(` ` + timeHTML("sw-muted sw-small", it.stamp(), "", template.HTML(template.HTMLEscapeString(it.When))))
			}
			b.WriteString(`</p>` + s.taskPresses(it, it.At.Before(dayStart)) + `</li>`) // today_nudge.go
		}
		b.WriteString(`</ul>`)
		return b.String()
	}
	var b strings.Builder
	if focus := s.focusSection(l, list); focus != "" { // today_focus.go
		b.WriteString(focus)
	} else {
		b.WriteString(s.lateNudge(l.Late, now)) // today_nudge.go
		b.WriteString(list("Late", l.Late) + list("Tasks", l.Tasks))
	}
	b.WriteString(list("Events", l.Events) + list("Reminders", l.Reminders))
	// What waits for the person, from the one list every reader reads
	// (chat/waiting.go), each kind in its own place.
	waiting := s.chatFor(r).Waiting()
	mail := s.questionSection(waiting) + s.inbox.Today(waiting) // proposal.go, inbox/inbox.go
	b.WriteString(mail)
	if l.count() == 0 && mail == "" {
		b.WriteString(string(s.part(ui.Empty{Message: "Nothing is due today, and nothing is late."})))
	}
	b.WriteString(s.dumpBox()) // today_focus.go
	s.page(w, r, "Today", template.HTML(b.String()), pageOptions{Lede: template.HTML(timeHTML("", now.Format("2006-01-02"), "", template.HTML(now.Format("Monday 2 January"))))})
}

// todayWhen is what an item on Today says beside its title: its time, if
// it has one; and for one from an earlier day, that day too, so a late
// task says when it was due, never a bare 3pm that reads as today's.
func todayWhen(at time.Time, allDay, earlier bool, now time.Time, h24 bool) string {
	switch {
	case earlier && allDay:
		return when.Day(at, now)
	case earlier:
		return when.At(at, now, h24)
	case allDay:
		return ""
	}
	return when.Clock(at, h24)
}

// stamp is an item's day as a machine reads it, for its <time>: the date
// of a whole day, the moment of one with a time.
func (it todayItem) stamp() string {
	if it.AllDay {
		return it.At.Format("2006-01-02")
	}
	return it.At.Format(time.RFC3339)
}

// todayListed is whether Today is in the navigation: when the workspace
// keeps something Today gathers (tasks, events or reminders).
func (s *Server) todayListed() bool {
	for _, typ := range []string{"task", "event", "reminder"} {
		if _, ok := s.app.Types.Get(typ); ok {
			return true
		}
	}
	return false
}
