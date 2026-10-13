package main

import "testing"

// A raw colour or a removed outline in a component's stylesheet fails,
// one problem each; tokens pass.
func TestComponentCSSUsesTokensOnly(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write(t, root, "ok/style.css", ".sw-ok { color: var(--sw-color-fg); }\n")
	write(t, root, "raw/style.css", ".sw-raw { color: #ff0000; background: rgba(0, 0, 0, 0.5); }\n")
	write(t, root, "bare/style.css", ".sw-bare:focus { outline: none; }\n")
	if p := checkComponentCSS("ok", root+"/ok/style.css"); len(p) != 0 {
		t.Errorf("tokens: %q", p)
	}
	if p := checkComponentCSS("raw", root+"/raw/style.css"); len(p) != 1 || !has(p, `raw colour "#ff0000"`) {
		t.Errorf("raw: %q", p)
	}
	if p := checkComponentCSS("bare", root+"/bare/style.css"); len(p) != 1 || !has(p, "removes the focus outline") {
		t.Errorf("bare: %q", p)
	}
}
