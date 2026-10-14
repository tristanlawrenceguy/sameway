package server_test

import (
	"strings"
	"testing"
)

// TestEntryDetailLedeCreationTimeIsLabelled asserts that an entry detail page's
// lede shows the creation timestamp with a visible "Created" label before it.
// Acceptance item 2.
func TestEntryDetailLedeCreationTimeIsLabelled(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	habitRec, err := a.Store.Create("habit", map[string]any{
		"name":   "Exercise",
		"target": 1.0,
	})
	if err != nil {
		t.Fatal(err)
	}

	entryRec, err := a.Store.Create("entry", map[string]any{
		"habit":  habitRec.ID,
		"at":     "2026-10-02T04:32:00Z",
		"amount": 3.0,
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/entry/"+entryRec.ID).Body.String()

	// The page must contain the visible text "Created" before the creation timestamp.
	if !strings.Contains(page, ">Created ") {
		t.Error("entry detail page lede should show \"Created\" before the creation timestamp; got an unlabeled time like \"3 days ago\"")
	}

	// The span must still exist with proper structure.
	if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
		t.Error("entry detail page lede should have a sw-detail__when span with muted styling")
	}
}
