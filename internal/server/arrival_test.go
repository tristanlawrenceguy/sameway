package server_test

import (
	"net/url"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// Several changes in one turn arrive one after another, in the order they
// were made, so a person has time to take each in. The conversation block
// is the person's own tool and never arrives.
func TestChangesArriveInTheOrderTheyWereMade(t *testing.T) {
	a, h := newApp(t)
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("add_component", map[string]any{"component": "heading", "props": map[string]any{"text": "Shopping"}}),
		toolCall("add_component", map[string]any{"component": "list", "props": map[string]any{"items": []string{"milk"}}}),
		{Text: "Done."},
	}}, nil
	get(t, h, "/")
	postForm(t, h, "/chat", url.Values{"message": {"make a shopping list"}, "from": {"/"}})
	page := parse(t, get(t, h, "/"))
	order := map[string]string{}
	for _, b := range page.WithAttr("data-block-id", "") {
		c, _ := htmltest.Attr(b, "data-block-component")
		n, _ := htmltest.Attr(b, "data-arrival")
		order[c] = n
	}
	if order["heading"] != "1" || order["list"] != "2" {
		t.Errorf("the heading was added first and arrives first, then the list; got %v", order)
	}
	if order["chat"] != "" {
		t.Errorf("the conversation does not arrive, got %q", order["chat"])
	}
}
