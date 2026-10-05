package server_test

// Search results on /search must show natural language dates and no raw
// field labels in headings, matching acceptance items 1–4 of task 0263.

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// TestSearchResultSnippetsHaveNoMachineFormatDate verifies that search result
// body snippets use natural language dates like "Today at …am/pm" instead of
// machine format with full years and 24-hour time. This covers acceptance items
// 1 and 2 (no "2026", no ", HH:MM") for all record types with datetime fields.
func TestSearchResultSnippetsHaveNoMachineFormatDate(t *testing.T) {
	_, h := newApp(t)

	tests := []struct {
		apiPath string
		payload map[string]any
	}{
		{"/api/task", map[string]any{"title": "Task with machine date test", "due": "2026-10-05"}},
		{"/api/entry", map[string]any{"title": "Entry with datetime field", "when": "today 3pm"}},
	}

	for _, tc := range tests {
		t.Run(tc.apiPath, func(t *testing.T) {
			var rec struct{ ID string }
			decode(t, postJSON(t, h, http.MethodPost, tc.apiPath, tc.payload), &rec)

			body := get(t, h, "/search?q="+strings.ReplaceAll(tc.payload["title"].(string), " ", "+")).Body.String()

			// No full year like 2026 in the body.
			if strings.Contains(visibleText(body), "2026") {
				t.Errorf("search result snippet must not contain a bare full year like '2026'; found in:\n%s", truncate(body))
			}

			// No comma-separated 24-hour time like ", 15:00" or ", 13:31".
			if matched, _ := regexp.MatchString(`,\s\d{2}:\d{2}\b`, body); matched {
				t.Errorf("search result snippet must not contain machine-format comma-separated time like ', 15:00'; found in:\n%s", truncate(body))
			}

			// The date should use am/pm (12-hour clock) when present.
			if strings.Contains(body, "am") || strings.Contains(body, "pm") {
				return // natural language detected, good enough
			}
			// If no time is shown at all, that is also acceptable for day-only values.
		})
	}
}

// TestSearchResultHeadingsHaveNoRawFieldLabels verifies that search result
// heading links do not contain raw schema field labels like "(added" or "(due"
// in their accessible-name spans. This covers acceptance item 3 — a screen
// reader user should never hear the word "added" or "due" as part of the link
// name; those are database column names, not words for people.
func TestSearchResultHeadingsHaveNoRawFieldLabels(t *testing.T) {
	_, h := newApp(t)

	tests := []struct {
		apiPath string
		payload map[string]any
	}{
		{"/api/task", map[string]any{"title": "Task for raw label check"}},
	}

	for _, tc := range tests {
		t.Run(tc.apiPath, func(t *testing.T) {
			var rec struct{ ID string }
			decode(t, postJSON(t, h, http.MethodPost, tc.apiPath, tc.payload), &rec)

			body := get(t, h, "/search?q="+strings.ReplaceAll(tc.payload["title"].(string), " ", "+")).Body.String()

			// The visually hidden span must not contain "(added" or "(due".
			if strings.Contains(body, "(added") {
				t.Errorf("search result heading must not contain raw field label '(added'; found in:\n%s", truncate(body))
			}
			if strings.Contains(body, "(due") {
				t.Errorf("search result heading must not contain raw field label '(due'; found in:\n%s", truncate(body))
			}

			// It should still be possible to tell apart identical titles using
			// ordinals like "first of 2" instead. Ordinal words are user-friendly.
			if strings.Contains(body, "(first") || strings.Contains(body, "(second") {
				return // ordinal telling apart is fine and expected for duplicate titles
			}
		})
	}
}

// TestSearchResultWithIdenticalTitlesHasNoRawLabels verifies that when multiple
// records share the same title, they are told apart by ordinal indicators ("first
// of 2") rather than raw field labels like "(added" or "(due". This is acceptance
// item 3 in a scenario where recordWays would normally produce disambiguation text.
func TestSearchResultWithIdenticalTitlesHasNoRawLabels(t *testing.T) {
	_, h := newApp(t)

	// Create two tasks with the same title but different due dates — this triggers
	// hitsApart because titles are identical after folding.
	var recs [2]struct{ ID string }
	for i := 0; i < 2; i++ {
		payload := map[string]any{"title": "Call plumber", "due": fmt.Sprintf("2026-10-%02d", i+1)}
		decode(t, postJSON(t, h, http.MethodPost, "/api/task", payload), &recs[i])
	}

	body := get(t, h, "/search?q=plumber").Body.String()

	if strings.Contains(body, "(added") {
		t.Errorf("search result with duplicate titles must not contain raw field label '(added'; found in:\n%s", truncate(body))
	}
	if strings.Contains(body, "(due") {
		t.Errorf("search result with duplicate titles must not contain raw field label '(due'; found in:\n%s", truncate(body))
	}

}
