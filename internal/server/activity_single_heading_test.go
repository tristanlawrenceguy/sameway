package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// TestActivityPageHasSingleActivityHeading checks that the /activity listing
// page has exactly one heading element containing the word "Activity", so a
// screen reader user does not hear it twice. The h1 from the layout template
// is the single authoritative heading for this page. (Acceptance 1, 2)
func TestActivityPageHasSingleActivityHeading(t *testing.T) {
	a, h := newApp(t)

	chat.Record(a.Store, "assistant", chat.Change{Action: "created", Component: "note", ID: "aaa1", Detail: "First note"})

	rec := get(t, h, "/activity")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	// The body content must not contain <h2>Activity</h2>.
	if strings.Contains(body, `<h2>Activity</h2>`) {
		t.Errorf("the /activity page body should NOT contain <h2>Activity</h2>; only the layout h1 provides the heading\n%s", truncate(body))
	}

	// The h1 from layout says "Activity"; no other visible heading should repeat it.
	h2Activity := strings.Count(body, `<h2>Activity</h2>`)
	if h2Activity > 0 {
		t.Errorf("expected 0 <h2>Activity</h2> in body content, got %d (the layout h1 is the only heading)\n%s", h2Activity, truncate(body))
	}

	// The page must still have its h1 with "Activity".
	if !strings.Contains(body, `<h1`) || !strings.Contains(body, ">Activity</h1>") {
		t.Errorf("the /activity page should contain <h1>Activity</h1> from the layout\n%s", truncate(body))
	}

	// The h2 date headings (e.g. "Monday 5 January") must still exist after the
	// overarching heading, confirming we only removed the duplicate.
	if !strings.Contains(body, `<h2 class="sw-small sw-muted"`) {
		t.Errorf("the /activity page should still contain date-grouped h2 headings\n%s", truncate(body))
	}

	// The individual entry h3s must still exist.
	if strings.Count(body, "<h3") != 1 {
		t.Errorf("expected exactly one <h3> for the single activity entry, got %d\n%s", strings.Count(body, "<h3"), truncate(body))
	}
}

// TestActivityPageEmptyStateSingleHeading checks that even with no activity
// records, the /activity page has exactly one heading "Activity" (from the
// layout h1) and no duplicate body heading. (Acceptance 1, 2)
func TestActivityPageEmptyStateSingleHeading(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/activity")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	// The body content must not contain <h2>Activity</h2>.
	if strings.Contains(body, `<h2>Activity</h2>`) {
		t.Errorf("the empty /activity page should NOT contain <h2>Activity</h2>\n%s", truncate(body))
	}

	// The h1 "Activity" from the layout must still be present.
	if !strings.Contains(body, ">Activity</h1>") {
		t.Errorf("the empty /activity page should contain <h1>Activity</h1> from the layout\n%s", truncate(body))
	}

	// Empty-state paragraph must still be present.
	if !strings.Contains(body, `<p class="sw-empty">`) || !strings.Contains(body, "Send a message") {
		t.Errorf("empty state should use <p class=\"sw-empty\"> and tell the person what creates activity\n%s", truncate(body))
	}

	// No h2 headings at all when there are no activities.
	if strings.Count(body, `<h2`) > 0 {
		t.Errorf("the empty /activity page body should have no <h2> elements, got %d\n%s", strings.Count(body, "<h2"), truncate(body))
	}
}
