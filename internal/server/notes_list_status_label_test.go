package server_test

// Published status values on notes list rows appear as badge chips without
// raw schema field name prefixes like "Status:". The badge communicates the value.

import (
	"net/http"
	"strings"
	"testing"
)

// TestNoteListShowsNoStatusLabelledStatus verifies that a published note's row on
// the notes list page shows just the "Published" badge without a raw "Status:" label.
func TestNoteListShowsNoStatusLabelledStatus(t *testing.T) {
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

	// The badge component should still render the Published value.
	if !strings.Contains(row, "Published") {
		t.Errorf("note list row must still mention that the note is published; found in:\n%s", truncate(row))
	}

	// No raw schema field name prefix before the badge.
	if strings.Contains(row, "Status: Published") {
		t.Errorf("note list row must not show 'Status:' label before published status; found in:\n%s", truncate(row))
	}
}

// TestNoteListScreenReaderHearsNoStatusLabelledStatus verifies that a notes list
// row for a published note does not contain a raw "Status:" prefix — only the
// badge chip with its value. (Acceptance: no machine-language labels.)
func TestNoteListScreenReaderHearsNoStatusLabelledStatus(t *testing.T) {
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

	if !strings.Contains(row, ">Published</span>") {
		t.Errorf("row must contain Published badge text; found in:\n%s", truncate(row))
		return
	}

	// No raw schema field name prefix.
	if strings.Contains(row, "Status:") {
		t.Errorf("row must not contain visible 'Status:' label; found in:\n%s", truncate(row))
	}
}

// TestNoteListDoesNotLabelDraftStatus verifies that a draft note (the default
// status) does NOT get a badge at all, since StateWorth suppresses rest
// values entirely.
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
	if strings.Contains(row, `sw-badge`) && strings.Contains(strings.ToLower(row), "draft") {
		t.Errorf("draft note list row must not render a Draft badge; found in:\n%s", truncate(row))
	}
}
