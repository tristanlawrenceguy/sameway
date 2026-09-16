package server_test

import (
	"net/url"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestChatMessagesListIsNotFocusable ensures the ol element wrapping message
// articles on /chat does not create an extra focus stop. A keyboard user tabbing
// from navigation to controls should not land on the entire messages list as one
// unit — only interactive elements (textarea, buttons) and individual messages
// with explicit links should be reachable. This pins down acceptance items 3
// and 4 of task: no ol has tabindex="0", message articles remain accessible.
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
	for _, n := range doc.Elements("ol") {
		if ti, ok := htmltest.Attr(n, "tabindex"); ok && ti != "-1" {
			t.Errorf("<ol> has tabindex=%q — it becomes a focus stop via Tab; remove or set to -1", ti)
		}
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
