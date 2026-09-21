package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// TestActivityPageHasOverarchingHeading checks that the /activity listing page
// renders an overarching <h2>Activity</h2> heading inside body content before
// any date-grouped headings. This gives screen reader users immediate context
// for what they are viewing, rather than jumping straight into dates like
// "Monday 5 January". (Acceptance 1)
func TestActivityPageHasOverarchingHeading(t *testing.T) {
	a, h := newApp(t)

	chat.Record(a.Store, "assistant", chat.Change{Action: "created", Component: "note", ID: "aaa1", Detail: "First note"})

	body := get(t, h, "/activity").Body.String()

	// The overarching heading must appear in the body.
	if !strings.Contains(body, `<h2>Activity</h2>`) {
		t.Errorf("the /activity page should contain an <h2>Activity</h2> heading in body content\n%s", truncate(body))
	}

	// It must come before any date-heading like "Monday 5 January".
	idxHeading := strings.Index(body, `<h2>Activity</h2>`)
	if idxHeading < 0 {
		return // already reported above
	}
	idxDateHeading := strings.Index(body, `<h2 class="sw-small sw-muted"`)
	if idxDateHeading >= 0 && idxHeading > idxDateHeading {
		t.Errorf("the overarching <h2>Activity</h2> must appear before any date-heading\n%s", truncate(body))
	}

	// The heading should be visible (no visually-hidden class).
	if strings.Contains(body, `<h2 class="sw-visually-hidden">Activity</h2>`) {
		t.Errorf("the overarching Activity heading on /activity should be visible, not visually hidden\n%s", truncate(body))
	}
}

// TestActivityPageOverarchingHeadingEmptyState checks that the /activity page
// shows an <h2>Activity</h2> heading even when there are no activity records,
// so screen readers still get context on the empty listing. (Acceptance 1)
func TestActivityPageOverarchingHeadingEmptyState(t *testing.T) {
	_, h := newApp(t)

	body := get(t, h, "/activity").Body.String()

	if !strings.Contains(body, `<h2>Activity</h2>`) {
		t.Errorf("the empty /activity page should still contain an <h2>Activity</h2> heading\n%s", truncate(body))
	}

	if !strings.Contains(body, `<p class="sw-empty">`) || !strings.Contains(body, "Send a message") {
		t.Errorf("empty state should use <p class=\"sw-empty\"> and tell the person what creates activity\n%s", truncate(body))
	}
}
