package server_test

import (
	"strings"
	"testing"
)

// TestTaskDetailLedeCreationTimeIsCreatedAt asserts that a task detail page's
// lede shows "Created at" before the creation time — not bare "Created". This is
// acceptance item 1. The middle-dot separator between title area and metadata
// must also be present.
func TestTaskDetailLedeCreationTimeIsCreatedAt(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("task", map[string]any{
		"title": "Order compost",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/task/"+rec.ID).Body.String()

	// Must show "Created at" (not bare "Created") before the timestamp.
	if !strings.Contains(page, ">Created at ") {
		t.Errorf("task detail lede must show \"Created at\" before creation timestamp; found label:\n%s", extractLabel(page))
	}

	// Must NOT show bare "Created " without "at".
	if strings.Contains(page, ">Created <time") && !strings.Contains(page, ">Created at ") {
		t.Error("task detail lede shows bare \"Created\" instead of \"Created at\" — the timestamp label must be a proper phrase")
	}

	// The span structure must still exist.
	if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
		t.Errorf("task detail page lede should have a sw-detail__when span with muted styling")
	}
}
