package server_test

import (
	"strings"
	"testing"
)

// TestReminderDetailHasExactlyOneTimestamp asserts that a reminder detail page
// shows exactly one <time> element in its lede — the ring-time badge from the
// glance. The duplicate whenMade span must be absent, matching how entry pages
// were fixed in goal 0121 (task 0340). Acceptance items 1–2: single time,
// single announcement on /t/reminder/<id>.
func TestReminderDetailHasExactlyOneTimestamp(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("reminder", map[string]any{
		"title": "Mum's morning medicine",
		"at":    "2026-10-09T08:16:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/reminder/"+rec.ID).Body.String()

	// Acceptance item 1: exactly one <time> element in the lede div.
	ledeStart := strings.Index(page, `class="sw-lede"`)
	if ledeStart == -1 {
		t.Fatal("reminder detail page should have a sw-lede div")
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
		t.Errorf("reminder detail lede should have exactly one <time> element; got %d: %s", timeCount, lede)
	}

	// Acceptance item 3: the remaining timestamp has a datetime attribute.
	if !strings.Contains(lede, `<time datetime="`) {
		t.Error("reminder detail lede should have one <time> element with a datetime attribute")
	}

	// The sw-detail__when span from whenMade must NOT be present on reminder pages.
	if strings.Contains(page, `class="sw-detail__when`) {
		t.Errorf("reminder detail page must not contain a sw-detail__when span (duplicate timestamp); lede: %s", lede)
	}

	// Acceptance item 2: the single time text is human-readable with "Rings" label.
	if !strings.Contains(lede, ">Rings") {
		t.Errorf("reminder detail lede should show the \"Rings\" badge; got: %s", lede)
	}
}

// TestReminderDetailLedeShowsRingTime asserts that the single remaining timestamp
// on a reminder detail page is the "Rings …" chip with its full text (e.g.
// "today at 8:16am"), confirming format preservation from acceptance item 3.
func TestReminderDetailLedeShowsRingTime(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("reminder", map[string]any{
		"title": "Take medicine",
		"at":    "2026-10-09T08:16:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/reminder/"+rec.ID).Body.String()
	lede := extractLede(page)

	// The glance chip must contain the "Rings" label text.
	if !strings.Contains(lede, ">Rings") {
		t.Errorf("reminder detail lede should show \"Rings\" from the glance chip; got: %s", lede)
	}

	// It must have a datetime attribute on the <time> element for accessibility.
	if !strings.Contains(lede, `datetime="`) {
		t.Error("reminder detail lede time element must have a datetime attribute")
	}

	// The page as a whole must not contain any sw-detail__when span (the duplicate).
	if strings.Contains(page, `class="sw-detail__when`) {
		t.Errorf("reminder detail page must not have a sw-detail__when span; got: %s", lede)
	}
}

// TestEntryDetailStillHasExactlyOneTimestamp asserts that entry detail pages
// still show exactly one timestamp after the reminder fix — no regression from
// goal 0121 (task 0340). Acceptance item 4.
func TestEntryDetailStillHasExactlyOneTimestamp(t *testing.T) {
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

	// Entry detail lede should have exactly one <time> element.
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

	// Entry pages must not have a sw-detail__when span.
	if strings.Contains(page, `class="sw-detail__when`) {
		t.Errorf("entry detail page must not contain a sw-detail__when span (regression); lede: %s", lede)
	}

	// The single time text should be the "Logged" chip.
	if !strings.Contains(lede, ">Logged") {
		t.Errorf("entry detail lede should show the \"Logged\" badge; got: %s", lede)
	}
}
