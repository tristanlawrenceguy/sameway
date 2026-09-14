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
