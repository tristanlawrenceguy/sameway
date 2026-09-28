package server_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/look"
)

// Anything can go anywhere: the header and footer are regions like the
// panes, search starts in the header, and a block can be shown whole, in
// less room, or as a glyph with its name that opens the whole thing.
func TestAnythingCanGoAnywhereAtAnySize(t *testing.T) {
	a, h := newApp(t)
	page := get(t, h, "/").Body.String()
	if !strings.Contains(page, `class="sw-strip sw-strip--header"`) || !strings.Contains(page, `data-component="search"`) {
		t.Fatal("a new workspace has search in the header")
	}
	if !strings.Contains(page, `role="search"`) {
		t.Error("the search block is a search landmark")
	}

	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("add_component", map[string]any{"component": "link", "props": map[string]any{"href": "/activity", "label": "Activity"}, "region": "footer", "size": "compact"}),
		toolCall("add_component", map[string]any{"component": "calendar", "props": map[string]any{"month": "2026-09", "caption": "September"}, "region": "header", "size": "icon"}),
		{Text: "Done."},
	}}, nil
	postForm(t, h, "/chat", url.Values{"message": {"arrange it"}, "from": {"/"}})
	page = get(t, h, "/").Body.String()
	if !strings.Contains(page, `class="sw-strip sw-strip--footer"`) || !strings.Contains(page, `data-size="compact"`) {
		t.Error("a block can sit in the footer at compact size")
	}
	if !strings.Contains(page, `data-size="icon"`) || !strings.Contains(page, `aria-label="September"`) || !strings.Contains(page, `>▦<`) {
		t.Errorf("an icon-sized block is its glyph with its name, got %.300s", page[strings.Index(page, "sw-strip--header"):])
	}
	o, _ := look.Page(page)
	if len(o.Problems) != 0 {
		t.Errorf("the page with strips and an icon should read cleanly, got %v", o.Problems)
	}
	var icon *look.Control
	for i := range o.Controls {
		if o.Controls[i].Name == "September" {
			icon = &o.Controls[i]
		}
	}
	if icon == nil || icon.Kind != "link" || !strings.HasPrefix(icon.Href, "/canvas/") {
		t.Errorf("the icon is a link to the block's own page, got %+v", icon)
	}
}
