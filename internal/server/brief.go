package server

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// The morning brief is Today as one notification, once a day at the time
// the owner sets on Help, desktop and phone alike, leading to the page.
// It interrupts, so it earns it: a day with nothing on sends nothing (W3C
// COGA, Limit interruptions; NN/g, irrelevant notifications are the ones
// people turn off), and when Today starts with three, the brief names
// those three, so the page and the brief say the same.

// briefWords is the morning brief as a notification says it: the counts,
// then the three to start with when Today has them, else the first few.
func briefWords(l todayLists, start []todayItem) (string, string) {
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
	if start != nil {
		var titles []string
		for _, it := range start {
			titles = append(titles, it.Title)
		}
		return "Today: " + strings.Join(parts, ", "), "Start with " + strings.Join(titles, "; ")
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

// briefIfDue sends today's brief when its time has come, it has not gone
// yet and there is something on, and says whether it sent it.
func (s *Server) briefIfDue(now time.Time) bool {
	at := s.app.Workspace.Config.Brief.At
	if at == "" || s.notify == nil || now.Format("15:04") < at || s.app.Store.Meta("brief:sent") == now.Format("2006-01-02") {
		return false
	}
	s.app.Store.SetMeta("brief:sent", now.Format("2006-01-02"))
	l := s.today(now)
	if l.count() == 0 {
		return false
	}
	start, _ := s.focus(l)
	title, text := briefWords(l, start)
	go s.notify(title, text, s.linkTo("/today"))
	return true
}

// briefLine is the brief's line on Help, for the owner.
func (s *Server) briefLine() string {
	at := s.app.Workspace.Config.Brief.At
	said := "Morning brief: none. Sameway can tell you what is on each morning, on this computer and your phone if reminders go there."
	if at != "" {
		said = "Morning brief: what is on today is sent at " + at + ", leading to Today. A day with nothing on sends nothing."
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
