package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// From the page, a person starts a new chat, goes back to an earlier one,
// and deletes one, from the menu at the top of the chat.
func TestAPersonMovesBetweenChats(t *testing.T) {
	a, h := newApp(t)
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{{Text: "Planned."}, {Text: "Noted."}}}, nil
	wantStatus(t, postForm(t, h, "/chat", url.Values{"message": {"plan the garden"}, "from": {"/chat"}}), http.StatusSeeOther)
	page := get(t, h, "/chat").Body.String()
	if !strings.Contains(page, `sw-chat__current">plan the garden<`) {
		t.Errorf("the menu names the current chat after what was first said\n%s", page)
	}
	first := a.Chat.Current()

	wantStatus(t, postForm(t, h, "/chat/new", url.Values{"from": {"/chat"}}), http.StatusSeeOther)
	page = get(t, h, "/chat").Body.String()
	if strings.Contains(page, "Planned.") || !strings.Contains(page, ">New chat<") {
		t.Errorf("a new chat shows none of the old messages\n%s", page)
	}
	wantStatus(t, postForm(t, h, "/chat", url.Values{"message": {"and the kitchen"}, "from": {"/chat"}}), http.StatusSeeOther)

	wantStatus(t, postForm(t, h, "/chat/open", url.Values{"id": {first}, "from": {"/chat"}}), http.StatusSeeOther)
	page = get(t, h, "/chat").Body.String()
	if !strings.Contains(page, "Planned.") || strings.Contains(page, "Noted.") {
		t.Errorf("opening the first chat brings its messages back and not the other's\n%s", page)
	}
	if !strings.Contains(page, `aria-label="Delete chat and the kitchen"`) {
		t.Error("every chat in the list can be deleted")
	}

	second := ""
	for _, c := range a.Chat.Conversations() {
		if c.ID != first {
			second = c.ID
		}
	}
	wantStatus(t, postForm(t, h, "/chat/delete", url.Values{"id": {second}, "from": {"/chat"}}), http.StatusSeeOther)
	if len(a.Chat.Conversations()) != 1 {
		t.Error("the deleted chat is gone")
	}
	if n, _ := a.Store.Count(chat.MessageType); n != 2 {
		t.Errorf("and its messages with it, leaving the first chat's two, got %d", n)
	}
}

// The chat is a block, so its menu can place the block: in a pane or the
// middle, and how wide. The move is logged as the person's.
func TestAPersonPlacesTheChat(t *testing.T) {
	a, h := newApp(t)
	page := get(t, h, "/").Body.String()
	if !strings.Contains(page, ">Place<") || !strings.Contains(page, `name="region" value="left"`) || !strings.Contains(page, `data-popout`) {
		t.Errorf("the chat on the canvas offers Place and Pop out\n%s", page)
	}
	blocks, _ := a.Store.List(chat.BlockType, store.ListOptions{})
	id := ""
	for _, b := range blocks {
		if b.Fields["component"] == chat.ComponentName {
			id = b.ID
		}
	}
	wantStatus(t, postForm(t, h, "/canvas/"+id+"/place", url.Values{"region": {"right"}, "from": {"/"}}), http.StatusSeeOther)
	blk, _ := a.Store.Get(chat.BlockType, id)
	if blk.Fields["region"] != "right" || blk.Fields["actor"] != "human" {
		t.Errorf("the chat moved to the right pane at the person's asking, got %v by %v", blk.Fields["region"], blk.Fields["actor"])
	}
	wantStatus(t, postForm(t, h, "/canvas/"+id+"/place", url.Values{"region": {"main"}, "span": {"6"}, "from": {"/"}}), http.StatusSeeOther)
	blk, _ = a.Store.Get(chat.BlockType, id)
	if blk.Fields["region"] != "main" || blk.Fields["span"] != int64(6) && blk.Fields["span"] != 6 && blk.Fields["span"] != float64(6) {
		t.Errorf("back to the middle, half wide, got %v %v", blk.Fields["region"], blk.Fields["span"])
	}
	if !strings.Contains(get(t, h, "/activity").Body.String(), "middle") {
		t.Error("the move is in the activity log")
	}
	page = get(t, h, "/chat").Body.String()
	if strings.Contains(page, ">Place<") || strings.Contains(page, "data-popout") {
		t.Error("the chat page itself is neither placed nor popped out")
	}
}
