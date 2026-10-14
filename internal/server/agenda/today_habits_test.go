package agenda_test

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

// Someone caring for a relative's tablets sees on Today what is not yet
// taken: each daily habit not met, with where it stands and Log; once met
// it leaves. A weekly habit, an archived one, and one that only records
// are not there. An agent reads the same list in /api/today.
func TestTodayListsDailyHabitsNotMet(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	pills, _ := a.Store.Create("habit", map[string]any{"name": "Mum's tablets", "target": 2})
	a.Store.Create("habit", map[string]any{"name": "Run 20 km", "cadence": "week", "target": 20, "unit": "km"})
	a.Store.Create("habit", map[string]any{"name": "Old stretches", "archived": true})
	a.Store.Create("habit", map[string]any{"name": "Weight", "aim": "record", "unit": "kg"})

	page := get(t, h, "/today").Body.String()
	i := strings.Index(page, "<h2>Habits</h2>")
	if i < 0 {
		t.Fatalf("Today has no Habits: %s", truncate(page))
	}
	habits := page[i : i+strings.Index(page[i:], "</ul>")]
	if !strings.Contains(habits, "Mum&#39;s tablets") || !strings.Contains(habits, `action="/habit/`+pills.ID+`/log"`) {
		t.Errorf("a daily habit not met is there with Log: %s", habits)
	}
	for _, not := range []string{"Run 20 km", "Old stretches", "Weight"} {
		if strings.Contains(habits, not) {
			t.Errorf("%s is not today's to do: %s", not, habits)
		}
	}

	var api struct {
		Habits []struct{ Title, Progress string }
	}
	json.Unmarshal(get(t, h, "/api/today").Body.Bytes(), &api)
	if len(api.Habits) != 1 || api.Habits[0].Title != "Mum's tablets" || api.Habits[0].Progress == "" {
		t.Errorf("an agent reads the same: %+v", api.Habits)
	}

	postForm(t, h, "/habit/"+pills.ID+"/log", url.Values{})
	if page := get(t, h, "/today").Body.String(); !strings.Contains(page, "Mum&#39;s tablets") {
		t.Error("one of two taken, it is still to do")
	}
	postForm(t, h, "/habit/"+pills.ID+"/log", url.Values{})
	if page := get(t, h, "/today").Body.String(); strings.Contains(page, "<h2>Habits</h2>") {
		t.Error("met, it leaves Today")
	}
}
