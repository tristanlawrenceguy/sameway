package server_test

import (
	"strings"
	"testing"
)

// TestReminderDetailLedeCreationTimeIsLabelled asserts that a reminder detail page's
// lede shows the creation timestamp with a visible "Added" label before it.
// Acceptance item 3.
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

	// The page must contain the visible text "Added" before the creation timestamp.
	if !strings.Contains(page, ">Added ") {
		t.Error("reminder detail page lede should show \"Added\" before the creation timestamp; got an unlabeled time")
	}

	// The span must still exist with proper structure.
	if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
		t.Error("reminder detail page lede should have a sw-detail__when span with muted styling")
	}
}
