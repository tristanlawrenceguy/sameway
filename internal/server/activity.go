package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// record logs a change a person made through a page, with the device it
// came from when that was not this machine.
func (s *Server) record(r *http.Request, c chat.Change) string {
	v := chat.VisitorOf(r.Context())
	c.Via, c.By, c.ByLogin = v.Device, v.Who(), v.Login
	if c.ByLogin == "" && v.Owner() {
		c.ByLogin = s.app.Chat.Owner.Login
	}
	return chat.Record(s.app.Store, "human", c)
}

// recentActivity renders the newest n actions inside a disclosure that is
// closed by default, so the log is there when wanted and silent otherwise.
// The full log is always on its own page, which is what a screen reader user
// or an agent uses when they do not want to open this. Workspaces without
// the activity type get nothing.
// from is the page the list is on, so an Undo returns to it.
func (s *Server) recentActivity(n int, from string) template.HTML {
	return s.recentActivityAbout(n, from, nil)
}

// recentActivityAbout is the recent activity about one thing only: a
// record's page shows what happened to that record, a list what happened
// to its kind. A page is about what it is about, so it does not list
// changes to everything else there is.
func (s *Server) recentActivityAbout(n int, from string, about func(target, id string) bool) template.HTML {
	if _, ok := s.app.Types.Get(chat.ActivityType); !ok {
		return ""
	}
	limit := n
	if about != nil {
		limit = 200
	}
	all, err := s.app.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: limit})
	if err != nil {
		return ""
	}
	var recs []*store.Record
	for _, r := range all {
		target, _ := r.Fields["target"].(string)
		id, _ := r.Fields["target_id"].(string)
		if about == nil || about(target, id) {
			recs = append(recs, r)
		}
		if len(recs) == n {
			break
		}
	}
	if len(recs) == 0 {
		return ""
	}
	total := len(recs)
	if about == nil {
		total, _ = s.app.Store.Count(chat.ActivityType)
	}
	var inner strings.Builder
	inner.WriteString(`<ol class="sw-plain sw-stack--tight" aria-label="Recent activity">`)
	for _, r := range recs {
		inner.WriteString(`<li>` + string(s.event(r, from, 3, true)) + `</li>`)
	}
	inner.WriteString(`</ol><p class="sw-small" style="margin:var(--sw-space-3) 0 0">`)
	inner.WriteString(string(s.component("link", map[string]any{"href": "/activity", "label": "All activity", "look": "button"})))
	inner.WriteString(`</p>`)

	body, err := s.app.Registry.RenderSlot("disclosure",
		map[string]any{"label": "Activity", "count": total, "id": "recent-activity"},
		template.HTML(inner.String()))
	if err != nil {
		return ""
	}
	return template.HTML(`<h2 class="sw-visually-hidden">Activity</h2>` + string(body))
}

// event renders one entry. One that can still be undone carries the way
// to undo it: a form posting to the entry, back to the page from. level
// makes its sentence a heading, for a log read heading by heading; dated
// gives its time the day, where no day's heading above says it.
func (s *Server) event(r *store.Record, from string, level int, dated bool) template.HTML {
	at := r.CreatedAt.Local().Format("15:04")
	if dated {
		at = messageTime(r.CreatedAt)
	}
	props := map[string]any{
		"actor":    r.Fields["actor"],
		"action":   r.Fields["action"],
		"time":     at,
		"datetime": r.CreatedAt.UTC().Format(time.RFC3339),
		"id":       "activity-" + r.ID,
	}
	if level > 0 {
		props["level"] = level
	}
	// Where it was done from, when not here, as the log's own summary says.
	if via, _ := r.Fields["via"].(string); via != "" {
		if !strings.HasPrefix(via, "through ") {
			via = "on " + via
		}
		props["via"] = via
	}
	if who, person := s.whoDid(r); who != "" {
		props["who"], props["person"] = who, person
	}
	if t, _ := r.Fields["target"].(string); t != "" {
		props["target"] = t
	}
	if d, _ := r.Fields["detail"].(string); d != "" {
		props["detail"] = d
	}
	if href := s.hrefFor(r); href != "" {
		props["href"] = href
	}
	// An undo reads as one: "You undid: Assistant added card Plan".
	if undoes, _ := r.Fields["undoes"].(string); undoes != "" {
		summary, _ := r.Fields["summary"].(string)
		for _, verb := range []string{" undid: ", " put back: "} {
			if _, after, ok := strings.Cut(summary, verb); ok {
				props["action"], props["detail"] = strings.TrimSpace(verb), after
				delete(props, "target")
				break
			}
		}
	}
	if s.app.Chat.Undoable(r) {
		props["undo"], props["from"] = "/activity/"+r.ID+"/undo", from
	}
	return s.component("event", props)
}

