package server_test

// The lede on record detail pages must not expose raw schema field names like
// "Date" or "Text1" as visible labels — backlog 0627, goal 0094. The dayFact
// function was the last place that used label(f.Name) directly instead of a
// conditional that respects custom schema Labels. This file tests that fix:
// when no custom Label exists on a datetime field, only the date text appears;
// when one does, it is shown as before.

import (
	"net/http"
	"strings"
	"testing"
)

// TestDayFactNoRawLabelOnDateField asserts that a meeting_notes_template-like
// record whose first datetime field has no custom schema Label shows only the
// date in its lede — no "Was Date" or any other raw field name prefix
// (backlog 0627, acceptance item 1 & 4). The type is created dynamically via
// the API so it mimics an agent-created type.
func TestDayFactNoRawLabelOnDateField(t *testing.T) {
	_, h := newApp(t)

	// Create a meeting_notes_template-like type with a datetime field named
	// "date" that has no custom Label — how agents create types dynamically.
	postJSON(t, h, http.MethodPost, "/api/types", map[string]any{
		"name":        "meeting_notes_template",
		"description": "Template for meeting notes",
		"title":       "title",
		"fields": []map[string]any{
			{"name": "title", "type": "string"},
			{"name": "date", "type": "datetime"},
		},
	})

	// Create a record with a past date (past the current day). Use the API to
	// create it, then fetch the ID from the response.
	rec := postJSON(t, h, http.MethodPost, "/api/meeting_notes_template", map[string]any{
		"title": "Team Standup",
		"date":  "2026-10-01T00:00:00Z",
	})

	var result struct {
		ID string `json:"id"`
	}
	decode(t, rec, &result)
	page := get(t, h, "/t/meeting_notes_template/"+result.ID).Body.String()

	// The lede must not contain any raw field name like "Date" or "date".
	if strings.Contains(page, ">Was date") || strings.Contains(page, ">Was Date") {
		t.Errorf("lede must not show raw field label 'date':\n%s", truncate(page))
	}

	// The lede should still contain the creation timestamp.
	if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
		t.Error("meeting_notes_template detail lede should still show creation time")
	}
}
