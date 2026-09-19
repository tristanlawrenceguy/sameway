package server_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// TestNoDuplicateActivityEntries seeds two activity entries via chat.Record(),
// GETs /chat, and verifies each entry appears exactly once — no duplicate
// text+timestamp pairs in the Activity section. (Backlog 0371; acceptance 2.)
func TestNoDuplicateActivityEntries(t *testing.T) {
	a, h := newApp(t)

	chat.Record(a.Store, "assistant", chat.Change{Action: "created", Component: "note", ID: "aaa1", Detail: "First note"})
	chat.Record(a.Store, "assistant", chat.Change{Action: "updated", Component: "note", ID: "aaa1", Detail: "Updated content"})

	body := get(t, h, "/chat").Body.String()

	for _, summary := range []string{"Assistant created note First note", "Assistant updated note Updated content"} {
		count := strings.Count(body, summary)
		if count != 1 {
			t.Errorf("activity entry %q should appear exactly once on /chat, got %d\n%s", summary, count, truncate(body))
		}
	}

	// Also check for duplicate <li> blocks with the same h3 heading text.
	events := strings.Split(body, `<li><h3 class="sw-event__heading"`)
	if len(events) > 1 {
		events = events[1:]
		type eventKey struct{ text string }
		var seen []eventKey
		for _, evt := range events {
			idxEnd := strings.Index(evt, `</h3>`)
			if idxEnd < 0 {
				continue
			}
			text := evt[:idxEnd]
			key := eventKey{text: text}
			for _, s := range seen {
				if s == key {
					t.Errorf("duplicate activity entry found in recent activity list:\n%q\n%s", text, truncate(body))
				}
			}
			seen = append(seen, key)
		}
	}

	// Also verify there is exactly one <h2 class="sw-visually-hidden">Activity</h2> heading.
	h2Count := strings.Count(body, `<h2 class="sw-visually-hidden">Activity</h2>`)
	if h2Count != 1 {
		t.Errorf("expected exactly 1 <h2 class=\"sw-visually-hidden\">Activity</h2> on /chat, got %d\n%s", h2Count, body)
	}

	// No visible (non-visually hidden) Activity heading should appear.
	if strings.Contains(body, `<h2>Activity</h2>`) {
		t.Errorf("no visible <h2>Activity</h2> should appear on /chat\n%s", body)
	}
}

// TestNoDuplicateActivityEntriesAfterModelAction seeds data via a simulated
// model action through chat.Send, then verifies no duplicate entries appear.
func TestNoDuplicateActivityEntriesAfterModelAction(t *testing.T) {
	a, h := newApp(t)

	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("add_component", map[string]any{"component": "card", "props": map[string]any{"title": "Shopping"}}),
		{Text: "Done."},
	}}, nil

	postForm(t, h, "/chat", url.Values{"message": {"add a card"}, "from": {"/"}})

	body := get(t, h, "/chat").Body.String()

	cardSummary := "Assistant added card Shopping"
	count := strings.Count(body, cardSummary)
	if count != 1 {
		t.Errorf("activity entry %q should appear exactly once on /chat after model action, got %d\n%s", cardSummary, count, truncate(body))
	}
}
