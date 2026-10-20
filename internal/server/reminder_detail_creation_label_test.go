package server_test

import (
	"strings"
	"testing"
)

// TestReminderDetailLedeCreationTimeIsLabelled asserts that a reminder detail page's
// lede shows the creation timestamp without any label text before it.
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

	lower := strings.ToLower(extractWhenMadeSpan(page))
	if strings.Contains(lower, "created at ") || strings.Contains(lower, "<span") {
		t.Error("reminder detail page lede should not contain \"Created\" label text or nested span before the creation timestamp; got an unlabeled time")
	}

	if !strings.Contains(extractWhenMadeSpan(page), `<time`) && !strings.Contains(extractWhenMadeSpan(page), `datetime="`) {
		t.Error("reminder detail page lede should have a <time> element with datetime for the creation timestamp")
	}

	// The span must still exist with proper structure.
	if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
		t.Error("reminder detail page lede should have a sw-detail__when span with muted styling")
	}
}
