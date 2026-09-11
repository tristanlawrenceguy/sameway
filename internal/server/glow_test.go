package server_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestRestingCanvasIsQuiet checks the default page carries no provenance
// chrome at all. The glow said who did what when it happened; once it has
// faded, the page is just the person's content.
func TestRestingCanvasIsQuiet(t *testing.T) {
	h, _ := canvasWithABlock(t)
	doc := parse(t, get(t, h, "/"))

	if n := len(doc.WithAttr("data-component", "badge")); n != 0 {
		t.Errorf("a resting canvas should carry no badges, got %d", n)
	}
	// The control bar exists, in the quiet layer, and holds actions only.
	bars := doc.WithAttr("class", "sw-bar sw-quiet")
	if len(bars) != 1 {
		t.Fatalf("expected one quiet control bar, got %d", len(bars))
	}
	sub := &htmltest.Doc{Root: bars[0]}
	if n := len(sub.Elements("a")) + len(sub.Elements("button")); n != 2 {
		t.Errorf("the control bar should hold Expand and Remove and nothing else, got %d controls", n)
	}
	// Provenance is still on the block for machines and screen readers.
	for _, n := range doc.WithAttr("data-block-id", "") {
		if _, ok := htmltest.Attr(n, "data-actor"); !ok {
			t.Errorf("every block must still carry data-actor")
		}
	}
	// And the header is down to the person's own content.
	main := doc.WithAttr("aria-label", "Main")
	if len(main) != 1 {
		t.Fatalf("expected one main nav")
	}
	links := (&htmltest.Doc{Root: main[0]}).Elements("a")
	if len(links) != 1 || htmltest.Text(links[0]) != "notes" {
		t.Errorf("the header should carry only content types, got %d links", len(links))
	}
}
