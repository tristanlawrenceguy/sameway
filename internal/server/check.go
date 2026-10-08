package server

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"
	"github.com/tristanlawrenceguy/sameway/internal/query"
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
	case trackerComponent, chartComponent:
		out = blocks.Resolve(s.app.Blocks, component, props, blocks.Place{})
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
	}
	k, _ := blocks.Of(component)
	return k.Shows(s.app.Blocks, props, out), ""
}

// noType says a type is not there, and what is (blocks.NoType).
func (s *Server) noType(name string) string { return s.app.Blocks.NoType(name) }

// collectionShows is a list in a few words: how many, which, in what
// order, and how: 3 tasks, not done, by due.
func (s *Server) collectionShows(props map[string]any) string {
	typeName, _ := props["type"].(string)
	t, _ := s.app.Types.Get(typeName)
	where := strs(props["where"])
	order, _ := props["order"].(string)
	recs, _ := query.Filter(s.app.Store, t, where, order, 0, time.Now())
	if len(recs) == 0 {
		return blocks.NothingYet(t.Name, where, "")
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
		return blocks.NothingYet(t.Name, where, t.FieldWords(field))
	}
	shows := schema.Count(n, t.Name) + " by " + t.FieldWords(field)
	if w := query.Words(t, where); w != "" {
		shows += ", " + w
	}
	return shows
}
