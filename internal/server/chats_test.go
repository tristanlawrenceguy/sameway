package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
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

// TestChatBuildsCanvasForPersonAndAgent runs one conversation through the
// JSON API and checks the result shows up on the HTML page, in the block
// API, and can be removed from the page by a person.
func TestChatBuildsCanvasForPersonAndAgent(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	model := &scripted{steps: []*llm.Response{
		toolCall("add_component", map[string]any{"component": "table", "props": map[string]any{"caption": "Plan", "columns": []string{"Task", "When"}, "rows": [][]string{{"Write tests", "today"}}}}),
		{Text: "Added a plan table."},
	}}
	a.Chat.Provider, a.Chat.ProviderErr = model, nil

	wantStatus(t, get(t, h, "/"), http.StatusOK)

	rec := postJSON(t, h, http.MethodPost, "/api/chat", map[string]any{"message": "make a plan table"})
	wantStatus(t, rec, http.StatusOK)
	var reply struct {
		OK    bool
		Reply struct{ Fields map[string]any }
	}
	decode(t, rec, &reply)
	if !reply.OK || reply.Reply.Fields["content"] != "Added a plan table." {
		t.Fatalf("chat reply: %+v", reply)
	}

	var blocks struct {
		Records []struct {
			ID     string
			Fields map[string]any
		}
	}
	decode(t, get(t, h, "/api/block"), &blocks)
	tableID := ""
	for _, b := range blocks.Records {
		if b.Fields["component"] == "table" {
			tableID = b.ID
		}
	}
	if tableID == "" {
		t.Fatalf("block API has no table: %+v", blocks)
	}

	page := parse(t, get(t, h, "/"))
	if len(page.WithAttr("data-block-component", "table")) != 1 {
		t.Fatalf("canvas should render the table block once")
	}
	if !strings.Contains(htmltest.Text(page.Root), "Write tests") {
		t.Errorf("table content missing from the page")
	}

	del := postForm(t, h, "/canvas/"+tableID+"/delete", nil)
	wantStatus(t, del, http.StatusSeeOther)
	if len(parse(t, get(t, h, "/")).WithAttr("data-block-component", "table")) != 0 {
		t.Errorf("the table block should be gone after a person removes it")
	}

	wantStatus(t, postForm(t, h, "/chat/clear", nil), http.StatusSeeOther)
	if n := len(parse(t, get(t, h, "/")).WithAttr("data-component", "message")); n != 0 {
		t.Errorf("clear should remove messages, %d left", n)
	}
}

// TestChatFormWithoutModelRecordsAnError covers the person's path when no
// model is reachable: the failure lands in the transcript, not a 500.
func TestChatFormWithoutModelRecordsAnError(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	wantStatus(t, get(t, h, "/"), http.StatusOK)
	rec := postForm(t, h, "/chat", url.Values{"message": {"hello"}, "from": {"/"}})
	wantStatus(t, rec, http.StatusSeeOther)
	if !strings.HasPrefix(rec.Header().Get("Location"), "/#msg-") {
		t.Errorf("redirect should jump to the newest message, got %q", rec.Header().Get("Location"))
	}
	page := parse(t, get(t, h, "/"))
	errs := page.WithAttr("data-role", "error")
	if len(errs) != 1 || !strings.Contains(htmltest.Text(errs[0]), "no model configured") {
		t.Errorf("expected one error message explaining the missing model")
	}
	skips := page.WithAttr("class", "sw-skip")
	if len(skips) < 2 {
		t.Errorf("with messages present there should be a skip link to the latest one")
	}
	empty := postForm(t, h, "/chat", url.Values{"message": {"   "}})
	wantStatus(t, empty, http.StatusSeeOther)
	if n := len(parse(t, get(t, h, "/")).WithAttr("data-component", "message")); n != 2 {
		t.Errorf("a blank message must not be recorded; have %d messages", n)
	}
}
