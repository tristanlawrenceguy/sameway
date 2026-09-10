package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// recentActivity renders the newest n actions inside a disclosure that is
// closed by default, so the log is there when wanted and silent otherwise.
// The full log is always on its own page, which is what a screen reader user
// or an agent uses when they do not want to open this. Workspaces without
// the activity type get nothing.
func (s *Server) recentActivity(n int) template.HTML {
	if _, ok := s.app.Types.Get(chat.ActivityType); !ok {
		return ""
	}
	total, _ := s.app.Store.Count(chat.ActivityType)
	if total == 0 {
		return ""
	}
	recs, err := s.app.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: n})
	if err != nil {
		return ""
	}
	var inner strings.Builder
	inner.WriteString(`<ol class="sw-plain sw-stack--tight" aria-label="Recent activity">`)
	for _, r := range recs {
		inner.WriteString("<li>" + string(s.event(r)) + "</li>")
	}
	inner.WriteString(`</ol><p class="sw-small" style="margin:var(--sw-space-3) 0 0">`)
	inner.WriteString(string(s.component("link", map[string]any{"href": "/activity", "label": "All activity"})))
	inner.WriteString(`</p>`)

	body, err := s.app.Registry.RenderSlot("disclosure",
		map[string]any{"label": "Activity", "count": total, "id": "recent-activity"},
		template.HTML(inner.String()))
	if err != nil {
		return ""
	}
	return template.HTML(`<h2 class="sw-visually-hidden">Activity</h2>` + string(body))
}

func (s *Server) event(r *store.Record) template.HTML {
	props := map[string]any{
		"actor":  r.Fields["actor"],
		"action": r.Fields["action"],
		"time":   r.CreatedAt.Local().Format("15:04"),
		"id":     "activity-" + r.ID,
	}
	if t, _ := r.Fields["target"].(string); t != "" {
		props["target"] = t
	}
	if d, _ := r.Fields["detail"].(string); d != "" {
		props["detail"] = d
	}
	return s.component("event", props)
}

// activityPage lists every recorded action, newest first, grouped by day.
func (s *Server) activityPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.app.Types.Get(chat.ActivityType); !ok {
		s.page(w, r, "Activity", s.component("alert", map[string]any{"kind": "info", "message": "This workspace has no activity type. Run `sameway init --force` to add schema/activity.yaml."}), pageOptions{})
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
		b.WriteString(`<p class="sw-empty">Nothing has happened yet.</p>`)
	}
	day := ""
	open := false
	for _, rec := range recs {
		d := rec.CreatedAt.Local().Format("Monday 2 January")
		if d != day {
			if open {
				b.WriteString("</ol>")
			}
			fmt.Fprintf(&b, `<h2 class="sw-small sw-muted" style="margin-top:var(--sw-space-8)">%s</h2><ol class="sw-plain sw-stack--tight sw-panel" aria-label="Activity on %s">`, template.HTMLEscapeString(d), template.HTMLEscapeString(d))
			day, open = d, true
		}
		b.WriteString("<li>" + string(s.event(rec)) + "</li>")
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
