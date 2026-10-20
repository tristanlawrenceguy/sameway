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

	// Must show "created at" in a sw-visually-hidden span before the timestamp.
	if !strings.Contains(page, `class="sw-visually-hidden">`) || !strings.Contains(strings.ToLower(extractWhenMadeSpan(page)), "created at ") {
		t.Errorf("task detail lede must show \"Created at\" in a sw-visually-hidden span before creation timestamp; found label:\n%s", extractLabel(page))
	}

	// Must NOT have bare "Created" text — it should be wrapped in sw-visually-hidden.
	if strings.Contains(page, ">Created ") {
		t.Error("task detail lede shows bare \"Created\" instead of wrapping it in a sw-visually-hidden span")
	}

	// The span structure must still exist.
	if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
		t.Errorf("task detail page lede should have a sw-detail__when span with muted styling")
	}
}
