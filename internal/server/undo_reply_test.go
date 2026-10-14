package server_test

import (
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A reply that made several changes offers to take them all back in one
// press, under its Changes made; pressed, every one is undone, and the
// person is told how many. A reply of one change has only its own Undo.
func TestAReplysChangesAreUndoneTogether(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "Seeds to buy"}}),
		toolCall("create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "Beds to dig"}}),
		toolCall("add_component", map[string]any{"component": "card", "props": map[string]any{"title": "Plan"}}),
		{Text: "Done."},
		toolCall("create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "One more"}}),
		{Text: "Done."},
	}}, nil
	postForm(t, h, "/chat", url.Values{"message": {"two notes and a card"}, "from": {"/chat"}})
	page := get(t, h, "/chat").Body.String()
	form := regexp.MustCompile(`<form class="sw-message__undo-all" method="post" action="(/chat/[a-z0-9]+/undo)">`).FindStringSubmatch(page)
	if form == nil || !strings.Contains(page, "Undo all 3<span") {
		t.Fatalf("the reply offers Undo all 3:\n%s", changesMade(page))
	}
	res := postForm(t, h, form[1], url.Values{"from": {"/chat"}})
	if res.Code >= 400 {
		t.Fatalf("undo all: %d %s", res.Code, res.Body.String())
	}
	if notes, _ := a.Store.List("note", store.ListOptions{}); len(notes) != 0 {
		t.Errorf("both notes are taken back, %d left", len(notes))
	}
	if blocks, _ := a.Store.List("block", store.ListOptions{}); len(blocks) != 0 {
		t.Errorf("the card is taken back, %d left", len(blocks))
	}

	postForm(t, h, "/chat", url.Values{"message": {"one more note"}, "from": {"/chat"}})
	if page := get(t, h, "/chat").Body.String(); strings.Contains(page[strings.LastIndex(page, "sw-message--assistant"):], "sw-message__undo-all") {
		t.Error("a reply of one change has only its own Undo")
	}
}
