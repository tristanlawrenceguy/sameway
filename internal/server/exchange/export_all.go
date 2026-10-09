package exchange

import (
	"html/template"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"
	"github.com/tristanlawrenceguy/sameway/internal/export"
	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// Everything with a day goes out as one calendar, and a list or a
// calendar on its own page as it is shown there.

// exportCalendar answers /export/all.ics: everything with a day, or the
// types named, ?type=task&type=reminder, as a calendar of several shows.
func (s *Service) exportCalendar(w http.ResponseWriter, r *http.Request) {
	var groups []export.Group
	only := r.URL.Query()["type"]
	for _, t := range s.app.Types.Types {
		if _, ok := export.ByExt(t, "ics"); !ok || !exportable(t) {
			continue
		}
		if len(only) > 0 && !slices.Contains(only, t.Name) || len(only) == 0 && !s.Listed(t) {
			continue
		}
		recs, err := s.app.Store.List(t.Name, store.ListOptions{})
		if err != nil {
			s.Fail(w, err)
			return
		}
		groups = append(groups, export.Group{Type: t, Records: recs})
	}
	w.Header().Set("Content-Type", export.ICS.Type)
	web.Attachment(w, s.app.Workspace.Config.Name+".ics")
	export.Calendar(w, s.app.Workspace.Config.Name, groups)
}

// BlockExport is a collection's or a calendar's way out, under it on its
// own page: the records it shows, with the choices made on it.
func (s *Service) BlockExport(name string, props map[string]any) template.HTML {
	typeName, _ := props["type"].(string)
	kinds := blocks.Strs(props["types"])
	if name == blocks.CalendarComponent && props["problem"] == nil && (typeName == "all" || len(kinds) > 0) {
		what, href := "everything on the calendar", "/export/all.ics"
		if len(kinds) > 0 && !slices.Contains(kinds, "all") {
			// The kinds shown, as the calendar shows them, and no more.
			var names []string
			for _, k := range kinds {
				names = append(names, schema.Plural(k))
			}
			what, href = "these "+blocks.AndList(names), href+"?"+url.Values{"type": kinds}.Encode()
		}
		return s.Component("export", map[string]any{"what": what, "items": []any{map[string]any{"href": href, "format": "ics"}}})
	}
	t, ok := s.app.Types.Get(typeName)
	all, _ := props["all"].(string)
	if !ok || props["problem"] != nil || all == "" || (name != blocks.CollectionComponent && name != blocks.CalendarComponent) {
		return ""
	}
	q := url.Values{}
	if _, raw, ok := strings.Cut(all, "?"); ok {
		q, _ = url.ParseQuery(raw)
	}
	recs, err := query.Filter(s.app.Store, t, q["where"], q.Get("order"), 0, s.Now())
	if err != nil || len(recs) == 0 {
		return ""
	}
	html := s.ExportLinks(t, q, recs)
	if name == blocks.CalendarComponent {
		// A calendar goes out as a calendar.
		html = string(s.Component("export", map[string]any{"what": "these " + schema.Plural(t.Name), "items": []any{map[string]any{
			"href": "/export/" + t.Name + ".ics" + strings.TrimPrefix(all, "/t/"+t.Name), "format": "ics", "size": s.exportSize(export.ICS, t, recs)}}}))
	}
	return template.HTML(html)
}
