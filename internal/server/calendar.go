package server

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
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
	return s.resolveCalendarAt(props, blockID, nil)
}

// resolveCalendarAt is resolveCalendar on a page a calendar of everything
// can be narrowed to one kind on: at says which, and its address holds
// the kind picked (calendar_kinds.go).
func (s *Server) resolveCalendarAt(props map[string]any, blockID string, at *collectionPlace) map[string]any {
	out := map[string]any{}
	for k, v := range props {
		out[k] = v
	}
	delete(out, "filter") // the server's to fill, never the block's
	now := time.Now()
	month, _ := out["month"].(string)
	if _, err := time.Parse("2006-01", month); err != nil {
		month = now.Format("2006-01")
		out["month"] = month
	}
	if d, _ := out["today"].(string); d == "" {
		out["today"] = now.Format("2006-01-02")
	}
	// One day instead of the month: its month is the one shown, and the
	// days either side are a link away, as the months are.
	day, _ := out["day"].(string)
	shownDay, dayErr := time.Parse("2006-01-02", day)
	if day != "" && dayErr != nil {
		delete(out, "day")
		day = ""
	}
	if day != "" {
		out["month"] = shownDay.Format("2006-01")
		month = out["month"].(string)
	}
	if blockID != "" {
		base := "/canvas/" + blockID
		out["dayBase"] = base + "?day="
		if day != "" {
			prev, next := shownDay.AddDate(0, 0, -1), shownDay.AddDate(0, 0, 1)
			out["nav"] = map[string]any{
				"previous": map[string]any{"href": base + "?day=" + prev.Format("2006-01-02"), "label": prev.Format("Mon 2 Jan")},
				"next":     map[string]any{"href": base + "?day=" + next.Format("2006-01-02"), "label": next.Format("Mon 2 Jan")},
				"month":    map[string]any{"href": base + "?month=" + month, "label": shownDay.Format("January")},
			}
		} else {
			shown, _ := time.Parse("2006-01", month)
			prev, next := shown.AddDate(0, -1, 0), shown.AddDate(0, 1, 0)
			out["nav"] = map[string]any{
				"previous": map[string]any{"href": base + "?month=" + prev.Format("2006-01"), "label": prev.Format("January 2006")},
				"next":     map[string]any{"href": base + "?month=" + next.Format("2006-01"), "label": next.Format("January 2006")},
				"today":    map[string]any{"href": base + "?day=" + now.Format("2006-01-02"), "label": "Today"},
			}
		}
	}
	typeName, _ := props["type"].(string)
	kinds := strs(props["types"])
	if typeName == "" && len(kinds) == 0 {
		return out
	}
	if day != "" {
		if add := s.logForDay(typeName, kinds, strs(props["where"]), shownDay); add != nil {
			out["add"] = add
		}
	}
	if typeName == "all" || len(kinds) > 0 {
		// Several kinds together: the ones named, or every listed type.
		only, problem := s.calendarTypes(kinds)
		if problem != "" {
			out["problem"] = problem
			return out
		}
		out["events"] = s.everyEvent(now, month, only)
		calendarKinds(out, at, blockID)
		// Told apart among what is shown, once narrowed to its kinds.
		if events, ok := out["events"].([]any); ok {
			out["events"] = eventsApart(events, month)
		}
		return out
	}
	// Set up wrong, it says so, rather than show an empty month, which
	// reads as nothing on.
	t, ok := s.app.Types.Get(typeName)
	if !ok {
		out["problem"] = s.noType(typeName)
		return out
	}
	field := dateField(t, props["date"])
	if field == "" {
		out["problem"] = noDateField(t, props["date"])
		return out
	}
	recs, err := query.Filter(s.app.Store, t, strs(props["where"]), field, 0, now)
	if err != nil {
		out["problem"] = err.Error()
		return out
	}
	events := make([]any, 0, len(recs))
	for _, rec := range recs {
		ev := s.eventOf(t, rec, field)
		if ev == nil {
			continue
		}
		if meta := s.showFields(t, rec, strs(props["show"])); meta != "" {
			ev["meta"] = meta
		}
		events = append(events, ev)
		events = append(events, s.repeatedIn(t, rec, field, ev, month)...)
	}
	out["events"] = eventsApart(events, month)
	out["all"] = listPath(t.Name, strs(props["where"]), field)
	return out
}

// calendarTypes is the types a calendar of several kinds shows: those
// named in types, each of which must have a date to place, or nil for
// every listed type (types empty, or holding all). Set up wrong, it says
// why, as the page would.
func (s *Server) calendarTypes(names []string) ([]*schema.Type, string) {
	if len(names) == 0 || slices.Contains(names, "all") {
		return nil, ""
	}
	var only []*schema.Type
	for _, name := range names {
		t, ok := s.app.Types.Get(name)
		if !ok {
			return nil, s.noType(name)
		}
		if dateField(t, nil) == "" {
			return nil, noDateField(t, nil)
		}
		if !slices.Contains(only, t) {
			only = append(only, t)
		}
	}
	return only, ""
}

// andList is names as said: tasks, reminders and entries.
func andList(names []string) string {
	if len(names) < 2 {
		return strings.Join(names, "")
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

// eventOf is one record on the calendar, or nil when its date will not
// read. A day with no time is stored as midnight UTC; it is that day
// everywhere, with no time to show. Anything else is a moment, shown in
// local time.
func (s *Server) eventOf(t *schema.Type, rec *store.Record, field string) map[string]any {
	v, _ := rec.Fields[field].(string)
	ts, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil
	}
	day, clock := ts.UTC().Format("2006-01-02"), ""
	if !strings.HasSuffix(v, "T00:00:00Z") {
		local := ts.Local()
		day, clock = local.Format("2006-01-02"), when.Clock(local)
	}
	ev := map[string]any{"date": day, "label": s.title(t, rec), "href": "/t/" + t.Name + "/" + rec.ID}
	if clock != "" {
		ev["time"] = clock
	}
	if actions := s.markActions(t, rec); actions != nil {
		ev["actions"] = actions
	}
	return ev
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
	for _, f := range t.Shown() {
		if f.Type == "datetime" {
			return f.Name
		}
	}
	return ""
}

// noDateField says why a type's records cannot go on a calendar: the
// field named is not a date, or the type has none.
func noDateField(t *schema.Type, named any) string {
	var dates []string
	for _, f := range t.Shown() {
		if f.Type == "datetime" {
			dates = append(dates, f.Name)
		}
	}
	if name, _ := named.(string); strings.TrimSpace(name) != "" {
		if len(dates) > 0 {
			return fmt.Sprintf("%s has no date field %q; its date fields are %s", t.Name, name, strings.Join(dates, ", "))
		}
		return fmt.Sprintf("%s has no date field %q, and no date field at all to place on a calendar", t.Name, name)
	}
	return t.Name + " has no date field to place on a calendar; show it as a list instead"
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
// they are; the stored block is not touched. An empty day is the month
// again.
func withMonth(props map[string]any, month, day string) map[string]any {
	out := map[string]any{}
	for k, v := range props {
		out[k] = v
	}
	if month != "" {
		out["month"] = month
	}
	if day != "" {
		out["day"] = day
	} else {
		delete(out, "day")
	}
	return out
}
