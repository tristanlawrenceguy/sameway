package render_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestComponentsNestInsideComponents is the promise that a button placed in
// a calendar is a real button: the same template, the same classes, the same
// accessibility contract, not a copy made by the calendar.
func TestComponentsNestInsideComponents(t *testing.T) {
	reg := builtins(t)
	out, err := reg.Render("calendar", map[string]any{
		"month": "2026-09", "today": "2026-09-11", "detail": "page",
		"events": []any{map[string]any{
			"date": "2026-09-11", "label": "Design review",
			"actions": []any{
				map[string]any{"component": "button", "props": map[string]any{"label": "Join", "context": "the design review", "variant": "quiet"}},
				map[string]any{"component": "badge", "props": map[string]any{"label": "30 min"}},
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	doc, err := htmltest.Parse(string(out))
	if err != nil {
		t.Fatal(err)
	}
	btns := doc.WithAttr("data-component", "button")
	if len(btns) != 1 {
		t.Fatalf("expected one nested button, got %d in %s", len(btns), out)
	}
	if name := doc.AccessibleName(btns[0]); name != "Join the design review" {
		t.Errorf("nested button accessible name = %q", name)
	}
	if len(doc.WithAttr("data-component", "badge")) != 1 {
		t.Errorf("expected the nested badge too: %s", out)
	}
}

// TestNestedComponentsAreLimitedByTheHostComponent: a calendar event says
// in its own manifest which components may sit inside it, so a model cannot
// drop a form into a calendar cell and trap someone there.
func TestNestedComponentsAreLimitedByTheHostComponent(t *testing.T) {
	reg := builtins(t)
	_, err := reg.Render("calendar", map[string]any{
		"month": "2026-09", "today": "2026-09-11", "detail": "page",
		"events": []any{map[string]any{"date": "2026-09-11", "label": "x",
			"actions": []any{map[string]any{"component": "textarea", "props": map[string]any{"name": "n", "label": "Notes"}}}}},
	})
	if err == nil {
		t.Fatal("a calendar should refuse a component its manifest does not list")
	}
	if !strings.Contains(err.Error(), "actions/0/component") {
		t.Errorf("the refusal should name the offending prop, got: %v", err)
	}
}

// TestNestingIsBounded: a component that contains itself stops at the limit
// with a note on the page, rather than recursing until the process dies.
// The built-in components cannot express this, so the test builds one.
func TestNestingIsBounded(t *testing.T) {
	dir := t.TempDir()
	nest := filepath.Join(dir, "nest")
	if err := os.MkdirAll(filepath.Join(nest, "examples"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(nest, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("manifest.json", `{"name":"nest","version":"1.0.0","description":"Test component that contains itself.",
		"props":{"type":"object","additionalProperties":false,"properties":{
			"inner":{"type":"object","properties":{"component":{"type":"string"},"props":{"type":"object"}}}}},
		"a11y":{},"machine":{},"examples":[]}`)
	write("template.html", `<div data-component="nest">{{child .inner}}</div>`)

	reg := builtins(t)
	if err := reg.LoadDir(dir, "workspace"); err != nil {
		t.Fatal(err)
	}
	spec := map[string]any{"component": "nest", "props": map[string]any{}}
	for i := 0; i < 5; i++ {
		spec = map[string]any{"component": "nest", "props": map[string]any{"inner": spec}}
	}
	out, err := reg.Render("nest", spec["props"].(map[string]any))
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(out), `data-component="nest"`); n != 3 {
		t.Errorf("expected nesting to stop after 3 levels, got %d: %s", n, out)
	}
	if !strings.Contains(string(out), "nested too deeply") {
		t.Errorf("the stop should say why, on the page: %s", out)
	}
}

// TestNestingRejectsUnknownComponents keeps a bad spec from a model to a
// readable note instead of a broken page or a template error.
func TestNestingRejectsUnknownComponents(t *testing.T) {
	dir := t.TempDir()
	nest := filepath.Join(dir, "nest")
	os.MkdirAll(nest, 0o755)
	os.WriteFile(filepath.Join(nest, "manifest.json"), []byte(`{"name":"nest","version":"1.0.0","description":"Test component that contains itself.",
		"props":{"type":"object","additionalProperties":false,"properties":{
			"inner":{"type":"object","properties":{"component":{"type":"string"},"props":{"type":"object"}}}}},
		"a11y":{},"machine":{},"examples":[]}`), 0o644)
	os.WriteFile(filepath.Join(nest, "template.html"), []byte(`<div data-component="nest">{{child .inner}}</div>`), 0o644)

	reg := builtins(t)
	if err := reg.LoadDir(dir, "workspace"); err != nil {
		t.Fatal(err)
	}
	out, err := reg.Render("nest", map[string]any{"inner": map[string]any{"component": "nonesuch", "props": map[string]any{}}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "sw-problem") || !strings.Contains(string(out), "nonesuch") {
		t.Errorf("an unknown nested component should be named on the page: %s", out)
	}
}
