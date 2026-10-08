package blocks

import (
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// Check resolves a block's records the way its page will, without
// drawing it: what it would show, in a few words, or why it cannot be
// shown, in the very words the page would say it, since it is the same
// resolving. Every write of a block goes through it (chat's CheckBlock),
// so a block that could only say it is set up wrong is refused when it is
// written, not found broken after the one who wrote it has said "done".
// A component with nothing to count says nothing either way.
func Check(w *Workspace, component string, props map[string]any) (shows, problem string) {
	k, ok := kinds[component]
	if !ok || k.Shows == nil {
		return "", ""
	}
	out := k.Resolve(w, props, Place{})
	if p, _ := out["problem"].(string); p != "" {
		return "", p
	}
	if p := w.meaningProblem(component, props); p != "" {
		return "", p
	}
	return k.Shows(w, props, out), ""
}

// meaningProblem is why props that fit and resolve would still not show
// what their writer meant, said when the block is written: conditions no
// record can meet, and a chart by a date with no period, which drew one
// bar for a month of days a model then called daily. A block stored
// before this check still renders as it did (a chart by a date without a
// period groups by month); only a new write is refused, since the one
// writing it is there to pick.
func (w *Workspace) meaningProblem(component string, props map[string]any) string {
	typeName, _ := props["type"].(string)
	t, ok := w.Store.Types().Get(typeName)
	if !ok {
		return "" // no type, or "all": nothing of one type to check
	}
	if p := query.Contradiction(t, Strs(props["where"])); p != "" {
		return p
	}
	if component != ChartComponent {
		return ""
	}
	by, _ := props["by"].(string)
	if period, _ := props["period"].(string); period != "" || !ByDate(t, by) {
		return ""
	}
	out := fmt.Sprintf("grouping by a date needs a period: day, week or month (period: day draws one bar a day of %s, week one a week, month one a month)", by)
	withDay := copyProps(props)
	withDay["period"] = "day"
	if first, last, n := DateSpan(resolveChart(w, withDay, Place{})); n > 0 {
		out += fmt.Sprintf("; the %s it counts fall on %d days, %s to %s", schema.Plural(t.Name), n, first, last)
	}
	return out
}
