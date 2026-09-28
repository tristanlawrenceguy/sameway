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

// TestInlineEditFormBodyTextareaReachable verifies acceptance items 1 and 3 of
// task 0435: the Body textarea (contentEditable editor) in the inline edit form
// must have tabindex="0" set explicitly so keyboard Tab navigation reaches it.
func TestInlineEditFormBodyTextareaReachable(t *testing.T) {
	// --- Acceptance item 1: contentEditable editor div has tabindex="0" ---

	editorSrc := readProseEditScript(t)

	if !strings.Contains(editorSrc, `editor.setAttribute("tabindex", "0")`) &&
		!strings.Contains(editorSrc, `editor.setAttribute('tabindex', '0')`) {
		t.Error("11-prose-edit.js: the contentEditable editor div must have tabindex=\"0\" set explicitly\n" +
			"so keyboard Tab navigation reaches the Body field in the inline edit form.\n" +
			"Add: editor.setAttribute(\"tabindex\", \"0\") after setting aria-label")
	}

	// --- Acceptance item 3: multiline textareas also have tabindex="0" ---

	editSrc := readEditScript(t)

	if !strings.Contains(editSrc, "input.setAttribute(\"tabindex\", \"0\")") &&
		!strings.Contains(editSrc, `input.setAttribute('tabindex', '0')`) {
		t.Error("08-edit.js: multiline textareas created by field() must have tabindex=\"0\" set explicitly\n" +
			"so keyboard Tab navigation reaches non-markdown Body fields (action body, payload, etc.).\n" +
			"Add: input.setAttribute(\"tabindex\", \"0\") inside the if (multiline) block")
	}

	// The textarea must get tabindex="0" only when multiline is true — single-line inputs should be left alone.
	if !strings.Contains(editSrc, "if (multiline)") {
		t.Error("08-edit.js: field() must check for multiline to create textareas; without it non-markdown fields are not editable")
	}

	// The tabindex attribute must be set inside the multiline branch (after the textarea is created).
	multilineIdx := strings.Index(editSrc, "if (multiline)")
	tabIndexSetIdx := -1
	if idx := strings.Index(editSrc, `input.setAttribute("tabindex", "0")`); idx >= 0 {
		tabIndexSetIdx = idx
	} else if idx := strings.Index(editSrc, `input.setAttribute('tabindex', '0')`); idx >= 0 {
		tabIndexSetIdx = idx
	}
	if multilineIdx >= 0 && tabIndexSetIdx >= 0 && tabIndexSetIdx < multilineIdx {
		t.Errorf("the tabindex=\"0\" assignment must appear after the if (multiline) check\n"+
			"so single-line inputs are not given a redundant tabindex.\n"+
			"Current: tabindex set at line %d, if (multiline) starts at line %d",
			strings.Count(editSrc[:tabIndexSetIdx], "\n")+1, strings.Count(editSrc[:multilineIdx], "\n")+1)
	}
}
