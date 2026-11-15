package agenda_test

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

// Weekly cadence habits appear in Today's Habits section when not yet met.
func TestWeeklyCadenceHabitsAppear(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)

	daily, _ := a.Store.Create("habit", map[string]any{"name": "Mum's tablets", "target": 2})
	weekly, _ := a.Store.Create("habit", map[string]any{
		"name": "Run 20 km", "cadence": "week", "target": 20, "unit": "km",
	})
	weeklyID := weekly.ID

	page := get(t, h, "/today").Body.String()
	i := strings.Index(page, "<h2>Habits</h2>")
	if i < 0 {
		t.Fatalf("Today has no Habits: %s", truncate(page))
	}
	habits := page[i : i+strings.Index(page[i:], "</ul>")]

	// Daily habit not met must be present.
	if !strings.Contains(habits, "Mum&#39;s tablets") {
		t.Errorf("daily habit not met should appear: %s", habits)
	}
	if !strings.Contains(habits, `action="/habit/`+daily.ID+`/log"`) {
		t.Errorf("daily habit must have Log form: %s", habits)
	}

	// Weekly cadence habit not yet met this week must now appear.
	if !strings.Contains(habits, "Run 20 km") {
		t.Errorf("weekly cadence habit (not met) should appear in Habits: %s", habits)
	}
	if !strings.Contains(habits, `action="/habit/`+weeklyID+`/log"`) {
		t.Errorf("weekly habit must have Log form: %s", habits)
	}

	// Agent reads the same list via /api/today.
	var api struct {
		Habits []struct{ Title, Progress string }
	}
	json.Unmarshal(get(t, h, "/api/today").Body.Bytes(), &api)
	if len(api.Habits) < 2 || api.Habits[0].Title == "Run 20 km" || api.Habits[1].Title != "Run 20 km" {
		t.Errorf("API habits must include weekly cadence: %+v", api.Habits)
	}

	// Archived habit must still be absent.
	a.Store.Create("habit", map[string]any{"name": "Old stretches", "archived": true})
	page2 := get(t, h, "/today").Body.String()
	i2 := strings.Index(page2, "<h2>Habits</h2>")
	habits2 := page2[i2 : i2+strings.Index(page2[i2:], "</ul>")]
	if strings.Contains(habits2, "Old stretches") {
		t.Errorf("archived habit must not appear: %s", habits2)
	}

	// After logging daily twice it is met and leaves.
	postForm(t, h, "/habit/"+daily.ID+"/log", url.Values{})
	if page := get(t, h, "/today").Body.String(); !strings.Contains(page, "Mum&#39;s tablets") {
		t.Error("partially logged must still appear: 1/2 done")
	} else {
		i := strings.Index(page, "<h2>Habits</h2>")
		if i < 0 || !strings.Contains(page[i:i+strings.Index(page[i:], "</ul>")], "Mum&#39;s tablets") {
			t.Error("partially logged must still appear in Habits: 1/2 done")
		}
	}
	postForm(t, h, "/habit/"+daily.ID+"/log", url.Values{})
	if page := get(t, h, "/today").Body.String(); strings.Contains(page, "Mum&#39;s tablets") {
		i := strings.Index(page, "<h2>Habits</h2>")
		if i >= 0 && strings.Contains(page[i:i+strings.Index(page[i:], "</ul>")], "Mum&#39;s tablets") {
			t.Error("met daily habit should leave Today's Habits: still in habits section")
		}
	}
}

// Record-type aim habits (no target) appear on Today because meets() returns false.
func TestRecordAimHabitsAppear(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)

	// Need a daily habit too so the Habits section renders at all.
	dailyID, _ := a.Store.Create("habit", map[string]any{"name": "Mum's tablets", "target": 2})

	record, _ := a.Store.Create("habit", map[string]any{
		"name": "Weight", "aim": "record", "unit": "kg",
	})
	recordID := record.ID

	page := get(t, h, "/today").Body.String()
	i := strings.Index(page, "<h2>Habits</h2>")
	if i < 0 {
		t.Fatalf("Today has no Habits: %s", truncate(page))
	}
	habits := page[i : i+strings.Index(page[i:], "</ul>")]

	// Daily baseline must be present with its Log form.
	if !strings.Contains(habits, "Mum&#39;s tablets") || !strings.Contains(habits, `action="/habit/`+dailyID.ID+`/log"`) {
		t.Errorf("daily habit not met should appear: %s", habits)
	}

	// Record-type aim habit with no entry → meets() false → must appear.
	if !strings.Contains(habits, "Weight") {
		t.Errorf("record-aim habit (not met) should appear in Habits: %s", habits)
	}
	if !strings.Contains(habits, `action="/habit/`+recordID+`/log"`) {
		t.Errorf("record-aim habit must have Log form: %s", habits)
	}

	// Agent reads it via /api/today.
	var api struct {
		Habits []struct{ Title string }
	}
	json.Unmarshal(get(t, h, "/api/today").Body.Bytes(), &api)
	found := false
	for _, hb := range api.Habits {
		if hb.Title == "Weight" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("API habits must include record-aim habit: %+v", api.Habits)
	}
}

