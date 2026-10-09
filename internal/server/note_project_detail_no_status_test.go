package server_test

import (
	"strings"
	"testing"
)

// TestNoteDetailPageHasNoStatusLabelInLede asserts that a note detail page
// does not expose its internal "Draft status" label in the lede text under
// the heading. The bug is that the raw enum value "Draft" renders with a
// visually-hidden context "status", producing "Draft status" to screen
// readers (backlog 0514). After the fix, no such badge should appear in the
// lede for note detail pages. Creation time is still shown.
func TestNoteDetailPageHasNoStatusLabelInLede(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title":  "Plans",
		"status": "draft",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/note/"+rec.ID).Body.String()

	// The lede should NOT contain the raw status label with its field context.
	if strings.Contains(page, `Draft<span class="sw-visually-hidden"> status</span>`) {
		t.Error("note detail lede should not show \"Draft status\" — internal enum value exposed to screen readers")
	}

	// Creation time must still appear in the page (createdAt == updatedAt on a fresh record).
	if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
		t.Error("note detail page should still show when the record was created")
	}
}

// TestProjectDetailPageHasNoStatusLabelInLede asserts that a project detail
// page does not expose its internal status label ("Active status" / "Done
// status") in the lede text under the heading (backlog 0514). After the fix,
// no such badge should appear in the lede for project detail pages. Creation/
// update times are still shown.
func TestProjectDetailPageHasNoStatusLabelInLede(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("project", map[string]any{
		"title": "My Project",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/project/"+rec.ID).Body.String()

	// The lede should NOT contain the raw status label with its field context.
	if strings.Contains(page, `<span class="sw-visually-hidden"> status</span>`) {
		t.Error("project detail lede should not expose \"status\" as a hidden field name")
	}

	// Creation time must still appear in the page (createdAt == updatedAt on a fresh record).
	if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
		t.Error("project detail page should still show when the record was created")
	}
}

// TestProjectDetailPageDoneStatusHasNoStatusLabelInLede asserts that a project
// marked as done does not expose its internal "Done status" label in the lede
// text under the heading (backlog 0514). After the fix, no such badge should
// appear for any project detail page regardless of status value.
func TestProjectDetailPageDoneStatusHasNoStatusLabelInLede(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("project", map[string]any{
		"title":  "Finished Project",
		"status": "done",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/project/"+rec.ID).Body.String()

	// The lede should NOT contain the raw status label with its field context.
	if strings.Contains(page, `<span class="sw-visually-hidden"> status</span>`) {
		t.Error("project detail lede should not expose \"status\" as a hidden field name")
	}

	// Creation time must still appear in the page.
	if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
		t.Error("project detail page should still show when the record was created")
	}
}
