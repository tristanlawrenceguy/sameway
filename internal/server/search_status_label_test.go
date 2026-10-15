package server_test

// Search result snippet text must label state/status enum values with a visible
// "Status:" prefix so they are not bare words separated by middots from
// surrounding content. Acceptance item 2 of task 0708 (backlog 0708).

import (
	"net/http"
	"strings"
	"testing"
)

// TestSearchResultSnippetLabelsStatusValue verifies that a note with status="draft"
// shows "Status: Draft" in the snippet, not bare "Draft". The label uses the field's
// display name so any enum kind (status, state) gets its own prefix.
func TestSearchResultSnippetLabelsStatusValue(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	var rec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{
		"title":  "Draft note for snippet check",
		"body":   "This is a draft note about the kitchen tap.",
		"status": "draft",
	}), &rec)

	body := get(t, h, "/search?q=draft").Body.String()

	if !strings.Contains(body, `href="/t/note/`+rec.ID) {
		t.Fatalf("could not find search result for note; found in:\n%s", truncate(body))
	}

	// The snippet must contain "Status:" as a visible prefix before the status value.
	if !strings.Contains(body, "Status:") {
		t.Errorf("search result snippet must show 'Status:' label before status value; found in:\n%s", truncate(body))
	}

	// If "Draft" appears as a standalone status value (not inside <mark> from title
	// highlighting), it should be preceded by "Status:" rather than floating bare.
	if strings.Contains(body, `· Draft`) ||
		strings.Contains(body, ">Draft</p>") {
		t.Errorf("search result snippet must not show bare 'Draft' — it should have a 'Status:' prefix; found in:\n%s", truncate(body))
	}

	// The card body div must exist (the note has searchable content).
	if !strings.Contains(body, `sw-card__body`) {
		t.Error("search result card must show its body/snippet")
	}
}

// TestSearchResultSnippetLabelsPublishedStatus verifies the same for status="published"
// on a note — "Status: Published" not bare "Published".
func TestSearchResultSnippetLabelsPublishedStatus(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	var rec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{
		"title":  "Published note for snippet check",
		"body":   "This published note is about the garden.",
		"status": "published",
	}), &rec)

	body := get(t, h, "/search?q=garden").Body.String()

	if !strings.Contains(body, `href="/t/note/`+rec.ID) {
		t.Fatalf("could not find search result for note; found in:\n%s", truncate(body))
	}

	// The snippet must contain "Status:" as a visible prefix.
	if !strings.Contains(body, "Status:") {
		t.Errorf("search result snippet must show 'Status:' label before status value; found in:\n%s", truncate(body))
	}

	// If "Published" appears as a standalone status value (not inside <mark> from title
	// highlighting), it should not be bare.
	if strings.Contains(body, `· Published`) ||
		strings.Contains(body, ">Published</p>") {
		t.Errorf("search result snippet must not show bare 'Published' — it should have a 'Status:' prefix; found in:\n%s", truncate(body))
	}

	// The card body div must exist.
	if !strings.Contains(body, `sw-card__body`) {
		t.Error("search result card must show its body/snippet")
	}
}

// TestSearchResultSnippetLabelsTaskStatus verifies that a task with a non-default
// status (e.g., "doing" instead of default "todo") shows the label in the snippet.
func TestSearchResultSnippetLabelsTaskStatus(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	var rec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{
		"title":  "Doing the tap fix",
		"status": "doing",
	}), &rec)

	body := get(t, h, "/search?q=tap").Body.String()

	if !strings.Contains(body, `href="/t/task/`+rec.ID) {
		t.Fatalf("could not find search result for task; found in:\n%s", truncate(body))
	}

	// The snippet must contain "Status:" as a visible prefix.
	if !strings.Contains(body, "Status:") {
		t.Errorf("search result snippet must show 'Status:' label before status value; found in:\n%s", truncate(body))
	}

	// If "Doing" appears as a standalone status value (not inside <mark> from title
	// highlighting), it should not be bare.
	if strings.Contains(body, `· Doing`) ||
		strings.Contains(body, ">Doing</p>") {
		t.Errorf("search result snippet must not show bare 'Doing' — it should have a 'Status:' prefix; found in:\n%s", truncate(body))
	}

	// The card body div must exist.
	if !strings.Contains(body, `sw-card__body`) {
		t.Error("search result card must show its body/snippet")
	}
}
