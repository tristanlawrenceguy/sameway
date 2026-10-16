package agenda_test

import (
	"strings"
	"testing"
)

// A daily habit usually kept says on Today the days of the last week it
// was not; two habits nothing tells apart are said once, with the press
// to keep the newest.
func TestTodaySaysTheDaysAHabitWasMissed(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	pills, _ := a.Store.Create("habit", map[string]any{"name": "Morning tablets"})
	day := func(n int) string { return tonight.AddDate(0, 0, n).Format("2006-01-02") }
	for _, n := range []int{-6, -5, -4, -1} { // kept four of the six days before, not -3 or -2
		a.Store.Create("entry", map[string]any{"habit": pills.ID, "at": day(n), "amount": 1})
	}
	a.Store.Create("habit", map[string]any{"name": "Water"})
	a.Store.Create("habit", map[string]any{"name": "Water"})
	page := get(t, h, "/today").Body.String()
	missed := tonight.AddDate(0, 0, -3).Format("Mon") + " and " + tonight.AddDate(0, 0, -2).Format("Mon")
	if !strings.Contains(page, "not kept "+missed) {
		t.Errorf("the days missed are said: want %q in %s", missed, truncate(page))
	}
	if !strings.Contains(page, "2 habits called Water, and nothing tells them apart.") || !strings.Contains(page, "Keep the newest") {
		t.Errorf("twins are said once, with the press: %s", truncate(page))
	}
}
