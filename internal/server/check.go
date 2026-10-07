package server

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/render"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// blockCheck resolves a block's records the way its page will, without
// drawing it: what it would show, in a few words, or why it cannot be
// shown, in the very words the page would say it, since it is the same
// resolving. Every write of a block goes through it (chat.Service.Check),
// so a block that could only say it is set up wrong is refused when it is
// written, not found broken after the one who wrote it has said "done".
func (s *Server) blockCheck(component string, props map[string]any) (shows, problem string) {
	var out map[string]any
	switch component {
	case collectionComponent:
		out = s.resolveCollection(props, "")
	case calendarComponent:
		out = s.resolveCalendar(props, "")
	case trackerComponent:
		out = s.resolveTracker(props)
	case chartComponent:
		out = s.resolveChart(props)
	default:
		return "", ""
	}
	if p, _ := out["problem"].(string); p != "" {
		return "", p
	}
	if p := s.meaningProblem(component, props); p != "" {
		return "", p
	}
	switch component {
	case collectionComponent:
		return s.collectionShows(props), ""
	case calendarComponent:
		return s.calendarShows(props, out), ""
	case trackerComponent:
		return trackerShows(props, out), ""
	}
	return s.chartShows(props, out), ""
}

// noType says a type is not there, and what is: the one likely meant,
// when the name is near one, then every one.
func (s *Server) noType(name string) string {
	names := s.app.Types.Names()
	out := "there is no content type " + name
	if near := render.Nearest(name, names); near != "" {
		out += " (did you mean " + near + "?)"
	}
	return out + "; the workspace has " + strings.Join(names, ", ")
}

// fieldsOfKind is the names of a type's fields of the kinds given, or
// every field when none is given.
func fieldsOfKind(t *schema.Type, kinds ...string) []string {
	var out []string
	for _, f := range t.Shown() {
		if len(kinds) == 0 {
			out = append(out, f.Name)
			continue
		}
		for _, k := range kinds {
			if f.Type == k {
				out = append(out, f.Name)
			}
		}
	}
	return out
}

// collectionShows is a list in a few words: how many, which, in what
// order, and how: 3 tasks, not done, by due.
func (s *Server) collectionShows(props map[string]any) string {
	typeName, _ := props["type"].(string)
	t, _ := s.app.Types.Get(typeName)
	where := strs(props["where"])
	order, _ := props["order"].(string)
	recs, _ := query.Filter(s.app.Store, t, where, order, 0, time.Now())
	if len(recs) == 0 {
		return nothingYet(t.Name, where, "")
	}
	out := schema.Count(len(recs), t.Name)
	if w := query.Words(t, where); w != "" {
		out += ", " + w
	}
	out += orderWords(t, order)
	switch props["as"] {
	case "board":
		if f, err := boardField(t, props["by"]); err == nil {
			out += ", as a board by " + strings.ToLower(f.Display())
		}
	case "table":
		out += ", as a table"
	case "cards":
		out += ", as cards"
	}
	return out
}

// chartShows is a chart in a few words: what it draws and over how many
// bars or points, and over a date, which days, weeks or months, so one
// bar for a month is never read as a month of days: Amount of entries by
// At: 30 days, 2026-09-01 to 2026-09-30, in glasses.
func (s *Server) chartShows(props, out map[string]any) string {
	series, _ := out["series"].([]any)
	caption, _ := out["caption"].(string)
	if caption == "" {
		caption = "the numbers given"
	}
	groups := fmt.Sprintf("%d groups", len(series))
	if len(series) == 1 {
		groups = "1 group"
	}
	if typeName, _ := props["type"].(string); typeName != "" {
		t, _ := s.app.Types.Get(typeName)
		by, _ := props["by"].(string)
		if len(series) == 0 {
			return nothingYet(t.Name, strs(props["where"]), "")
		}
		if byDate(t, by) {
			period, _ := props["period"].(string)
			if period == "" {
				period = "month"
			}
			first, last, n := dateSpan(out)
			groups = fmt.Sprintf("%d %ss, %s to %s", n, period, first, last)
			if n == 1 {
				groups = "1 " + period + ", " + first
			}
		}
	}
	shows := caption + ": " + groups
	if unit, _ := props["unit"].(string); unit != "" {
		shows += ", in " + unit
	}
	return shows
}

// calendarShows is a calendar in a few words: whose dates, and how many.
func (s *Server) calendarShows(props, out map[string]any) string {
	typeName, _ := props["type"].(string)
	events, _ := out["events"].([]any)
	if kinds := strs(props["types"]); len(kinds) > 0 && !slices.Contains(kinds, "all") {
		var names []string
		for _, k := range kinds {
			names = append(names, schema.Plural(k))
		}
		return andList(names) + " with a date, " + fmt.Sprint(len(events)) + " in all"
	}
	switch typeName {
	case "":
		return fmt.Sprintf("the %d events given", len(events))
	case "all":
		return everyKind(events)
	}
	t, _ := s.app.Types.Get(typeName)
	field := dateField(t, props["date"])
	where := strs(props["where"])
	recs, _ := query.Filter(s.app.Store, t, where, field, 0, time.Now())
	n := 0
	for _, rec := range recs {
		if v, _ := rec.Fields[field].(string); v != "" {
			n++
		}
	}
	if n == 0 {
		return nothingYet(t.Name, where, t.FieldWords(field))
	}
	shows := schema.Count(n, t.Name) + " by " + t.FieldWords(field)
	if w := query.Words(t, where); w != "" {
		shows += ", " + w
	}
	return shows
}

// trackerShows is a tracker in a few words: how many habits, and which.
func trackerShows(props, out map[string]any) string {
	habits, _ := out["habits"].([]any)
	shows := schema.Count(len(habits), HabitType)
	if tags := strs(props["tags"]); len(tags) > 0 {
		shows += " tagged " + strings.Join(tags, " or ")
	}
	if len(strs(props["habits"])) > 0 {
		var names []string
		for _, h := range habits {
			if m, ok := h.(map[string]any); ok {
				names = append(names, str(m["name"], ""))
			}
		}
		shows += ": " + andList(names)
	}
	return shows
}
