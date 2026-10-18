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

// TestEntryDetailLedeHabitNameNotRunningIntoCreated asserts that the habit name
// badge on an entry detail page is visually separated from the "Created" label by
// visible punctuation (a middle dot) so screen readers do not hear them as one
// concatenated string. Acceptance items 1 and 3.
func TestEntryDetailLedeHabitNameNotRunningIntoCreated(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	habitRec, err := a.Store.Create("habit", map[string]any{
		"name":   "Mum's evening medicine",
		"target": 1.0,
	})
	if err != nil {
		t.Fatal(err)
	}

	entryRec, err := a.Store.Create("entry", map[string]any{
		"habit":  habitRec.ID,
		"at":     "2026-10-02T04:32:00Z",
		"amount": 2.0,
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/entry/"+entryRec.ID).Body.String()

	// The lede div must contain a middle-dot separator between the habit badge
	// and the whenMade span. This ensures the habit name does not run directly
	// into "Created" on screen or in a screen reader's linear text flow.
	if !strings.Contains(page, `</span> · <span class="sw-detail__when`) {
		t.Errorf("entry detail page lede should separate the habit badge from \"Created\" with visible punctuation; got no middle dot between \"%s\" and whenMade span", extractLede(page))
	}

	// The lede must not contain the pattern "Habit Name Created" as adjacent
	// text without any separator element or punctuation between them.
	lede := extractLede(page)
	if strings.Contains(lede, `</span>`+"\n"+"Created") ||
		(strings.Contains(lede, ">Mum's evening medicine<") && !strings.Contains(lede, " · ")) {
		t.Error("habit name should not run directly into \"Created\" without visible separation in the lede")
	}
}

// TestEntryDetailLedeCreationTimestampHasProperLabel asserts that on an entry
// detail page, the creation timestamp is preceded by a label like "Created at"
// so screen readers announce it with clear context. Acceptance item 2.
func TestEntryDetailLedeCreationTimestampHasProperLabel(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	habitRec, err := a.Store.Create("habit", map[string]any{
		"name":   "Drink water",
		"target": 1.0,
	})
	if err != nil {
		t.Fatal(err)
	}

	entryRec, err := a.Store.Create("entry", map[string]any{
		"habit":  habitRec.ID,
		"at":     "2026-10-02T04:32:00Z",
		"amount": 5.0,
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/entry/"+entryRec.ID).Body.String()

	// The whenMade span must contain "Created at" (not just "Created") before the
	// time element, so a screen reader hears a proper label for the timestamp.
	if !strings.Contains(page, ">Created at ") {
		t.Error("entry detail page lede should show \"Created at\" before the creation timestamp; got bare \"Created\" without \"at\"")
	}
}

// extractLede returns the text content of the sw-lede div from an HTML page.
func extractLede(page string) string {
	start := strings.Index(page, `class="sw-lede"`)
	if start == -1 {
		return ""
	}
	bodyStart := start + len(`class="sw-lede">`)
	end := strings.Index(page[bodyStart:], "</div>")
	if end == -1 {
		return page[bodyStart:]
	}
	return page[bodyStart : bodyStart+end]
}
