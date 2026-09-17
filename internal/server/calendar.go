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
// a calendar never has to be told what day it is; the way to the months
// either side, on the block's own page; and the events, from the records
// of a type when one is named.
func (s *Server) resolveCalendar(props map[string]any, blockID string) map[string]any {
	out := map[string]any{}
	for k, v := range props {
		out[k] = v
	}
	now := time.Now()
	month, _ := out["month"].(string)
	if _, err := time.Parse("2006-01", month); err != nil {
		month = now.Format("2006-01")
		out["month"] = month
	}
	if d, _ := out["today"].(string); d == "" {
		out["today"] = now.Format("2006-01-02")
	}
	if blockID != "" {
		shown, _ := time.Parse("2006-01", month)
		prev, next := shown.AddDate(0, -1, 0), shown.AddDate(0, 1, 0)
		out["nav"] = map[string]any{
			"previous": map[string]any{"href": "/canvas/" + blockID + "?month=" + prev.Format("2006-01"), "label": prev.Format("January 2006")},
			"next":     map[string]any{"href": "/canvas/" + blockID + "?month=" + next.Format("2006-01"), "label": next.Format("January 2006")},
		}
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
		// A day with no time is stored as midnight UTC; it is that day
		// everywhere, with no time to show. Anything else is a moment,
		// shown in local time.
		day, clock := ts.UTC().Format("2006-01-02"), ""
		if !strings.HasSuffix(v, "T00:00:00Z") {
			local := ts.Local()
			day, clock = local.Format("2006-01-02"), local.Format("15:04")
		}
		ev := map[string]any{"date": day, "label": titleOf(t, rec), "href": "/t/" + t.Name + "/" + rec.ID}
		if clock != "" {
			ev["time"] = clock
		}
		if meta := s.showFields(t, rec, strs(props["show"])); meta != "" {
			ev["meta"] = meta
		}
		if actions := markActions(t, rec); actions != nil {
			ev["actions"] = actions
		}
		events = append(events, ev)
	}
	out["events"] = events
	out["all"] = listPath(t.Name, strs(props["where"]), field)
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

// withMonth is the block's props with another month shown, the rest as
// they are; the stored block is not touched.
func withMonth(props map[string]any, month string) map[string]any {
	out := map[string]any{}
	for k, v := range props {
		out[k] = v
	}
	out["month"] = month
	return out
}
