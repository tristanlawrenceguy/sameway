package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// recentActivity renders the newest n activity records as event components.
// Workspaces without the activity type get an empty list.
func (s *Server) recentActivity(n int) []template.HTML {
	if _, ok := s.app.Types.Get(chat.ActivityType); !ok {
		return nil
	}
	recs, err := s.app.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: n})
	if err != nil {
		return nil
	}
	var out []template.HTML
	for _, r := range recs {
		out = append(out, s.event(r))
	}
	return out
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
