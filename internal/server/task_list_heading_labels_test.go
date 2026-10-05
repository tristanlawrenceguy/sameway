package server_test

// Task list page heading links must not embed raw field labels such as
// "(added …)" or "(due …)" in their visible text, nor in any span that is
// adjacent to the link (acceptance items 1 of task 0681 / goal 0098).
// The context used to tell apart duplicate titles must use only natural-
// language words without schema column names.

import (
	"strings"
	"testing"
)

func TestTaskListPageNoRawDueLabelInHeading(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("task", map[string]any{
		"title": "Buy milk",
		"due":   "2026-10-03T14:00:00Z", // future date so it's not done
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/task").Body.String()

	rowStart := strings.Index(page, `href="/t/task/`+rec.ID)
	if rowStart < 0 {
		t.Fatalf("could not find task row in page\n%s", truncate(page))
	}
	rowEnd := strings.Index(page[rowStart:], "</li>")
	if rowEnd < 0 {
		t.Fatalf("could not find end of task row\n%s", truncate(page))
	}
	row := page[rowStart : rowStart+rowEnd]

	// The row must NOT contain the raw field label "due" followed by a date.
	if strings.Contains(strings.ToLower(row), ">due ") {
		t.Errorf("task list row must not show raw field label 'due'; found in:\n%s", truncate(row))
	}

	// The visually hidden context span must also not contain "(due …)".
	if strings.Contains(row, "sw-visually-hidden") && strings.Contains(strings.ToLower(row), "(due ") {
		t.Errorf("task list row visually hidden context must not start with '(due '; found in:\n%s", truncate(row))
	}
}

func TestTaskListPageNoRawAddedLabelInHeading(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("task", map[string]any{
		"title": "Buy milk 2",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/task").Body.String()

	rowStart := strings.Index(page, `href="/t/task/`+rec.ID)
	if rowStart < 0 {
		t.Fatalf("could not find task row in page\n%s", truncate(page))
	}
	rowEnd := strings.Index(page[rowStart:], "</li>")
	if rowEnd < 0 {
		t.Fatalf("could not find end of task row\n%s", truncate(page))
	}
	row := page[rowStart : rowStart+rowEnd]

	// The row must NOT contain "(added …)" as context.
	if strings.Contains(row, "sw-visually-hidden") && strings.Contains(strings.ToLower(row), "(added ") {
		t.Errorf("task list row visually hidden context must not start with '(added '; found in:\n%s", truncate(row))
	}
}
