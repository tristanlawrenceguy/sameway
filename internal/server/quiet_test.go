package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// canvasWithABlock puts one assistant-made block on the canvas.
func canvasWithABlock(t *testing.T) (http.Handler, string) {
	t.Helper()
	a, h := newApp(t)
	a.Chat.Provider, a.Chat.ProviderErr = &scripted{steps: []*llm.Response{
		toolCall("add_component", map[string]any{"component": "card", "props": map[string]any{"title": "Shopping"}}),
		{Text: "Added a card."},
	}}, nil
	postForm(t, h, "/chat", url.Values{"message": {"add a card"}})
	var blocks struct {
		Records []struct{ ID string }
	}
	decode(t, get(t, h, "/api/block"), &blocks)
	if len(blocks.Records) != 1 {
		t.Fatalf("expected one block, got %d", len(blocks.Records))
	}
	return h, blocks.Records[0].ID
}

// TestQuietControlsStayAvailableToEveryone is the core promise of the quiet
// layer: chrome is visually faded for sighted pointer users but is still in
// the DOM, in the tab order, and in the accessibility tree, so screen reader
// users and agents keep it. Fading is done with opacity, never display,
// visibility, hidden, or aria-hidden.
func TestQuietControlsStayAvailableToEveryone(t *testing.T) {
	h, id := canvasWithABlock(t)
	doc := parse(t, get(t, h, "/"))

	bars := doc.WithAttr("class", "sw-bar sw-quiet")
	if len(bars) != 1 {
		t.Fatalf("expected one quiet control bar, got %d", len(bars))
	}
	bar := bars[0]
	sub := &htmltest.Doc{Root: bar}

	// Nothing in the quiet layer may be hidden from assistive technology.
	sub.Walk(func(n *html.Node) {
		for _, bad := range []string{"aria-hidden", "hidden", "inert"} {
			if v, ok := htmltest.Attr(n, bad); ok && v != "false" {
				t.Errorf("<%s> in the quiet layer sets %s; that hides it from screen readers too", n.Data, bad)
			}
		}
		if style, ok := htmltest.Attr(n, "style"); ok {
			for _, bad := range []string{"display:none", "display: none", "visibility:hidden", "visibility: hidden"} {
				if strings.Contains(style, bad) {
					t.Errorf("<%s> in the quiet layer uses %s inline", n.Data, bad)
				}
			}
		}
		if ti, ok := htmltest.Attr(n, "tabindex"); ok && ti == "-1" {
			t.Errorf("<%s> in the quiet layer is removed from the tab order", n.Data)
		}
	})

	// The controls are the real thing: focusable, named, and wired up.
	edit := findByName(t, sub, "a", "Edit card")
	if href, _ := htmltest.Attr(edit, "href"); href != "/t/block/"+id+"/edit" {
		t.Errorf("Edit link points at %q", href)
	}
	remove := findByName(t, sub, "button", "Remove card")
	if typ, _ := htmltest.Attr(remove, "type"); typ != "submit" {
		t.Errorf("Remove should submit its form, got type %q", typ)
	}
	for _, n := range []*html.Node{edit, remove} {
		if !htmltest.Focusable(n) {
			t.Errorf("<%s> in the quiet layer is not focusable", n.Data)
		}
	}

	// The resting page says nothing about provenance, on screen or in the
	// control bar. It is still complete in the accessibility tree.
	if len(sub.WithAttr("data-component", "badge")) != 0 {
		t.Errorf("the control bar should carry actions only, no provenance label")
	}
	block := doc.WithAttr("data-block-id", id)[0]
	hidden := (&htmltest.Doc{Root: block}).WithAttr("class", "sw-visually-hidden")
	found := false
	for _, n := range hidden {
		if strings.Contains(htmltest.Text(n), "Added by the assistant") {
			found = true
		}
	}
	if !found {
		t.Errorf("a block should still state who made it in the accessibility tree")
	}
	if actor, _ := htmltest.Attr(block, "data-actor"); actor != "assistant" {
		t.Errorf("and in data-actor for machines, got %q", actor)
	}

	// And a person can actually use them.
	if rec := postForm(t, h, "/canvas/"+id+"/delete", nil); rec.Code != http.StatusSeeOther {
		t.Errorf("removing a block through the quiet control returned %d", rec.Code)
	}
}

