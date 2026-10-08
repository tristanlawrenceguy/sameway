package blocks

import (
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A thing that repeats is on the calendar on each day it falls in the
// month shown, as a calendar shows a meeting every Tuesday on every
// Tuesday, not only on the next one. Each says that it repeats in words,
// "Repeats every Tuesday", once, beside its name: not an icon alone, which
// is not read out. The days are worked out for the month shown and no
// further, so a repeat that goes on for ever is a month's worth of days.
// The later ones lead to the record but carry no tick: ticking one would
// tick the one due now.

// mostInMonth is the most times one repeat is shown in a month: every day.
const mostInMonth = 31

// repeatedIn is a record's later times in the month shown, as events, and
// ev, its own event, says it repeats too. Nothing for a record that does
// not repeat, or a calendar drawn from a day it does not move.
func (ws *Workspace) repeatedIn(t *schema.Type, rec *store.Record, field string, ev map[string]any, month string) []any {
	repeat, day, ok := t.Repeats()
	rule, _ := rec.Fields[repeat].(string)
	v, _ := rec.Fields[field].(string)
	if !ok || day != field || rule == "" || ev == nil {
		return nil
	}
	said := "Repeats " + when.RepeatText(rule)
	ev["repeats"] = said
	first, err := time.Parse("2006-01", month)
	if err != nil {
		return nil
	}
	var out []any
	for _, at := range when.Occurrences(rule, v, first.Format("2006-01-02"), first.AddDate(0, 1, -1).Format("2006-01-02"), mostInMonth) {
		again := *rec
		again.Fields = map[string]any{}
		for k, x := range rec.Fields {
			again.Fields[k] = x
		}
		again.Fields[field] = at
		later := ws.eventOf(t, &again, field)
		if later == nil {
			continue
		}
		delete(later, "actions")
		later["repeats"] = said
		for _, k := range []string{"meta", "kind"} {
			if v, ok := ev[k]; ok {
				later[k] = v
			}
		}
		out = append(out, later)
	}
	return out
}
