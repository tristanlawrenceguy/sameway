package server_test

import (
	"regexp"
	"strings"
	"testing"
)

// TestTaskDetailLedeNoUpdatedLabel asserts that the task detail page lede does
// not include an " · Updated ..." suffix — the last-change time always appeared
// with a field-name label ("Updated") and must be removed too. Acceptance item 2.
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

	re := regexp.MustCompile(`class="sw-detail__when[^"]*">Created <time[^>]*>([^<]+)</time>`)
	matches := re.FindStringSubmatch(body)
	if len(matches) < 2 {
		t.Fatal("task detail page should have a sw-detail__when span with \"Created\" label in the lede")
	}
	whenText := matches[1]

	// The span must still contain relative time text (no "Updated" suffix).
	if strings.Contains(whenText, " · Updated") {
		t.Errorf("task detail lede should not contain \" · Updated\"; got %q", whenText)
	}

	if !strings.Contains(whenText, "ago") && !strings.HasPrefix(whenText, "today") {
		t.Errorf("task detail lede should start with a relative timestamp; got %q", whenText)
	}
}
