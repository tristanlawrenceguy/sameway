package server_test

import (
	"strings"
	"testing"
)

// TestTaskDetailLedeNoRawDueLabel asserts that a task detail page lede does not
// show the raw schema field name "due" as a visible label on its datetime chip,
// and instead shows only the date text (acceptance items 1 & 4).
func TestTaskDetailLedeNoRawDueLabel(t *testing.T) {
	a, h := newApp(t)

	_, err := a.Store.Create("task", map[string]any{
		"title": "Garden compost",
		"due":   "2026-10-02T00:00:00Z", // future date, not past
	})
	if err != nil {
		t.Fatal(err)
	}

	var result struct {
		Records []struct{ ID string } `json:"records"`
	}
	decode(t, get(t, h, "/api/task"), &result)
	if len(result.Records) == 0 {
		t.Fatal("no task records found")
	}

	page := get(t, h, "/t/task/"+result.Records[0].ID+"?show=fields").Body.String()

	// The lede must NOT contain a badge with the raw "due" label.
	if strings.Contains(page, ">Due ") || strings.Contains(page, ">was due") {
		t.Error("task detail page lede must not show raw field name \"due\":\n" + truncate(page))
	}

	// The date text should still be present as a chip (no label prefix).
	if !strings.Contains(page, "sw-badge") && !strings.Contains(page, "sw-when") {
		t.Error("task detail page lede should still show the due date as a badge or span")
	}

	// The creation timestamp via whenMade should still be present.
	if !strings.Contains(page, ">Added ") {
		t.Error("task detail page lede should still show creation time")
	}
}

// TestTaskListRowNoRawDatetimeLabel asserts that list rows for tasks do not
// display the raw schema column name "Due" in their dayFact cells (acceptance
// items 3 & 4). This covers both chip mode (detail lede) and row mode.
func TestTaskListRowNoRawDatetimeLabel(t *testing.T) {
	a, h := newApp(t)

	_, err := a.Store.Create("task", map[string]any{
		"title": "Past task",
		"due":   "2024-10-01T00:00:00Z", // past date
	})
	if err != nil {
		t.Fatal(err)
	}

	listPage := get(t, h, "/t/task?order=due").Body.String()

	// The row must NOT contain "Was due" — the raw field name should not appear.
	if strings.Contains(listPage, ">Was due") || strings.Contains(listPage, ">was due") {
		t.Error("task list row must not show raw schema field label \"Due\" in dayFact output")
	}

}
