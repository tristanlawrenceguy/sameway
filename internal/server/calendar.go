package server

import (
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// calendarComponent is the month block. Given a content type, its records
// with a date are the events, read when the page renders, so the month a
// person glances at is what is true now; without one, the events are
// whatever the block carries.
const calendarComponent = "calendar"

// resolveCalendar fills what the block left out: the month and today, so
// a calendar never has to be told what day it is; and the events, from
// the records of a type when one is named.
func (s *Server) resolveCalendar(props map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range props {
		out[k] = v
	}
	now := time.Now()
	if m, _ := out["month"].(string); len(m) != 7 {
		out["month"] = now.Format("2006-01")
	}
	if d, _ := out["today"].(string); d == "" {
		out["today"] = now.Format("2006-01-02")
	}
	typeName, _ := props["type"].(string)
	if typeName == "" {
		return out
	}
	t, ok := s.app.Types.Get(typeName)
	if !ok {
		out["events"] = []any{}
		return out
	}
	field := dateField(t, props["date"])
	if field == "" {
		out["events"] = []any{}
		return out
	}
	recs, err := query.Filter(s.app.Store, t, strs(props["where"]), field, 0, now)
	if err != nil {
		out["events"] = []any{}
		out["caption"] = err.Error()
		return out
	}
	events := make([]any, 0, len(recs))
	for _, rec := range recs {
		v, _ := rec.Fields[field].(string)
		ts, err := time.Parse(time.RFC3339, v)
		if err != nil {
			continue
		}
		local := ts.Local()
		ev := map[string]any{"date": local.Format("2006-01-02"), "label": titleOf(t, rec), "href": "/t/" + t.Name + "/" + rec.ID}
		if local.Hour() != 0 || local.Minute() != 0 {
			ev["time"] = local.Format("15:04")
		}
		if meta := s.showFields(t, rec, strs(props["show"])); meta != "" {
			ev["meta"] = meta
		}
		events = append(events, ev)
	}
	out["events"] = events
	return out
}

// dateField is the field the days come from: the one named, or the
// type's first datetime field.
func dateField(t *schema.Type, named any) string {
	if name, _ := named.(string); strings.TrimSpace(name) != "" {
		if f, ok := t.Field(name); ok && f.Type == "datetime" {
			return name
		}
		return ""
	}
	for _, f := range t.Fields {
		if f.Type == "datetime" {
			return f.Name
		}
	}
	return ""
}

// showFields is the fields a person asked to see beside each event, as
// their values: a ref by the title it points at, a bool by yes or no.
func (s *Server) showFields(t *schema.Type, rec *store.Record, names []string) string {
	var parts []string
	for _, name := range names {
		f, ok := t.Field(name)
		if !ok {
			continue
		}
		v := display(*f, rec.Fields[name])
		if f.Type == "ref" {
			v = s.refTitle(*f, v)
		}
		if v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, " · ")
}
