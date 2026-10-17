package server_test

import (
	"golang.org/x/net/html"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestASearchResultShowsItsWholeTitle: a result is recognised by its title,
// so the whole of it is shown, a heading under Results, not cut to six
// words with the rest in a tooltip a keyboard or a finger cannot reach.
func TestASearchResultShowsItsWholeTitle(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	title := "Ask the fontanero about the cistern upstairs before Friday afternoon"
	a.Store.Create("note", map[string]any{"title": title})
	doc := parse(t, get(t, h, "/search?q=fontanero"))
	var found bool
	for _, node := range doc.Elements("h3") {
		if class, _ := htmltest.Attr(node, "class"); class != "sw-card__title" {
			continue
		}
		found = true
		if got := strings.TrimSpace(htmltest.VisibleText(node)); got != title {
			t.Errorf("the result should show its whole title, got %q", got)
		}
		// The accessible name comes from the link's aria-label now.
		var aNode *html.Node
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && c.Data == "a" {
				aNode = c
				break
			}
		}
		if aNode != nil {
			for _, a := range aNode.Attr {
				if a.Key == "aria-label" && a.Val != title {
					t.Errorf("the result's heading aria-label should be the title for a screen reader, got %q", a.Val)
				}
			}
		}
		for _, a := range node.FirstChild.Attr {
			if a.Key == "title" {
				t.Errorf("the result link should carry no tooltip")
			}
		}
	}
	if !found {
		t.Errorf("a result is an h3 card title under the Results h2")
	}
}
