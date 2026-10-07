package server

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// meaningProblem is why props that fit and resolve would still not show
// what their writer meant, said when the block is written: conditions no
// record can meet, and a chart by a date with no period, which drew one
// bar for a month of days a model then called daily. A block stored
// before this check still renders as it did (a chart by a date without a
// period groups by month); only a new write is refused, since the one
// writing it is there to pick.
func (s *Server) meaningProblem(component string, props map[string]any) string {
	typeName, _ := props["type"].(string)
	t, ok := s.app.Types.Get(typeName)
	if !ok {
		return "" // no type, or "all": nothing of one type to check
	}
	if p := query.Contradiction(t, strs(props["where"])); p != "" {
		return p
	}
	if component != chartComponent {
		return ""
	}
	by, _ := props["by"].(string)
	if period, _ := props["period"].(string); period != "" || !byDate(t, by) {
		return ""
	}
	out := fmt.Sprintf("grouping by a date needs a period: day, week or month (period: day draws one bar a day of %s, week one a week, month one a month)", by)
	withDay := map[string]any{}
	for k, v := range props {
		withDay[k] = v
	}
	withDay["period"] = "day"
	if first, last, n := dateSpan(s.resolveChart(withDay)); n > 0 {
		out += fmt.Sprintf("; the %s it counts fall on %d days, %s to %s", schema.Plural(t.Name), n, first, last)
	}
	return out
}

// byDate is whether a chart groups by a date: created_at, updated_at or
// a date field.
func byDate(t *schema.Type, by string) bool {
	if by == "created_at" || by == "updated_at" {
		return true
	}
	f, ok := t.Field(by)
	return ok && f.Type == "datetime"
}

// dateSpan is the first and last group of a chart by a date, and how
// many groups there are.
func dateSpan(out map[string]any) (first, last string, n int) {
	series, _ := out["series"].([]any)
	if len(series) == 0 {
		return "", "", 0
	}
	label := func(v any) string {
		m, _ := v.(map[string]any)
		l, _ := m["label"].(string)
		return l
	}
	return label(series[0]), label(series[len(series)-1]), len(series)
}

// nothingYet is a block that shows nothing, said so a model cannot read
// it as done: which records it waits for (has: a field they need set), and why it is not refused.
// Records may be added later (a list of books to read, before the first
// book), so it is written; if some should show now, the conditions are
// wrong, and the writer is the one to know.
func nothingYet(typeName string, where []string, has string) string {
	one := schema.Words(typeName)
	out := "nothing yet: there are no " + schema.Plural(typeName)
	switch {
	case has != "" && len(where) > 0:
		out = "nothing yet: no " + one + " has a " + has + " and matches " + strings.Join(where, " and ")
	case has != "":
		out = "nothing yet: no " + one + " has a " + has
	case len(where) > 0:
		out = "nothing yet: no " + one + " matches " + strings.Join(where, " and ")
	}
	return out + " (it fills in as records are added; if some should show now, change the conditions)"
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
