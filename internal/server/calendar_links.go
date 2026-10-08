package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/ingest"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// Calendars: the link a calendar elsewhere gives for subscribing (Google's
// secret address in iCal format, Outlook's published ICS link, iCloud's
// public calendar) is pasted once, and its events are kept in step every
// hour (ingest.SyncCalendar): new ones added, changed ones changed, gone
// ones taken away, never the person's own. Each sync that changed anything
// is one line in the log.

// calendarLink is one calendar kept in step.
type calendarLink struct {
	ID   string    `json:"id"`
	Name string    `json:"name"`
	URL  string    `json:"url"`
	UIDs []string  `json:"uids"`
	At   time.Time `json:"at"`
	Err  string    `json:"err,omitempty"`
}

func (s *Server) calendarLinks() []calendarLink {
	var out []calendarLink
	json.Unmarshal([]byte(s.app.Store.Meta("calendar:links")), &out)
	return out
}

func (s *Server) saveCalendarLinks(l []calendarLink) {
	b, _ := json.Marshal(l)
	s.app.Store.SetMeta("calendar:links", string(b))
}

// syncCalendar fetches one calendar and keeps its events in step.
func (s *Server) syncCalendar(ctx context.Context, l *calendarLink) {
	l.At, l.Err = time.Now(), ""
	t, ok := s.app.Types.Get(records.EventType)
	if !ok {
		l.Err = "this workspace has no events"
		return
	}
	address := strings.Replace(l.URL, "webcal://", "https://", 1)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		l.Err = err.Error()
		return
	}
	res, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		l.Err = "could not reach the calendar: " + err.Error()
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		l.Err = "the calendar answered " + res.Status + "; the link may have changed"
		return
	}
	data, _ := io.ReadAll(io.LimitReader(res.Body, 20<<20))
	tb, err := ingest.ReadICS(data)
	if err != nil {
		l.Err = err.Error()
		return
	}
	done, err := ingest.SyncCalendar(s.app.Store, t, tb, l.UIDs)
	if err != nil {
		l.Err = err.Error()
		return
	}
	l.UIDs = done.UIDs
	if said := done.String(); said != "" {
		records.Record(s.app.Store, "system", records.Change{Action: "kept in step", Component: "calendar", Detail: l.Name + ": " + said, Href: "/t/event"})
	}
}

// KeepCalendars keeps every calendar link in step, every hour, while ctx
// lasts.
func (s *Server) KeepCalendars(ctx context.Context) {
	go func() {
		tick := time.NewTicker(time.Hour)
		defer tick.Stop()
		for {
			links := s.calendarLinks()
			for i := range links {
				s.syncCalendar(ctx, &links[i])
			}
			if len(links) > 0 {
				s.mergeCalendarLinks(links)
			}
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
		}
	}()
}

// mergeCalendarLinks keeps what syncs found for the links still there: one
// removed while a sync ran stays removed.
func (s *Server) mergeCalendarLinks(synced []calendarLink) {
	byID := map[string]calendarLink{}
	for _, l := range synced {
		byID[l.ID] = l
	}
	now := s.calendarLinks()
	for i, l := range now {
		if got, ok := byID[l.ID]; ok {
			now[i] = got
		}
	}
	s.saveCalendarLinks(now)
}

// calendarHow is where each calendar gives its link.
var calendarHow = []struct{ app, how string }{
	{"Google Calendar", "Settings, then the calendar under Settings for my calendars, then Secret address in iCal format."},
	{"Outlook", "Settings, Calendar, Shared calendars, then Publish a calendar, and copy the ICS link."},
	{"iCloud", "in Calendar, share the calendar, turn on Public Calendar, and copy the link."},
}

func (s *Server) calendarsPage(w http.ResponseWriter, r *http.Request) {
	esc := template.HTMLEscapeString
	var b strings.Builder
	links := s.calendarLinks()
	if len(links) > 0 {
		b.WriteString(`<h2>Kept in step</h2><ul class="sw-plain sw-rows">`)
		for _, l := range links {
			state := schema.Count(len(l.UIDs), records.EventType) + ", checked " + when.Sent(l.At, time.Now())
			if l.Err != "" {
				state = "Not up to date: " + l.Err
			}
			b.WriteString(`<li class="sw-stack"><strong>` + esc(l.Name) + `</strong> <span class="sw-small sw-muted">` + esc(state) + `</span><form method="post" action="/calendars/remove"><input type="hidden" name="id" value="` + l.ID + `">` +
				string(s.component("button", map[string]any{"label": "Stop keeping it", "context": l.Name, "type": "submit", "variant": "quiet"})) + `</form></li>`)
		}
		b.WriteString(`</ul>`)
	}
	b.WriteString(`<h2>Add a calendar</h2><p>Its events come in and are kept up to date every hour. Change them in the calendar they come from: a change here is put back at the next hour.</p><dl class="sw-stack">`)
	for _, h := range calendarHow {
		b.WriteString(`<dt><strong>` + esc(h.app) + `</strong></dt><dd>` + esc(h.how) + `</dd>`)
	}
	b.WriteString(`</dl><form method="post" action="/calendars/add" class="sw-stack">`)
	b.WriteString(string(s.component("text-field", map[string]any{"label": "The calendar's link", "name": "url", "type": "url", "required": true, "hint": "It begins https:// or webcal://. Keep it to yourself: whoever has it can read the calendar."})))
	b.WriteString(string(s.component("text-field", map[string]any{"label": "A name for it", "name": "name", "hint": "Such as Work or Family."})))
	b.WriteString(string(s.component("button", map[string]any{"label": "Keep it in step", "type": "submit"})) + `</form>`)
	s.page(w, r, "Calendars", template.HTML(b.String()), pageOptions{Lede: "Your calendars from Google, Outlook or iCloud, kept in step."})
}

func (s *Server) calendarAdd(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	link := strings.TrimSpace(r.PostForm.Get("url"))
	if !strings.HasPrefix(link, "https://") && !strings.HasPrefix(link, "webcal://") && !strings.HasPrefix(link, "http://") {
		s.failed(w, r, "Not added", errors.New("paste the calendar's link, which begins https:// or webcal://"), "/calendars")
		return
	}
	name := strings.TrimSpace(r.PostForm.Get("name"))
	if name == "" {
		name = "Calendar"
	}
	l := calendarLink{ID: linkID(), Name: name, URL: link}
	ctx, cancel := context.WithTimeout(r.Context(), 40*time.Second)
	defer cancel()
	s.syncCalendar(ctx, &l)
	if l.Err != "" {
		s.failed(w, r, "Not added", errors.New(l.Err), "/calendars")
		return
	}
	s.saveCalendarLinks(append(s.calendarLinks(), l))
	s.tellAt(w, r, outcome{Title: name + " is kept in step", Text: fmt.Sprintf("%d events are here now, and are checked again every hour.", len(l.UIDs))}, "/calendars")
}

func (s *Server) calendarRemove(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var kept []calendarLink
	gone := ""
	for _, l := range s.calendarLinks() {
		if l.ID == r.PostForm.Get("id") {
			gone = l.Name
			continue
		}
		kept = append(kept, l)
	}
	s.saveCalendarLinks(kept)
	s.tellAt(w, r, outcome{Title: "No longer kept in step", Text: gone + "'s events stay as they are; nothing more comes from it."}, "/calendars")
}

func (s *Server) calendarRoutes(m *http.ServeMux) {
	m.HandleFunc("GET /calendars", s.calendarsPage)
	m.HandleFunc("POST /calendars/add", s.calendarAdd)
	m.HandleFunc("POST /calendars/remove", s.calendarRemove)
}

func linkID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}
