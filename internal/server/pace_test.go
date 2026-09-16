package server_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A person sets how changes arrive by asking, not in a settings page: the
// assistant's set_pace lands on the root element for the motion rules to
// read. The page invites questions, and carries the script that lets a
// person who has caught up show everything at once.
func TestPaceIsSetByAskingAndReadByThePage(t *testing.T) {
	a, h := newApp(t)
	if page := get(t, h, "/").Body.String(); !strings.Contains(page, `data-pace="calm"`) {
		t.Error("the default pace is calm and is on the root element")
	}
	if page := get(t, h, "/chat").Body.String(); !strings.Contains(page, "ask what something on the page is") {
		t.Error("the composer should invite questions about the page")
	}
	if js := get(t, h, "/design/sameway.js").Body.String(); !strings.Contains(js, "data-show-all") {
		t.Error("every page should carry the arrival script")
	}

	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("set_pace", map[string]any{"pace": "quick"}),
		{Text: "Faster from now on."},
	}}, nil
	postForm(t, h, "/chat", url.Values{"message": {"a bit faster please"}, "from": {"/"}})
	if page := get(t, h, "/").Body.String(); !strings.Contains(page, `data-pace="quick"`) {
		t.Error("the pace the assistant set should be on the page")
	}
	if a.Workspace.Config.UI.Pace != "quick" {
		t.Errorf("the workspace should remember the pace, got %q", a.Workspace.Config.UI.Pace)
	}
	if !logged(t, h, "Assistant set pace quick") {
		t.Error("setting the pace is a change like any other, in the log")
	}
}
