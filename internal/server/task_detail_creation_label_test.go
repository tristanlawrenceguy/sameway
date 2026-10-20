package server_test

import (
	"strings"
	"testing"
	"time"
)

// TestTaskDetailLedeCreationTimeIsLabelled asserts that a task detail page's
// lede shows the creation timestamp without any label text before it,
// so a person reading encounters only the timestamp. Acceptance item 1.
func TestTaskDetailLedeCreationTimeIsLabelled(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("task", map[string]any{
		"title": "Buy milk",
		"due":   time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/task/"+rec.ID).Body.String()

	lower := strings.ToLower(extractWhenMadeSpan(page))
	// The page must NOT contain "created at" label text or nested span.
	if strings.Contains(lower, "created at ") || strings.Contains(lower, "<span") {
		t.Error("task detail page lede should not show \"Created\" label text before the creation timestamp; got an unlabeled time like \"3 days ago\"")
	}

	// The page must NOT contain bare "Created" text.
	if strings.Contains(page, ">Created ") {
		t.Error("task detail page lede shows bare \"Created\"")
	}

	// The span must still exist with proper structure and a <time> element.
	if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
		t.Error("task detail page lede should have a sw-detail__when span with muted styling")
	}

	if !strings.Contains(extractWhenMadeSpan(page), `<time`) && !strings.Contains(extractWhenMadeSpan(page), `datetime="`) {
		t.Error("task detail page lede should have a <time> element with datetime for the creation timestamp")
	}

	// The relative timestamp text must still be present inside <time>.
	if !strings.Contains(page, "ago") && !strings.Contains(page, ">Today") && !strings.Contains(page, ">Yesterday") {
		t.Error("task detail page lede should show a relative timestamp like \"3 days ago\" or \"Today at…\"")
	}

	// The due date chip must be labelled "Due" (not just raw text).
	if !strings.Contains(page, ">Due ") && !strings.Contains(page, ">Overdue") {
		t.Error("task detail page should show a labelled due-date chip like \"Due …\" or \"Overdue…\" in the lede, not an unlabeled date")
	}
}
