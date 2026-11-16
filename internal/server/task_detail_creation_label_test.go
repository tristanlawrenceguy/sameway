package server_test

import (
	"strings"
	"testing"
	"time"
)

// TestCreationLabelIsCreatedNotAdded asserts that a task detail page's lede no
// longer carries the creation-time span — matching how entry and reminder pages
// were fixed (goals 0121/0341). The due-date chip must still be present.
func TestTaskDetailNoDuplicateTimestampInLede(t *testing.T) {
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

	// The sw-detail__when span must NOT be present (creation-time duplicate removed).
	if strings.Contains(page, `class="sw-detail__when`) {
		t.Error("task detail lede must not contain a sw-detail__when span")
	}

	// Must NOT have bare "Created" text.
	if strings.Contains(page, ">Created ") {
		t.Error("task detail page lede shows bare \"Created\"")
	}

	// The due date chip must be labelled "Due" (not just raw text).
	if !strings.Contains(page, ">Due ") && !strings.Contains(page, ">Overdue") {
		t.Error("task detail page should show a labelled due-date chip like \"Due …\" or \"Overdue…\" in the lede, not an unlabeled date")
	}
}
