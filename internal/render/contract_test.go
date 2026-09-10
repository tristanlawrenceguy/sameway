package render_test

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

func builtins(t *testing.T) *render.Registry {
	t.Helper()
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}
	return reg
}

// forEachExample renders every example of every built-in component.
func forEachExample(t *testing.T, fn func(c *render.Component, ex render.Example, doc *htmltest.Doc)) {
	t.Helper()
	for _, c := range builtins(t).Components() {
		for _, ex := range c.Manifest.Examples {
			out, err := c.Render(ex.Props)
			if err != nil {
				t.Errorf("%s/%s: %v", c.Manifest.Name, ex.Name, err)
				continue
			}
			doc, err := htmltest.Parse(string(out))
			if err != nil {
				t.Errorf("%s/%s: parse: %v", c.Manifest.Name, ex.Name, err)
				continue
			}
			fn(c, ex, doc)
		}
	}
}

// TestMachineSelectorResolves checks the manifest's machine.selector finds
// an element in every example, which is what a browser-driving agent relies on.
func TestMachineSelectorResolves(t *testing.T) {
	forEachExample(t, func(c *render.Component, ex render.Example, doc *htmltest.Doc) {
		var machine struct{ Selector string }
		json.Unmarshal(c.Manifest.Machine, &machine)
		matches := resolve(doc, machine.Selector)
		if len(matches) == 0 {
			t.Errorf("%s/%s: machine.selector %q matches nothing", c.Manifest.Name, ex.Name, machine.Selector)
			return
		}
		for _, m := range matches {
			if htmltest.Focusable(m) && doc.AccessibleName(m) == "" {
				t.Errorf("%s/%s: focusable element <%s> has no accessible name", c.Manifest.Name, ex.Name, m.Data)
			}
		}
	})
}

// resolve supports the selector shapes manifests use:
// [data-component="x"] and [data-component="x"] <tag>.
func resolve(doc *htmltest.Doc, selector string) []*html.Node {
	parts := strings.Fields(selector)
	re := regexp.MustCompile(`^\[data-component="([^"]+)"\]$`)
	m := re.FindStringSubmatch(parts[0])
	if m == nil {
		return nil
	}
	roots := doc.WithAttr("data-component", m[1])
	if len(parts) == 1 {
		return roots
	}
	var out []*html.Node
	for _, r := range roots {
		sub := &htmltest.Doc{Root: r}
		out = append(out, sub.Elements(parts[1])...)
	}
	return out
}

// TestStructureContract checks the invariants every rendered example must
// hold for screen readers and agents alike.
func TestStructureContract(t *testing.T) {
	forEachExample(t, func(c *render.Component, ex render.Example, doc *htmltest.Doc) {
		where := c.Manifest.Name + "/" + ex.Name
		if len(doc.WithAttr("data-component", c.Manifest.Name)) == 0 {
			t.Errorf("%s: no element with data-component=%q", where, c.Manifest.Name)
		}
		seen := map[string]bool{}
		for _, n := range doc.WithAttr("id", "") {
			id, _ := htmltest.Attr(n, "id")
			if seen[id] {
				t.Errorf("%s: duplicate id %q", where, id)
			}
			seen[id] = true
		}
		for _, key := range []string{"aria-describedby", "aria-labelledby"} {
			for _, n := range doc.WithAttr(key, "") {
				refs, _ := htmltest.Attr(n, key)
				for _, id := range strings.Fields(refs) {
					if doc.ByID(id) == nil {
						t.Errorf("%s: %s points at missing id %q", where, key, id)
					}
				}
			}
		}
		for _, l := range doc.Elements("label") {
			target, _ := htmltest.Attr(l, "for")
			if doc.ByID(target) == nil {
				t.Errorf("%s: label for=%q has no target", where, target)
			}
		}
		for _, tag := range []string{"input", "select", "textarea"} {
			for _, n := range doc.Elements(tag) {
				if doc.AccessibleName(n) == "" {
					t.Errorf("%s: <%s> has no label", where, tag)
				}
			}
		}
		for _, n := range doc.Elements("table") {
			if doc.AccessibleName(n) == "" {
				t.Errorf("%s: table has no caption", where)
			}
		}
		for _, n := range doc.Elements("img") {
			if _, ok := htmltest.Attr(n, "alt"); !ok {
				t.Errorf("%s: img without alt", where)
			}
		}
	})
}

