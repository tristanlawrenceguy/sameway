package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A miss is said without blame: the days before the first entry are not
// misses, a day short says how far it got, and the day after one miss
// says today gets back on track rather than a run of nought. After two,
// the row says nothing of it.
func TestAHabitSaysAMissWithoutBlame(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	water, err := a.Store.Create(server.HabitType, map[string]any{"name": "Water", "cadence": "day", "target": 8, "unit": "glasses"})
	if err != nil {
		t.Fatal(err)
	}
	stretch, _ := a.Store.Create(server.HabitType, map[string]any{"name": "Stretch"})
	day := func(n int) string { return when.Store(tonight.AddDate(0, 0, -n), false) }
	a.Store.Create(server.EntryType, map[string]any{"habit": water.ID, "at": day(3), "amount": 8})
	a.Store.Create(server.EntryType, map[string]any{"habit": water.ID, "at": day(2), "amount": 8})
	a.Store.Create(server.EntryType, map[string]any{"habit": water.ID, "at": day(1), "amount": 5})
	a.Store.Create(server.EntryType, map[string]any{"habit": stretch.ID, "at": day(3)})

	page := get(t, h, "/t/"+server.HabitType+"/"+water.ID).Body.String()
	for _, want := range []string{
		`Missed yesterday; today gets back on track`,
		`aria-label="The last 4 days of Water"`, // from the first entry, not seven
		`Tue 10 Mar: 5 of 8 glasses`,
		`Reached the target of 8 glasses on 2 of the 4 days since Sun 8 Mar. Best run: 2 days in a row.`,
		`The 4 days so far`,
		`Amount (glasses)`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("a habit's page says a miss without blame, missing %q", want)
		}
	}
	for _, not := range []string{`Sat 7 Mar: not met`, `0 days in a row`, `Glasses (glasses)`} {
		if strings.Contains(page, not) {
			t.Errorf("a habit's page should not say %q", not)
		}
	}
	page = get(t, h, "/t/"+server.HabitType+"/"+stretch.ID).Body.String()
	if strings.Contains(page, "back on track") || !strings.Contains(page, "Tue 10 Mar: nothing logged") {
		t.Error("after two misses the row says nothing of a run, and a day with nothing says so")
	}
}