// Running miles: cadence=week, aim=record — the original bug case.
func TestRunningMilesAppears(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)

	// Need a daily habit so the Habits section renders at all.
	daily, _ := a.Store.Create("habit", map[string]any{"name": "Mum's tablets", "target": 2})

	miles, _ := a.Store.Create("habit", map[string]any{
		"name": "Running miles", "cadence": "week", "aim": "record", "unit": "km",
	})
	milesID := miles.ID

	page := get(t, h, "/today").Body.String()
	i := strings.Index(page, "<h2>Habits</h2>")
	if i < 0 {
		t.Fatalf("Today has no Habits: %s", truncate(page))
	}
	habits := page[i : i+strings.Index(page[i:], "</ul>")]

	// Daily baseline must be present.
	if !strings.Contains(habits, "Mum&#39;s tablets") {
		t.Errorf("daily habit not met should appear: %s", habits)
	}

	// A weekly record-aim habit with no entry is not met → must appear.
	if !strings.Contains(habits, "Running miles") {
		t.Errorf("weekly cadence + record aim should appear in Habits: %s", habits)
	}
	if !strings.Contains(habits, `action="/habit/`+milesID+`/log"`) {
		t.Errorf("running miles must have Log form: %s", habits)
	}

	var api struct {
		Habits []struct{ Title string }
	}
	json.Unmarshal(get(t, h, "/api/today").Body.Bytes(), &api)
	found := false
	for _, hb := range api.Habits {
		if hb.Title == "Running miles" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("API habits must include Running miles: %+v", api.Habits)
	}

	// Log an entry for this week — still not met (record aim always Met=false).
	a.Store.Create("entry", map[string]any{
		"habit":  milesID,
		"amount": 5.0,
		"at":     tonight.Format("2006-01-02T15:04:05Z"),
	})
	page = get(t, h, "/today").Body.String()
	i = strings.Index(page, "<h2>Habits</h2>")
	if i < 0 {
		t.Fatalf("Today has no Habits after logging: %s", truncate(page))
	}
	habits = page[i : i+strings.Index(page[i:], "</ul>")]

	// Record aim always returns Met=false, so it stays in "To do" even with entries.
	if !strings.Contains(habits, "Running miles") {
		t.Errorf("record-aim habit must stay after logging (meets()=false): %s", habits)
	}

	// Daily should still be there too.
	if !strings.Contains(habits, "Mum&#39;s tablets") {
		t.Errorf("daily baseline should still appear: %s", habits)
	}
	// After logging twice, daily is met and leaves.
	postForm(t, h, "/habit/"+daily.ID+"/log", url.Values{})
	if page := get(t, h, "/today").Body.String(); !strings.Contains(page, "Mum&#39;s tablets") {
		t.Error("partially logged must still appear: 1/2 done")
	} else {
		i := strings.Index(page, "<h2>Habits</h2>")
		if i < 0 || !strings.Contains(page[i:i+strings.Index(page[i:], "</ul>")], "Mum&#39;s tablets") {
			t.Error("partially logged must still appear in Habits: 1/2 done")
		}
	}
	postForm(t, h, "/habit/"+daily.ID+"/log", url.Values{})
	if page := get(t, h, "/today").Body.String(); strings.Contains(page, "Mum&#39;s tablets") {
		i := strings.Index(page, "<h2>Habits</h2>")
		if i >= 0 && strings.Contains(page[i:i+strings.Index(page[i:], "</ul>")], "Mum&#39;s tablets") {
			t.Error("met daily habit should leave Today's Habits: still in habits section")
		}
	}
	// Running miles stays because record aim always Met=false.
	if page := get(t, h, "/today").Body.String(); !strings.Contains(page, "Running miles") {
		t.Errorf("record-aim must stay after logging: %s", truncate(page))
	}
}
