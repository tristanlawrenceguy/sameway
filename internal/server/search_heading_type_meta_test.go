package server_test

// Search result headings must not show a visible bare type-name paragraph.
// Acceptance item 1 of task 0708 (backlog 0761, 0762).

import (
	"net/http"
	"strings"
	"testing"
)

// TestSearchResultHasNoVisibleTypeMeta verifies that search result cards do not
// render a visible <p class="sw-card__meta"> paragraph showing the bare type name
// such as "Note", "Task", or "Block". The visually hidden span for screen readers
// is kept; only the visible meta line is removed. (Acceptance 1.)
func TestSearchResultHasNoVisibleTypeMeta(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	var rec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{
		"title": "Call the plumber",
		"body":  "About the kitchen tap.",
	}), &rec)

	body := get(t, h, "/search?q=plumber").Body.String()

	if !strings.Contains(body, `href="/t/note/`+rec.ID) {
		t.Fatalf("could not find search result for note; found in:\n%s", truncate(body))
	}

	// The card must NOT contain a visible <p class="sw-card__meta"> paragraph.
	if strings.Contains(body, `<p class="sw-card__meta"`) {
		t.Errorf("search result card must not show a visible type-name meta paragraph; found in:\n%s", truncate(body))
	}

	// The visually hidden span for screen readers should still be present.
	if !strings.Contains(body, `class="sw-visually-hidden"`) {
		t.Error("visually hidden span for the kind must still exist for screen readers")
	}

	// The card title link must include the kind in its accessible name via
	// the visually hidden span so a screen reader says "Call the plumber — Note".
	if !strings.Contains(body, `— Note`) && !strings.Contains(body, "— note") {
		t.Error("result links should include both the record type and its title in the accessible name, e.g., 'Call the plumber — Note'")
	}

	// The visible card body must not contain a standalone bare word matching
	// the type name as a paragraph.
	if strings.Contains(body, ">Note</p>") || strings.Contains(body, "<p>Note</p>") {
		t.Errorf("visible card must not show a bare 'Note' paragraph; found in:\n%s", truncate(body))
	}
}

// TestSearchResultNoVisibleTypeMetaForTask verifies the same for tasks and other
// record types — no visible <p class="sw-card__meta"> showing "Task".
func TestSearchResultNoVisibleTypeMetaForTask(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	var rec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{
		"title": "Fix the tap",
	}), &rec)

	body := get(t, h, "/search?q=fix").Body.String()

	if !strings.Contains(body, `href="/t/task/`+rec.ID) {
		t.Fatalf("could not find search result for task; found in:\n%s", truncate(body))
	}

	if strings.Contains(body, `<p class="sw-card__meta"`) {
		t.Errorf("search result card must not show a visible type-name meta paragraph; found in:\n%s", truncate(body))
	}

	// The kind should still be in the accessible name.
	if !strings.Contains(body, `— Task`) && !strings.Contains(body, "— task") {
		t.Error("result links should include both the record type and its title in the accessible name")
	}
}
