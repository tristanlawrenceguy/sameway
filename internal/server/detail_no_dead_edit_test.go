package server_test

import (
	"strings"
	"testing"
)

// TestDetailPageHasNoDeadEditButton checks that note detail pages no longer
// render a server-side <button data-inline-edit> — the working Edit button is
// created client-side by 08-edit.js via progressive enhancement. Acceptance item
// 1: views.go must not contain data-inline-edit in detailPage().
func TestDetailPageHasNoDeadEditButton(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{"title": "Test note"})
	if err != nil {
		t.Fatal(err)
	}
	body := get(t, h, "/t/note/"+rec.ID).Body.String()

	if strings.Contains(body, `data-inline-edit`) {
		t.Errorf("note detail page should not contain a data-inline-edit button (dead UI — 08-edit.js creates the real one)\n%s", truncate(body))
	}
}

// TestDetailPageActivityHasNoDeadEditButton checks that activity detail pages
// also do not render a server-side <button data-inline-edit>. Acceptance item 1.
func TestDetailPageActivityHasNoDeadEditButton(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("activity", map[string]any{
		"actor": "human", "action": "added", "detail": "A test activity",
	})
	if err != nil {
		t.Fatal(err)
	}
	body := get(t, h, "/t/activity/"+rec.ID).Body.String()

	if strings.Contains(body, `data-inline-edit`) {
		t.Errorf("activity detail page should not contain a data-inline-edit button\n%s", truncate(body))
	}
}

// TestDetailPageProposalHasNoDeadEditButton checks that proposal detail pages
// also do not render a server-side <button data-inline-edit>. Acceptance item 1.
func TestDetailPageProposalHasNoDeadEditButton(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("proposal", map[string]any{
		"summary": "Should we refactor this?", "action": `{"tool":"test"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	body := get(t, h, "/t/proposal/"+rec.ID).Body.String()

	if strings.Contains(body, `data-inline-edit`) {
		t.Errorf("proposal detail page should not contain a data-inline-edit button\n%s", truncate(body))
	}
}

// TestDetailPageHasNoSwClusterAfterEditButtonRemoval checks that the empty
// sw-cluster div that existed solely to hold the dead Edit button is also gone.
// Acceptance item 1: after removing the button, no orphan cluster should remain.
func TestDetailPageHasNoSwClusterAfterEditButtonRemoval(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{"title": "Test note"})
	if err != nil {
		t.Fatal(err)
	}
	body := get(t, h, "/t/note/"+rec.ID).Body.String()

	// sw-cluster should not appear on detail pages after the dead button is removed.
	if strings.Contains(body, `class="sw-cluster"`) {
		t.Errorf("note detail page should not have a sw-cluster div (it was created only to hold the dead Edit button)\n%s", truncate(body))
	}
}
