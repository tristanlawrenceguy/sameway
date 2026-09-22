package server_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestProseSourceNotHiddenOrDisabledAtStartup verifies acceptance items 2 and 3 of
// task 0412: the source textarea must not start hidden or disabled so that a
// keyboard user can reach it via Tab from the Markdown button (or any previous
// focusable element). The fix is in design/base/11-prose-edit.js.
func TestProseSourceNotHiddenOrDisabledAtStartup(t *testing.T) {
	src := readProseEditScript(t)

	// Acceptance items 2 and 3: the source textarea must NOT be set to
	// hidden=true or disabled=true at startup, because those attributes
	// remove it from the tab order and prevent keyboard focus entirely.
	sourceCreateIdx := strings.Index(src, "var source = document.createElement(\"textarea\")")
	if sourceCreateIdx < 0 {
		t.Fatal("swProseField must create a textarea element for the Markdown source")
	}

	var afterSource string
	if idx := strings.Index(src[sourceCreateIdx:], "\n\n"); idx >= 0 {
		afterSource = src[sourceCreateIdx : sourceCreateIdx+idx]
	} else {
		end := sourceCreateIdx + 800
		if end > len(src) {
			end = len(src)
		}
		afterSource = src[sourceCreateIdx:end]
	}

	if strings.Contains(afterSource, "source.hidden = true") {
		t.Error("11-prose-edit.js: source.hidden = true at startup removes the Body textarea from the tab order;\n" +
			"a keyboard user cannot reach it via Tab. Remove this assignment so the textarea is visible and focusable by default.")
	}

	if strings.Contains(afterSource, "source.disabled = true") {
		t.Error("11-prose-edit.js: source.disabled = true at startup prevents keyboard focus on the Body textarea;\n" +
			"a keyboard user cannot reach it via Tab. Remove this assignment so the textarea is always enabled.")
	}

	// The initial setup must create the textarea with no hidden/disabled overrides.
	// Lines 59-66 of swProseField should only set className, name, rows, value, and aria-labelledby.

	// The first block of assignments for `source` should NOT include hidden or disabled.
	if strings.Contains(afterSource, "hidden") || strings.Contains(afterSource, "disabled") {
		t.Errorf("11-prose-edit.js: the source textarea's initial setup must not set hidden or disabled;\n"+
			"found in these first assignments:\n%s", afterSource)
	}

	// The Markdown toggle switcher button itself must still be focusable (it is).
	if !strings.Contains(src, `toggle.textContent = "Markdown"`) {
		t.Error("11-prose-edit.js: expected the toggle to say \"Markdown\" when in rich text mode")
	}
}

// TestProseToggleUsesHiddenOnly verifies that the Markdown/rich-text toggle
// handlers only use source.hidden (not source.disabled) for visibility, because
// source.disabled is no longer set at startup and must not be used to hide.
func TestProseToggleUsesHiddenOnly(t *testing.T) {
	src := readProseEditScript(t)

	// In the click handler, switching TO markdown mode should only set hidden=false.
	toggleToMarkdown := strings.Index(src, "source.hidden = source.disabled")
	if toggleToMarkdown >= 0 {
		t.Errorf("11-prose-edit.js: the toggle handlers still reference source.disabled;\n" +
			"since source is no longer disabled at startup, use source.hidden only.\n" +
			"This assignment should read just: source.hidden = false; (for markdown mode)\nor:  source.hidden = true; (for rich text mode)")
	}

	// When switching TO markdown mode (source becomes visible), hidden must go to false.
	if !strings.Contains(src, "source.hidden = false") {
		t.Error("11-prose-edit.js: when toggling to Markdown view, source.hidden must be set to false\n" +
			"so the textarea becomes visible and Tab-reachable")
	}

	// When switching TO rich text mode (source is hidden again), hidden must go to true.
	if !strings.Contains(src, "source.hidden = true") {
		t.Error("11-prose-edit.js: when toggling back to rich text view, source.hidden must be set to true\n" +
			"so the textarea is hidden but remains enabled for next time")
	}

	// The check on whether we're in markdown mode should use hidden (not disabled).
	if strings.Contains(src, "if (source.disabled)") {
		t.Error("11-prose-edit.js: the toggle click handler must check source.hidden instead of\n" +
			"source.disabled to decide which direction to switch; disabled is no longer used")
	}
}

func readProseEditScript(t *testing.T) string {
	t.Helper()
	root := findRepoRoot(t)
	path := filepath.Join(root, "design", "base", "11-prose-edit.js")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read 11-prose-edit.js: %v", err)
	}
	return string(data)
}
