package agenda_test

import (
	"encoding/json"
	"testing"
)

// A team lead planning on a Sunday evening reads the week in one answer:
// /api/today?days=7 adds the six days after today, each as Today will list
// it on the day; without days, the answer is today alone.
func TestTodayReadsTheDaysAhead(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	day := func(n int) string { return tonight.AddDate(0, 0, n).Format("2006-01-02") }
	a.Store.Create("task", map[string]any{"title": "Send the rota", "due": day(2)})
	a.Store.Create("project", map[string]any{"title": "Quarterly report", "due": day(4)})
	a.Store.Create("event", map[string]any{"title": "Team standup", "starts": day(1) + "T09:00:00Z"})
	a.Store.Create("task", map[string]any{"title": "Too far off", "due": day(9)})

	var week struct {
		Ahead []struct {
			Date      string
			Due       []struct{ Title string }
			Deadlines []struct{ Title string }
			Events    []struct{ Title string }
		}
	}
	json.Unmarshal(get(t, h, "/api/today?days=7").Body.Bytes(), &week)
	if len(week.Ahead) != 6 {
		t.Fatalf("seven days are today and the six after: %d ahead", len(week.Ahead))
	}
	if d := week.Ahead[0]; d.Date != day(1) || len(d.Events) != 1 || d.Events[0].Title != "Team standup" {
		t.Errorf("tomorrow has its event: %+v", d)
	}
	if d := week.Ahead[1]; len(d.Due) != 1 || d.Due[0].Title != "Send the rota" {
		t.Errorf("the day after has its task: %+v", d)
	}
	if d := week.Ahead[3]; len(d.Deadlines) != 1 {
		t.Errorf("a project's deadline is on its day: %+v", d)
	}
	var today map[string]any
	json.Unmarshal(get(t, h, "/api/today").Body.Bytes(), &today)
	if _, ok := today["ahead"]; ok {
		t.Error("without days, the answer is today alone")
	}
}
