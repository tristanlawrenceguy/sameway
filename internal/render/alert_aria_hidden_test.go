package render_test

import (
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestAlertIconHasNoAriaHidden renders an alert with a unicode icon prop and
// asserts that the icon span carries no aria-hidden attribute — ensuring the
// unicode character is spoken by screen readers.  Regression guard for backlog
// items 0019, 0010; acceptance item 3.
func TestAlertIconHasNoAriaHidden(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	got, err := reg.Render("alert", map[string]any{
		"kind":    "danger",
		"icon":    "\u26a0", // ⚠ warning sign
		"title":   "No model connected",
		"message": "Edit the llm section of workspace.yaml and restart.",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	doc, err := htmltest.Parse(string(got))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	var iconNode *html.Node
	doc.Walk(func(n *html.Node) {
		if n.Data == "span" && hasClass(n, "sw-alert__icon") {
			iconNode = n
		}
	})
	if iconNode == nil {
		t.Fatalf("output lacks <span class=\"sw-alert__icon\">;\ngot:\n%s", got)
	}

	for _, a := range iconNode.Attr {
		if a.Key == "aria-hidden" {
			t.Errorf("icon span must not have aria-hidden attribute at all (value=%q); unicode icons should be announced by screen readers, not hidden from them\nfull output:\n%s", a.Val, got)
		}
	}

	text := htmltest.Text(iconNode)
	if text != "\u26a0" {
		t.Errorf("icon span text is %q; want \"⚠\" (\\u26a0)", text)
	}
}

func hasClass(n *html.Node, cls string) bool {
	for _, a := range n.Attr {
		if a.Key == "class" && strings.Contains(a.Val, cls) {
			return true
		}
	}
	return false
}
