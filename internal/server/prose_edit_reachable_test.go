package server_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestProseSourceNotHiddenOrDisabledAtStartup was superseded by task 0424:
// the source textarea now starts hidden (source.hidden = true) so that only one
// Body field is visible at startup. The toggle button provides keyboard access
// to switch between rich text and Markdown source views. This test remains as a
// no-op placeholder since its original assertions conflict with task 0424's fix.
func TestProseSourceNotHiddenOrDisabledAtStartup(t *testing.T) {
	// Superseded by task 0424: source now starts hidden to prevent duplicate Body textboxes.
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
