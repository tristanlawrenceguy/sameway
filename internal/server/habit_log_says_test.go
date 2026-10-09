package server_test

import (
	"strings"
	"testing"
)

// Logging says what was logged and where it stands now, with Undo there,
// not only in the activity log.
func TestLoggingSaysWhatAndWhereItStands(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	habit, err := a.Store.Create("habit", map[string]any{"name": "Water", "cadence": "day", "target": 8, "unit": "glasses"})
	if err != nil {
		t.Fatal(err)
	}
	page := after(t, h, postForm(t, h, "/habit/"+habit.ID+"/log", nil)).Body.String()
	for _, want := range []string{"Water: 1 glass logged.", "Now 1 of 8 glasses.", `/undo" class="sw-outcome__undo"`, "Undo<span class=\"sw-visually-hidden\"> Water 1 glass</span>"} {
		if !strings.Contains(page, want) {
			t.Errorf("logging should say %q\n%s", want, truncate(page))
		}
	}
	bad := after(t, h, postForm(t, h, "/habit/"+habit.ID+"/log", map[string][]string{"amount": {"lots"}})).Body.String()
	if !strings.Contains(bad, "Water not logged") || !strings.Contains(bad, `data-field="log-`+habit.ID+`"`) {
		t.Errorf("a bad amount names the habit and leads to its field\n%s", truncate(bad))
	}
}
