package server_test

// No note list row at /t/note shows bare "Changed" text before any timestamp.
// Each row's timestamp is preceded by "Updated" instead of the unlabeled verb.
// Acceptance items 1–2 of task 0302.

import (
	"net/http"
	"strings"
	"testing"
)

// TestNoteListRowNoBareChanged verifies that no note list row contains the
// bare word "Changed" followed by a time element — the verb label has been
// replaced with a human-readable alternative. (Acceptance 1.)
func TestNoteListRowNoBareChanged(t *testing.T) {
	t.Parallel()

	_, h := newApp(t)

	var rec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{
		"title": "Test note for bare Changed check",
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

	if !strings.Contains(row, `<span class="sw-muted">Updated <time`) {
		t.Errorf("note list row should show \"Updated\" before the timestamp; found in:\n%s", truncate(row))
	}

	if strings.Contains(row, `>Changed <time`) {
		t.Errorf("note list row must not contain bare \"Changed\" before timestamp;\nfound: %s", truncate(row))
	}
}

// TestNoteListRowUpdatedLabelIsPresent verifies that the "Updated" label is
// rendered as a span with class sw-muted, wrapping the human-readable time.
// (Acceptance 2.)
func TestNoteListRowUpdatedLabelIsPresent(t *testing.T) {
	t.Parallel()

	_, h := newApp(t)

	var rec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{
		"title": "Test note for updated label check",
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

	if !strings.Contains(row, `<span class="sw-muted">Updated`) {
		t.Errorf("note list row should contain a span with class=\"sw-muted\" and text \"Updated\"; found in:\n%s", truncate(row))
	}
}
