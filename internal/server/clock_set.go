package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// Setting, cancelling and putting off a reminder. Each says what it did in
// words, and when the reminder now rings, because the time was typed the
// way a person says it and read by the server: 7 is 07:00, and a time
// already past is tomorrow. A reading nobody is told of goes unnoticed
// until the alarm does not ring. Each can be undone.

// clockSet makes a reminder from the clock's forms: minutes from now as
// a timer, or a time of day (today, or tomorrow if it has passed) as an
// alarm, named for what it is for.
func (s *Server) clockSet(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	now := time.Now()
	fields := map[string]any{"state": "set"}
	var at time.Time
	if minutes, err := strconv.Atoi(r.PostForm.Get("minutes")); err == nil && r.PostForm.Get("at") == "" {
		if minutes < 1 || minutes > 1440 {
			s.tell(w, r, outcome{Failed: true, Title: "Timer not set", Text: "A timer takes between 1 and 1440 minutes."}, "/")
			return
		}
		at = now.Add(time.Duration(minutes) * time.Minute)
		fields["kind"], fields["at"] = "timer", when.Store(at, false)
		fields["title"] = fmt.Sprintf("%d minute timer", minutes)
	} else {
		// The time the way a person says it: 7:30, 7pm, tomorrow 6am. A
		// time already past today is tomorrow.
		var ok, dayOnly bool
		at, dayOnly, ok = when.Parse(r.PostForm.Get("at"), now)
		if !ok || dayOnly {
			s.tell(w, r, outcome{Failed: true, Title: "Alarm not set", Text: "An alarm needs a time of day, such as 7:30 or 7pm."}, "/")
			return
		}
		if !at.After(now) {
			at = at.AddDate(0, 0, 1)
		}
		fields["kind"], fields["at"] = "alarm", when.Store(at, false)
		fields["title"] = strings.TrimSpace(r.PostForm.Get("title"))
		if fields["title"] == "" {
			fields["title"] = "Alarm"
		}
	}
	// A reminder set from a record's page is about it, and named for it.
	if about := r.PostForm.Get("about"); about != "" {
		if _, _, ok := s.aboutOf(about); ok {
			fields["about"] = about
			if title := strings.TrimSpace(r.PostForm.Get("title")); title != "" {
				fields["title"] = title
			}
		}
	}
	rec, err := s.app.Store.Create(ReminderType, fields)
	if err != nil {
		s.failed(w, r, "Not set", err, "/")
		return
	}
	title, _ := fields["title"].(string)
	undo := s.record(r, chat.Change{Action: "created", Component: ReminderType, ID: rec.ID, Detail: title})
	what := "Alarm set"
	if fields["kind"] == "timer" {
		what = "Timer set"
	}
	s.tellAt(w, r, outcome{Title: what, Text: title + " rings " + ringsWhen(at, now) + ".", Undo: undo, Of: title}, backFrom(r))
}

// clockDone dismisses a reminder, rung or not.
func (s *Server) clockDone(w http.ResponseWriter, r *http.Request) {
	s.setReminder(w, r, map[string]any{"state": "done"}, "done")
}

// clockSnooze gives a reminder five more minutes.
func (s *Server) clockSnooze(w http.ResponseWriter, r *http.Request) {
	s.setReminder(w, r, map[string]any{"state": "set", "at": when.Store(time.Now().Add(5*time.Minute), false)}, "snoozed")
}

func (s *Server) setReminder(w http.ResponseWriter, r *http.Request, fields map[string]any, action string) {
	rec, err := s.app.Store.Get(ReminderType, r.PathValue("id"))
	if err != nil {
		s.failed(w, r, "Reminder not changed", err, "/")
		return
	}
	if _, err := s.app.Store.Update(ReminderType, rec.ID, fields); err != nil {
		s.failed(w, r, "Reminder not changed", err, "/")
		return
	}
	t, _ := s.app.Types.Get(ReminderType)
	title := s.title(t, rec)
	undo := s.record(r, chat.Change{Action: action, Component: ReminderType, ID: rec.ID, Detail: title, Before: rec.Fields})
	o := outcome{Title: title + " dismissed", Undo: undo}
	switch {
	case action == "snoozed":
		now := time.Now()
		o.Title, o.Text = title+": 5 more minutes", "Rings again "+ringsWhen(now.Add(5*time.Minute), now)+"."
		o.Of = o.Title
	case rec.Fields["state"] == "set":
		o.Title = title + " cancelled"
	}
	s.tellAt(w, r, o, backFrom(r))
}

// backFrom is the page a clock form came from, when it is one of ours.
func backFrom(r *http.Request) string {
	if u, err := url.Parse(r.Referer()); err == nil && u.Host == r.Host && strings.HasPrefix(u.Path, "/") {
		return u.RequestURI()
	}
	return "/"
}

// dayOf is the day of a moment in the words the clock uses beside its
// time: nothing for today, Tomorrow, a weekday within the week, a date
// after that.
func dayOf(at, now time.Time) string {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	at = at.In(now.Location())
	d := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, now.Location())
	switch diff := int(d.Sub(today).Hours() / 24); {
	case diff == 0:
		return ""
	case diff == 1:
		return "Tomorrow"
	case diff > 1 && diff < 7:
		return d.Format("Monday")
	}
	return d.Format("2 Jan")
}

// ringsWhen is when a reminder rings, as it ends a sentence: at 14:30,
// tomorrow at 07:00, on Monday at 09:00.
func ringsWhen(at, now time.Time) string {
	clock := "at " + at.In(now.Location()).Format("15:04")
	switch day := dayOf(at, now); day {
	case "":
		return clock
	case "Tomorrow":
		return "tomorrow " + clock
	default:
		return "on " + day + " " + clock
	}
}
