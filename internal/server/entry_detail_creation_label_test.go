package server_test

import (
	"strings"
	"testing"
)

// TestEntryDetailLedeCreationTimeIsLabelled asserts that an entry detail page's
// lede does NOT show a separate creation timestamp span, since the "Logged"
// chip already names and shows the time. Acceptance item 2: only one announcement.
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

	// The entry's creation timestamp is not shown separately — the "Logged …" chip
	// from the glance already provides this with its label. No sw-detail__when span.
	if strings.Contains(page, `class="sw-detail__when`) {
		t.Error("entry detail page must not have a sw-detail__when span (creation time is in the Logged chip)")
	}

	// The lede should contain the "Logged" chip with its time.
	if !strings.Contains(page, ">Logged") {
		t.Error("entry detail page lede should show \"Logged\" from the glance chip")
	}
}

// TestEntryDetailLedeHabitNameAbsent asserts that an entry detail page's lede
// does not contain the habit name; it is rendered in the fields section below.
// Acceptance item 1.
func TestEntryDetailLedeHabitNameAbsent(t *testing.T) {
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
	lede := extractLede(page)

	// The habit name must not appear in the lede at all.
	if strings.Contains(lede, "Mum's evening medicine") {
		t.Errorf("entry detail page lede should not contain the habit name; got: %s", lede)
	}

	// The sw-detail__when span no longer exists on entry pages, so this pattern
	// cannot appear. This assertion is a regression guard against its return.
	if strings.Contains(lede, `</span> · <span class="sw-detail__when`) {
		t.Errorf("entry detail page lede should not have a sw-detail__when span; got: %s", lede)
	}
}

// TestEntryDetailLedeCreationTimestampHasProperLabel asserts that on an entry
// detail page, there is no separate "Created at" span — the "Logged …" chip
// from the glance provides the time with its label instead. Acceptance item 2.
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

	// No separate whenMade span exists on entry pages — the Logged chip covers it.
	if strings.Contains(page, `class="sw-detail__when`) {
		t.Error("entry detail page must not have a sw-detail__when span (creation time is in the Logged chip)")
	}

	// The lede should contain "Logged" from the glance chip.
	if !strings.Contains(page, ">Logged") {
		t.Error("entry detail page lede should show \"Logged\" with its time from the glance chip")
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
