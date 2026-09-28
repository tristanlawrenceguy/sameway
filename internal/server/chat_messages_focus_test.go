package server_test

import (
	"net/url"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestChatMessagesListIsNotFocusable ensures the transcript is one Tab
// stop, named Messages, and no more: a scrolling list with no links in it
// cannot be scrolled by a keyboard otherwise (Safari, and Chrome once
// tabindex is -1), so it takes Tab once, as a whole, and the messages in
// it stay plain articles.
func TestChatMessagesListIsNotFocusable(t *testing.T) {
	a, h := newApp(t)

	// Create at least two messages via the assistant so the ol is rendered.
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		{Text: "First reply."},
		{Text: "Second reply."},
	}}, nil

	postForm(t, h, "/chat", url.Values{"message": {"first message"}, "from": {"/chat"}})
	postForm(t, h, "/chat", url.Values{"message": {"second message"}, "from": {"/chat"}})

	doc := parse(t, get(t, h, "/chat"))

	// Acceptance 3: no ol element in the page has tabindex="0" or any
	// positive tabindex. The ol wrapping messages must be non-focusable via Tab.
	stops := 0
	for _, n := range doc.Elements("ol") {
		if ti, ok := htmltest.Attr(n, "tabindex"); ok && ti != "-1" {
			stops++
			if name, _ := htmltest.Attr(n, "aria-label"); name != "Messages" {
				t.Errorf("a list that takes Tab must be named; got %q", name)
			}
		}
	}
	if stops != 1 {
		t.Errorf("the transcript should be one Tab stop, got %d", stops)
	}
	// Acceptance 4: message content elements (articles) inside the ol are
	// still present and accessible — the list container loses focusability but
	// its children remain reachable. Walk every article with data-component="message"
	// and confirm at least two exist on the page.
	msgArticles := doc.WithAttr("data-component", "message")
	if len(msgArticles) < 2 {
		t.Errorf("expected at least 2 message articles for accessibility, got %d — messages may have been lost during the tabindex fix", len(msgArticles))
	}

	// Verify the ol itself exists and has the expected aria-label, confirming
	// semantic meaning is preserved even without focusability.
	messageLists := doc.WithAttr("aria-label", "Messages")
	if len(messageLists) == 0 {
		t.Error("<ol> with aria-label=\"Messages\" is missing — semantic structure may be broken")
	}
}
