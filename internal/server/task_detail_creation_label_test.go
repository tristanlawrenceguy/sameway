package server_test

import (
	"strings"
	"testing"
	"time"
)

// TestTaskDetailLedeCreationTimeIsLabelled asserts that a task detail page's
// lede shows the creation timestamp with a visible "Created" label before it,
// so a person reading never encounters an unlabeled timestamp. Acceptance item 1.
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

	// The page must contain the visible text "Created" followed by a relative timestamp.
	if !strings.Contains(page, ">Created ") {
		t.Error("task detail page lede should show \"Created\" before the creation timestamp; got an unlabeled time like \"3 days ago\"")
	}

	// The span must still exist with proper structure.
	if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
		t.Error("task detail page lede should have a sw-detail__when span with muted styling")
	}

	// The relative timestamp text must still be present inside <time>.
	if !strings.Contains(page, "ago") && !strings.Contains(page, ">Today") && !strings.Contains(page, ">Yesterday") {
		t.Error("task detail page lede should show a relative timestamp like \"3 days ago\" or \"Today at…\" after the label")
	}

	// The due date chip must be labelled "Due" (not just raw text).
	if !strings.Contains(page, ">Due ") && !strings.Contains(page, ">Overdue") {
		t.Error("task detail page should show a labelled due-date chip like \"Due …\" or \"Overdue…\" in the lede, not an unlabeled date")
	}
}
