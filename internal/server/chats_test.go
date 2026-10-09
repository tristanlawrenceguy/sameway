package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// From the page, a person starts a new chat, goes back to an earlier one,
// and deletes one, from the menu at the top of the chat.
func TestAPersonMovesBetweenChats(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{{Text: "Planned."}, {Text: "Noted."}}}, nil
	wantStatus(t, postForm(t, h, "/chat", url.Values{"message": {"plan the garden"}, "from": {"/chat"}}), http.StatusSeeOther)
	page := get(t, h, "/chat").Body.String()
	if !strings.Contains(page, `sw-chat__name">plan the garden<`) {
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
	if !strings.Contains(page, `>Delete<span class="sw-visually-hidden"> chat and the kitchen</span></button>`) {
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
	if n, _ := a.Store.Count(records.MessageType); n != 2 {
		t.Errorf("and its messages with it, leaving the first chat's two, got %d", n)
	}
}

// Where the chat sits is the assistant's to arrange, as for any block: a
// person asks, and the chat's menu has no Place of its own. Pop out stays
// on the canvas, and the chat page itself has neither.
func TestTheChatIsPlacedByAsking(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	page := get(t, h, "/").Body.String()
	if strings.Contains(page, ">Place<") || strings.Contains(page, "/place\"") || !strings.Contains(page, `data-popout`) {
		t.Errorf("the chat on the canvas offers Pop out and no Place\n%s", page)
	}
	page = get(t, h, "/chat").Body.String()
	if strings.Contains(page, "data-popout") {
		t.Error("the chat page itself is not popped out")
	}
}
