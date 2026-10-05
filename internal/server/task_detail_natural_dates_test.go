package server_test

// Task detail page due dates and entries list headings must show natural language
// timestamps — no machine-format like "Mon 5 Oct 2026, 10:00" or "(at Wed 30 Sep, 13:31)".
// Acceptance items 1–3 of task 0262 (backlog 0676).

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// TestTaskDetailLedeDueDateNaturalLanguage asserts that the due date shown on
// a task detail page reads in natural language (e.g., "Tomorrow at 10am")
// rather than machine format like "Mon 5 Oct 2026, 10:00" (acceptance item 1).
func TestTaskDetailLedeDueDateNaturalLanguage(t *testing.T) {
	a, h := newApp(t)

	now := time.Now()
	future := when.Store(now.Add(48*time.Hour), false) // moment in the future, not day-only

	_, err := a.Store.Create("task", map[string]any{
		"title": "Buy coffee",
		"due":   future,
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

	// Extract the lede paragraph.
	ledeStart := strings.Index(page, `<div class="sw-lede">`)
	if ledeStart == -1 {
		t.Fatal("no lede found in task detail page")
	}
	ledeEnd := strings.Index(page[ledeStart:], `</div>`)
	if ledeEnd == -1 {
		t.Fatal("no closing tag for lede")
	}
	lede := page[ledeStart : ledeStart+ledeEnd]

	// The date must NOT contain a 4-digit year (machine format indicator).
	machineYearRe := regexp.MustCompile(`\d{4}`)
	if machineYearRe.MatchString(lede) {
		t.Errorf("task detail lede due date must not contain a 4-digit year (machine format); found in:\n%s", truncate(lede))
	}

	// The date must NOT match the weekday+month+year pattern like "Mon 5 Oct 2026".
	machineDateRe := regexp.MustCompile(`[A-Z][a-z]{2}\s+\d{1,2}\s+[A-Z][a-z]{2}\s+\d{4}`)
	if machineDateRe.MatchString(lede) {
		t.Errorf("task detail lede must not contain machine-format date like 'Mon 5 Oct 2026'; found in:\n%s", truncate(lede))
	}

	// The date must NOT have a comma-separated time pattern like ", 10:00".
	machineTimeRe := regexp.MustCompile(`,\s*\d{1,2}:\d{2}\b`)
	if machineTimeRe.MatchString(lede) {
		t.Errorf("task detail lede must not contain comma-separated time like ', 10:00'; found in:\n%s", truncate(lede))
	}

	// The date text should use a 12-hour clock (am/pm), not a bare 24-hour format.
	if !strings.Contains(strings.ToLower(lede), "am") && !strings.Contains(strings.ToLower(lede), "pm") {
		t.Errorf("task detail lede due date should use 12-hour clock with am/pm; found in:\n%s", truncate(lede))
	}
}
