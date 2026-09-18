package server_test

import (
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestEditButtonUsesBlockLabel tests the Edit button's accessible name comes
// from data-block-label, not from data-block-component.  The server adds the
// attribute and the script reads it; this test verifies both pieces.
func TestEditButtonUsesBlockLabel(t *testing.T) {
	h, _ := canvasWithABlock(t)

	// Step A: the block element must carry a data-block-label with the title.
	doc := parse(t, get(t, h, "/"))
	blocks := doc.WithAttr("data-block-id", "")
	if len(blocks) == 0 {
		t.Fatal("no canvas blocks on the page")
	}
	label := ""
	for _, b := range blocks {
		if v, ok := htmltest.Attr(b, "data-block-label"); ok && v != "" {
			label = v
		}
	}
	if label == "" {
		t.Errorf("canvas block must have a data-block-label attribute with the note title")
	}

	// Step B: the Edit button script must read that attribute.
	script := get(t, h, "/design/base/08-edit.js")
	body := script.Body.String()
	if !strings.Contains(body, `getAttribute("data-block-label")`) {
		t.Errorf("08-edit.js must read data-block-label for the Edit button's accessible name\nbody: %s", truncate(body))
	}

	// Step C: it must fall back to data-block-component if data-block-label is absent.
	if !strings.Contains(body, "data-block-component") {
		t.Errorf("08-edit.js must still reference data-block-component as a fallback\nbody: %s", truncate(body))
	}
}

// TestExpandLinkAccessibleNameIncludesTitle checks that the Expand link on each
// canvas card carries the note title in its accessible name, not "card".
func TestExpandLinkAccessibleNameIncludesTitle(t *testing.T) {
	h, _ := canvasWithABlock(t)
	doc := parse(t, get(t, h, "/"))

	// Find all expand links on the page.
	var expands []*htmltest.Doc
	doc.Walk(func(n *html.Node) {
		if n.Data == "a" {
			d := &htmltest.Doc{Root: n}
			name := d.AccessibleName(n)
			if strings.HasPrefix(name, "Expand") {
				expands = append(expands, d)
			}
		}
	})

	if len(expands) == 0 {
		t.Fatal("no Expand links found on the canvas page")
		return
	}

	for _, e := range expands {
		name := e.AccessibleName(e.Root)
		if strings.Contains(name, "Expand card") {
			t.Errorf("expand link should include the note title, not say %q", name)
		}
		// The visible text is "Expand"; the rest must be in a visually hidden span.
		hidden := (&htmltest.Doc{Root: e.Root}).WithAttr("class", "sw-visually-hidden")
		if len(hidden) != 1 {
			t.Errorf("expand link %q should carry its context in one sw-visually-hidden span", name)
			continue
		}
		hiddenText := strings.TrimSpace(htmltest.Text(hidden[0]))
		if hiddenText == "card" || hiddenText == "" {
			t.Errorf("the visually hidden span on expand link %q should contain the note title, not %q or empty", name, hiddenText)
		}
	}
}

// TestRemoveButtonAccessibleNameIncludesTitle checks that the Remove button on
// each canvas card carries the note title in its accessible name.
func TestRemoveButtonAccessibleNameIncludesTitle(t *testing.T) {
	h, _ := canvasWithABlock(t)
	doc := parse(t, get(t, h, "/"))

	// Find all remove buttons on the page.
	var removes []*htmltest.Doc
	for _, n := range doc.Elements("button") {
		name := doc.AccessibleName(n)
		if strings.HasPrefix(name, "Remove") {
			removes = append(removes, &htmltest.Doc{Root: n})
		}
	}

	if len(removes) == 0 {
		t.Fatal("no Remove buttons found on the canvas page")
		return
	}

	for _, r := range removes {
		name := r.AccessibleName(r.Root)
		if strings.Contains(name, "Remove card") {
			t.Errorf("remove button should include the note title, not say %q", name)
		}
		// The visible text is "Remove"; the rest must be in a visually hidden span.
		hidden := (&htmltest.Doc{Root: r.Root}).WithAttr("class", "sw-visually-hidden")
		if len(hidden) != 1 {
			t.Errorf("remove button %q should carry its context in one sw-visually-hidden span", name)
			continue
		}
		hiddenText := strings.TrimSpace(htmltest.Text(hidden[0]))
		if hiddenText == "card" || hiddenText == "" {
			t.Errorf("the visually hidden span on remove button %q should contain the note title, not %q or empty", name, hiddenText)
		}
	}
}

// TestExpandLinkAccessibleNameForCalendar checks that a calendar block's expand
// link uses its caption as context rather than "calendar".
func TestExpandLinkAccessibleNameForCalendar(t *testing.T) {
	h, id := canvasWithACalendar(t)
	doc := parse(t, get(t, h, "/"))

	// Find the expand link for the calendar block by matching its href.
	var expands []*html.Node
	for _, n := range doc.Elements("a") {
		if href, _ := htmltest.Attr(n, "href"); href == "/canvas/"+id {
			name := doc.AccessibleName(n)
			if strings.HasPrefix(name, "Expand") {
				expands = append(expands, n)
			}
		}
	}

	if len(expands) != 1 {
		t.Fatalf("expected one Expand link for the calendar block, got %d", len(expands))
		return
	}

	name := doc.AccessibleName(expands[0])
	if strings.Contains(name, "Expand calendar") {
		t.Errorf("expand link should include the calendar's caption, not say %q", name)
	}
	if !strings.Contains(name, "September 2026") && !strings.HasPrefix(name, "Expand September") {
		t.Errorf("expand link for the calendar block should read as something like 'Expand September 2026', got %q", name)
	}
}

// TestBlockItemHasDataBlockLabel asserts that every canvas block element carries
// a data-block-label attribute with its summarised title.
func TestBlockItemHasDataBlockLabel(t *testing.T) {
	h, id := canvasWithABlock(t)
	doc := parse(t, get(t, h, "/"))

	block := doc.WithAttr("data-block-id", id)
	if len(block) == 0 {
		t.Fatal("block not found on page")
	}

	v, ok := htmltest.Attr(block[0], "data-block-label")
	if !ok || v == "" {
		t.Errorf("canvas block %q must have a non-empty data-block-label attribute", id)
		return
	}
	if v == "card" {
		t.Errorf("data-block-label should be the note title, not the component type; got %q", v)
	}
	if v != "Shopping" {
		t.Errorf("expected data-block-label to be 'Shopping', got %q", v)
	}
}

// TestCanvasControlsAreDistinguishable asserts that on a page with multiple
// canvas blocks, each control's accessible name is unique — the core bug fix.
func TestCanvasControlsAreDistinguishable(t *testing.T) {
	a, h := newApp(t)
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("add_component", map[string]any{"component": "card", "props": map[string]any{"title": "First Card"}}),
		{Text: "Added first card."},
	}}, nil
	postForm(t, h, "/chat", url.Values{"message": {"add a card"}})

	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("add_component", map[string]any{"component": "card", "props": map[string]any{"title": "Second Card"}}),
		{Text: "Added second card."},
	}}, nil
	postForm(t, h, "/chat", url.Values{"message": {"and a second one"}})

	doc := parse(t, get(t, h, "/"))

	// Collect all Remove button accessible names.
	var removeNames []string
	for _, n := range doc.Elements("button") {
		name := doc.AccessibleName(n)
		if strings.HasPrefix(name, "Remove") {
			removeNames = append(removeNames, name)
		}
	}

	if len(removeNames) < 2 {
		t.Fatalf("expected at least two Remove buttons, got %d", len(removeNames))
	}

	// Each should be unique.
	for i := 0; i < len(removeNames); i++ {
		for j := i + 1; j < len(removeNames); j++ {
			if removeNames[i] == removeNames[j] {
				t.Errorf("Remove buttons %d and %d have the same accessible name %q — they should be distinguishable", i, j, removeNames[i])
			}
		}
	}

	for i := 0; i < len(removeNames); i++ {
		if strings.Contains(removeNames[i], "Remove card") {
			t.Errorf("remove button %d says %q instead of 'Remove [title]'", i, removeNames[i])
		}
	}
}
