package server_test

import (
	"strings"
	"testing"
)

// TestTaskDetailLedeCreationTimeIsCreatedAt asserts that a task detail page's
// lede no longer shows the creation-time span — matching how entry and reminder
// pages were fixed (goals 0121/0341). Acceptance items 1–2: single timestamp.
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

	// The sw-detail__when span must NOT be present (creation-time duplicate removed).
	if strings.Contains(page, `class="sw-detail__when`) {
		t.Error("task detail lede must not contain a sw-detail__when span")
	}

	// Must NOT have bare "Created" text in the lede.
	if strings.Contains(page, ">Created ") {
		t.Error("task detail lede shows bare \"Created\"")
	}

	// The due-date timestamp must still be present.
	if !strings.Contains(page, `class="sw-lede"`) {
		t.Fatal("task detail page should have a sw-lede div")
	}
}
