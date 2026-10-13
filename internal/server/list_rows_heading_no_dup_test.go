package server_test

// List row headings must not announce date metadata twice in their accessible name.
// A heading's accessible name should contain each piece of information exactly once —
// no duplicate "(added …)" when an aria-label and a sibling text node both carry it.
//
// Note: this test file exists for regression protection. The original task (backlog 0718)
// described a bug where heading accessible names contained duplicate date metadata, but
// code review confirmed that on current main the row links have no explicit aria-label
// attribute and the sibling meta paragraph is outside the h3 element, so duplication
// does not occur. These tests verify that future changes do not reintroduce it.
// Acceptance items 1–3: /t/task, /t/entry, /t/reminder lists must not double-announce
// the date inside heading accessible names.

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestTaskListHeadingNoDuplicateDateInAccessibleName verifies that each task row's
// h3 heading accessible name contains "(added" at most once — no duplicate date
// metadata in the heading's accessible name. It creates a single task with a due
// date on /t/task, fetches the page, finds all h3 headings and checks them.
// (Acceptance 1.)
func TestTaskListHeadingNoDuplicateDateInAccessibleName(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	wantStatus(t, postJSON(t, h, "POST", "/api/task", map[string]any{
		"title": "Buy groceries",
		"due":   "2026-10-01",
	}), 201)

	body := get(t, h, "/t/task").Body.String()
	doc, err := htmltest.Parse(body)
	if err != nil {
		t.Fatal(err)
	}

	h3s := doc.Elements("h3")
	if len(h3s) == 0 {
		t.Fatal("expected at least one task row heading (h3)")
	}

	for _, n := range h3s {
		name := doc.AccessibleName(n)
		if strings.Count(name, "(added") > 1 {
			t.Errorf("h3 accessible name %q contains duplicate date metadata; \"(added\" appears %d times", name, strings.Count(name, "(added"))
		}
		if !strings.Contains(name, "Buy groceries") {
			t.Errorf("h3 accessible name %q should contain the record title", name)
		}
	}
}

// TestEntryListHeadingNoDuplicateDateInAccessibleName verifies that each entry row's
// heading accessible name contains "(added" at most once — no duplicate date
// metadata in the heading's accessible name. It creates a habit and an entry on
// /t/entry, fetches the page, finds all h2 headings (entry list uses grouped by
// day with h2 group headings) and checks them. (Acceptance 2.)
func TestEntryListHeadingNoDuplicateDateInAccessibleName(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	var habit struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/habit", map[string]any{
		"name":    "Exercise",
		"unit":    "minutes",
		"cadence": "day",
		"target":  30,
	}), &habit)

	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/entry", map[string]any{
		"habit":  habit.ID,
		"amount": float64(25),
		"at":     "2026-10-14T08:00:00Z",
	}), http.StatusCreated)

	body := get(t, h, "/t/entry").Body.String()
	doc, err := htmltest.Parse(body)
	if err != nil {
		t.Fatal(err)
	}

	headings := doc.Elements("h2")
	if len(headings) == 0 {
		h3s := doc.Elements("h3")
		t.Fatalf("expected at least one entry heading (h2); found %d h2 and %d h3", len(doc.Elements("h2")), len(h3s))
	}

	for _, n := range headings {
		name := doc.AccessibleName(n)
		if strings.Count(name, "(added") > 1 {
			t.Errorf("entry heading accessible name %q contains duplicate date metadata; \"(added\" appears %d times", name, strings.Count(name, "(added"))
		}
	}

	h3s := doc.Elements("h3")
	for _, n := range h3s {
		name := doc.AccessibleName(n)
		if strings.Count(name, "(added") > 1 {
			t.Errorf("entry row heading accessible name %q contains duplicate date metadata; \"(added\" appears %d times", name, strings.Count(name, "(added"))
		}
	}
}

// TestReminderListHeadingNoDuplicateDateInAccessibleName verifies that each reminder row's
// heading accessible name contains "(added" at most once — no duplicate date
// metadata in the heading's accessible name. It creates a single reminder on /t/reminder,
// fetches the page, finds all h3 headings and checks them. (Acceptance 3.)
func TestReminderListHeadingNoDuplicateDateInAccessibleName(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	wantStatus(t, postJSON(t, h, "POST", "/api/reminder", map[string]any{
		"title": "Submit tax return",
		"at":    "2026-10-15T09:00:00Z",
	}), 201)

	body := get(t, h, "/t/reminder").Body.String()
	doc, err := htmltest.Parse(body)
	if err != nil {
		t.Fatal(err)
	}

	h3s := doc.Elements("h3")
	if len(h3s) == 0 {
		h2s := doc.Elements("h2")
		h1s := doc.Elements("h1")
		t.Fatalf("expected at least one reminder heading; found %d h1, %d h2, %d h3", len(h1s), len(h2s), len(h3s))
	}

	for _, n := range h3s {
		name := doc.AccessibleName(n)
		if strings.Count(name, "(added") > 1 {
			t.Errorf("h3 accessible name %q contains duplicate date metadata; \"(added\" appears %d times", name, strings.Count(name, "(added"))
		}
		if !strings.Contains(name, "Submit tax return") {
			t.Errorf("h3 accessible name %q should contain the record title", name)
		}
	}
}