// TestKeyboardContract requires a keyboard map for any component that
// renders something focusable.
func TestKeyboardContract(t *testing.T) {
	forEachExample(t, func(c *render.Component, ex render.Example, doc *htmltest.Doc) {
		focusable := 0
		doc.Walk(func(n *html.Node) {
			if htmltest.Focusable(n) {
				focusable++
			}
		})
		if focusable == 0 {
			return
		}
		var a11y struct{ Keyboard []struct{ Key, Does string } }
		json.Unmarshal(c.Manifest.A11y, &a11y)
		if len(a11y.Keyboard) == 0 {
			t.Errorf("%s: renders focusable elements but a11y.keyboard is empty", c.Manifest.Name)
		}
		for _, k := range a11y.Keyboard {
			if k.Key == "" || k.Does == "" {
				t.Errorf("%s: keyboard entry needs key and does: %+v", c.Manifest.Name, k)
			}
		}
	})
}

// TestEveryEnumValueHasAnExample makes each way a component can be used
// visible: every enum prop value must appear in an example or be the default.
func TestEveryEnumValueHasAnExample(t *testing.T) {
	for _, c := range builtins(t).Components() {
		var schema struct {
			Properties map[string]struct {
				Enum    []any `json:"enum"`
				Default any   `json:"default"`
			} `json:"properties"`
		}
		if err := json.Unmarshal(c.Manifest.Props, &schema); err != nil {
			t.Fatalf("%s: %v", c.Manifest.Name, err)
		}
		for prop, def := range schema.Properties {
			if len(def.Enum) == 0 {
				continue
			}
			used := map[string]bool{}
			for _, ex := range c.Manifest.Examples {
				if v, ok := ex.Props[prop]; ok {
					used[fmt.Sprint(v)] = true
				} else if def.Default != nil {
					used[fmt.Sprint(def.Default)] = true
				}
			}
			for _, v := range def.Enum {
				if !used[fmt.Sprint(v)] {
					t.Errorf("%s: no example uses %s=%v", c.Manifest.Name, prop, v)
				}
			}
		}
	}
}

// TestManifestSections checks the human-facing contract is filled in.
func TestManifestSections(t *testing.T) {
	for _, c := range builtins(t).Components() {
		var a11y struct {
			Role string
			WCAG struct{ Target, Notes string } `json:"wcag"`
		}
		var machine struct{ Selector, Identify, Operate string }
		json.Unmarshal(c.Manifest.A11y, &a11y)
		json.Unmarshal(c.Manifest.Machine, &machine)
		if c.Manifest.Description == "" || a11y.Role == "" || a11y.WCAG.Target == "" || a11y.WCAG.Notes == "" {
			t.Errorf("%s: description, a11y.role, and a11y.wcag must be filled in", c.Manifest.Name)
		}
		if machine.Selector == "" || machine.Identify == "" || machine.Operate == "" {
			t.Errorf("%s: machine.selector, identify, and operate must be filled in", c.Manifest.Name)
		}
		readme, err := c.ReadFile("README.md")
		if err != nil || len(strings.TrimSpace(string(readme))) < 40 {
			t.Errorf("%s: README.md missing or too short", c.Manifest.Name)
		}
	}
}

// TestCSSUsesTokensOnly forbids raw colours in component stylesheets.
func TestCSSUsesTokensOnly(t *testing.T) {
	raw := regexp.MustCompile(`(?i)#[0-9a-f]{3,8}\b|rgba?\(|hsla?\(`)
	for _, c := range builtins(t).Components() {
		if m := raw.FindString(c.CSS); m != "" {
			t.Errorf("%s: style.css uses raw colour %q; use a --sw- token", c.Manifest.Name, m)
		}
		if strings.Contains(c.CSS, "outline: none") || strings.Contains(c.CSS, "outline:none") {
			t.Errorf("%s: style.css removes the focus outline", c.Manifest.Name)
		}
	}
}
