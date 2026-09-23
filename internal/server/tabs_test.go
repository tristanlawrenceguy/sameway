package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// Tabs are canvases: a second canvas is a second page of blocks with the
// tab bar linking them, the current one marked; a new tab opens on its own
// chat; what is said on a tab is built on that tab; and a block's own page
// leads back to the tab it lives on.
func TestTabsAreSeparateCanvases(t *testing.T) {
	a, h := newApp(t)
	if page := get(t, h, "/").Body.String(); strings.Contains(page, `aria-label="Canvases"`) {
		t.Error("with only Home there is no tab bar to show")
	}
	var garden struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/canvas", map[string]any{"name": "Garden"}), &garden)

	home := parse(t, get(t, h, "/"))
	tabs := home.WithAttr("aria-label", "Canvases")
	if len(tabs) != 1 {
		t.Fatalf("with two canvases the page should carry one tab bar, got %d", len(tabs))
	}
	if links := home.WithAttr("href", "/c/"+garden.ID); len(links) == 0 {
		t.Error("the tab bar should link to the Garden canvas")
	}
	current := false
	for _, l := range home.WithAttr("href", "/") {
		if cur, _ := htmltest.Attr(l, "aria-current"); cur == "page" {
			current = true
		}
	}
	if !current {
		t.Error("Home should be marked as the current tab")
	}

	// The new tab opens on its own chat, and shows nothing from Home.
	postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "heading", "props": map[string]any{"text": "Only on Home"}})
	rec := get(t, h, "/c/"+garden.ID)
	wantStatus(t, rec, http.StatusOK)
	tab := rec.Body.String()
	if strings.Contains(tab, "Only on Home") || !strings.Contains(tab, `data-block-component="chat"`) {
		t.Error("a tab shows its own blocks only, and opens on a chat")
	}
	if !strings.Contains(tab, `value="/c/`+garden.ID+`"`) {
		t.Error("the chat on a tab should post from that tab")
	}
	wantStatus(t, get(t, h, "/c/nope"), http.StatusNotFound)

	// What is said on the tab is built on the tab, and the person stays there.
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("add_component", map[string]any{"component": "card", "props": map[string]any{"title": "Beds to dig"}}),
		{Text: "Added the beds."},
	}}, nil
	sent := postForm(t, h, "/chat", url.Values{"message": {"add the beds"}, "from": {"/c/" + garden.ID}})
	wantStatus(t, sent, http.StatusSeeOther)
	if loc := sent.Header().Get("Location"); !strings.HasPrefix(loc, "/c/"+garden.ID) {
		t.Errorf("after talking on a tab the person should stay on it, got %q", loc)
	}
	// The rendered card, not the transcript, which every tab's chat shares.
	if page := get(t, h, "/c/"+garden.ID).Body.String(); !strings.Contains(page, ">Beds to dig</h2>") {
		t.Error("the block should be on the Garden tab")
	}
	if page := get(t, h, "/").Body.String(); strings.Contains(page, ">Beds to dig</h2>") {
		t.Error("and not on Home")
	}

	// A block's own page leads back to its tab.
	var blocks struct {
		Records []struct {
			ID     string
			Fields map[string]any
		}
	}
	decode(t, get(t, h, "/api/block"), &blocks)
	for _, b := range blocks.Records {
		if b.Fields["canvas"] == garden.ID && b.Fields["component"] == "card" {
			focus := get(t, h, "/canvas/"+b.ID).Body.String()
			if !strings.Contains(focus, `href="/c/`+garden.ID+`"`) {
				t.Error("the way back from an expanded block is its own tab")
			}
		}
	}
}
