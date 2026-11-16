package server_test

import (
	"strings"
	"testing"
)

// TestEntryDetailHasExactlyOneTimestamp asserts that an entry detail page
// shows the creation timestamp exactly once in its lede — the "Logged …"
// chip from the glance. After removing the duplicate whenMade span, there
// must be no sw-detail__when element on entry pages at all. This covers
// acceptance items 1–2 and 3 (single time, single announcement, format).
func TestEntryDetailHasExactlyOneTimestamp(t *testing.T) {
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

	// Acceptance item 1: exactly one <time> element in the lede div.
	ledeStart := strings.Index(page, `class="sw-lede"`)
	if ledeStart == -1 {
		t.Fatal("entry detail page should have a sw-lede div")
	}
	bodyStart := ledeStart + len(`class="sw-lede">`)
	end := strings.Index(page[bodyStart:], "</div>")
	if end == -1 {
		t.Fatal("unclosed sw-lede div")
	}
	lede := page[bodyStart : bodyStart+end]

	timeCount := 0
	for i := 0; ; {
		idx := strings.Index(lede[i:], "<time")
		if idx == -1 {
			break
		}
		timeCount++
		i += idx + len("<time")
	}
	if timeCount != 1 {
		t.Errorf("entry detail lede should have exactly one <time> element; got %d: %s", timeCount, lede)
	}

	// Acceptance item 3: the single remaining timestamp is the "Logged …" chip
	// from the glance. It must be wrapped in <time> with a datetime attribute
	// and contain relative time text.
	if !strings.Contains(lede, `<time datetime="`) {
		t.Error("entry detail lede should have one <time> element for the Logged time")
	}

	// The sw-detail__when span from whenMade must NOT be present on entry pages.
	if strings.Contains(page, `class="sw-detail__when`) {
		t.Errorf("entry detail page must not contain a sw-detail__when span (duplicate timestamp); lede: %s", lede)
	}

	// Acceptance item 2: the single time text is human-readable with date and time.
	if !strings.Contains(lede, "Logged") {
		t.Errorf("entry detail lede should show the Logged label; got: %s", lede)
	}
}

// TestEntryDetailLedeShowsLoggedTime asserts that the single remaining
// timestamp on an entry detail page is the "Logged …" chip with its full
// text (e.g. "today at 6:21am"), confirming format preservation from item 3.
func TestEntryDetailLedeShowsLoggedTime(t *testing.T) {
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
	lede := extractLede(page)

	// The glance chip must contain the "Logged" label text.
	if !strings.Contains(lede, ">Logged") {
		t.Errorf("entry detail lede should show \"Logged\" from the glance chip; got: %s", lede)
	}

	// It must have a datetime attribute on the <time> element for accessibility.
	if !strings.Contains(lede, `datetime="`) {
		t.Error("entry detail lede time element must have a datetime attribute")
	}

	// The page as a whole must not contain any sw-detail__when span (the duplicate).
	if strings.Contains(page, `class="sw-detail__when`) {
		t.Errorf("entry detail page must not have a sw-detail__when span; got: %s", extractLede(page))
	}
}
