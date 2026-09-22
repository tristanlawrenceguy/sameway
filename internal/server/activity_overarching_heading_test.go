package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// TestActivityPageHasOverarchingHeading checks that the /activity listing page
// has exactly one visible heading "Activity" — the h1 from the layout template.
// The duplicate <h2>Activity</h2> in body content was removed to avoid screen
// reader duplication. Date-grouped headings come after, and no other visible
// heading repeats "Activity". (Acceptance 1)
func TestActivityPageHasOverarchingHeading(t *testing.T) {
	a, h := newApp(t)

	chat.Record(a.Store, "assistant", chat.Change{Action: "created", Component: "note", ID: "aaa1", Detail: "First note"})

	body := get(t, h, "/activity").Body.String()

	// The overarching heading is the layout h1, not a body-level duplicate.
	if !strings.Contains(body, ">Activity</h1>") {
		t.Errorf("the /activity page should contain an <h1>Activity</h1> from the layout\n%s", truncate(body))
	}

	// No duplicate <h2>Activity</h2> in body content.
	if strings.Count(body, `<h2>Activity</h2>`) > 0 {
		t.Errorf("the /activity page should NOT contain a duplicate <h2>Activity</h2>\n%s", truncate(body))
	}

	// The h1 must come before any date-heading like "Monday 5 January".
	idxH1 := strings.Index(body, `<h1`)
	if idxH1 >= 0 {
		idxDateHeading := strings.Index(body, `<h2 class="sw-small sw-muted"`)
		if idxDateHeading >= 0 && idxH1 > idxDateHeading {
			t.Errorf("the <h1>Activity</h1> must appear before any date-heading\n%s", truncate(body))
		}
	}

	// The heading should be visible (not visually hidden).
	if strings.Contains(body, `<h2 class="sw-visually-hidden">Activity</h2>`) {
		t.Errorf("the overarching Activity heading on /activity should not be visually hidden\n%s", truncate(body))
	}
}

// TestActivityPageOverarchingHeadingEmptyState checks that the /activity page
// still has the layout h1 "Activity" even when there are no activity records,
// and that no duplicate <h2>Activity</h2> was introduced. (Acceptance 1)
func TestActivityPageOverarchingHeadingEmptyState(t *testing.T) {
	_, h := newApp(t)

	body := get(t, h, "/activity").Body.String()

	if !strings.Contains(body, ">Activity</h1>") {
		t.Errorf("the empty /activity page should still contain <h1>Activity</h1>\n%s", truncate(body))
	}

	// No duplicate heading in body content.
	if strings.Count(body, `<h2>Activity</h2>`) > 0 {
		t.Errorf("the empty /activity page should NOT contain a duplicate <h2>Activity</h2>\n%s", truncate(body))
	}

	if !strings.Contains(body, `<p class="sw-empty">`) || !strings.Contains(body, "Send a message") {
		t.Errorf("empty state should use <p class=\"sw-empty\"> and tell the person what creates activity\n%s", truncate(body))
	}
}
