package server_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// TestChatPageRecentActivityHasH3Headings checks that the /chat page's recent
// activity section wraps each entry in an <h3 class="sw-event__heading"> so a
// screen reader user can jump between activities.
func TestChatPageRecentActivityHasH3Headings(t *testing.T) {
	a, h := newApp(t)

	// Seed two distinct activity entries via chat.Record (the same way chat does).
	chat.Record(a.Store, "assistant", chat.Change{Action: "created", Component: "note", ID: "aaa1", Detail: "First note", Href: "/t/note/aaa1"})
	chat.Record(a.Store, "assistant", chat.Change{Action: "created", Component: "note", ID: "bbb2", Detail: "Second note", Href: "/t/note/bbb2"})

	body := get(t, h, "/chat").Body.String()

	// Acceptance 3: each entry is wrapped in its own <h3 class="sw-event__heading">.
	if !strings.Contains(body, `<h3 class="sw-event__heading"`) {
		t.Errorf("recent activity on /chat should wrap entries in <h3 class=\"sw-event__heading\">\n%s", truncate(body))
	}

	// Acceptance 4: the h3 text contains the full summary (e.g., "Assistant created note First note").
	if !strings.Contains(body, "<h3") || !strings.Contains(body, "Assistant created note First note") {
		t.Errorf("the <h3> should contain the full activity summary 'Assistant created note First note'\n%s", truncate(body))
	}

	// Two entries = two h3s.
	h3Count := strings.Count(body, `<h3 class="sw-event__heading"`)
	if h3Count != 2 {
		t.Errorf("expected 2 <h3 class=\"sw-event__heading\"> elements for 2 activities on /chat, got %d\n%s", h3Count, truncate(body))
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
		t.Error("no empty <h3 class=\"sw-event__heading\"> should appear on /chat")
	}
}

// TestChatPageRecentActivitySummaryFromHeadingText checks that the heading text
// comes from the summary field (e.g., "Assistant added card Shopping") rather
// than just repeating the action verb.
func TestChatPageRecentActivitySummaryFromHeadingText(t *testing.T) {
	a, h := newApp(t)

	// Create a card block via chat so we get an assistant activity with a
	// summary like "Assistant added card Shopping".
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("add_component", map[string]any{"component": "card", "props": map[string]any{"title": "Shopping"}}),
		{Text: "Done."},
	}}, nil
	postForm(t, h, "/chat", url.Values{"message": {"add a card"}, "from": {"/"}})

	body := get(t, h, "/chat").Body.String()

	// The summary for an assistant-added card is "Assistant added card Shopping".
	if !strings.Contains(body, "<h3") || !strings.Contains(body, "Assistant added card Shopping") {
		t.Errorf("the <h3> should contain the full summary 'Assistant added card Shopping'\n%s", truncate(body))
	}

	// The heading must include both actor and action and target — not just "added".
	if strings.Contains(body, "<h3>added</h3>") {
		t.Error("the <h3> should contain the full summary, not just the verb")
	}
}

// TestChatPageRecentActivityEmptyStateHasNoHeading checks that when there are no
// activities, /chat does not render an orphaned <h3>.
func TestChatPageRecentActivityEmptyStateHasNoHeading(t *testing.T) {
	_, h := newApp(t)

	body := get(t, h, "/chat").Body.String()

	// No <h3 class="sw-event__heading"> elements when there are no activities.
	h3Count := strings.Count(body, `<h3 class="sw-event__heading"`)
	if h3Count != 0 {
		t.Errorf("empty /chat should have 0 <h3 class=\"sw-event__heading\"> elements, got %d\n%s", h3Count, truncate(body))
	}

	// The "no model configured" notice is still present.
	if !strings.Contains(body, "Connect the assistant to an AI model") {
		t.Errorf("empty /chat should say the assistant needs a model\n%s", truncate(body))
	}
}
