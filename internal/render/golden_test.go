package render_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// TestGolden renders every manifest example and compares it with the checked
// in example file. Run with UPDATE_GOLDEN=1 to rewrite the example files.
func TestGolden(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}
	update := os.Getenv("UPDATE_GOLDEN") != ""
	for _, c := range reg.Components() {
		if len(c.Manifest.Examples) == 0 {
			t.Errorf("component %s has no examples", c.Manifest.Name)
		}
		for _, ex := range c.Manifest.Examples {
			got, err := c.Render(ex.Props)
			if err != nil {
				t.Errorf("%s example %s: %v", c.Manifest.Name, ex.Name, err)
				continue
			}
			if !strings.Contains(string(got), `data-component="`+c.Manifest.Name+`"`) {
				t.Errorf("%s example %s: output lacks data-component attribute", c.Manifest.Name, ex.Name)
			}
			path := filepath.Join("..", "..", "design", c.Dir(), ex.File)
			if update {
				os.MkdirAll(filepath.Dir(path), 0o755)
				if err := os.WriteFile(path, []byte(string(got)+"\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				continue
			}
			want, err := c.ReadFile(ex.File)
			if err != nil {
				t.Errorf("%s example %s: missing %s (run make golden)", c.Manifest.Name, ex.Name, ex.File)
				continue
			}
			if strings.TrimSpace(string(want)) != strings.TrimSpace(string(got)) {
				t.Errorf("%s example %s differs from %s\n got: %s\nwant: %s", c.Manifest.Name, ex.Name, ex.File, got, want)
			}
		}
	}
}

func TestRenderRejectsBadProps(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}
	if _, err := reg.Render("button", map[string]any{}); err == nil {
		t.Error("expected an error for a button without a label")
	}
	if _, err := reg.Render("button", map[string]any{"label": "x", "bogus": 1}); err == nil {
		t.Error("expected an error for an unknown prop")
	}
	if _, err := reg.Render("heading", map[string]any{"text": "x", "level": 9}); err == nil {
		t.Error("expected an error for heading level 9")
	}
	if _, err := reg.Render("nope", nil); err == nil {
		t.Error("expected an error for an unknown component")
	}
}

func TestRenderEscapes(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}
	out, err := reg.Render("text", map[string]any{"content": "<script>alert(1)</script>"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "<script>") {
		t.Errorf("content was not escaped: %s", out)
	}
	out, err = reg.Render("link", map[string]any{"href": "javascript:alert(1)", "label": "x"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "javascript:") {
		t.Errorf("unsafe href was not neutralised: %s", out)
	}
}
