package render_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestNestedComponentValidationErrorIsHumanized checks that when a nested
// component spec has invalid props, the error rendered on the page uses plain
// language instead of raw JSON Schema terminology. This covers acceptance item 3:
// "The alert block still conveys what is wrong and which component or field has
// the problem — just in plain words". The nest test component renders a child;
// if that child's props are invalid, the error must be humanized.
func TestNestedComponentValidationErrorIsHumanized(t *testing.T) {
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
	write("manifest.json", `{
		"name": "nest",
		"version": "1.0.0",
		"description": "Test component that nests another.",
		"props": {"type": "object", "properties": {
			"inner": {"type": "object", "properties": {"component": {"type": "string"}, "props": {"type": "object"}}}
		}},
		"a11y": {},
		"machine": {},
		"examples": []
	}`)
	write("template.html", `<div data-component="nest">{{child .inner}}</div>`)

	reg := builtins(t)
	if err := reg.LoadDir(dir, "workspace"); err != nil {
		t.Fatal(err)
	}

	// Render the nest component with an inner button that has no label (invalid).
	out, err := reg.Render("nest", map[string]any{
		"inner": map[string]any{
			"component": "button",
			"props":     map[string]any{}, // missing required 'label'
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	outStr := string(out)

	// The nested component error should be rendered as a sw-render-problem span.
	if !strings.Contains(outStr, `class="sw-render-problem"`) {
		t.Errorf("nested validation failure should render in sw-render-problem: %s", outStr)
	}

	// Acceptance 1: Raw JSON Schema terms must not appear on the page.
	for _, raw := range []string{"missing property"} {
		if strings.Contains(outStr, raw) {
			t.Errorf("nested error should not expose %q on the page: %s", raw, outStr)
		}
	}

	// Acceptance 3: The span must still convey what is wrong — it names the field.
	doc, err := htmltest.Parse(outStr)
	if err != nil {
		t.Fatal(err)
	}
	problems := doc.WithAttr("class", "sw-render-problem")
	if len(problems) == 0 {
		t.Fatalf("expected a sw-render-problem span in output: %s", outStr)
	}
	text := htmltest.Text(problems[0])
	if !strings.Contains(text, "Label") && !strings.Contains(text, "label") {
		t.Errorf("nested error should mention the field that is wrong: %q", text)
	}
}
