package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestSkipLinkTargetsLatestMessage verifies acceptance item 1: the "Skip to
// latest message" link on /chat has an href that targets the actual newest
// message article (the one with id="msg-<id>"). A person clicking it should
// be taken directly to their latest reply, not a dead anchor.
func TestSkipLinkTargetsLatestMessage(t *testing.T) {
	a, h := newApp(t)

	// Create two messages so we can identify the latest one.
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		{Text: "First reply."},
		{Text: "Second reply."},
	}}, nil

	postForm(t, h, "/chat", map[string][]string{"message": {"first"}, "from": {"/chat"}})
	postForm(t, h, "/chat", map[string][]string{"message": {"second"}, "from": {"/chat"}})

	doc := parse(t, get(t, h, "/chat"))

	// Find the skip link with label "Skip to latest message".
	skips := doc.WithAttr("class", "sw-skip")
	var target string
	for _, s := range skips {
		name, _ := htmltest.Attr(s, "aria-label")
		if name == "" {
			name = strings.TrimSpace(htmltest.Text(s))
		}
		if name == "Skip to latest message" {
			target, _ = htmltest.Attr(s, "href")
			break
		}
	}
	if target == "" {
		t.Fatal("no skip link with label 'Skip to latest message' found on /chat")
	}

	// The href must start with #msg- and point to an actual article.
	if !strings.HasPrefix(target, "#msg-") {
		t.Errorf("skip link href = %q; want #msg-<id>", target)
	}

	targetID := strings.TrimPrefix(target, "#")
	msgArticle := doc.ByID(targetID)
	if msgArticle == nil {
		t.Errorf("skip link targets id=%q but no element with that id exists on the page", targetID)
		return
	}

	// The targeted article must have data-component="message" to be a real message.
	comp, ok := htmltest.Attr(msgArticle, "data-component")
	if !ok || comp != "message" {
		t.Errorf("element with id=%q is not a message (data-component=%q)", targetID, comp)
	}

	// The targeted article must be focusable (tabindex="-1") so the browser can
	// place keyboard focus on it after anchor navigation — acceptance item 2.
	if ti, ok := htmltest.Attr(msgArticle, "tabindex"); !ok || ti != "-1" {
		t.Errorf("message article id=%q needs tabindex=\"-1\" for keyboard focus after skip-link navigation; got tabindex=%q (or missing)", targetID, ti)
	}
}

// TestSkipLinkAbsentWhenNoMessages verifies that when there are zero messages
// on /chat, the "Skip to latest message" skip link is not rendered — it has no
// target to point to.
func TestSkipLinkAbsentWhenNoMessages(t *testing.T) {
	_, h := newApp(t)

	doc := parse(t, get(t, h, "/chat"))

	skips := doc.WithAttr("class", "sw-skip")
	for _, s := range skips {
		name, _ := htmltest.Attr(s, "aria-label")
		if name == "" {
			name = strings.TrimSpace(htmltest.Text(s))
		}
		if name == "Skip to latest message" {
			href, _ := htmltest.Attr(s, "href")
			t.Errorf("skip link should not appear when there are no messages; found href=%q", href)
		}
	}

	msgs := doc.WithAttr("data-component", "message")
	if len(msgs) != 0 {
		t.Errorf("expected zero messages on fresh /chat, got %d", len(msgs))
	}
}
