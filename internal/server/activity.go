package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// headingSummary returns the activity summary if present, otherwise builds a
// fallback from actor + action + target + detail so blank-summary records get
// readable heading text like "You said hello" instead of an empty h3.
func (s *Server) headingSummary(r *store.Record) string {
	if s2, _ := r.Fields["summary"].(string); s2 != "" {
		return template.HTMLEscapeString(s2)
	}
	actor, _ := r.Fields["actor"].(string)
	action, _ := r.Fields["action"].(string)
	target, _ := r.Fields["target"].(string)
	detail, _ := r.Fields["detail"].(string)

	who := map[string]string{"human": "You", "assistant": "Assistant", "system": "System"}[actor]
	if who == "" {
		who = actor
	}
	parts := []string{who, action}
	if target != "" {
		parts = append(parts, target)
	}
	if detail != "" {
		parts = append(parts, detail)
	}
	return strings.Join(parts, " ")
}

// recentActivity renders the newest n actions inside a disclosure that is
// closed by default, so the log is there when wanted and silent otherwise.
// The full log is always on its own page, which is what a screen reader user
// or an agent uses when they do not want to open this. Workspaces without
// the activity type get nothing.
// from is the page the list is on, so an Undo returns to it.
func (s *Server) recentActivity(n int, from string) template.HTML {
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
		summary := s.headingSummary(r)
		inner.WriteString(`<li><h3 class="sw-event__heading">` + summary + `</h3>` + string(s.event(r, from)) + `</li>`)
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
// to undo it: a form posting to the entry, back to the page from.
func (s *Server) event(r *store.Record, from string) template.HTML {
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
		b.WriteString(`<p class="sw-empty">Nothing has happened yet.</p>`)
	}
	b.WriteString(`<h2>Activity</h2>`)
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
		summary := s.headingSummary(rec)
		b.WriteString(`<li><h3 class="sw-event__heading">` + summary + `</h3>` + string(s.event(rec, "/activity")) + `</li>`)
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
