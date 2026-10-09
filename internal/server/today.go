package server

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"time"

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
			at, allDay, ok := when.Parse(v, now)
			if v == "" || !ok {
				continue
			}
			at = at.In(time.Local)
			list := keep(r, at, at.Format("2006-01-02") == day)
			if list == nil {
				continue
			}
			it := todayItem{Title: s.title(t, r), Href: "/t/" + typ + "/" + r.ID, At: at, Type: typ, ID: r.ID, AllDay: allDay}
			if !allDay {
				it.When = when.Clock(at, s.h24())
			} else if at.Format("2006-01-02") < day {
				it.When = when.Day(at, now)
			}
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
			b.WriteString(`<li><a class="sw-link" href="` + template.HTMLEscapeString(it.Href) + `">` + template.HTMLEscapeString(it.Title) + `</a>`)
			if it.When != "" {
				b.WriteString(` <span class="sw-muted sw-small">` + template.HTMLEscapeString(it.When) + `</span>`)
			}
			b.WriteString(s.taskPresses(it, it.At.Before(dayStart)) + `</li>`) // today_nudge.go
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
	mail := s.mailSection() + s.suggestedSection(s.sortingRefs()) + s.tagSection() // mail_sort.go, suggested.go, classify_today.go
	b.WriteString(mail)
	if l.count() == 0 && mail == "" {
		b.WriteString(string(s.part(ui.Empty{Message: "Nothing is due today, and nothing is late."})))
	}
	b.WriteString(s.dumpBox()) // today_focus.go
	s.page(w, r, "Today", template.HTML(b.String()), pageOptions{Lede: template.HTML(now.Format("Monday 2 January"))})
}

// briefWords is the morning brief as a notification says it.
func briefWords(l todayLists) (string, string) {
	var parts []string
	for _, p := range []struct {
		n         int
		one, many string
	}{{len(l.Tasks), "task", "tasks"}, {len(l.Events), "event", "events"}, {len(l.Reminders), "reminder", "reminders"}, {len(l.Late), "late", "late"}} {
		if p.n == 1 {
			parts = append(parts, "1 "+p.one)
		} else if p.n > 1 {
			parts = append(parts, fmt.Sprintf("%d %s", p.n, p.many))
		}
	}
	if len(parts) == 0 {
		return "Today", "Nothing is due today, and nothing is late."
	}
	var first []string
	for _, l := range [][]todayItem{l.Events, l.Tasks, l.Reminders, l.Late} {
		for _, it := range l {
			if len(first) < 3 {
				first = append(first, strings.TrimSpace(it.Title+" "+it.When))
			}
		}
	}
	return "Today: " + strings.Join(parts, ", "), strings.Join(first, "; ")
}

// KeepBrief sends the morning brief at the time brief.at says, once a
// day, while ctx lasts.
func (s *Server) KeepBrief(ctx context.Context) {
	go func() {
		tick := time.NewTicker(30 * time.Second)
		defer tick.Stop()
		for {
			s.briefIfDue(s.now())
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
		}
	}()
}

// briefIfDue sends today's brief when its time has come and it has not
// gone yet, and says whether it sent it.
func (s *Server) briefIfDue(now time.Time) bool {
	at := s.app.Workspace.Config.Brief.At
	if at == "" || s.notify == nil || now.Format("15:04") < at || s.app.Store.Meta("brief:sent") == now.Format("2006-01-02") {
		return false
	}
	s.app.Store.SetMeta("brief:sent", now.Format("2006-01-02"))
	title, text := briefWords(s.today(now))
	go s.notify(title, text, s.linkTo("/today"))
	return true
}

// briefLine is the brief's line on Help, for the owner.
func (s *Server) briefLine() string {
	at := s.app.Workspace.Config.Brief.At
	said := "Morning brief: none. Sameway can tell you what is on each morning, on this computer and your phone if reminders go there."
	if at != "" {
		said = "Morning brief: what is on today is sent at " + at + ", leading to Today."
	}
	return template.HTMLEscapeString(said) + " " + string(s.form(ui.Form{Action: "/brief", Class: "sw-stack",
		Body:   s.part(ui.TextField{Label: "Time of the brief", Name: "at", ID: "brief-at", Value: at, Hint: "Hours and minutes, such as 07:30. Empty sends none."}),
		Button: &ui.Button{Label: "Set the morning brief", Variant: ui.Secondary}}))
}

func (s *Server) briefSet(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	at := strings.TrimSpace(r.PostForm.Get("at"))
	if at != "" {
		if _, err := time.Parse("15:04", at); err != nil {
			s.failed(w, r, "Not set", errors.New("give the time as hours and minutes, such as 07:30"), "/help")
			return
		}
	}
	if s.app.Records.SetSetting == nil {
		s.failed(w, r, "Not set", errors.New("this workspace has no settings file"), "/help")
		return
	}
	if err := s.app.Records.SetSetting("brief.at", at); err != nil {
		s.failed(w, r, "Not set", err, "/help")
		return
	}
	o := outcome{Title: "Morning brief at " + at, Text: "What is on today comes at " + at + " each day, leading to Today."}
	if at == "" {
		o = outcome{Title: "No morning brief", Text: "Today still shows what is on."}
	}
	s.tell(w, r, o, "/help")
}
