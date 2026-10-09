package blocks

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// What a list or a calendar shows, in a few words, said to the one who
// writes the block.

// collectionShows is a list in a few words: how many, which, in what
// order, and how: 3 tasks, not done, by due.
func (ws *Workspace) collectionShows(props map[string]any) string {
	typeName, _ := props["type"].(string)
	t, _ := ws.Store.Types().Get(typeName)
	where := Strs(props["where"])
	order, _ := props["order"].(string)
	recs, _ := query.Filter(ws.Store, t, where, order, 0, ws.now())
	if len(recs) == 0 {
		return NothingYet(t.Name, where, "")
	}
	out := schema.Count(len(recs), t.Name)
	if w := query.Words(t, where); w != "" {
		out += ", " + w
	}
	out += OrderWords(t, order)
	switch props["as"] {
	case "board":
		if f, err := BoardField(t, props["by"]); err == nil {
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
func (ws *Workspace) calendarShows(props, out map[string]any) string {
	typeName, _ := props["type"].(string)
	events, _ := out["events"].([]any)
	if kinds := Strs(props["types"]); len(kinds) > 0 && !slices.Contains(kinds, "all") {
		var names []string
		for _, k := range kinds {
			names = append(names, schema.Plural(k))
		}
		return AndList(names) + " with a date, " + fmt.Sprint(len(events)) + " in all"
	}
	switch typeName {
	case "":
		return fmt.Sprintf("the %d events given", len(events))
	case "all":
		return everyKind(events)
	}
	t, _ := ws.Store.Types().Get(typeName)
	field := dateField(t, props["date"])
	where := Strs(props["where"])
	recs, _ := query.Filter(ws.Store, t, where, field, 0, ws.now())
	n := 0
	for _, rec := range recs {
		if v, _ := rec.Fields[field].(string); v != "" {
			n++
		}
	}
	if n == 0 {
		return NothingYet(t.Name, where, t.FieldWords(field))
	}
	shows := schema.Count(n, t.Name) + " by " + t.FieldWords(field)
	if w := query.Words(t, where); w != "" {
		shows += ", " + w
	}
	return shows
}

// everyKind is a calendar of everything in a few words: how many events,
// and how many of each kind, most first, so a kind that floods the rest
// (a habit's entries, day after day, over a few tasks) is seen when the
// block is written, not only on the page: 44 events: 32 entries, 9 tasks,
// 3 reminders.
func everyKind(events []any) string {
	if len(events) == 0 {
		return "nothing yet: no record has a date (it fills in as dated records are added)"
	}
	counts := map[string]int{}
	var kinds []string
	for _, e := range events {
		ev, _ := e.(map[string]any)
		kind, _ := ev["kind"].(string) // the type's name, as everyEvent sets it
		if kind == "" {
			kind = "other"
		}
		if counts[kind] == 0 {
			kinds = append(kinds, kind)
		}
		counts[kind]++
	}
	sort.SliceStable(kinds, func(i, j int) bool { return counts[kinds[i]] > counts[kinds[j]] })
	out := fmt.Sprintf("every record with a date, %d in all", len(events))
	if len(kinds) < 2 {
		return out + ": " + schema.Count(len(events), kinds[0])
	}
	parts := make([]string, len(kinds))
	for i, k := range kinds {
		parts[i] = schema.Count(counts[k], k)
	}
	out += ": " + strings.Join(parts, ", ")
	if top := kinds[0]; counts[top]*2 > len(events) {
		// The kinds it buries, named as types takes them.
		rest := make([]string, 0, len(kinds)-1)
		for _, k := range kinds[1:] {
			if k != "other" {
				rest = append(rest, strconv.Quote(k))
			}
		}
		out += fmt.Sprintf("; mostly %s, which crowd out the rest: a calendar of only the kinds wanted (types: [%s]) leaves them out", schema.Plural(top), strings.Join(rest, ", "))
	}
	return out
}
