package server_test

import (
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// The activity page's headings: one for the page, one for the log, one
// for each entry, said by its summary, and none for a log with nothing in it.

// TestActivityPageHasIndividualHeadings checks that each activity entry on
// /activity is wrapped in its own <h3> element containing the summary text,
// so a screen reader user can jump directly between activities.
func TestActivityPageHasIndividualHeadings(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	// Seed two distinct activity entries via records.Record (the same way chat does).
	records.Record(a.Store, "assistant", records.Change{Action: "created", Component: "note", ID: "aaa1", Detail: "First note", Href: "/t/note/aaa1"})
	records.Record(a.Store, "assistant", records.Change{Action: "created", Component: "note", ID: "bbb2", Detail: "Second note", Href: "/t/note/bbb2"})

	body := get(t, h, "/activity").Body.String()

	// Acceptance 1: each entry is wrapped in its own <h3> element.
	if !strings.Contains(body, "<h3") {
		t.Errorf("each activity entry should be wrapped in an <h3> element\n%s", truncate(body))
	}

	// Acceptance 2: the h3 text contains the full summary (e.g., "Assistant created note First note"), not just the verb.
	if !anyH3Says(body, "Assistant created note First note") {
		t.Errorf("the <h3> should contain the full activity summary 'Assistant created note First note'\n%s", truncate(body))
	}
	if !anyH3Says(body, "Assistant created note Second note") {
		t.Errorf("the second entry <h3> should contain 'Assistant created note Second note'\n%s", truncate(body))
	}

	// Acceptance 3: there is one h3 per activity entry (two entries = two h3s).
	h3Count := strings.Count(body, "<h3")
	if h3Count != 2 {
		t.Errorf("expected 2 <h3> elements for 2 activities, got %d\n%s", h3Count, truncate(body))
	}

	// Each h3 is the event component's own sentence, said once: it sits
	// inside the entry's event component rather than repeating it above.
	if strings.Count(body, `<li><div class="sw-event" data-component="event"`) != 2 || strings.Count(body, eventHeading) != 2 {
		t.Errorf("each entry should be an event component whose sentence is its <h3>\n%s", truncate(body))
	}
}

// TestActivityPageHeadingUsesSummaryNotVerb checks that the heading text
// comes from the summary field (e.g., "Assistant created note X") rather
// than just repeating the action verb.
func TestActivityPageHeadingUsesSummaryNotVerb(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	// Create a card block via chat so we get an assistant activity with a
	// summary like "Assistant added card Shopping".
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("add_component", map[string]any{"component": "card", "props": map[string]any{"title": "Shopping"}}),
		{Text: "Done."},
	}}, nil
	postForm(t, h, "/chat", url.Values{"message": {"add a card"}, "from": {"/"}})

	body := get(t, h, "/activity").Body.String()

	// The summary for an assistant-added card is "Assistant added card Shopping".
	if !anyH3Says(body, "Assistant added card Shopping") {
		t.Errorf("the <h3> should contain the full summary 'Assistant added card Shopping'\n%s", truncate(body))
	}

	// The heading must include both actor and action and target — not just "added".
	if slices.Contains(h3Said(body), "added") {
		t.Error("the <h3> should contain the full summary, not just the verb")
	}
}

// TestActivityPageEmptyStateHasNoHeading checks that when there are no
// activities, the page does not render an orphaned <h3> — only a paragraph
// saying nothing has happened yet.
func TestActivityPageEmptyStateHasNoHeading(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	body := get(t, h, "/activity").Body.String()

	if !strings.Contains(body, `data-component="empty"`) {
		t.Errorf("empty state should use the empty component\n%s", truncate(body))
	}

	// No <h3> elements at all when there are no activities.
	h3Count := strings.Count(body, "<h3")
	if h3Count != 0 {
		t.Errorf("empty activity page should have 0 <h3> elements, got %d\n%s", h3Count, truncate(body))
	}

	// The empty state tells the person what creates activity.
	if !strings.Contains(body, "Send a message") || !strings.Contains(body, "canvas") {
		t.Errorf("empty state should tell the person what creates activity:\n%s", truncate(body))
	}
}

// TestActivityPageHasOverarchingHeading checks that the /activity listing page
// has exactly one visible heading "Activity" — the h1 from the layout template.
// The duplicate <h2>Activity</h2> in body content was removed to avoid screen
// reader duplication. Date-grouped headings come after, and no other visible
// heading repeats "Activity". (Acceptance 1)
func TestActivityPageHasOverarchingHeading(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	records.Record(a.Store, "assistant", records.Change{Action: "created", Component: "note", ID: "aaa1", Detail: "First note"})

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
	t.Parallel()
	_, h := newApp(t)

	body := get(t, h, "/activity").Body.String()

	if !strings.Contains(body, ">Activity</h1>") {
		t.Errorf("the empty /activity page should still contain <h1>Activity</h1>\n%s", truncate(body))
	}

	// No duplicate heading in body content.
	if strings.Count(body, `<h2>Activity</h2>`) > 0 {
		t.Errorf("the empty /activity page should NOT contain a duplicate <h2>Activity</h2>\n%s", truncate(body))
	}

	if !strings.Contains(body, `data-component="empty"`) || !strings.Contains(body, "Send a message") {
		t.Errorf("empty state should use the empty component and tell the person what creates activity\n%s", truncate(body))
	}
}

// TestActivityPageHasSingleActivityHeading checks that the /activity listing
// page has exactly one heading element containing the word "Activity", so a
// screen reader user does not hear it twice. The h1 from the layout template
// is the single authoritative heading for this page. (Acceptance 1, 2)
func TestActivityPageHasSingleActivityHeading(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	records.Record(a.Store, "assistant", records.Change{Action: "created", Component: "note", ID: "aaa1", Detail: "First note"})

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
	t.Parallel()
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
	if !strings.Contains(body, `data-component="empty"`) || !strings.Contains(body, "Send a message") {
		t.Errorf("empty state should use the empty component and tell the person what creates activity\n%s", truncate(body))
	}

	// No h2 headings at all when there are no activities.
	if strings.Count(body, `<h2`) > 0 {
		t.Errorf("the empty /activity page body should have no <h2> elements, got %d\n%s", strings.Count(body, "<h2"), truncate(body))
	}
}
