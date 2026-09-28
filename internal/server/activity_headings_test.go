package server_test

import (
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// TestActivityPageHasIndividualHeadings checks that each activity entry on
// /activity is wrapped in its own <h3> element containing the summary text,
// so a screen reader user can jump directly between activities.
func TestActivityPageHasIndividualHeadings(t *testing.T) {
	a, h := newApp(t)

	// Seed two distinct activity entries via chat.Record (the same way chat does).
	chat.Record(a.Store, "assistant", chat.Change{Action: "created", Component: "note", ID: "aaa1", Detail: "First note", Href: "/t/note/aaa1"})
	chat.Record(a.Store, "assistant", chat.Change{Action: "created", Component: "note", ID: "bbb2", Detail: "Second note", Href: "/t/note/bbb2"})

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
