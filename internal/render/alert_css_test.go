package render_test

// Tests for alert component CSS rules — specifically that .sw-alert__icon uses
// design tokens (not raw colours) for font-size and margin-right.

import (
	"regexp"
	"strings"
	"testing"
)

// TestAlertIconCSSRuleExists asserts that the alert component's compiled CSS
// contains a .sw-alert__icon rule block with token-based properties.
func TestAlertIconCSSRuleExists(t *testing.T) {
	reg := builtins(t)
	c, ok := reg.Get("alert")
	if !ok {
		t.Fatal("alert component not found in registry")
	}

	ruleRe := regexp.MustCompile(`(?s)\.sw-alert__icon\s*\{([^}]*)\}`)
	matches := ruleRe.FindStringSubmatch(c.CSS)
	if len(matches) < 2 {
		t.Fatal("alert: could not find .sw-alert__icon rule block in CSS")
	}

	ruleBody := matches[1]

	fsRe := regexp.MustCompile(`font-size\s*:\s*(var\(--sw-size-text-[^)]+\))`)
	if !fsRe.MatchString(ruleBody) {
		t.Errorf("alert: .sw-alert__icon has no font-size using --sw-size-text-* token; got %q", fsRe.FindStringSubmatch(ruleBody))
	}

	mrRe := regexp.MustCompile(`margin-right\s*:\s*(var\(--sw-space-[^)]+\))`)
	if !mrRe.MatchString(ruleBody) {
		t.Errorf("alert: .sw-alert__icon has no margin-right using --sw-space-* token; got %q", mrRe.FindStringSubmatch(ruleBody))
	}
}

// TestAlertIconFontSizeToken asserts the CSS file content includes
// `font-size: var(--sw-size-text-`.
func TestAlertIconFontSizeToken(t *testing.T) {
	reg := builtins(t)
	c, ok := reg.Get("alert")
	if !ok {
		t.Fatal("alert component not found in registry")
	}

	ruleRe := regexp.MustCompile(`(?s)\.sw-alert__icon\s*\{([^}]*)\}`)
	matches := ruleRe.FindStringSubmatch(c.CSS)
	if len(matches) < 2 {
		t.Fatal("alert: could not find .sw-alert__icon rule block in CSS")
	}

	ruleBody := matches[1]
	if !strings.Contains(ruleBody, "font-size: var(--sw-size-text-") {
		t.Errorf("alert: .sw-alert__icon font-size does not use --sw-size-text-* token; got %q", ruleBody)
	}
}

// TestAlertIconMarginRightToken asserts the CSS file content includes
// `margin-right: var(--sw-space-`.
func TestAlertIconMarginRightToken(t *testing.T) {
	reg := builtins(t)
	c, ok := reg.Get("alert")
	if !ok {
		t.Fatal("alert component not found in registry")
	}

	ruleRe := regexp.MustCompile(`(?s)\.sw-alert__icon\s*\{([^}]*)\}`)
	matches := ruleRe.FindStringSubmatch(c.CSS)
	if len(matches) < 2 {
		t.Fatal("alert: could not find .sw-alert__icon rule block in CSS")
	}

	ruleBody := matches[1]
	if !strings.Contains(ruleBody, "margin-right: var(--sw-space-") {
		t.Errorf("alert: .sw-alert__icon margin-right does not use --sw-space-* token; got %q", ruleBody)
	}
}
