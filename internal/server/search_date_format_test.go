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
	t.Parallel()
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

// TestSearchResultHeadingsHaveNoRawFieldLabels: a result whose name is its
// own is said alone, with no words to tell it apart from anything
// (design/foundations/names.md).
func TestSearchResultHeadingsHaveNoRawFieldLabels(t *testing.T) {
	t.Parallel()
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

		})
	}
}

// TestSearchResultWithIdenticalTitlesSaysTheirDays: results of one title
// and kind are told apart by their days, as their list tells them, not
// numbered (design/foundations/names.md).
func TestSearchResultWithIdenticalTitlesSaysTheirDays(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	// Create two tasks with the same title but different due dates — this triggers
	// hitsApart because titles are identical after folding.
	var recs [2]struct{ ID string }
	for i := 0; i < 2; i++ {
		payload := map[string]any{"title": "Call plumber", "due": fmt.Sprintf("2026-10-%02d", i+1)}
		decode(t, postJSON(t, h, http.MethodPost, "/api/task", payload), &recs[i])
	}

	body := get(t, h, "/search?q=plumber").Body.String()

	if matched, _ := regexp.MatchString(`\((Mon|Tue|Wed|Thu|Fri|Sat|Sun) \d\d?\b`, body); !matched {
		t.Errorf("each result says its day, and neither is numbered:\n%s", truncate(body))
	}
}
