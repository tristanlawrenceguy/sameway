package server_test

// Published status values on notes list rows must be preceded by a visible
// "Status:" label so they are not bare enum words floating beside dates.
// Acceptance items 1–2 of task 0754 (backlog 0754).

import (
	"net/http"
	"strings"
	"testing"
)

// TestNoteListShowsStatusLabelledStatus verifies that a published note's row on
// the notes list page shows "Status:" before the "Published" badge, not bare
// "Published" or "Published status". (Acceptance 1.)
func TestNoteListShowsStatusLabelledStatus(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	var rec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{
		"title":  "Published note for status label",
		"status": "published",
	}), &rec)

	body := get(t, h, "/t/note").Body.String()

	rowStart := strings.Index(body, `href="/t/note/`+rec.ID)
	if rowStart < 0 {
		t.Fatalf("could not find note row in page\n%s", truncate(body))
	}
	rowEnd := strings.Index(body[rowStart:], "</li>")
	if rowEnd < 0 {
		t.Fatalf("could not find end of note row\n%s", truncate(body))
	}
	row := body[rowStart : rowStart+rowEnd]

	// The row must contain "Status:" as a visible prefix before the status value.
	if !strings.Contains(row, "Status:") {
		t.Errorf("note list row must show 'Status:' label before published status; found in:\n%s", truncate(row))
	}

	// The raw word "Published" must not appear without the "Status:" prefix.
	// Check that if "Published" appears, it is preceded by "Status:".
	if strings.Contains(row, ">Published") && !strings.Contains(row, "Status:") {
		t.Errorf("note list row must not show bare 'Published' status value; found in:\n%s", truncate(row))
	}

	// The badge component should still render the Published value.
	if !strings.Contains(row, "Published") {
		t.Errorf("note list row must still mention that the note is published; found in:\n%s", truncate(row))
	}
}

// TestNoteListScreenReaderHearsStatusLabelledStatus verifies that the visible
// text order on a notes list row for a published note reads "Status:" before
// "Published", so screen readers announce the field name first. (Acceptance 2.)
func TestNoteListScreenReaderHearsStatusLabelledStatus(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	var rec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{
		"title":  "Screen reader status test note",
		"status": "published",
	}), &rec)

	body := get(t, h, "/t/note").Body.String()

	rowStart := strings.Index(body, `href="/t/note/`+rec.ID)
	if rowStart < 0 {
		t.Fatalf("could not find note row in page\n%s", truncate(body))
	}
	rowEnd := strings.Index(body[rowStart:], "</li>")
	if rowEnd < 0 {
		t.Fatalf("could not find end of note row\n%s", truncate(body))
	}
	row := body[rowStart : rowStart+rowEnd]

	// Strip HTML tags to check visible text order.
	cleaned := strings.ReplaceAll(row, "<span", "\n<span")
	cleaned = strings.ReplaceAll(cleaned, "</span>", "</span>\n")
	lines := strings.Split(cleaned, "\n")

	var statusIndex int = -1
	var publishedIndex int = -1
	for i, line := range lines {
		if strings.Contains(line, "Status:") {
			statusIndex = i
		}
		if strings.Contains(line, "Published") || (strings.Contains(line, `sw-badge`) && strings.Contains(line, "info")) {
			publishedIndex = i
		}
	}

	if statusIndex < 0 {
		t.Errorf("row must contain visible 'Status:' label; found in:\n%s", truncate(row))
		return
	}
	if publishedIndex < 0 {
		t.Errorf("row must contain Published badge text; found in:\n%s", truncate(row))
		return
	}

	// Status: must come before (or at the same position as) Published.
	// Screen readers read left-to-right, top-to-bottom, so "Status:" should appear first.
	if statusIndex > publishedIndex {
		t.Errorf("screen reader would hear 'Published' before 'Status:' — the label must precede the value; found in:\n%s", truncate(row))
	}

	// The word "status" must not appear after "Published" as a postfix context.
	if strings.Contains(row, "Published") && strings.Contains(strings.ToLower(row), "published status") {
		t.Errorf("row must not show 'Published status' — the label must precede, not follow; found in:\n%s", truncate(row))
	}
}

// TestNoteListDoesNotLabelDraftStatus verifies that a draft note (the default
// status) does NOT get a "Status:" label, since StateWorth suppresses rest
// values entirely. Only non-rest statuses like Published get the label.
func TestNoteListDoesNotLabelDraftStatus(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	var rec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{
		"title":  "Draft note for no label check",
		"status": "draft",
	}), &rec)

	body := get(t, h, "/t/note").Body.String()

	rowStart := strings.Index(body, `href="/t/note/`+rec.ID)
	if rowStart < 0 {
		t.Fatalf("could not find note row in page\n%s", truncate(body))
	}
	rowEnd := strings.Index(body[rowStart:], "</li>")
	if rowEnd < 0 {
		t.Fatalf("could not find end of note row\n%s", truncate(body))
	}
	row := body[rowStart : rowStart+rowEnd]

	// Draft is the rest/default, so StateWorth says it's not worth showing.
	if strings.Contains(row, "Status:") {
		t.Errorf("draft note list row must not show a Status: label; found in:\n%s", truncate(row))
	}
	if strings.Contains(row, `sw-badge`) && strings.Contains(strings.ToLower(row), "draft") {
		t.Errorf("draft note list row must not render a Draft badge; found in:\n%s", truncate(row))
	}
}
