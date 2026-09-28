package server_test

import (
	"os"
	"strings"
	"testing"
)

// TestEditJSDoesNotInterceptHashAnchors verifies acceptance item 3 of task
// 0303: design/base/08-edit.js must not intercept hash anchor clicks for scroll
// behavior. The browser handles native anchor navigation on its own; adding JS
// to prevent default or manually scrolling would break the skip link and
// duplicate work the browser already does correctly (once tabindex="-1" is
// present on target articles).
func TestEditJSDoesNotInterceptHashAnchors(t *testing.T) {
	data, err := os.ReadFile("../../design/base/08-edit.js")
	if err != nil {
		t.Fatalf("read 08-edit.js: %v", err)
	}

	src := string(data)

	// These patterns would indicate the script intercepts hash navigation.
	for _, pattern := range []string{"hashchange", "scrollIntoView"} {
		if strings.Contains(src, pattern) {
			t.Errorf("08-edit.js must not contain %q — it should not interfere with native anchor scrolling;\nthe browser handles scroll-to-anchor natively when tabindex=\"-1\" is on the target", pattern)
		}
	}

	// A preventDefault on <a> elements would break skip-link navigation.
	// Only flag preventDefault that appears in context of href/hash/anchor
	// handling — unrelated uses (form submission, key handlers) are fine.
	lines := strings.Split(src, "\n")
	for i, line := range lines {
		if !strings.Contains(line, "preventDefault") {
			continue
		}
		// Look up to 10 preceding lines for hash/anchor context.
		start := 0
		if i > 10 {
			start = i - 10
		}
		context := strings.Join(lines[start:i], "\n")
		if strings.Contains(context, ".href") ||
			strings.Contains(context, "window.location.hash") ||
			strings.Contains(context, `getAttribute("href")`) ||
			strings.Contains(context, `querySelector("a`) {
			t.Errorf("08-edit.js line %d: preventDefault near anchor/hash navigation — may intercept skip-link behavior;\nline %d: %s", i+1, i+1, lines[i])
		}
	}
}
