package blocks

import (
	"slices"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// logForDay is what a calendar's day view offers when it shows what was
// logged: each habit it shows, as it stood that day, with Log filled in
// for that day, so a day missed can be logged where it is seen. A
// calendar of one habit's entries (where habit=<id>) offers that habit;
// one of all entries, of everything, or of several kinds with entry among
// them (types), offers every habit kept.
func (ws *Workspace) logForDay(typeName string, kinds, where []string, day time.Time) []any {
	shows := kinds // types, when given, are what the calendar shows
	if len(shows) == 0 {
		shows = []string{typeName}
	}
	if !slices.Contains(shows, records.EntryType) && !slices.Contains(shows, "all") {
		return nil
	}
	if _, ok := ws.Store.Types().Get(records.HabitType); !ok {
		return nil
	}
	only := ""
	for _, w := range where {
		if v, ok := strings.CutPrefix(w, "habit="); ok {
			only = v
		}
	}
	recs, _ := ws.Store.List(records.HabitType, store.ListOptions{OrderBy: "created_at"})
	asOf := time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 0, time.Local)
	on := day.Format("2006-01-02")
	habits := []any{}
	for _, rec := range recs {
		if archived, _ := rec.Fields["archived"].(bool); archived || (only != "" && rec.ID != only) {
			continue
		}
		item := Standing(ws.Store, rec, asOf)
		item["dated"], item["on"] = true, on
		habits = append(habits, item)
	}
	if len(habits) == 0 {
		return nil
	}
	label := "Log for " + day.Format("Mon 2 Jan")
	return []any{map[string]any{"component": TrackerComponent, "props": map[string]any{"label": label, "level": 4, "habits": habits}}}
}
