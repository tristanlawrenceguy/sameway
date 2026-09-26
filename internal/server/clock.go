package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/store"
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
// is ringing, and what is coming.
func (s *Server) resolveClock(props map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range props {
		out[k] = v
	}
	now := time.Now()
	out["now"] = now.Format(time.RFC3339)
	out["time"] = now.Format("15:04")
	out["date"] = now.Format("Monday 2 January")
	ringing := []any{}
	var next []coming
	if t, ok := s.app.Types.Get(ReminderType); ok {
		recs, _ := query.Filter(s.app.Store, t, nil, "at", 0, now)
		for _, rec := range recs {
			item := map[string]any{"id": rec.ID, "title": s.title(t, rec), "href": "/t/" + ReminderType + "/" + rec.ID}
			switch rec.Fields["state"] {
			case "rang":
				if text, _ := s.ringWords(rec); rec.Fields["about"] != nil && text != item["title"] {
					item["text"] = text
				}
				ringing = append(ringing, item)
			case "set":
				v, _ := rec.Fields["at"].(string)
				at, err := time.Parse(time.RFC3339, v)
				if err != nil {
					continue
				}
				item["day"], item["time"] = dayOf(at, now), at.In(now.Location()).Format("15:04")
				item["kind"], _ = rec.Fields["kind"].(string)
				next = append(next, coming{at, item})
			}
		}
	}
	next = append(next, s.onToday(now)...)
	out["ringing"], out["upcoming"] = ringing, soonest(next, 8)
	return out
}

// coming is one thing in Coming up, with the moment it is sorted by.
type coming struct {
	at   time.Time
	item map[string]any
}

// soonest is what is coming in time order, a reminder and a task at the
// same hour side by side whatever they are, the first n of them.
func soonest(all []coming, n int) []any {
	sort.SliceStable(all, func(i, j int) bool { return all[i].at.Before(all[j].at) })
	out := []any{}
	for i, c := range all {
		if i == n {
			break
		}
		out = append(out, c.item)
	}
	return out
}

// onToday is what falls today across every listed type with a day: the
// calendar's view of the day. A thing with no time of its own is all day,
// and comes first.
func (s *Server) onToday(now time.Time) []coming {
	day := now.Format("2006-01-02")
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var items []coming
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
			item := map[string]any{"title": s.title(t, rec), "href": "/t/" + t.Name + "/" + rec.ID, "kind": "event", "time": "All day"}
			at := start
			if strings.HasSuffix(v, "T00:00:00Z") {
				if ts.UTC().Format("2006-01-02") != day {
					continue
				}
			} else {
				if ts.Local().Format("2006-01-02") != day {
					continue
				}
				at = ts.Local()
				item["time"] = at.Format("15:04")
			}
			items = append(items, coming{at, item})
		}
	}
	return items
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
			// What it is about, in words and as a link, as the machine's own
			// notification says it: Water, 3 of 8 glasses so far.
			text, link := s.ringWords(rec)
			body, _ := json.Marshal(map[string]any{"id": rec.ID, "title": s.title(t, rec), "href": "/t/" + ReminderType + "/" + rec.ID, "text": text, "url": link})
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
