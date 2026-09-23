package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// The clock: the time, and reminders. A reminder is a record (an alarm
// at a time, or a timer that ends at one) that the clock sets, lists,
// and rings when its time comes. Ringing happens here: a page with a
// clock listens on /clock/stream, and every few seconds the server marks
// what is due as rung and tells the page, which shows it, sounds, and
// notifies. A rung reminder stays until dismissed or given five more
// minutes, and it shows on the calendar like anything with a day.

const clockComponent = "clock"

// ReminderType is the content type the clock sets and rings.
const ReminderType = "reminder"

// resolveClock fills what the block leaves to the moment: the time, what
// is ringing, what is coming, and what is on today.
func (s *Server) resolveClock(props map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range props {
		out[k] = v
	}
	now := time.Now()
	out["now"] = now.Format(time.RFC3339)
	out["time"] = now.Format("15:04")
	out["date"] = now.Format("Monday 2 January")
	ringing, upcoming := []any{}, []any{}
	if t, ok := s.app.Types.Get(ReminderType); ok {
		recs, _ := query.Filter(s.app.Store, t, nil, "at", 0, now)
		for _, rec := range recs {
			item := map[string]any{"id": rec.ID, "title": s.title(t, rec), "href": "/t/" + ReminderType + "/" + rec.ID}
			switch rec.Fields["state"] {
			case "rang":
				ringing = append(ringing, item)
			case "set":
				at, _ := rec.Fields["at"].(string)
				item["when"] = when.Short(at, now)
				item["kind"], _ = rec.Fields["kind"].(string)
				if len(upcoming) < 6 {
					upcoming = append(upcoming, item)
				}
			}
		}
	}
	out["ringing"], out["upcoming"], out["today"] = ringing, upcoming, s.onToday(now)
	return out
}

// onToday is what falls today across every listed type with a day, in
// time order: the calendar's view of the day, in a few lines.
func (s *Server) onToday(now time.Time) []any {
	day := now.Format("2006-01-02")
	var items []map[string]any
	for _, t := range s.app.Types.Types {
		if t.Internal || t.Name == ReminderType || !s.listed(t) {
			continue
		}
		field := dateField(t, nil)
		if field == "" {
			continue
		}
		recs, err := query.Filter(s.app.Store, t, nil, field, 0, now)
		if err != nil {
			continue
		}
		for _, rec := range recs {
			v, _ := rec.Fields[field].(string)
			ts, err := time.Parse(time.RFC3339, v)
			if err != nil {
				continue
			}
			item := map[string]any{"label": s.title(t, rec), "href": "/t/" + t.Name + "/" + rec.ID}
			if strings.HasSuffix(v, "T00:00:00Z") {
				if ts.UTC().Format("2006-01-02") != day {
					continue
				}
			} else {
				if ts.Local().Format("2006-01-02") != day {
					continue
				}
				item["time"] = ts.Local().Format("15:04")
			}
			items = append(items, item)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		a, _ := items[i]["time"].(string)
		b, _ := items[j]["time"].(string)
		return a < b
	})
	out := make([]any, 0, len(items))
	for i, it := range items {
		if i == 8 {
			break
		}
		out = append(out, it)
	}
	return out
}

