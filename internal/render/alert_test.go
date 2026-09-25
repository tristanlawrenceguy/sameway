package render_test

import (
	"encoding/json"
	"fmt"
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
// asserts no sw-alert__icon span appears for success kind (which has neither a
// text prefix nor a default icon). Info, warning, and danger kinds get default
// icons so they do render .sw-alert__icon even without the prop.
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

		propsKind := fmt.Sprintf("%v", ex.Props["kind"])
		if propsKind == "success" {
			if iconNode != nil {
				t.Errorf("%s: success output should not contain <span class=\"sw-alert__icon\">;\ngot:\n%s", ex.Name, got)
			}
		}
	}
}

// TestAlertIconIsHTMLEscaped verifies that an icon value containing HTML markup
// is escaped by the template engine. Acceptance item 4.
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

// TestAlertKeyboard verifies keyboard accessibility and ARIA attributes of the
// alert component. It renders two variants — with dismiss enabled
// and without — then asserts on focusability, role, aria-live. Acceptance items
// 2–4. Manifest documentation checks are in alert_doc_test.go.
func TestAlertKeyboard(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	// --- Render A: dismiss enabled (success kind → role="status") ---

	got, err := reg.Render("alert", map[string]any{
		"kind":    "success",
		"message": "Changes saved",
		"dismiss": true,
	})
	if err != nil {
		t.Fatalf("render alert with dismiss: %v", err)
	}

	out := string(got)

	// 1. The close button exists (Acceptance item 2).
	if !strings.Contains(out, `sw-alert__close`) {
		t.Errorf("alert with dismiss=true should render .sw-alert__close;\ngot:\n%s", out)
		return
	}

	doc, err := htmltest.Parse(out)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	// 2. The close button is focusable — native <button> without tabindex="-1"
	//    is always in the tab order (Acceptance item 2).
	closeBtns := doc.WithAttr("class", "sw-alert__close")
	if len(closeBtns) == 0 {
		t.Error("no element with class sw-alert__close found in parsed HTML")
	} else {
		btn := closeBtns[0]
		if !htmltest.Focusable(btn) {
			t.Errorf("close button is not focusable (Tab should reach it);\nnode: %#v", btn)
		}
	}

	if strings.Contains(out, `tabindex="-1"`) {
		t.Errorf("alert with dismiss must not use tabindex=\"-1\" (blocks keyboard focus);\ngot:\n%s", out)
	}

	// 3. The root div has role="status" for success kind (Acceptance item 4).
	rootEls := doc.WithAttr("data-component", "alert")
	if len(rootEls) == 0 {
		t.Error("no element with data-component=\"alert\" found in parsed HTML")
	} else {
		root := rootEls[0]
		role, hasRole := htmltest.Attr(root, "role")
		if !hasRole || role != "status" {
			t.Errorf("alert success should have role=\"status\"; got %q (attr present: %v)", role, hasRole)
		}

		// 4. The root div has aria-live="polite" for screen-reader announcement
		//    (Acceptance item 4).
		ariaLive, hasAria := htmltest.Attr(root, "aria-live")
		if !hasAria {
			t.Error("alert root should have aria-live attribute (screen reader announcement);\ngot:\n" + out)
		} else if ariaLive != "polite" {
			t.Errorf("alert role=\"status\" should have aria-live=\"polite\"; got %q", ariaLive)
		}
	}

	// --- Render B: dismiss not set (info kind, no close button) ---

	got2, err := reg.Render("alert", map[string]any{
		"kind":    "info",
		"message": "No changes.",
	})
	if err != nil {
		t.Fatalf("render alert without dismiss: %v", err)
	}

	out2 := string(got2)

	// 5. No close button appears when dismiss is not set (Acceptance item 3).
	if strings.Contains(out2, "sw-alert__close") {
		t.Errorf("alert without dismiss must NOT render .sw-alert__close;\ngot:\n%s", out2)
	}

	if strings.Contains(out2, `data-dismiss`) {
		t.Errorf("alert without dismiss must NOT have data-dismiss attribute;\ngot:\n%s", out2)
	}
}
