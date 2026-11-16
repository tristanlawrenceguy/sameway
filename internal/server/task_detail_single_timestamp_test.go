package server_test

import (
	"strings"
	"testing"
)

// TestTaskDetailHasExactlyOneTimestamp asserts that a task detail page shows
// exactly one <time> element in its lede — the due/ring-time badge from the
// glance. The duplicate whenMade span must be absent, matching how entry and
// reminder pages were fixed (goals 0121/0341). Acceptance items 1–2: single
// time, single announcement on /t/task/<id>.
func TestTaskDetailHasExactlyOneTimestamp(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	rec, err := a.Store.Create("task", map[string]any{
		"title": "Order compost",
		"due":   "2026-10-09T08:16:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/task/"+rec.ID).Body.String()

	// Acceptance item 1: exactly one <time> element in the lede div.
	ledeStart := strings.Index(page, `class="sw-lede"`)
	if ledeStart == -1 {
		t.Fatal("task detail page should have a sw-lede div")
	}
	bodyStart := ledeStart + len(`class="sw-lede">`)
	end := strings.Index(page[bodyStart:], "</div>")
	if end == -1 {
		t.Fatal("unclosed sw-lede div")
	}
	lede := page[bodyStart : bodyStart+end]

	timeCount := 0
	for i := 0; ; {
		idx := strings.Index(lede[i:], "<time")
		if idx == -1 {
			break
		}
		timeCount++
		i += idx + len("<time")
	}
	if timeCount != 1 {
		t.Errorf("task detail lede should have exactly one <time> element; got %d: %s", timeCount, lede)
	}

	// Acceptance item 3: the remaining timestamp has a datetime attribute.
	if !strings.Contains(lede, `<time datetime="`) {
		t.Error("task detail lede should have one <time> element with a datetime attribute")
	}

	// The sw-detail__when span from whenMade must NOT be present on task pages.
	if strings.Contains(page, `class="sw-detail__when`) {
		t.Errorf("task detail page must not contain a sw-detail__when span (duplicate timestamp); lede: %s", lede)
	}

	// Acceptance item 2: the single time text is human-readable with "Due" context.
	if !strings.Contains(strings.ToLower(lede), "due") {
		t.Errorf("task detail lede should show a \"due\" badge; got: %s", lede)
	}
}