// hrefFor is the page of the thing an activity entry is about, when it
// still exists: a record's page, a block's own page, or a tab. Worked out
// when the entry is shown, not when it was written, so an entry about
// something since removed leads nowhere instead of to a 404.
func (s *Server) hrefFor(r *store.Record) string {
	target, _ := r.Fields["target"].(string)
	id, _ := r.Fields["target_id"].(string)
	if target == "" || id == "" {
		return ""
	}
	if target == chat.CanvasType {
		if _, err := s.app.Store.Get(chat.CanvasType, id); err == nil {
			return chat.CanvasPath(id)
		}
		return ""
	}
	if _, ok := s.app.Types.Get(target); ok {
		if _, err := s.app.Store.Get(target, id); err == nil {
			return "/t/" + target + "/" + id
		}
		return ""
	}
	if _, err := s.app.Store.Get(chat.BlockType, id); err == nil {
		return "/canvas/" + id
	}
	return ""
}

// activityPage lists every recorded action, newest first, grouped by day.
func (s *Server) activityPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.app.Types.Get(chat.ActivityType); !ok {
		s.page(w, r, "Activity", s.component("alert", map[string]any{"kind": "info", "message": "This workspace keeps no log yet. Run sameway init --force to add one."}), pageOptions{})
		return
	}
	recs, err := s.app.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 500})
	if err != nil {
		s.fail(w, err)
		return
	}
	var b strings.Builder
	b.WriteString(`<p class="sw-muted sw-prose">Every change to the canvas, by you or the assistant, newest first. The same log is at <a href="/api/activity">/api/activity</a>.</p>`)
	if len(recs) == 0 {
		b.WriteString(string(s.component("empty", map[string]any{
			"message": "No activity yet. What you and the assistant change on the canvas shows up here.", "action": map[string]any{"href": "/chat", "label": "Send a message"},
		})))
	}
	day := ""
	open := false
	for _, rec := range recs {
		d := dayHeading(rec.CreatedAt)
		if d != day {
			if open {
				b.WriteString("</ol>")
			}
			id := "day-" + rec.CreatedAt.Local().Format("2006-01-02")
			fmt.Fprintf(&b, `<h2 class="sw-small sw-muted" style="margin-top:var(--sw-space-8)" id="%s">%s</h2><ol class="sw-plain sw-stack--tight sw-panel" aria-labelledby="%s">`, id, template.HTMLEscapeString(d), id)
			day, open = d, true
		}
		b.WriteString(`<li>` + string(s.event(rec, "/activity", 3, false)) + `</li>`)
	}
	if open {
		b.WriteString("</ol>")
	}
	s.page(w, r, "Activity", template.HTML(b.String()), pageOptions{JSONURL: "/api/activity"})
}

// script serves the concatenated component enhancement scripts.
func (s *Server) script(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(s.js)
}

// baseFile serves an individual file from design/base/.
func (s *Server) baseFile(w http.ResponseWriter, r *http.Request) {
	file := r.PathValue("file")
	if !strings.HasSuffix(file, ".js") {
		http.NotFound(w, r)
		return
	}
	data, err := design.FS.ReadFile("base/" + file)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(data)
}

// dayHeading is a day of the log as its heading says it: Today, Yesterday,
// or the day, with its year when it is not this one.
func dayHeading(at time.Time) string {
	at, now := at.Local(), time.Now()
	y, m, d := now.Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, now.Location())
	switch day := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, now.Location()); {
	case day.Equal(today):
		return "Today, " + at.Format("Monday 2 January")
	case day.Equal(today.AddDate(0, 0, -1)):
		return "Yesterday, " + at.Format("Monday 2 January")
	case at.Year() != now.Year():
		return at.Format("Monday 2 January 2006")
	}
	return at.Format("Monday 2 January")
}
