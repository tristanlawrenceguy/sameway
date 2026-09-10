package render_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render"
)

func writeComponent(t *testing.T, root, name, template string) {
	t.Helper()
	dir := filepath.Join(root, name)
	os.MkdirAll(filepath.Join(dir, "examples"), 0o755)
	manifest := `{"name":"` + name + `","version":"0.1.0","description":"test","props":{"type":"object","additionalProperties":false,"required":["text"],"properties":{"text":{"type":"string"}}},"a11y":{"role":"note","keyboard":[],"states":[],"wcag":{"target":"AA","notes":"test"}},"machine":{"selector":"[data-component=\"` + name + `\"]","identify":"text","operate":"none"},"examples":[{"name":"default","props":{"text":"Hi"},"file":"examples/default.html"}]}`
	os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644)
	os.WriteFile(filepath.Join(dir, "template.html"), []byte(template), 0o644)
	os.WriteFile(filepath.Join(dir, "style.css"), []byte(".sw-"+name+"{}"), 0o644)
}

// TestWorkspaceComponentsExtendAndOverride covers the two ways a person or
// agent adds their own component: a new name, or replacing a built-in.
func TestWorkspaceComponentsExtendAndOverride(t *testing.T) {
	root := t.TempDir()
	writeComponent(t, root, "callout", `<aside class="sw-callout" data-component="callout">{{.text}}</aside>`)
	writeComponent(t, root, "button", `<button class="custom" data-component="button">{{.text}}</button>`)

	reg := builtins(t)
	if err := reg.LoadDir(root, "workspace"); err != nil {
		t.Fatal(err)
	}
	c, ok := reg.Get("callout")
	if !ok || c.Source != "workspace" {
		t.Fatalf("callout not loaded from workspace: %+v", c)
	}
	out, err := reg.Render("callout", map[string]any{"text": "Hello"})
	if err != nil || !strings.Contains(string(out), "<aside") {
		t.Errorf("callout render: %v %s", err, out)
	}
	b, _ := reg.Get("button")
	if b.Source != "workspace" {
		t.Errorf("built-in button should be overridden by the workspace copy")
	}
	out, err = reg.Render("button", map[string]any{"text": "Go"})
	if err != nil || !strings.Contains(string(out), `class="custom"`) {
		t.Errorf("override not used: %v %s", err, out)
	}
	if !strings.Contains(reg.CSS(), ".sw-callout{}") {
		t.Errorf("workspace component css missing from bundle")
	}
}

func TestLoadDirMissingIsFine(t *testing.T) {
	reg := render.New()
	if err := reg.LoadDir(filepath.Join(t.TempDir(), "nope"), "workspace"); err != nil {
		t.Fatalf("missing components dir should not be an error: %v", err)
	}
}

func TestBrokenComponentReportsWhichFile(t *testing.T) {
	root := t.TempDir()
	writeComponent(t, root, "broken", `<div>{{.text`)
	reg := render.New()
	err := reg.LoadDir(root, "workspace")
	if err == nil || !strings.Contains(err.Error(), "broken") || !strings.Contains(err.Error(), "template.html") {
		t.Errorf("expected an error naming broken/template.html, got %v", err)
	}
	root2 := t.TempDir()
	os.MkdirAll(filepath.Join(root2, "mismatch"), 0o755)
	os.WriteFile(filepath.Join(root2, "mismatch", "manifest.json"), []byte(`{"name":"other","props":{"type":"object"}}`), 0o644)
	os.WriteFile(filepath.Join(root2, "mismatch", "template.html"), []byte(`<div></div>`), 0o644)
	if err := render.New().LoadDir(root2, "workspace"); err == nil || !strings.Contains(err.Error(), "must match the folder name") {
		t.Errorf("expected folder/name mismatch error, got %v", err)
	}
}
