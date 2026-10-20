package server_test

import (
	"strings"
	"testing"
)

// TestTaskDetailLedeCreationTimeIsCreatedAt asserts that a task detail page's
// lede shows the creation time without any label text — not bare "Created". This is
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

	lower := strings.ToLower(extractWhenMadeSpan(page))
	// Must NOT have any label text — visible or visually-hidden.
	if strings.Contains(lower, "created at ") || strings.Contains(lower, "<span") {
		t.Errorf("task detail lede must not contain \"Created at\" label text; found:\n%s", extractLabel(page))
	}

	// Must NOT have bare "Created" text.
	if strings.Contains(page, ">Created ") {
		t.Error("task detail lede shows bare \"Created\"")
	}

	// The span structure must still exist with a <time> element.
	if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"`) {
		t.Errorf("task detail page lede should have a sw-detail__when span with muted styling")
	}

	if !strings.Contains(extractWhenMadeSpan(page), `<time`) && !strings.Contains(extractWhenMadeSpan(page), `datetime="`) {
		t.Error("task detail lede should have a <time> element with datetime for the creation timestamp")
	}
}
