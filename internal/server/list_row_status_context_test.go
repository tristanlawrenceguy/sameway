package server_test

// State badges on list rows must include their field name as context so screen
// readers announce "Published status" instead of just "Published". Without it,
// a bare enum value heard out of place is meaningless. Acceptance 2 of task
// 0286 (backlog 0754).

import (
	"net/http"
	"strings"
	"testing"
)

// TestListRowStateBadgeIncludesFieldContext verifies that when a note's status
// enum value appears on a list row, the badge includes "Status" as visually
// hidden context so screen readers say "Published status".
func TestListRowStateBadgeIncludesFieldContext(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	var rec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{
		"title":  "Published note",
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

	// The badge should include the field name as visually hidden context.
	if strings.Contains(row, "Published") && !strings.Contains(row, `<span class="sw-visually-hidden"> status</span>`) {
		t.Errorf("state badge on list row must include its field name as context; found in:\n%s", truncate(row))
	}
}

// TestListRowStateBadgeContextForActiveProject verifies that a project with
// status "active" shows the field context on its list row.
func TestListRowStateBadgeContextForActiveProject(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	var rec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{
		"title":  "Garden redesign",
		"status": "active",
	}), &rec)

	body := get(t, h, "/t/project").Body.String()

	rowStart := strings.Index(body, `href="/t/project/`+rec.ID)
	if rowStart < 0 {
		t.Fatalf("could not find project row in page\n%s", truncate(body))
	}
	rowEnd := strings.Index(body[rowStart:], "</li>")
	if rowEnd < 0 {
		t.Fatalf("could not find end of note row\n%s", truncate(body))
	}
	row := body[rowStart : rowStart+rowEnd]

	if strings.Contains(row, "Active") && !strings.Contains(row, `<span class="sw-visually-hidden"> status</span>`) {
		t.Errorf("project state badge on list row must include its field name as context; found in:\n%s", truncate(row))
	}
}

// TestListRowStateBadgeContextOmittedWhenNoValue verifies that when a note's
// status is the default (Draft), which StateWorth says is not worth showing,
// no badge appears at all — so there is nothing to check for context.
func TestListRowStateBadgeContextOmittedWhenNotWorth(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	var rec struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{
		"title":  "Draft note",
		"status": "draft", // default, not worth showing
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

	// Draft is the default, so no badge should appear. The only "draft" in this
	// row should be from the note title itself ("Draft note"), not as a status
	// badge. Check for sw-badge which would indicate an unwanted enum badge.
	if strings.Contains(row, `sw-badge`) {
		t.Errorf("a draft status (default) must not appear as a badge on list rows; found in:\n%s", truncate(row))
	}
}
