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
		toolCall("create_record", map[string]any{"type": "note", "fields": map[string]any{"title": "Seeds to buy", "body": "Ten packets"}}),
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

// Undo all after the person changed part of one of the reply's changes
// again keeps what they did, and says so; the rest is taken back.
func TestUndoAllKeepsWhatWasChangedSince(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	seeds, _ := a.Store.Create("note", map[string]any{"title": "Seeds"})
	beds, _ := a.Store.Create("note", map[string]any{"title": "Beds"})
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("update_record", map[string]any{"type": "note", "id": seeds.ID, "fields": map[string]any{"title": "Seeds to buy", "body": "Ten packets"}}),
		toolCall("update_record", map[string]any{"type": "note", "id": beds.ID, "fields": map[string]any{"title": "Beds to dig"}}),
		{Text: "Done."},
	}}, nil
	postForm(t, h, "/chat", url.Values{"message": {"name them better"}, "from": {"/chat"}})
	a.Store.Update("note", seeds.ID, map[string]any{"title": "Seeds, by Friday"})
	page := get(t, h, "/chat").Body.String()
	form := regexp.MustCompile(`action="(/chat/[a-z0-9]+/undo)"`).FindStringSubmatch(page)
	if form == nil {
		t.Fatalf("the reply offers Undo all:\n%s", changesMade(page))
	}
	back := after(t, h, postForm(t, h, form[1], url.Values{"from": {"/chat"}})).Body.String()
	if !strings.Contains(back, "All 2 changes of that reply are taken back. Kept as changed since: title of Seeds, by Friday.") {
		t.Errorf("the outcome says what was kept:\n%s", truncate(back))
	}
	if got, _ := a.Store.Get("note", seeds.ID); got.Fields["title"] != "Seeds, by Friday" || got.Fields["body"] == "Ten packets" {
		t.Errorf("the title typed since is kept, the words written by the reply go: %v", got.Fields)
	}
	if got, _ := a.Store.Get("note", beds.ID); got.Fields["title"] != "Beds" {
		t.Errorf("the other is taken back: %v", got.Fields["title"])
	}
}
