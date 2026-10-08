package server

import (
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"
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
	if period, _ := props["period"].(string); period != "" || !blocks.ByDate(t, by) {
		return ""
	}
	out := fmt.Sprintf("grouping by a date needs a period: day, week or month (period: day draws one bar a day of %s, week one a week, month one a month)", by)
	withDay := map[string]any{}
	for k, v := range props {
		withDay[k] = v
	}
	withDay["period"] = "day"
	if first, last, n := blocks.DateSpan(blocks.Resolve(s.app.Blocks, chartComponent, withDay, blocks.Place{})); n > 0 {
		out += fmt.Sprintf("; the %s it counts fall on %d days, %s to %s", schema.Plural(t.Name), n, first, last)
	}
	return out
}
