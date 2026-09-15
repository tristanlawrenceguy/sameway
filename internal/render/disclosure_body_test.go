package render_test

import (
	"regexp"
	"testing"
)

// TestDisclosureBodyHasExplicitColor checks that .sw-disclosure__body declares
// an explicit color token, so body content text always has a known contrast
// ratio against its background regardless of what the summary or page sets as
// the inherited foreground colour. Without this, axe-core cannot reliably
// compute contrast for body text and may flag the element. Follow-up to backlog 0161.
func TestDisclosureBodyHasExplicitColor(t *testing.T) {
	reg := builtins(t)
	c, ok := reg.Get("disclosure")
	if !ok {
		t.Fatal("disclosure component not found in registry")
	}

	ruleRe := regexp.MustCompile(`(?s)\.sw-disclosure__body\s*\{([^}]*)\}`)
	matches := ruleRe.FindStringSubmatch(c.CSS)
	if len(matches) < 2 {
		t.Fatal("disclosure: could not find .sw-disclosure__body rule block in CSS")
	}

	ruleBody := matches[1]
	colRe := regexp.MustCompile(`color\s*:\s*(var\(--sw-color-fg(?:-muted)?\))`)
	if !colRe.MatchString(ruleBody) {
		t.Errorf("disclosure: .sw-disclosure__body has no explicit color token; expected var(--sw-color-fg-muted) or var(--sw-color-fg), got %q", colRe.FindStringSubmatch(ruleBody))
	}
}
