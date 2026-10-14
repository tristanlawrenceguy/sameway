package server_test

import (
	"golang.org/x/net/html"
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// A tab with a long name is one line in the tab bar, cut as a title is
// anywhere one line holds it, and its own page is headed with the whole
// name. A name is at most 60 characters (schema/canvas.yaml), so a single
// long word is shown whole and wraps (the tabs component's long example,
// checked at phone width in the browser).
func TestALongTabNameIsCutInTheBarAndWholeOnItsPage(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	long := "Everything about the house move in March and the boxes"
	var move struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/canvas", map[string]any{"name": long}), &move)
	if over := postJSON(t, h, http.MethodPost, "/api/canvas", map[string]any{"name": strings.Repeat("a", 61)}); over.Code < 400 {
		t.Errorf("a name over 60 characters is refused: %d", over.Code)
	}

	// The tab bar's link; the page's recent changes may link to the tab too.
	var links []*html.Node
	for _, l := range parse(t, get(t, h, "/")).WithAttr("href", "/c/"+move.ID) {
		if inTabBar(l) {
			links = append(links, l)
		}
	}
	if len(links) != 1 {
		t.Fatalf("one tab links to the move, got %d", len(links))
	}
	if said := htmltest.Text(links[0]); said != "Everything about the house move in…" {
		t.Errorf("the tab bar cuts the name to a line: %q", said)
	}
	page := parse(t, get(t, h, "/c/"+move.ID))
	if h1 := page.Elements("h1"); len(h1) == 0 || !strings.Contains(htmltest.Text(h1[0]), long) {
		t.Errorf("the tab's own page is headed with its whole name")
	}
}

// inTabBar says a node sits in the tab bar.
func inTabBar(n *html.Node) bool {
	for p := n.Parent; p != nil; p = p.Parent {
		for _, a := range p.Attr {
			if a.Key == "aria-label" && a.Val == "Canvases" {
				return true
			}
		}
	}
	return false
}
