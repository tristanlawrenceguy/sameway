package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// TestNoteDetailPageRecentActivityHasH3Headings checks that the note detail page's
// recent activity section wraps each entry in an <h3 class="sw-event__heading"> so a
// screen reader user can jump between activities.
func TestNoteDetailPageRecentActivityHasH3Headings(t *testing.T) {
	a, h := newApp(t)

	// Create a real note so the detail page exists and has content.
	noteRec, err := a.Store.Create("note", map[string]any{"title": "Test Note"})
	if err != nil {
		t.Fatal(err)
	}

	// Seed two distinct activity entries via chat.Record (the same way chat does).
	chat.Record(a.Store, "assistant", chat.Change{Action: "created", Component: "note", ID: noteRec.ID, Detail: "Test Note"})
	chat.Record(a.Store, "assistant", chat.Change{Action: "updated", Component: "note", ID: noteRec.ID, Detail: "Added content"})

	body := get(t, h, "/t/note/"+noteRec.ID).Body.String()

	// Acceptance 1: each entry is wrapped in its own <h3 class="sw-event__heading">.
	if !strings.Contains(body, `<h3 class="sw-event__heading"`) {
		t.Errorf("recent activity on note detail page should wrap entries in <h3 class=\"sw-event__heading\">\n%s", truncate(body))
	}

	// Acceptance 2: the h3 text contains the full summary (e.g., "Assistant created note Test Note").
	if !strings.Contains(body, "<h3") || !strings.Contains(body, "Assistant created note Test Note") {
		t.Errorf("the <h3> should contain the full activity summary 'Assistant created note Test Note'\n%s", truncate(body))
	}

	// Two entries = two h3s.
	h3Count := strings.Count(body, `<h3 class="sw-event__heading"`)
	if h3Count != 2 {
		t.Errorf("expected 2 <h3 class=\"sw-event__heading\"> elements for 2 activities on note detail page, got %d\n%s", h3Count, truncate(body))
	}

	// Each h3 should appear before its corresponding event component.
	idxH3 := strings.Index(body, `<h3 class="sw-event__heading"`)
	if idxH3 >= 0 {
		idxEvent := strings.Index(body[idxH3:], `data-component="event"`)
		if idxEvent < 0 {
			t.Errorf("the <h3> should precede the event component for its entry\n%s", truncate(body))
		}
	}

	// Acceptance 5: no empty <h3 class="sw-event__heading"> elements.
	if strings.Contains(body, `<h3 class="sw-event__heading"></h3>`) {
		t.Error("no empty <h3 class=\"sw-event__heading\"> should appear on note detail page")
	}
}

// TestNoteDetailPageRecentActivitySummaryFromHeadingText checks that the heading text
// comes from the summary field rather than just repeating the action verb.
func TestNoteDetailPageRecentActivitySummaryFromHeadingText(t *testing.T) {
	a, h := newApp(t)

	// Create a real note so we have something to visit on its detail page.
	noteRec, err := a.Store.Create("note", map[string]any{"title": "My Note"})
	if err != nil {
		t.Fatal(err)
	}

	// Seed an assistant activity about this note, with a whole summary.
	chat.Record(a.Store, "assistant", chat.Change{Action: "updated", Component: "note", ID: noteRec.ID, Detail: "Shopping"})

	body := get(t, h, "/t/note/"+noteRec.ID).Body.String()

	// The summary should be in the heading.
	if !strings.Contains(body, "<h3") || !strings.Contains(body, "Assistant updated note Shopping") {
		t.Errorf("the <h3> should contain the full summary 'Assistant updated note Shopping'\n%s", truncate(body))
	}

	// The heading must include both actor and action and target — not just "added".
	if strings.Contains(body, "<h3>added</h3>") {
		t.Error("the <h3> should contain the full summary, not just the verb")
	}
}

// TestNoteDetailPageRecentActivityEmptyStateHasNoHeading checks that when there are no
// activities on a note detail page, it does not render an orphaned <h3>.
func TestNoteDetailPageRecentActivityEmptyStateHasNoHeading(t *testing.T) {
	a, h := newApp(t)

	// Create a note so we have something to visit.
	rec, err := a.Store.Create("note", map[string]any{"title": "A note"})
	if err != nil {
		t.Fatal(err)
	}
	body := get(t, h, "/t/note/"+rec.ID).Body.String()

	// No <h3 class="sw-event__heading"> elements when there are no activities.
	h3Count := strings.Count(body, `<h3 class="sw-event__heading"`)
	if h3Count != 0 {
		t.Errorf("empty note detail page should have 0 <h3 class=\"sw-event__heading\"> elements, got %d\n%s", h3Count, truncate(body))
	}

	// The title of the note is still present.
	if !strings.Contains(body, "A note") {
		t.Errorf("note detail page should show the note title\n%s", truncate(body))
	}
}
