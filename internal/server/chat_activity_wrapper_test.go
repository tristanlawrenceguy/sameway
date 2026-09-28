package server_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// TestChatActivityHasSwActivityWrapper checks that the Activity section on
// /chat is wrapped in <div class="sw-activity"> so JS settle() can find it
// and update in-place instead of creating a duplicate. (Acceptance 1.)
func TestChatActivityHasSwActivityWrapper(t *testing.T) {
	a, h := newApp(t)

	chat.Record(a.Store, "assistant", chat.Change{Action: "created", Component: "note", ID: "aaa1", Detail: "First note"})

	body := get(t, h, "/chat").Body.String()

	if !strings.Contains(body, `<div class="sw-activity">`) {
		t.Errorf("/chat should wrap Activity in <div class=\"sw-activity\"> (missing opening tag)\n%s", truncate(body))
	}

	// The closing </div> must come after the wrapper's opening tag so it actually
	// encloses the activity content.  A HasSuffix check would be wrong because the
	// full page ends with </html>, not </div>.
	openIdx := strings.Index(body, `<div class="sw-activity">`)
	closeIdx := strings.LastIndex(body, `</div>`)
	if openIdx < 0 || closeIdx < 0 || closeIdx <= openIdx {
		t.Errorf("/chat should close the .sw-activity wrapper with </div>\n%s", truncate(body))
	}

	// The opening tag must come before the Activity heading so the wrapper
	// actually encloses it.
	h2Idx := strings.Index(body, `<h2 class="sw-visually-hidden">Activity</h2>`)
	if openIdx < 0 || h2Idx < 0 || openIdx > h2Idx {
		t.Errorf("the .sw-activity wrapper must appear before the Activity heading\n%s", truncate(body))
	}
}

// TestChatActivityWrapperIsEmptyWhenNoActivity checks that when there are no
// activity records, /chat does not render a stray empty <div class="sw-activity">.
func TestChatActivityWrapperIsEmptyWhenNoActivity(t *testing.T) {
	a, h := newApp(t)

	body := get(t, h, "/chat").Body.String()

	if strings.Contains(body, `<div class="sw-activity">`) {
		t.Errorf("/chat with no activity should not render a .sw-activity wrapper\n%s", truncate(body))
	}
	_ = a // ensure app is created so the workspace is properly initialised
}

// TestChatActivityWrapperAfterModelAction checks that after the model makes a
// change through chat.Send, the page still has the .sw-activity wrapper.
func TestChatActivityWrapperAfterModelAction(t *testing.T) {
	a, h := newApp(t)

	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("add_component", map[string]any{"component": "card", "props": map[string]any{"title": "Shopping"}}),
		{Text: "Done."},
	}}, nil

	postForm(t, h, "/chat", url.Values{"message": {"add a card"}, "from": {"/"}})

	body := get(t, h, "/chat").Body.String()

	if !strings.Contains(body, `<div class="sw-activity">`) {
		t.Errorf("/chat after model action should wrap Activity in <div class=\"sw-activity\">\n%s", truncate(body))
	}
}
