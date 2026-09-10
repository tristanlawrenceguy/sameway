package tokens_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/tokens"
)

const sample = `{
  "$comment": "ignored",
  "color": { "bg": { "light": "#fff", "dark": "#000" }, "fg": { "light": "#111", "dark": "#eee" } },
  "space": { "1": "0.25rem" }
}`

func TestGenerateEmitsLightDarkAndToggle(t *testing.T) {
	css, err := tokens.Generate([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"--sw-color-bg: #fff;", "--sw-color-fg: #111;", "--sw-space-1: 0.25rem;",
		"@media (prefers-color-scheme: dark)", `:root:not([data-theme="light"])`, `:root[data-theme="dark"]`,
		"--sw-color-bg: #000;", "color-scheme: light dark;",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("css missing %q", want)
		}
	}
	if strings.Count(css, "--sw-color-bg: #000;") != 2 {
		t.Errorf("dark values should appear in both the media query and the explicit theme block")
	}
	again, _ := tokens.Generate([]byte(sample))
	if again != css {
		t.Errorf("output must be deterministic")
	}
}

func TestGenerateRejectsHalfThemedColours(t *testing.T) {
	_, err := tokens.Generate([]byte(`{"color":{"bg":{"light":"#fff"}}}`))
	if err == nil || !strings.Contains(err.Error(), "needs light and dark") {
		t.Errorf("expected light/dark error, got %v", err)
	}
	if _, err := tokens.Generate([]byte(`{"color":"nope"}`)); err == nil {
		t.Errorf("a non-object group must be rejected")
	}
	if _, err := tokens.Generate([]byte(`not json`)); err == nil {
		t.Errorf("bad JSON must be rejected")
	}
}
