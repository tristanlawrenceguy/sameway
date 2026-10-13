package agenda

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// The morning brief is Today as one notification, once a day at the time
// the owner sets on Help, desktop and phone alike, leading to the page.
// It interrupts, so it earns it: a day with nothing on sends nothing (W3C
// COGA, Limit interruptions; NN/g, irrelevant notifications are the ones
// people turn off), and when Today starts with three, the brief names
// those three, so the page and the brief say the same. What waits for the
// person is counted from the list Today shows (chat/waiting.go), so a day
// with emails to answer and nothing due still says so.

// briefWords is the morning brief as a notification says it: the counts,
// then the three to start with when Today has them, else the first few,
// else the first few things waiting.
func briefWords(l todayLists, start []todayItem, waiting []chat.Waiting) (string, string) {
	var parts []string
	for _, p := range []struct {
		n         int
		one, many string
	}{{len(l.Tasks), "task", "tasks"}, {len(l.Events), "event", "events"}, {len(l.Reminders), "reminder", "reminders"}, {len(l.Late), "late", "late"}, {len(waiting), "waiting for you", "waiting for you"}} {
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
	for _, w := range waiting {
		if len(first) < 3 && l.count() == 0 {
			first = append(first, w.Words)
		}
	}
	return "Today: " + strings.Join(parts, ", "), strings.Join(first, "; ")
}

// BriefNext is when the brief is next due after now: the time brief.at
// says, today or tomorrow, read each time so a change applies. It is the
// zero time when none is set.
func (s *Service) BriefNext(now time.Time) time.Time {
	at, err := time.Parse("15:04", s.app.Workspace.Config.Brief.At)
	if err != nil {
		return time.Time{}
	}
	next := time.Date(now.Year(), now.Month(), now.Day(), at.Hour(), at.Minute(), 0, 0, now.Location())
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

// BriefIfDue sends today's brief when its time has come, it has not gone
// yet and there is something on, and says whether it sent it.
func (s *Service) BriefIfDue(now time.Time) bool {
	at := s.app.Workspace.Config.Brief.At
	if at == "" || !s.CanNotify() || now.Format("15:04") < at || s.app.Store.Meta("brief:sent") == now.Format("2006-01-02") {
		return false
	}
	s.app.Store.SetMeta("brief:sent", now.Format("2006-01-02"))
	l, waiting := s.today(now), s.app.Chat.Waiting()
	if l.count() == 0 && len(waiting) == 0 {
		return false
	}
	start, _ := s.focus(l)
	title, text := briefWords(l, start, waiting)
	s.Notify(title, text, "/today")
	return true
}

// BriefLine is the brief's line on Help, for the owner.
func (s *Service) BriefLine() string {
	at := s.app.Workspace.Config.Brief.At
	said := "Morning brief: none. Sameway can tell you what is on each morning, on this computer and your phone if reminders go there."
	if at != "" {
		said = "Morning brief: what is on today is sent at " + at + ", leading to Today. A day with nothing on sends nothing."
	}
	return template.HTMLEscapeString(said) + " " + string(s.Form(ui.Form{Action: "/brief", Class: "sw-stack",
		Body:   s.Part(ui.TextField{Label: "Time of the brief", Name: "at", ID: "brief-at", Value: at, Hint: "Hours and minutes, such as 07:30. Empty sends none."}),
		Button: &ui.Button{Label: "Set the morning brief", Variant: ui.Secondary}}))
}

func (s *Service) briefSet(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	at := strings.TrimSpace(r.PostForm.Get("at"))
	if at != "" {
		if _, err := time.Parse("15:04", at); err != nil {
			s.Failed(w, r, "Not set", errors.New("give the time as hours and minutes, such as 07:30"), "/help")
			return
		}
	}
	if s.app.Records.SetSetting == nil {
		s.Failed(w, r, "Not set", errors.New("this workspace has no settings file"), "/help")
		return
	}
	if err := s.app.Records.SetSetting("brief.at", at); err != nil {
		s.Failed(w, r, "Not set", err, "/help")
		return
	}
	o := web.Outcome{Title: "Morning brief at " + at, Text: "What is on today comes at " + at + " each day, leading to Today."}
	if at == "" {
		o = web.Outcome{Title: "No morning brief", Text: "Today still shows what is on."}
	}
	s.Tell(w, r, o, "/help")
}
