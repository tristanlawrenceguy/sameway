package blocks

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

// CalendarComponent is the month block. Given a content type, its records
// with a date are the events, read when the page renders, so the month a
// person glances at is what is true now; without one, the events are
// whatever the block carries.
const CalendarComponent = "calendar"

// resolveCalendar fills what the block left out: the month and today, so
// a calendar never has to be told what day it is; the way to the months
// either side, on the block's own page; and the events, from the records
// of a type when one is named. On a page a calendar of everything can be
// narrowed to one kind on, the page's address holds the kind picked
// (calendar_kinds.go).
func resolveCalendar(ws *Workspace, props map[string]any, at Place) map[string]any {
	out := copyProps(props)
	delete(out, "filter") // the server's to fill, never the block's
	now := time.Now()
	month, day, shownDay := shownMonth(out, now)
	if at.Block != "" {
		calendarNav(out, at.Block, month, day, shownDay, now)
	}
	typeName, _ := props["type"].(string)
	kinds := Strs(props["types"])
	if typeName == "" && len(kinds) == 0 {
		return out
	}
	if day != "" {
		if add := ws.logForDay(typeName, kinds, Strs(props["where"]), shownDay); add != nil {
			out["add"] = add
		}
	}
	if typeName == "all" || len(kinds) > 0 {
		// Several kinds together: the ones named, or every listed type.
		only, problem := ws.calendarTypes(kinds)
		if problem != "" {
			out["problem"] = problem
			return out
		}
		out["events"] = ws.everyEvent(now, month, only)
		calendarKinds(out, at.Page, at.Block)
		// Told apart among what is shown, once narrowed to its kinds.
		if events, ok := out["events"].([]any); ok {
			out["events"] = eventsApart(events, month)
		}
		return out
	}
	ws.typeEvents(out, props, typeName, month, now)
	return out
}

// typeEvents fills a calendar of one type's records on their days. Set
// up wrong, it says so, rather than show an empty month, which reads as
// nothing on.
func (ws *Workspace) typeEvents(out, props map[string]any, typeName, month string, now time.Time) {
	t, ok := ws.Store.Types().Get(typeName)
	if !ok {
		out["problem"] = ws.NoType(typeName)
		return
	}
	field := dateField(t, props["date"])
	if field == "" {
		out["problem"] = noDateField(t, props["date"])
		return
	}
	recs, err := query.Filter(ws.Store, t, Strs(props["where"]), field, 0, now)
	if err != nil {
		out["problem"] = err.Error()
		return
	}
	events := make([]any, 0, len(recs))
	for _, rec := range recs {
		ev := ws.eventOf(t, rec, field)
		if ev == nil {
			continue
		}
		if meta := ws.showFields(t, rec, Strs(props["show"])); meta != "" {
			ev["meta"] = meta
		}
		events = append(events, ev)
		events = append(events, ws.repeatedIn(t, rec, field, ev, month)...)
	}
	out["events"] = eventsApart(events, month)
	out["all"] = listPath(t.Name, Strs(props["where"]), field)
}

// calendarTypes is the types a calendar of several kinds shows: those
// named in types, each of which must have a date to place, or nil for
// every listed type (types empty, or holding all). Set up wrong, it says
// why, as the page would.
func (ws *Workspace) calendarTypes(names []string) ([]*schema.Type, string) {
	if len(names) == 0 || slices.Contains(names, "all") {
		return nil, ""
	}
	var only []*schema.Type
	for _, name := range names {
		t, ok := ws.Store.Types().Get(name)
		if !ok {
			return nil, ws.NoType(name)
		}
		if !t.HasDay() {
			return nil, noDateField(t, nil)
		}
		if !slices.Contains(only, t) {
			only = append(only, t)
		}
	}
	return only, ""
}

// eventOf is one record on the calendar, or nil when its date will not
// read. A day with no time is stored as midnight UTC; it is that day
// everywhere, with no time to show. Anything else is a moment, shown in
// local time.
func (ws *Workspace) eventOf(t *schema.Type, rec *store.Record, field string) map[string]any {
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
	ev := map[string]any{"date": day, "label": ws.title(t, rec), "href": "/t/" + t.Name + "/" + rec.ID}
	if clock != "" {
		ev["time"] = clock
	}
	if actions := ws.MarkActions(t, rec); actions != nil {
		ev["actions"] = actions
	}
	return ev
}

// dateField is the field the days come from: the one named, or the
// type's day (schema DayField), as a list, an export and a record's
// related things have it.
func dateField(t *schema.Type, named any) string {
	if name, _ := named.(string); strings.TrimSpace(name) != "" {
		if f, ok := t.Field(name); ok && f.Type == "datetime" {
			return name
		}
		return ""
	}
	return t.DayField()
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
func (ws *Workspace) showFields(t *schema.Type, rec *store.Record, names []string) string {
	var parts []string
	for _, name := range names {
		f, ok := t.Field(name)
		if !ok {
			continue
		}
		v := Display(*f, rec.Fields[name])
		if f.Type == "ref" {
			v = ws.refTitle(*f, v)
		}
		if v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, " · ")
}

// everyEvent is the records of several types on their days, for a
// calendar that shows them together: the types in only, or every listed
// type when only is nil. Each event carries its kind, the type's name,
// for the kinds to narrow by and to tell two of one name apart; a thing
// that repeats is on each of its days in the month shown.
func (ws *Workspace) everyEvent(now time.Time, month string, only []*schema.Type) []any {
	events := []any{}
	types := only
	if types == nil {
		for _, t := range ws.Store.Types().Types {
			if !t.Internal && ws.Listed(t) {
				types = append(types, t)
			}
		}
	}
	for _, t := range types {
		field := t.DayField()
		if field == "" {
			continue
		}
		recs, err := query.Filter(ws.Store, t, nil, field, 0, now)
		if err != nil {
			continue
		}
		for _, rec := range recs {
			if ev := ws.eventOf(t, rec, field); ev != nil {
				ev["kind"] = t.Name
				events = append(events, ev)
				events = append(events, ws.repeatedIn(t, rec, field, ev, month)...)
			}
		}
	}
	return events
}
