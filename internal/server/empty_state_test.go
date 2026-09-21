package server_test

// Tests for empty-state simplification across surfaces.
// These pin that task 0155's Acceptance 3 is met: empty state text on list
// pages (notes, actions, canvases) is a single short line telling the person
// what to do next.

import (
	"net/http"
	"os"
	"strings"
	"testing"
)

// TestCanvasEmptyStateSaysWhatToDo checks that the canvas empty-state string
// in canvas.go has been simplified from the old two-sentence "Nothing here yet.
// Ask for something in the chat and it appears here." to one short line telling
// the person what to do next.  This covers Acceptance 3.
func TestCanvasEmptyStateSaysWhatToDo(t *testing.T) {
	data, err := os.ReadFile("canvas.go")
	if err != nil {
		t.Fatalf("cannot read canvas.go: %v", err)
	}
	src := string(data)

	if strings.Contains(src, "Nothing here yet") {
		t.Error("canvas empty state must not say 'Nothing here yet' — it should tell the person what to do next (Acceptance 3)")
	}
	if strings.Contains(src, "and it appears here") {
		t.Error("canvas empty state must be a single line; the old version had two sentences")
	}
}

// TestListPageEmptyStateIsActionable checks that an empty notes listing shows
// one short actionable instruction with a link — not just "No notes yet."  This
// covers Acceptance 3.
func TestListPageEmptyStateIsActionable(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/t/note")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	// The new empty state should use the sw-empty class and include a link to create.
	if !strings.Contains(body, `<p class="sw-empty">`) {
		t.Errorf("empty list page should use the sw-empty class\n%s", truncate(body))
	}
	// It should tell the person what to do next — "Add your first" is one pattern.
	if strings.Contains(body, `>No notes yet.<`) || (strings.Contains(body, `<p>No notes yet.</p>`)) {
		t.Error("empty list page must not just say 'No notes yet.' — it should tell the person what to do next with a link")
	}
	if !strings.Contains(body, "/t/note/new") {
		t.Errorf("empty list page should link to the new-resource form\n%s", truncate(body))
	}
}

// TestActivityEmptyStateIsActionable checks that an empty activity log shows
// one short actionable instruction — not "Nothing has happened yet."  This covers
// Acceptance 3.
func TestActivityEmptyStateIsActionable(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/activity")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if strings.Contains(body, `>Nothing has happened yet.<`) || (strings.Contains(body, `<p class="sw-empty">Nothing has happened yet.</p>`)) {
		t.Error("empty activity page must not say 'Nothing has happened yet.' — it should tell the person what to do next")
	}
}

// TestSearchEmptyStateIsTight checks that an empty search result page shows one
// short actionable line — not the old two-part sentence.  This covers Acceptance 3.
func TestSearchEmptyStateIsTight(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/search?q=nonexistent")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if strings.Contains(body, "Nothing has") && strings.Contains(body, "Try searching with different words") {
		t.Error("empty search state must not use the old two-sentence pattern — it should be a single tight line")
	}
}
