package server_test

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// A calendar given a content type shows that type's records on their
// days, as links to their pages, and never needs telling what day it is.
func TestACalendarShowsRecordsOnTheirDays(t *testing.T) {
	_, h := newApp(t)
	now := time.Now()
	day := func(d int) string { return now.AddDate(0, 0, d).Format("2006-01-02") + "T00:00:00Z" }
	var soon struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Order compost", "due": now.Add(2 * time.Hour).UTC().Format(time.RFC3339)}), &soon)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Call the dentist", "due": day(1), "done": true}), http.StatusCreated)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Plant garlic"}), http.StatusCreated)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "calendar", "props": map[string]any{"type": "task", "where": []string{"done=false"}, "caption": "Due"},
	}), http.StatusCreated)

	page := get(t, h, "/").Body.String()
	for _, want := range []string{`data-month="` + now.Format("2006-01") + `"`, `aria-current="date"`, `href="/t/task/` + soon.ID + `">Order compost</a>`, "<caption>Due</caption>"} {
		if !strings.Contains(page, want) {
			t.Errorf("the calendar should show this month, today, and the task as a link, missing %s", want)
		}
	}
	for _, not := range []string{"Call the dentist", "Plant garlic"} {
		if strings.Contains(page, not) {
			t.Errorf("%s is done or has no day and should not show", not)
		}
	}

	// A calendar with no type keeps the events it was given, and still
	// learns the month and today.
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "calendar", "props": map[string]any{"caption": "Given", "events": []map[string]any{{"date": now.Format("2006-01-02"), "label": "Market day"}}},
	}), http.StatusCreated)
	if page := get(t, h, "/").Body.String(); !strings.Contains(page, "Market day") || strings.Count(page, `aria-current="date"`) < 2 {
		t.Error("a calendar with its own events shows them, with today marked")
	}
}
