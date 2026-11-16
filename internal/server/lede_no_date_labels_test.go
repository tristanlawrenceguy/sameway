package server_test

import (
	"strings"
	"testing"
)

// TestTaskDetailLedeNoUpdatedLabel asserts that the task detail page lede does
// not include a "sw-detail__when" span — which would carry an unlabelled creation
// timestamp. With the span removed entirely, there is no " · Updated" problem to
// worry about. Acceptance items 1–2: single timestamp with clear context.
func TestTaskDetailLedeNoUpdatedLabel(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("task", map[string]any{
		"title": "Buy milk",
	})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/t/task/"+rec.ID).Body.String()

	// The sw-detail__when span must be absent (the creation-time duplicate is gone).
	if strings.Contains(body, `class="sw-detail__when`) {
		t.Error("task detail lede must not contain a sw-detail__when span")
	}

	// No " · Updated" text should appear anywhere in the page lede.
	ledeStart := strings.Index(body, `class="sw-lede"`)
	if ledeStart == -1 {
		t.Fatal("task detail page should have a sw-lede div")
	}
	bodyStart := ledeStart + len(`class="sw-lede">`)
	end := strings.Index(body[bodyStart:], "</div>")
	if end == -1 {
		t.Fatal("unclosed sw-lede div")
	}
	lede := body[bodyStart : bodyStart+end]
	if strings.Contains(lede, " · Updated") {
		t.Errorf("task detail lede should not contain \" · Updated\"; got: %s", lede)
	}
}
