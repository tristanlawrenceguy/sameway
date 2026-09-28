package server

import (
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// logForDay is what a calendar's day view offers when it shows what was
// logged: each habit it shows, as it stood that day, with Log filled in
// for that day, so a day missed can be logged where it is seen. A
// calendar of one habit's entries (where habit=<id>) offers that habit;
// one of all entries, or of everything, offers every habit kept.
func (s *Server) logForDay(typeName string, where []string, day time.Time) []any {
	if typeName != EntryType && typeName != "all" {
		return nil
	}
	if _, ok := s.app.Types.Get(HabitType); !ok {
		return nil
	}
	only := ""
	for _, w := range where {
		if v, ok := strings.CutPrefix(w, "habit="); ok {
			only = v
		}
	}
	recs, _ := s.app.Store.List(HabitType, store.ListOptions{OrderBy: "created_at"})
	asOf := time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 0, time.Local)
	on := day.Format("2006-01-02")
	habits := []any{}
	for _, rec := range recs {
		if archived, _ := rec.Fields["archived"].(bool); archived || (only != "" && rec.ID != only) {
			continue
		}
		item := s.standing(rec, asOf)
		item["dated"], item["on"] = true, on
		habits = append(habits, item)
	}
	if len(habits) == 0 {
		return nil
	}
	label := "Log for " + day.Format("Mon 2 Jan")
	return []any{map[string]any{"component": trackerComponent, "props": map[string]any{"label": label, "habits": habits}}}
}
