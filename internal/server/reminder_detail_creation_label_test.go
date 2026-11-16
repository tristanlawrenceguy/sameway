package server_test

import (
	"strings"
	"testing"
)

// TestReminderDetailLedeCreationTimeIsLabelled asserts that a reminder detail page's
// lede does NOT show the creation timestamp at all — only the ring time. The
// whenMade span must be absent for reminders, matching the duplicate-timestamp fix.
func TestReminderDetailLedeCreationTimeIsLabelled(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("reminder", map[string]any{
		"title": "Take medicine",
		"at":    "2026-10-09T06:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/reminder/"+rec.ID).Body.String()

	// The whenMade span must NOT be present on reminder pages — the creation time
	// is noise; only the ring-time badge from the glance should appear.
	if strings.Contains(page, `class="sw-detail__when`) {
		t.Error("reminder detail page must not contain a sw-detail__when span (creation timestamp removed to avoid duplicate)")
	}

	// No <time> element for creation should exist in the whenMade area.
	if !strings.Contains(extractWhenMadeSpan(page), "(no span found)") {
		t.Error("reminder detail lede should not have a whenMade span at all; got: " + extractWhenMadeSpan(page))
	}
}
