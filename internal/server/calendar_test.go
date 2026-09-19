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

// A day with no time shows no time and stays on its day wherever the
// server is; the block's own page reaches the months either side and
// the list the calendar draws from.
func TestACalendarMovesBetweenMonths(t *testing.T) {
	_, h := newApp(t)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Order compost", "due": "2026-09-19T00:00:00Z"}), http.StatusCreated)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Harvest", "due": "2026-10-03T00:00:00Z"}), http.StatusCreated)
	var block struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "calendar", "props": map[string]any{"type": "task", "month": "2026-09", "detail": "page"},
	}), &block)

	page := get(t, h, "/canvas/"+block.ID).Body.String()
	for _, want := range []string{
		`data-month="2026-09"`, `>Order compost</a>`,
		`<nav class="sw-calendar__months" aria-label="Other months">`,
		`href="/canvas/` + block.ID + `?month=2026-08" rel="prev">&larr; August 2026</a>`,
		`href="/canvas/` + block.ID + `?month=2026-10" rel="next">October 2026 &rarr;</a>`,
		`<a class="sw-link sw-link--button" href="/t/task?order=due">See the list</a>`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the block page should carry %s", want)
		}
	}
	if strings.Contains(page, "<time>") || strings.Contains(page, "Harvest") {
		t.Error("a day with no time shows no time, and next month's task is not in this month")
	}

	next := get(t, h, "/canvas/"+block.ID+"?month=2026-10").Body.String()
	if !strings.Contains(next, `data-month="2026-10"`) || !strings.Contains(next, ">Harvest</a>") || strings.Contains(next, "Order compost") {
		t.Error("?month= shows that month's records on the block's page")
	}
	if !strings.Contains(next, `?month=2026-11" rel="next">November 2026`) {
		t.Error("the months either side follow the month shown")
	}
	// The canvas keeps the block's own month; the stored block is untouched.
	if got := get(t, h, "/").Body.String(); !strings.Contains(got, `data-month="2026-09"`) {
		t.Error("the canvas shows the block's own month")
	}
}
