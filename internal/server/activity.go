package server

import (
	stdcmp "cmp"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// who made a request's change: a person on a page, with the device it
// came from when that was not this machine, or an agent posting the
// page's form (agents.go), by the name it gave.
func (s *Server) who(r *http.Request) records.Who {
	if pageAction(r) || records.VisitorOf(r.Context()).Agent {
		return apiAgent(r).As()
	}
	v := records.VisitorOf(r.Context())
	w := records.Who{Actor: "human", Via: v.Device, By: v.Who(), ByLogin: v.Login}
	if w.ByLogin == "" && v.Owner() {
		w.ByLogin = s.app.Records.Owner.Login
	}
	return w
}

// record logs a change made through a page, as whoever made it.
func (s *Server) record(r *http.Request, c records.Change) string {
	w := s.who(r)
	c.Via, c.By, c.ByLogin = w.Via, w.By, w.ByLogin
	return records.Record(s.app.Store, w.Actor, c)
}

// apply makes a change through a page: it writes the change's ops, all
// or none, and logs the change, with them, as whoever made it. A change
// that makes its thing is named by the id it was given. It returns the
// log entry and the ops as made.
func (s *Server) apply(r *http.Request, c records.Change, ops ...records.Op) (string, []records.Op, error) {
	done, err := s.app.Records.Apply(ops...)
	if err != nil {
		return "", nil, err
	}
	if c.ID == "" && len(done) > 0 {
		c.ID = done[len(done)-1].ID
	}
	c.Ops = done
	return s.record(r, c), done, nil
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
	if _, ok := s.app.Types.Get(records.ActivityType); !ok {
		return ""
	}
	limit := n
	if about != nil {
		limit = 200
	}
	all, err := s.app.Store.List(records.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: limit})
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
	// The count is what is inside; the whole log's size is on its link.
	everything := "All activity"
	if about == nil {
		if total, _ := s.app.Store.Count(records.ActivityType); total > len(recs) {
			everything = fmt.Sprintf("All activity, %d changes", total)
		}
	}
	var inner strings.Builder
	inner.WriteString(`<ol class="sw-plain sw-stack--tight" aria-label="Recent activity">`)
	told := s.entriesApart(recs)
	for i, r := range recs {
		inner.WriteString(`<li>` + string(s.event(r, from, 3, true, told[i])) + `</li>`)
	}
	inner.WriteString(`</ol><p class="sw-small" style="margin:var(--sw-space-3) 0 0">`)
	inner.WriteString(string(s.part(ui.Link{Href: "/activity", Label: everything, Look: ui.LookButton})))
	inner.WriteString(`</p>`)

	body, err := s.app.Registry.RenderSlot("disclosure",
		map[string]any{"label": "Activity", "count": len(recs), "of": oneOrMany(len(recs), "change", "changes"), "id": "recent-activity"},
		template.HTML(inner.String()))
	if err != nil {
		return ""
	}
	return template.HTML(`<h2 class="sw-visually-hidden">Activity</h2>` + string(body))
}

// event renders one entry, as line says it, with its time and its anchor.
// level makes its sentence a heading, for a log read heading by heading;
// dated gives its time the day, where no day's heading above says it.
// told tells it from another entry shown that says the same, or is "".
func (s *Server) event(r *store.Record, from string, level int, dated bool, told string) template.HTML {
	at := when.Clock(r.CreatedAt.Local(), s.h24())
	if dated {
		at = when.Sent(r.CreatedAt, s.now(), s.h24())
	}
	props := s.line(r, true)
	if props["undo"] != nil {
		props["from"] = from
	}
	props["time"], props["datetime"], props["id"] = at, r.CreatedAt.UTC().Format(time.RFC3339), "activity-"+r.ID
	if level > 0 {
		props["level"] = level
	}
	if told != "" {
		props["context"] = told
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
	if target == records.CanvasType {
		if _, err := s.app.Store.Get(records.CanvasType, id); err == nil {
			return records.CanvasPath(id)
		}
		return ""
	}
	if _, ok := s.app.Types.Get(target); ok {
		if _, err := s.app.Store.Get(target, id); err == nil {
			return "/t/" + target + "/" + id
		}
		return ""
	}
	if _, err := s.app.Store.Get(records.BlockType, id); err == nil {
		return "/canvas/" + id
	}
	return ""
}

// activityPage lists every recorded action, newest first, grouped by day.
func (s *Server) activityPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.app.Types.Get(records.ActivityType); !ok {
		s.page(w, r, "Activity", s.part(ui.Alert{Kind: ui.Info, Message: "This workspace keeps no log yet. Run sameway init --force to add one."}), pageOptions{})
		return
	}
	all, err := s.app.Store.List(records.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true})
	if err != nil {
		s.fail(w, err)
		return
	}
	// Narrowed by who, what and when, as the address says, a page at a
	// time; Undo on an entry comes back to this same narrowing.
	recs, filters, f := s.narrowLog(all, r.URL.Query(), s.now())
	pg := pageOf(r, len(recs), activityPageSize)
	from := r.URL.RequestURI()
	var b strings.Builder
	b.WriteString(`<p class="sw-muted sw-prose">Every change, by you or the assistant, newest first.</p>`)
	if len(all) > 0 {
		b.WriteString(string(s.component("filters", filters)))
	}
	switch {
	case len(all) == 0:
		b.WriteString(string(s.part(ui.Empty{Message: "No activity yet. What you and the assistant change on the canvas shows up here.",
			Action: &ui.EmptyAction{Href: "/chat", Label: "Send a message"}})))
	case len(recs) == 0:
		b.WriteString(string(s.part(ui.Empty{Message: "No changes match: " + filters["showing"].(string) + ".",
			Action: &ui.EmptyAction{Href: str(filters["reset"], ""), Label: "Show every change"}})))
	}
	recs = recs[pg.lo:pg.hi]
	day := ""
	open := false
	told := s.entriesApart(recs)
	for i, rec := range recs {
		d := when.DayHeading(rec.CreatedAt, s.now())
		if d != day {
			if open {
				b.WriteString("</ol>")
			}
			id := "day-" + rec.CreatedAt.Local().Format("2006-01-02")
			fmt.Fprintf(&b, `<h2 class="sw-small sw-muted" style="margin-top:var(--sw-space-8)" id="%s">%s</h2><ol class="sw-plain sw-stack--tight sw-panel" aria-labelledby="%s">`, id, template.HTMLEscapeString(d), id)
			day, open = d, true
		}
		b.WriteString(`<li>` + string(s.event(rec, from, 3, false, told[i])) + `</li>`)
	}
	if open {
		b.WriteString("</ol>")
	}
	b.WriteString(string(s.pageNav(r, pg, "Pages of activity")))
	said := "Activity"
	if f.active() {
		said = fmt.Sprintf("Activity: %s, %s", filters["showing"], stdcmp.Or(filters["count"].(string), "no changes"))
	}
	s.page(w, r, "Activity", template.HTML(b.String()), pageOptions{JSONURL: "/api/activity", Said: pg.title(said)})
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
