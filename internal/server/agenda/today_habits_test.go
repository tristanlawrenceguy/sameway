package agenda_test

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

// Someone caring for a relative's tablets sees on Today what is not yet
// taken: each habit not met, with where it stands and Log; once met it
// leaves. A weekly cadence habit (not met) and one that only records both
// appear because they are not met today. An archived habit stays absent.
// An agent reads the same list in /api/today.
func TestTodayListsDailyHabitsNotMet(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	pills, _ := a.Store.Create("habit", map[string]any{"name": "Mum's tablets", "target": 2})
	_, _ = a.Store.Create("habit", map[string]any{"name": "Run 20 km", "cadence": "week", "target": 20, "unit": "km"})
	a.Store.Create("habit", map[string]any{"name": "Old stretches", "archived": true})
	_, _ = a.Store.Create("habit", map[string]any{"name": "Weight", "aim": "record", "unit": "kg"})

	page := get(t, h, "/today").Body.String()
	i := strings.Index(page, "<h2>Habits</h2>")
	if i < 0 {
		t.Fatalf("Today has no Habits: %s", truncate(page))
	}
	habits := page[i : i+strings.Index(page[i:], "</ul>")]
	if !strings.Contains(habits, "Mum&#39;s tablets") || !strings.Contains(habits, `action="/habit/`+pills.ID+`/log"`) {
		t.Errorf("a daily habit not met is there with Log: %s", habits)
	}
	// Weekly cadence and record-aim habits are not met → must appear.
	for _, want := range []string{"Run 20 km", "Weight"} {
		if !strings.Contains(habits, want) {
			t.Errorf("unmet %s should be in Habits: %s", want, habits)
		}
	}
	// Archived habit must not appear.
	if strings.Contains(habits, "Old stretches") {
		t.Errorf("archived habit must not appear: %s", habits)
	}

	var api struct {
		Habits []struct{ Title, Progress string }
	}
	json.Unmarshal(get(t, h, "/api/today").Body.Bytes(), &api)
	if len(api.Habits) < 3 || api.Habits[0].Progress == "" {
		t.Errorf("an agent reads the same: %+v", api.Habits)
	}

	postForm(t, h, "/habit/"+pills.ID+"/log", url.Values{})
	if page := get(t, h, "/today").Body.String(); !strings.Contains(page, "Mum&#39;s tablets") {
		t.Error("one of two taken, it is still to do")
	}
	postForm(t, h, "/habit/"+pills.ID+"/log", url.Values{})
	if page := get(t, h, "/today").Body.String(); !strings.Contains(page, "<h2>Habits</h2>") {
		t.Error("habits section should still exist with weekly/record unmet")
	} else {
		i2 := strings.Index(page, "<h2>Habits</h2>")
		habits2 := page[i2 : i2+strings.Index(page[i2:], "</ul>")]
		if !strings.Contains(habits2, "Run 20 km") || !strings.Contains(habits2, "Weight") {
			t.Errorf("weekly cadence and record-aim should still appear after daily is met: %s", habits2)
		}
	}
}
