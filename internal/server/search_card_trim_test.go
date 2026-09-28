package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestASearchResultShowsItsWholeTitle: a result is recognised by its title,
// so the whole of it is shown, a heading under Results, not cut to six
// words with the rest in a tooltip a keyboard or a finger cannot reach.
func TestASearchResultShowsItsWholeTitle(t *testing.T) {
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
		if got := strings.TrimSpace(htmltest.Text(node)); got != title+" — Note" {
			t.Errorf("the result's heading should end with its kind for a screen reader, got %q", got)
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
