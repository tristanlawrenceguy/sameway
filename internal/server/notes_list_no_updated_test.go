package server_test

// No list row on /t/note, /t/action, or /t/person shows a bare "Updated"
// label before its timestamp. Acceptance items 1–2 of task 0924.

import (
	"net/http"
	"strings"
	"testing"
)

// TestNoteListRowHasNoBareUpdatedLabel verifies that no note list row on
// /t/note contains the bare word "Updated" as a standalone label before
// its time element. The date itself remains, but without the redundant
// "Updated" prefix. (Acceptance 1.)
func TestNoteListRowHasNoBareUpdatedLabel(t *testing.T) {
	t.Parallel()

	_, h := newApp(t)

	var rec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{
		"title": "Test note for updated label removal",
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

	if strings.Contains(row, `<span class="sw-muted">Updated`) {
		t.Errorf("note list row must not contain bare \"Updated\" label; found in:\n%s", truncate(row))
	}
}

// TestNoteListRowStillShowsTimeElement verifies that removing the bare
// "Updated" label does not remove the time element itself — the date is
// still present, just without the redundant prefix. (Acceptance 1.)
func TestNoteListRowStillShowsTimeElement(t *testing.T) {
	t.Parallel()

	_, h := newApp(t)

	var rec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{
		"title": "Test note for time element preservation",
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

	if !strings.Contains(row, `<time`) {
		t.Errorf("note list row must still show a time element; found in:\n%s", truncate(row))
	}
}

// TestActionListRowHasNoBareUpdatedLabel verifies that no action list row
// on /t/action contains the bare word "Updated" as a standalone label before
// its time element. (Acceptance 2.)
func TestActionListRowHasNoBareUpdatedLabel(t *testing.T) {
	t.Parallel()

	_, h := newApp(t)

	var act struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/action", map[string]any{
		"title": "Test action for updated label removal",
	}), &act)

	body := get(t, h, "/t/action").Body.String()

	rowStart := strings.Index(body, `href="/t/action/`+act.ID)
	if rowStart < 0 {
		t.Fatalf("could not find action row in page\n%s", truncate(body))
	}
	rowEnd := strings.Index(body[rowStart:], "</li>")
	if rowEnd < 0 {
		t.Fatalf("could not find end of action row\n%s", truncate(body))
	}
	row := body[rowStart : rowStart+rowEnd]

	if strings.Contains(row, `<span class="sw-muted">Updated`) {
		t.Errorf("action list row must not contain bare \"Updated\" label; found in:\n%s", truncate(row))
	}
}

// TestPersonListRowHasNoBareUpdatedLabel verifies that no person list row
// on /t/person contains the bare word "Updated" as a standalone label before
// its time element. (Acceptance 2.)
func TestPersonListRowHasNoBareUpdatedLabel(t *testing.T) {
	t.Parallel()

	_, h := newApp(t)

	var per struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/person", map[string]any{
		"name": "Test Person for label removal",
	}), &per)

	body := get(t, h, "/t/person").Body.String()

	rowStart := strings.Index(body, `href="/t/person/`+per.ID)
	if rowStart < 0 {
		t.Fatalf("could not find person row in page\n%s", truncate(body))
	}
	rowEnd := strings.Index(body[rowStart:], "</li>")
	if rowEnd < 0 {
		t.Fatalf("could not find end of person row\n%s", truncate(body))
	}
	row := body[rowStart : rowStart+rowEnd]

	if strings.Contains(row, `<span class="sw-muted">Updated`) {
		t.Errorf("person list row must not contain bare \"Updated\" label; found in:\n%s", truncate(row))
	}
}
