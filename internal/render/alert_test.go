package render_test

import (
	"encoding/json"
	"regexp"
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
		if iconNode != nil {
			t.Errorf("%s: output should not contain <span class=\"sw-alert__icon\">;\ngot:\n%s", ex.Name, got)
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

// TestAlertIconCSSRuleExists asserts that the alert component's compiled CSS
// contains a .sw-alert__icon rule block. Acceptance item 1.
func TestAlertIconCSSRuleExists(t *testing.T) {
	reg := builtins(t)
	c, ok := reg.Get("alert")
	if !ok {
		t.Fatal("alert component not found in registry")
	}

	ruleRe := regexp.MustCompile(`(?s)\.sw-alert__icon\s*\{([^}]*)\}`)
	matches := ruleRe.FindStringSubmatch(c.CSS)
	if len(matches) < 2 {
		t.Fatal("alert: could not find .sw-alert__icon rule block in CSS")
	}

	ruleBody := matches[1]

	fsRe := regexp.MustCompile(`font-size\s*:\s*(var\(--sw-size-text-[^)]+\))`)
	if !fsRe.MatchString(ruleBody) {
		t.Errorf("alert: .sw-alert__icon has no font-size using --sw-size-text-* token; got %q", fsRe.FindStringSubmatch(ruleBody))
	}

	mrRe := regexp.MustCompile(`margin-right\s*:\s*(var\(--sw-space-[^)]+\))`)
	if !mrRe.MatchString(ruleBody) {
		t.Errorf("alert: .sw-alert__icon has no margin-right using --sw-space-* token; got %q", mrRe.FindStringSubmatch(ruleBody))
	}
}

// TestAlertIconFontSizeToken asserts the CSS file content includes
// `font-size: var(--sw-size-text-`. Acceptance item 2.
func TestAlertIconFontSizeToken(t *testing.T) {
	reg := builtins(t)
	c, ok := reg.Get("alert")
	if !ok {
		t.Fatal("alert component not found in registry")
	}

	ruleRe := regexp.MustCompile(`(?s)\.sw-alert__icon\s*\{([^}]*)\}`)
	matches := ruleRe.FindStringSubmatch(c.CSS)
	if len(matches) < 2 {
		t.Fatal("alert: could not find .sw-alert__icon rule block in CSS")
	}

	ruleBody := matches[1]
	if !strings.Contains(ruleBody, "font-size: var(--sw-size-text-") {
		t.Errorf("alert: .sw-alert__icon font-size does not use --sw-size-text-* token; got %q", ruleBody)
	}
}

// TestAlertIconMarginRightToken asserts the CSS file content includes
// `margin-right: var(--sw-space-`. Acceptance item 3.
func TestAlertIconMarginRightToken(t *testing.T) {
	reg := builtins(t)
	c, ok := reg.Get("alert")
	if !ok {
		t.Fatal("alert component not found in registry")
	}

	ruleRe := regexp.MustCompile(`(?s)\.sw-alert__icon\s*\{([^}]*)\}`)
	matches := ruleRe.FindStringSubmatch(c.CSS)
	if len(matches) < 2 {
		t.Fatal("alert: could not find .sw-alert__icon rule block in CSS")
	}

	ruleBody := matches[1]
	if !strings.Contains(ruleBody, "margin-right: var(--sw-space-") {
		t.Errorf("alert: .sw-alert__icon margin-right does not use --sw-space-* token; got %q", ruleBody)
	}
}
