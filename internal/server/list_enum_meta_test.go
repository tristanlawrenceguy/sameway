package server_test

// Note, project and file list pages must not show enum badges (status chips)
// in row meta text. Acceptance items 1–3 of task 0217 (backlog 0514, 0526).

import (
	"net/http"
	"strings"
	"testing"
)

// TestNoteListDoesNotShowStatusBadge verifies that the notes listing page
// does not render a badge showing the note's status enum value such as "Draft".
func TestNoteListDoesNotShowStatusBadge(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	var rec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{
		"title":  "Shopping list",
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

	if strings.Contains(row, "Draft") || strings.Contains(row, `sw-badge`) {
		t.Errorf("note list row must not show status badge 'Draft'; found in:\n%s", truncate(row))
	}
	if !strings.Contains(row, "sw-muted") {
		t.Errorf("note list row should still show the timestamp; found in:\n%s", truncate(row))
	}
}

// TestProjectListDoesNotShowStatusBadge verifies that the projects listing page
// does not render a badge showing the project's status enum value such as "Active".
func TestProjectListDoesNotShowStatusBadge(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	var rec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{
		"title":  "Garden",
		"status": "active",
	}), &rec)

	body := get(t, h, "/t/project").Body.String()

	rowStart := strings.Index(body, `href="/t/project/`+rec.ID)
	if rowStart < 0 {
		t.Fatalf("could not find project row in page\n%s", truncate(body))
	}
	rowEnd := strings.Index(body[rowStart:], "</li>")
	if rowEnd < 0 {
		t.Fatalf("could not find end of project row\n%s", truncate(body))
	}
	row := body[rowStart : rowStart+rowEnd]

	if strings.Contains(row, "Active") || strings.Contains(row, `sw-badge`) {
		t.Errorf("project list row must not show status badge 'Active'; found in:\n%s", truncate(row))
	}
	if !strings.Contains(row, "sw-muted") {
		t.Errorf("project list row should still show the timestamp; found in:\n%s", truncate(row))
	}
}

// TestFileListDoesNotShowStatusBadge verifies that the files listing page does not
// render a badge showing the file's status enum value such as "Ready" or
// "Could not be read".
func TestFileListDoesNotShowStatusBadge(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	var rec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/file", map[string]any{
		"title": "Receipts",
	}), &rec)

	body := get(t, h, "/t/file").Body.String()

	rowStart := strings.Index(body, `href="/t/file/`+rec.ID)
	if rowStart < 0 {
		t.Fatalf("could not find file row in page\n%s", truncate(body))
	}
	rowEnd := strings.Index(body[rowStart:], "</li>")
	if rowEnd < 0 {
		t.Fatalf("could not find end of file row\n%s", truncate(body))
	}
	row := body[rowStart : rowStart+rowEnd]

	if strings.Contains(row, "Ready") || strings.Contains(row, `sw-badge`) {
		t.Errorf("file list row must not show status badge 'Ready'; found in:\n%s", truncate(row))
	}
	if !strings.Contains(row, "sw-muted") {
		t.Errorf("file list row should still show the timestamp; found in:\n%s", truncate(row))
	}
}