// clockSet makes a reminder from the clock's forms: minutes from now as
// a timer, or a time of day (today, or tomorrow if it has passed) as an
// alarm, named for what it is for.
func (s *Server) clockSet(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	now := time.Now()
	fields := map[string]any{"state": "set"}
	if minutes, err := strconv.Atoi(r.PostForm.Get("minutes")); err == nil && r.PostForm.Get("at") == "" {
		if minutes < 1 || minutes > 1440 {
			s.app.Chat.Notice("A timer takes between 1 and 1440 minutes.")
			http.Redirect(w, r, backFrom(r), http.StatusSeeOther)
			return
		}
		fields["kind"], fields["at"] = "timer", when.Store(now.Add(time.Duration(minutes)*time.Minute), false)
		fields["title"] = fmt.Sprintf("%d minute timer", minutes)
		if minutes == 1 {
			fields["title"] = "1 minute timer"
		}
	} else {
		// The time the way a person says it: 7:30, 7pm, tomorrow 6am. A
		// time already past today is tomorrow.
		at, dayOnly, ok := when.Parse(r.PostForm.Get("at"), now)
		if !ok || dayOnly {
			s.app.Chat.Notice("An alarm needs a time of day, such as 7:30 or 7pm.")
			http.Redirect(w, r, backFrom(r), http.StatusSeeOther)
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
		s.app.Chat.Notice(err.Error())
	} else {
		title, _ := fields["title"].(string)
		s.record(r, chat.Change{Action: "created", Component: ReminderType, ID: rec.ID, Detail: title})
	}
	http.Redirect(w, r, backFrom(r), http.StatusSeeOther)
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
		s.fail(w, err)
		return
	}
	if _, err := s.app.Store.Update(ReminderType, rec.ID, fields); err != nil {
		s.app.Chat.Notice(err.Error())
	} else {
		t, _ := s.app.Types.Get(ReminderType)
		s.record(r, chat.Change{Action: action, Component: ReminderType, ID: rec.ID, Detail: s.title(t, rec), Before: rec.Fields})
	}
	http.Redirect(w, r, backFrom(r), http.StatusSeeOther)
}

// backFrom is the page a clock form came from, when it is one of ours.
func backFrom(r *http.Request) string {
	if u, err := url.Parse(r.Referer()); err == nil && u.Host == r.Host && strings.HasPrefix(u.Path, "/") {
		return u.RequestURI()
	}
	return "/"
}

// clockStream tells an open page about reminders as they ring, as
// server-sent events: every few seconds, whatever has rung and this page
// has not been told of yet. The ringing itself is the server's, in
// ring.go, whether or not a page is open.
func (s *Server) clockStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming is not possible here", http.StatusNotImplemented)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "event: hello\ndata: {}\n\n")
	flusher.Flush()
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	told := map[string]bool{}
	for {
		for _, rec := range s.ringing() {
			if told[rec.ID] {
				continue
			}
			told[rec.ID] = true
			t, _ := s.app.Types.Get(ReminderType)
			body, _ := json.Marshal(map[string]any{"id": rec.ID, "title": s.title(t, rec), "href": "/t/" + ReminderType + "/" + rec.ID})
			fmt.Fprintf(w, "event: ring\ndata: %s\n\n", body)
			flusher.Flush()
		}
		select {
		case <-r.Context().Done():
			return
		case <-tick.C:
		}
	}
}

// ringing is every reminder that has rung and not been dismissed.
func (s *Server) ringing() []*store.Record {
	t, ok := s.app.Types.Get(ReminderType)
	if !ok {
		return nil
	}
	recs, err := query.Filter(s.app.Store, t, []string{"state=rang"}, "at", 0, time.Now())
	if err != nil {
		return nil
	}
	return recs
}

// ring marks every reminder whose time has come as rung, once, and says
// which. The log has it too, as the system's doing.
func (s *Server) ring(now time.Time) []*store.Record {
	t, ok := s.app.Types.Get(ReminderType)
	if !ok {
		return nil
	}
	recs, err := query.Filter(s.app.Store, t, []string{"state=set"}, "at", 0, now)
	if err != nil {
		return nil
	}
	rang := s.nudges(now)
	for _, rec := range recs {
		at, _ := rec.Fields["at"].(string)
		ts, err := time.Parse(time.RFC3339, at)
		if err != nil || ts.After(now) {
			continue
		}
		if _, err := s.app.Store.Update(ReminderType, rec.ID, map[string]any{"state": "rang"}); err != nil {
			log.Printf("clock: %v", err)
			continue
		}
		chat.Record(s.app.Store, "system", chat.Change{Action: "rang", Component: ReminderType, ID: rec.ID, Detail: s.title(t, rec)})
		rang = append(rang, rec)
	}
	return rang
}
