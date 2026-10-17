package server_test

// No note list row at /t/note shows bare "Changed" or "Updated" text before any timestamp.
// Acceptance items 1–2 of task 0302; also covers task 0924 (no bare "Updated").

import (
	"net/http"
	"strings"
	"testing"
)

// TestNoteListRowNoBareChanged verifies that no note list row contains the
// bare word "Changed" followed by a time element. It also checks that the
// later fix of replacing "Changed" with "Updated" (task 0302) was itself
// removed, leaving only the raw timestamp. (Acceptance 1–2.)
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

	if strings.Contains(row, `>Changed <time`) {
		t.Errorf("note list row must not contain bare \"Changed\" before timestamp;\nfound: %s", truncate(row))
	}

	if strings.Contains(row, `<span class="sw-muted">Updated`) {
		t.Errorf("note list row must not contain bare \"Updated\" label; found in:\n%s", truncate(row))
	}
}
