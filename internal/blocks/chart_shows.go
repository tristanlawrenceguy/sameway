package blocks

import (
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// chartShows is a chart in a few words: what it draws and over how many
// bars or points, and over a date, which days, weeks or months, so one
// bar for a month is never read as a month of days: Amount of entries by
// At: 30 days, 2026-09-01 to 2026-09-30, in glasses.
func chartShows(w *Workspace, props, out map[string]any) string {
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
		t, _ := w.Store.Types().Get(typeName)
		by, _ := props["by"].(string)
		if len(series) == 0 {
			return NothingYet(t.Name, Strs(props["where"]), "")
		}
		if ByDate(t, by) {
			period, _ := props["period"].(string)
			if period == "" {
				period = "month"
			}
			first, last, n := DateSpan(out)
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

// ByDate is whether a chart groups by a date: created_at, updated_at or
// a date field.
func ByDate(t *schema.Type, by string) bool {
	if by == "created_at" || by == "updated_at" {
		return true
	}
	f, ok := t.Field(by)
	return ok && f.Type == "datetime"
}

// DateSpan is the first and last group of a chart by a date, and how
// many groups there are.
func DateSpan(out map[string]any) (first, last string, n int) {
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
