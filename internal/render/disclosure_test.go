package render_test

import (
	"regexp"
	"strings"
	"testing"
)

// TestDisclosureSummaryHasExplicitBackground checks that the disclosure
// component's .sw-disclosure__summary rule declares an explicit background
// matching var(--sw-color-bg), so axe-core never sees a transparent summary
// element with insufficient contrast against its parent (backlog 0161).
func TestDisclosureSummaryHasExplicitBackground(t *testing.T) {
	reg := builtins(t)
	c, ok := reg.Get("disclosure")
	if !ok {
		t.Fatal("disclosure component not found in registry")
	}

	// Extract the .sw-disclosure__summary rule block from the CSS.
	ruleRe := regexp.MustCompile(`(?s)\.sw-disclosure__summary\s*\{([^}]*)\}`)
	matches := ruleRe.FindStringSubmatch(c.CSS)
	if len(matches) < 2 {
		t.Fatal("disclosure: could not find .sw-disclosure__summary rule block in CSS")
	}

	ruleBody := matches[1]
	if !strings.Contains(ruleBody, "background:") && !strings.Contains(ruleBody, "background :") {
		t.Error("disclosure: .sw-disclosure__summary has no background property; axe-core may flag insufficient contrast when the summary inherits a transparent parent")
	}

	// The value must be --sw-color-bg (a design token), not a raw colour.
	bgRe := regexp.MustCompile(`background\s*:\s*(var\(--sw-color-bg\))`)
	if !bgRe.MatchString(ruleBody) {
		t.Errorf("disclosure: .sw-disclosure__summary background is %q; expected var(--sw-color-bg)", bgRe.FindStringSubmatch(ruleBody))
	}
}

// TestDisclosureSummaryHasExplicitColor checks that the disclosure component's
// .sw-disclosure__summary rule declares an explicit color matching
// var(--sw-color-fg-muted), so axe-core has a stable reference for contrast
// calculations on child elements (backlog 0161).
func TestDisclosureSummaryHasExplicitColor(t *testing.T) {
	reg := builtins(t)
	c, ok := reg.Get("disclosure")
	if !ok {
		t.Fatal("disclosure component not found in registry")
	}

	ruleRe := regexp.MustCompile(`(?s)\.sw-disclosure__summary\s*\{([^}]*)\}`)
	matches := ruleRe.FindStringSubmatch(c.CSS)
	if len(matches) < 2 {
		t.Fatal("disclosure: could not find .sw-disclosure__summary rule block in CSS")
	}

	ruleBody := matches[1]
	colRe := regexp.MustCompile(`color\s*:\s*(var\(--sw-color-fg-muted\))`)
	if !colRe.MatchString(ruleBody) {
		t.Errorf("disclosure: .sw-disclosure__summary has no explicit color token; expected var(--sw-color-fg-muted), got %q", colRe.FindStringSubmatch(ruleBody))
	}
}

// TestDisclosurePseudoElementUsesToken checks that the disclosure component's
// .sw-disclosure__summary::before pseudo-element uses an explicit token value
// instead of currentColor, which can cause axe-core to flag a colour-contrast
// failure because computed contrast depends on inherited color at hover state
// or browser-specific rendering (backlog 0161).
func TestDisclosurePseudoElementUsesToken(t *testing.T) {
	reg := builtins(t)
	c, ok := reg.Get("disclosure")
	if !ok {
		t.Fatal("disclosure component not found in registry")
	}

	ruleRe := regexp.MustCompile(`(?s)\.sw-disclosure__summary::before\s*\{([^}]*)\}`)
	matches := ruleRe.FindStringSubmatch(c.CSS)
	if len(matches) < 2 {
		t.Fatal("disclosure: could not find .sw-disclosure__summary::before rule block in CSS")
	}

	ruleBody := matches[1]
	if strings.Contains(ruleBody, "currentColor") {
		t.Errorf("disclosure: .sw-disclosure__summary::before uses currentColor which can cause contrast failures; replace with var(--sw-color-fg-muted)")
	}

	borderRe := regexp.MustCompile(`border-right\s*:\s*2px solid (var\(--sw-color-fg-muted\))`)
	if !borderRe.MatchString(ruleBody) {
		t.Errorf("disclosure: .sw-disclosure__summary::before border-right does not use var(--sw-color-fg-muted); got %q", borderRe.FindStringSubmatch(ruleBody))
	}

	borderBottomRe := regexp.MustCompile(`border-bottom\s*:\s*2px solid (var\(--sw-color-fg-muted\))`)
	if !borderBottomRe.MatchString(ruleBody) {
		t.Errorf("disclosure: .sw-disclosure__summary::before border-bottom does not use var(--sw-color-fg-muted); got %q", borderBottomRe.FindStringSubmatch(ruleBody))
	}
}

// TestDisclosureCountContrast checks that the .sw-disclosure__count element has
// both an explicit color and a background declared with token values, so the
// count badge text always meets contrast requirements against its background.
func TestDisclosureCountContrast(t *testing.T) {
	reg := builtins(t)
	c, ok := reg.Get("disclosure")
	if !ok {
		t.Fatal("disclosure component not found in registry")
	}

	ruleRe := regexp.MustCompile(`(?s)\.sw-disclosure__count\s*\{([^}]*)\}`)
	matches := ruleRe.FindStringSubmatch(c.CSS)
	if len(matches) < 2 {
		t.Fatal("disclosure: could not find .sw-disclosure__count rule block in CSS")
	}

	ruleBody := matches[1]

	colRe := regexp.MustCompile(`color\s*:\s*(var\(--sw-color-fg-muted\))`)
	if !colRe.MatchString(ruleBody) {
		t.Errorf("disclosure: .sw-disclosure__count has no explicit color token; expected var(--sw-color-fg-muted), got %q", colRe.FindStringSubmatch(ruleBody))
	}

	bgRe := regexp.MustCompile(`background\s*:\s*(var\(--sw-color-bg-muted\))`)
	if !bgRe.MatchString(ruleBody) {
		t.Errorf("disclosure: .sw-disclosure__count has no explicit background token; expected var(--sw-color-bg-muted), got %q", bgRe.FindStringSubmatch(ruleBody))
	}
}

// TestDisclosureBodyHasExplicitBackground checks that the disclosure component's
// .sw-disclosure__body rule declares an explicit background matching
// var(--sw-color-bg), so axe-core never sees a transparent body with insufficient
// contrast when nested inside other elements.
func TestDisclosureBodyHasExplicitBackground(t *testing.T) {
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
	bgRe := regexp.MustCompile(`background\s*:\s*(var\(--sw-color-bg\))`)
	if !bgRe.MatchString(ruleBody) {
		t.Errorf("disclosure: .sw-disclosure__body has no explicit background token; expected var(--sw-color-bg), got %q", bgRe.FindStringSubmatch(ruleBody))
	}
}
