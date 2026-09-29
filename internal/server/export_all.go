package server

import (
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/export"
	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Everything with a day goes out as one calendar, and a list or a
// calendar on its own page as it is shown there.

// exportCalendar answers /export/all.ics: everything with a day.
func (s *Server) exportCalendar(w http.ResponseWriter, r *http.Request) {
	var groups []export.Group
	for _, t := range s.app.Types.Types {
		if _, ok := export.ByExt(t, "ics"); !ok || !exportable(t) || !s.listed(t) {
			continue
		}
		recs, err := s.app.Store.List(t.Name, store.ListOptions{})
		if err != nil {
			s.fail(w, err)
			return
		}
		groups = append(groups, export.Group{Type: t, Records: recs})
	}
	w.Header().Set("Content-Type", export.ICS.Type)
	attachment(w, s.app.Workspace.Config.Name+".ics")
	export.Calendar(w, s.app.Workspace.Config.Name, groups)
}

// blockExport is a collection's or a calendar's way out, under it on its
// own page: the records it shows, with the choices made on it.
func (s *Server) blockExport(name string, props map[string]any) template.HTML {
	typeName, _ := props["type"].(string)
	if name == calendarComponent && typeName == "all" {
		return s.component("export", map[string]any{"what": "everything on the calendar", "items": []any{map[string]any{"href": "/export/all.ics", "format": "ics"}}})
	}
	t, ok := s.app.Types.Get(typeName)
	all, _ := props["all"].(string)
	if !ok || props["problem"] != nil || all == "" || (name != collectionComponent && name != calendarComponent) {
		return ""
	}
	q := url.Values{}
	if _, raw, ok := strings.Cut(all, "?"); ok {
		q, _ = url.ParseQuery(raw)
	}
	recs, err := query.Filter(s.app.Store, t, q["where"], q.Get("order"), 0, time.Now())
	if err != nil || len(recs) == 0 {
		return ""
	}
	html := s.exportLinks(t, q, recs)
	if name == calendarComponent {
		// A calendar goes out as a calendar.
		html = string(s.component("export", map[string]any{"what": "these " + plural(t.Name), "items": []any{map[string]any{
			"href": "/export/" + t.Name + ".ics" + strings.TrimPrefix(all, "/t/"+t.Name), "format": "ics", "size": s.exportSize(export.ICS, t, recs)}}}))
	}
	return template.HTML(html)
}
