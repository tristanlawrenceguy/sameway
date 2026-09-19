package render_test

import (
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestAlertIconOrder renders an alert with kind=danger and icon ⚠, then asserts
// the icon span appears before the kind span in the title paragraph. Acceptance item 6.
func TestAlertIconOrder(t *testing.T) {
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
		if n.Data == "span" {
			for _, a := range n.Attr {
				if a.Key == "class" && strings.Contains(a.Val, "sw-alert__icon") {
					iconNode = n
				}
			}
		}
	})
	if iconNode == nil {
		t.Fatalf("output lacks <span class=\"sw-alert__icon\">;\ngot:\n%s", got)
	}

	hasAriaHidden := false
	for _, a := range iconNode.Attr {
		if a.Key == "aria-hidden" && a.Val == "true" {
			hasAriaHidden = true
		}
	}
	if hasAriaHidden {
		t.Error("icon span must not have aria-hidden=\"true\"")
	}

	text := htmltest.Text(iconNode)
	if text != "\u26a0" {
		t.Errorf("icon span text is %q; want \"⚠\"", text)
	}

	// Confirm icon span appears before kind span in title paragraph.
	var pTitle *html.Node
	doc.Walk(func(n *html.Node) {
		if n.Data == "p" {
			for _, a := range n.Attr {
				if a.Key == "class" && strings.Contains(a.Val, "sw-alert__title") {
					pTitle = n
					return
				}
			}
		}
	})
	if pTitle == nil {
		t.Fatalf("no <p class=\"sw-alert__title\"> in output")
	}

	var order struct{ icon, kind int }
	i := 0
	for c := pTitle.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "span" {
			for _, a := range c.Attr {
				if a.Key == "class" {
					switch {
					case strings.Contains(a.Val, "sw-alert__icon"):
						order.icon = i
					case strings.Contains(a.Val, "sw-alert__kind"):
						order.kind = i
					}
				}
			}
		}
		i++
	}
	if order.icon > 0 && order.icon >= order.kind {
		t.Errorf("icon span (pos %d) must appear before kind span (pos %d)", order.icon, order.kind)
	}
}

// TestAlertInfoKindAriaLabel renders the alert with kind=info and icon ℹ,
// then asserts the icon span does NOT carry aria-label="info". Acceptance item 5.
func TestAlertInfoKindAriaLabel(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	got, err := reg.Render("alert", map[string]any{
		"kind":    "info",
		"icon":    "\u2139", // ℹ
		"message": "A note with an icon.",
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
		if n.Data == "span" {
			for _, a := range n.Attr {
				if a.Key == "class" && strings.Contains(a.Val, "sw-alert__icon") {
					iconNode = n
				}
			}
		}
	})
	if iconNode == nil {
		t.Fatalf("output lacks <span class=\"sw-alert__icon\">;\ngot:\n%s", got)
	}

	hasAriaLabel := false
	for _, a := range iconNode.Attr {
		if a.Key == "aria-label" && a.Val == "info" {
			hasAriaLabel = true
		}
	}
	if hasAriaLabel {
		t.Error("icon span must not have aria-label")
	}

	text := htmltest.Text(iconNode)
	if text != "\u2139" {
		t.Errorf("icon span text is %q; want \"ℹ\"", text)
	}
}
