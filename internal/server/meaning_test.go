package server_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// entries logs a habit on n days in a row, ending today, and returns the
// habit's id.
func entries(t *testing.T, a *app.App, n int) string {
	t.Helper()
	withHabit(t, a)
	habits, _ := a.Store.List("habit", store.ListOptions{})
	y, m, d := time.Now().Date()
	today := time.Date(y, m, d, 12, 0, 0, 0, time.Local)
	for i := 0; i < n; i++ {
		at := today.AddDate(0, 0, -i).Format(time.RFC3339)
		if _, err := a.Store.Create("entry", map[string]any{"habit": habits[0].ID, "at": at, "amount": 2}); err != nil {
			t.Fatal(err)
		}
	}
	return habits[0].ID
}

// Props that fit and resolve but could not mean what was written are
// refused with a fix, the same through the assistant's tools and the API:
// a chart by a date with no period (it drew one bar for a month of days,
// which a model called daily), and one field asked for two values (a
// list that could never show a book).
func TestABlockThatCouldNotMeanWhatWasWrittenIsRefused(t *testing.T) {
	a, h := newApp(t)
	entries(t, a, 30)
	first := time.Now().AddDate(0, 0, -29).Format("2006-01-02")
	for _, c := range []struct {
		component string
		props     map[string]any
		why       string
	}{
		{"chart", map[string]any{"type": "entry", "by": "at", "sum": "amount", "unit": "glasses"}, "grouping by a date needs a period: day, week or month (period: day draws one bar a day of at, week one a week, month one a month); the entries it counts fall on 30 days, " + first + " to "},
		{"chart", map[string]any{"type": "task", "by": "created_at"}, "grouping by a date needs a period"},
		{"collection", map[string]any{"type": "task", "where": []any{"status=doing", "status=todo"}}, "status=doing and status=todo can never both hold, since a task has one status, so it would never show anything; use one value, or one block per value (one list per status)"},
		{"calendar", map[string]any{"type": "task", "where": []any{"done=true", "done=false"}}, "done=true and done=false can never both hold"},
		{"chart", map[string]any{"type": "task", "by": "status", "where": []any{"status=doing", "status=done"}}, "status=doing and status=done can never both hold"},
	} {
		said, isErr := call(t, a, "add_component", map[string]any{"component": c.component, "props": c.props})
		if !isErr || !strings.Contains(said, "not saved: this "+c.component+" could not be shown: "+c.why) {
			t.Errorf("add_component %s %v should be refused with %q, got %v %q", c.component, c.props, c.why, isErr, said)
		}
		res := postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": c.component, "props": c.props})
		wantStatus(t, res, http.StatusUnprocessableEntity)
		if body := res.Body.String(); !strings.Contains(body, jsonText(c.why)) {
			t.Errorf("POST %s %v should say %q: %s", c.component, c.props, c.why, body)
		}
	}
	// With the period, the line says which days, so a month is never
	// read as days.
	said, isErr := call(t, a, "add_component", map[string]any{"component": "chart", "props": map[string]any{"type": "entry", "by": "at", "period": "day", "sum": "amount", "unit": "glasses"}})
	if want := "; it shows Amount of entries by At: 30 days, " + first + " to " + time.Now().Format("2006-01-02") + ", in glasses"; isErr || !strings.HasSuffix(strings.SplitN(said, "\n", 2)[0], want) {
		t.Errorf("a chart by day says its days, want %q, got %q", want, said)
	}
	// Stored before the check, a chart by a date with no period still
	// renders, by month.
	storedBlock(t, a, "chart", map[string]any{"type": "entry", "by": "at", "sum": "amount"})
	if page := get(t, h, "/").Body.String(); strings.Contains(page, "cannot be shown") {
		t.Error("a chart stored without a period still renders")
	}
}

// A block that shows nothing is written, since records may come later,
// but its line says so loudly, with what it waits for, so a model cannot
// read "0 books" as done.
func TestABlockThatShowsNothingSaysSoLoudly(t *testing.T) {
	a, h := newApp(t)
	for _, c := range []struct {
		component string
		props     map[string]any
		shows     string
	}{
		{"collection", map[string]any{"type": "task", "where": []any{"title~pond", "done=false"}}, "; it shows nothing yet: no task matches title~pond and done=false (it fills in as records are added; if some should show now, change the conditions)"},
		{"collection", map[string]any{"type": "reminder"}, "; it shows nothing yet: there are no reminders ("},
		{"calendar", map[string]any{"type": "task", "where": []any{"done=false"}}, "; it shows nothing yet: no task has a due and matches done=false ("},
		{"chart", map[string]any{"type": "task", "by": "status", "where": []any{"title~pond"}}, "; it shows nothing yet: no task matches title~pond ("},
	} {
		said, isErr := call(t, a, "add_component", map[string]any{"component": c.component, "props": c.props})
		if isErr || !strings.Contains(said, c.shows) {
			t.Errorf("add_component %s %v should say %q, got %v %q", c.component, c.props, c.shows, isErr, said)
		}
	}
	var made struct{ Shows string }
	res := postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "collection", "props": map[string]any{"type": "task", "where": []any{"title~pond"}}})
	wantStatus(t, res, http.StatusCreated)
	decode(t, res, &made)
	if !strings.HasPrefix(made.Shows, "nothing yet: no task matches title~pond") {
		t.Errorf("the API says a block shows nothing too, got %q", made.Shows)
	}
}

// A calendar of everything says how many of each kind it shows, most
// first, and when one kind floods the rest, says so and how to show one.
func TestACalendarOfEverythingSaysWhatItShowsByKind(t *testing.T) {
	a, _ := newApp(t)
	entries(t, a, 8)
	for _, title := range []string{"Dig", "Weed"} {
		if _, err := a.Store.Create("task", map[string]any{"title": title, "due": time.Now().UTC().Format(time.RFC3339)}); err != nil {
			t.Fatal(err)
		}
	}
	said, isErr := call(t, a, "add_component", map[string]any{"component": "calendar", "props": map[string]any{"type": "all"}})
	want := "; it shows every record with a date, 10 in all: 8 entries, 2 tasks; mostly entries, which crowd out the rest: a calendar of one type (type: task) shows only that type"
	if isErr || !strings.HasSuffix(strings.SplitN(said, "\n", 2)[0], want) {
		t.Errorf("a calendar of everything says its kinds, want %q, got %q", want, said)
	}
}
