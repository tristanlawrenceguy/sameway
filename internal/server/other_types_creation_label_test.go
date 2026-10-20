package server_test

import (
	"strings"
	"testing"
)

// TestNoteDetailLedeCreationTimeIsLabelled asserts that a note detail page's
// lede shows the creation timestamp with a visible "Created" label before it.
// Acceptance item 4.
func TestNoteDetailLedeCreationTimeIsLabelled(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{"title": "Plans"})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/note/"+rec.ID).Body.String()

	if !strings.Contains(page, `class="sw-visually-hidden">`) || !strings.Contains(strings.ToLower(extractWhenMadeSpan(page)), "created at ") {
		t.Error("note detail page lede should show \"Created\" in a sw-visually-hidden span before the creation timestamp; got an unlabeled time")
	}

	if strings.Contains(page, ">Created ") {
		t.Error("note detail page lede shows bare \"Created\" instead of wrapping it in a sw-visually-hidden span")
	}
	if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
		t.Error("note detail page lede should have a sw-detail__when span with muted styling")
	}
}

// TestFileDetailLedeCreationTimeIsLabelled asserts that a file detail page's
// lede shows the creation timestamp with a visible "Created" label before it.
// Acceptance item 4.
func TestFileDetailLedeCreationTimeIsLabelled(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("file", map[string]any{"title": "Readme"})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/file/"+rec.ID).Body.String()

	if !strings.Contains(page, `class="sw-visually-hidden">`) || !strings.Contains(strings.ToLower(extractWhenMadeSpan(page)), "created at ") {
		t.Error("file detail page lede should show \"Created\" in a sw-visually-hidden span before the creation timestamp; got an unlabeled time")
	}

	if strings.Contains(page, ">Created ") {
		t.Error("file detail page lede shows bare \"Created\" instead of wrapping it in a sw-visually-hidden span")
	}
	if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
		t.Error("file detail page lede should have a sw-detail__when span with muted styling")
	}
}

// TestActionDetailLedeCreationTimeIsLabelled asserts that an action detail page's
// lede shows the creation timestamp with a visible "Created" label before it.
// Acceptance item 4.
func TestActionDetailLedeCreationTimeIsLabelled(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("action", map[string]any{"title": "Sync data"})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/action/"+rec.ID).Body.String()

	if !strings.Contains(page, `class="sw-visually-hidden">`) || !strings.Contains(strings.ToLower(extractWhenMadeSpan(page)), "created at ") {
		t.Error("action detail page lede should show \"Created\" in a sw-visually-hidden span before the creation timestamp; got an unlabeled time")
	}

	if strings.Contains(page, ">Created ") {
		t.Error("action detail page lede shows bare \"Created\" instead of wrapping it in a sw-visually-hidden span")
	}
	if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
		t.Error("action detail page lede should have a sw-detail__when span with muted styling")
	}
}

// TestProjectDetailLedeCreationTimeIsLabelled asserts that a project detail page's
// lede shows the creation timestamp with a visible "Created" label before it.
// Acceptance item 4.
func TestProjectDetailLedeCreationTimeIsLabelled(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("project", map[string]any{"title": "My Project"})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/project/"+rec.ID).Body.String()

	if !strings.Contains(page, `class="sw-visually-hidden">`) || !strings.Contains(strings.ToLower(extractWhenMadeSpan(page)), "created at ") {
		t.Error("project detail page lede should show \"Created\" in a sw-visually-hidden span before the creation timestamp; got an unlabeled time")
	}

	if strings.Contains(page, ">Created ") {
		t.Error("project detail page lede shows bare \"Created\" instead of wrapping it in a sw-visually-hidden span")
	}
	if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
		t.Error("project detail page lede should have a sw-detail__when span with muted styling")
	}
}
