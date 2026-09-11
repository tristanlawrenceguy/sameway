package render_test

import (
	"encoding/json"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestAlertIconPropDeclared checks the alert manifest declares an optional
// "icon" string prop — type is string, not required, no default. Acceptance item 1.
func TestAlertIconPropDeclared(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}
	c, ok := reg.Get("alert")
	if !ok {
		t.Fatal("no alert component registered")
	}

	var schema struct {
		Properties map[string]map[string]any `json:"properties"`
		Required   []string                  `json:"required,omitempty"`
	}
	if err := json.Unmarshal(c.Manifest.Props, &schema); err != nil {
		t.Fatal(err)
	}

	iconDef, hasIcon := schema.Properties["icon"]
	if !hasIcon {
		t.Fatalf("alert manifest missing 'icon' property")
	}
	if typ, _ := iconDef["type"].(string); typ != "string" {
		t.Errorf("icon type is %q; want \"string\"", typ)
	}

	reqMap := map[string]bool{}
	for _, r := range schema.Required {
		reqMap[r] = true
	}
	if reqMap["icon"] {
		t.Error("'icon' should not be in required")
	}
}

// TestAlertNoIconSpanWithoutProp renders the alert without an icon prop and
// asserts no sw-alert__icon span appears. Acceptance item 2 (golden test covers
// byte-identical output; this checks structural absence).
func TestAlertNoIconSpanWithoutProp(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	alert, ok := reg.Get("alert")
	if !ok {
		t.Fatal("no alert component")
	}

	for _, ex := range alert.Manifest.Examples {
		if strings.Contains(ex.Name, "-icon") {
			continue // these have an icon prop by definition
		}
		got, err := reg.Render("alert", ex.Props)
		if err != nil {
			t.Fatalf("%s: render: %v", ex.Name, err)
		}

		doc, err := htmltest.Parse(string(got))
		if err != nil {
			t.Fatalf("%s: parse: %v", ex.Name, err)
		}

		found := false
		doc.Walk(func(n *html.Node) {
			if n.Data == "span" {
				for _, a := range n.Attr {
					if a.Key == "class" && strings.Contains(a.Val, "sw-alert__icon") {
						found = true
					}
				}
			}
		})
		if found {
			t.Errorf("%s: output contains sw-alert__icon span without icon prop", ex.Name)
		}
	}
}

// TestAlertIconSpanWithProp renders the alert with a danger kind and icon ⚠,
// then asserts the icon span structure. Acceptance item 3.
func TestAlertIconSpanWithProp(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	got, err := reg.Render("alert", map[string]any{
		"kind":    "danger",
		"icon":    "\u26a0", // ⚠
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
		if a.Key == "aria-label" {
			t.Errorf("icon span must not have aria-label, found %q", a.Val)
		}
	}
	if !hasAriaHidden {
		t.Error("icon span missing aria-hidden=\"true\"")
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

// TestAlertIconIsHTMLEscaped verifies that an icon value containing HTML
// markup is escaped by the template engine rather than injected raw. Acceptance
// item 4.
func TestAlertIconIsHTMLEscaped(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	got, err := reg.Render("alert", map[string]any{
		"kind":    "danger",
		"icon":    "<script>alert(1)</script>",
		"message": "test message",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	out := string(got)
	if strings.Contains(out, "<script>") {
		t.Errorf("icon value was not HTML-escaped — raw <script> found in output:\n%s", out)
	}

	doc, err := htmltest.Parse(out)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	foundScript := false
	doc.Walk(func(n *html.Node) {
		if n.Data == "script" {
			foundScript = true
		}
	})
	if foundScript {
		t.Error("a <script> element appeared in the DOM from icon prop")
	}
}
