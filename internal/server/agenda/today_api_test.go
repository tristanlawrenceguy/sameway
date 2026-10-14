package agenda_test

import (
	"encoding/json"
	"fmt"
	"testing"
)

// GET /api/today is the Today page as an agent reads it: what is late and
// due, and on a long day the three to start with, led by what the
// assistant said to start with, and how many more wait.
func TestTodayIsReadAsJSON(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	day := func(n int) string { return tonight.AddDate(0, 0, n).Format("2006-01-02") }
	for i := 1; i <= 7; i++ {
		a.Store.Create("task", map[string]any{"title": fmt.Sprintf("Late errand %d", i), "due": day(-i)})
	}
	a.Store.Create("task", map[string]any{"title": "Call the dentist", "tags": []any{"start"}})
	var got struct {
		Day       string `json:"day"`
		Page      string `json:"page"`
		Late      []struct{ Title, Page, When string }
		StartWith []struct {
			Title   string `json:"title"`
			Matters bool   `json:"matters"`
		} `json:"start_with"`
		Rest int `json:"rest"`
	}
	res := get(t, h, "/api/today")
	if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil || res.Code != 200 {
		t.Fatalf("today as JSON: %d %v %s", res.Code, err, res.Body)
	}
	if got.Page != "/today" || got.Day == "" || len(got.Late) != 7 || got.Late[0].Page == "" || got.Late[0].When == "" {
		t.Errorf("the day, its page, and what is late with pages and when: %+v", got)
	}
	if len(got.StartWith) != 3 || got.StartWith[0].Title != "Call the dentist" || !got.StartWith[0].Matters || got.Rest != 5 {
		t.Errorf("three to start with, led by the one tagged start, and the rest counted: %+v rest %d", got.StartWith, got.Rest)
	}
}