// findByName locates one element of a tag whose accessible name matches,
// the way a screen reader user or an agent would address it.
func findByName(t *testing.T, doc *htmltest.Doc, tag, name string) *html.Node {
	t.Helper()
	for _, n := range doc.Elements(tag) {
		if doc.AccessibleName(n) == name {
			return n
		}
	}
	var got []string
	for _, n := range doc.Elements(tag) {
		got = append(got, doc.AccessibleName(n))
	}
	t.Fatalf("no <%s> named %q; found %v", tag, name, got)
	return nil
}

// TestCompactLabelsKeepFullAccessibleNames checks the context prop: short
// visible text, complete accessible name.
func TestCompactLabelsKeepFullAccessibleNames(t *testing.T) {
	h, _ := canvasWithABlock(t)
	doc := parse(t, get(t, h, "/"))
	remove := findByName(t, doc, "button", "Remove card")
	// The visible run of text is just the label; the rest is for machines.
	visible := ""
	for c := remove.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			visible += c.Data
		}
	}
	if strings.TrimSpace(visible) != "Remove" {
		t.Errorf("visible text should be short, got %q", visible)
	}
	hiddenSpans := (&htmltest.Doc{Root: remove}).WithAttr("class", "sw-visually-hidden")
	if len(hiddenSpans) != 1 || strings.TrimSpace(htmltest.Text(hiddenSpans[0])) != "card" {
		t.Errorf("the context should be carried in a visually hidden span")
	}
}

// TestActivityIsHiddenButNotLost checks the log is collapsed by default and
// still reachable in full for anyone who does not want to open it.
func TestActivityIsHiddenButNotLost(t *testing.T) {
	h, _ := canvasWithABlock(t)
	doc := parse(t, get(t, h, "/"))

	details := doc.WithAttr("data-component", "disclosure")
	if len(details) != 1 {
		t.Fatalf("expected the activity log in one disclosure, got %d", len(details))
	}
	if _, open := htmltest.Attr(details[0], "open"); open {
		t.Errorf("the activity log should start closed")
	}
	if details[0].Data != "details" {
		t.Errorf("disclosure should be a native details element, got <%s>", details[0].Data)
	}
	summaries := (&htmltest.Doc{Root: details[0]}).Elements("summary")
	if len(summaries) != 1 || !strings.Contains(htmltest.Text(summaries[0]), "Activity") {
		t.Errorf("the summary should name what is inside")
	}
	// The count tells a person whether opening it is worth it.
	if !strings.ContainsAny(htmltest.Text(summaries[0]), "0123456789") {
		t.Errorf("the summary should carry a count: %q", htmltest.Text(summaries[0]))
	}
	// Nothing is lost: the whole log has its own page and its own JSON.
	wantStatus(t, get(t, h, "/activity"), http.StatusOK)
	full := parse(t, get(t, h, "/activity"))
	if len(full.WithAttr("data-component", "event")) < 2 {
		t.Errorf("the activity page should list every action")
	}
	var log struct{ Count int }
	decode(t, get(t, h, "/api/activity"), &log)
	if log.Count < 2 {
		t.Errorf("the activity API should list every action, got %d", log.Count)
	}
}

// TestControlsVisibleSetting checks a workspace can pin the chrome on for
// people who do not want it to fade.
func TestControlsVisibleSetting(t *testing.T) {
	a, h := newApp(t)
	doc := parse(t, get(t, h, "/"))
	if v, _ := htmltest.Attr(doc.Elements("html")[0], "data-controls"); v != "auto" {
		t.Errorf("default should be data-controls=auto, got %q", v)
	}

	a.Workspace.Config.UI.Controls = "visible"
	doc = parse(t, get(t, h, "/"))
	if v, _ := htmltest.Attr(doc.Elements("html")[0], "data-controls"); v != "visible" {
		t.Errorf("ui.controls: visible should reach the root element, got %q", v)
	}
}
