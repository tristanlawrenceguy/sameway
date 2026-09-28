package server_test

import (
	"strings"
	"testing"
)

// TestProseSourceTextareaStartsHidden verifies acceptance items 1 and 2 of
// task 0424: the inline edit form on a note's detail page must show exactly one
// Body textarea when "Edit block" is clicked. The duplicate Body textbox comes
// from both the contentEditable editor div AND the Markdown source textarea
// being visible at startup. Hiding the source textarea initially gives the user
// exactly one editable field (the rich text editor).
func TestProseSourceTextareaStartsHidden(t *testing.T) {
	src := readProseEditScript(t)

	// The source textarea must be set to hidden=true right after creation so that
	// only the contentEditable editor div is visible when "Edit block" opens.
	// Without this, both are visible — two Body textboxes with identical labels.
	sourceCreateIdx := strings.Index(src, `var source = document.createElement("textarea")`)
	if sourceCreateIdx < 0 {
		t.Fatal("swProseField must create a textarea element for the Markdown source")
	}

	var afterSource string
	end := sourceCreateIdx + 600
	if end > len(src) {
		end = len(src)
	}
	afterSource = src[sourceCreateIdx:end]

	if !strings.Contains(afterSource, "source.hidden = true") &&
		!strings.Contains(afterSource, "source.hidden=true") {
		t.Error("11-prose-edit.js: the source textarea must start hidden (source.hidden = true) so that only one Body field is visible;\n" +
			"without this both the contentEditable editor and the Markdown source textarea appear simultaneously,\n" +
			"creating two textboxes labeled \"Body\" — see backlog 0424")
	}

	// The hidden assignment must come AFTER creating the textarea but BEFORE appending it.
	sourceAppendIdx := strings.Index(src, "wrap.appendChild(source)")
	if sourceAppendIdx < 0 {
		t.Fatal("swProseField must append the source textarea to the wrap")
	}

	hiddenTrueIdx := strings.Index(afterSource, "source.hidden = true")
	if hiddenTrueIdx < 0 {
		// Try without spaces
		hiddenTrueIdx = strings.Index(afterSource, "source.hidden=true")
	}
	if hiddenTrueIdx >= 0 && sourceAppendIdx < (strings.Index(src, afterSource)+hiddenTrueIdx) {
		t.Error("11-prose-edit.js: source.hidden = true must appear before wrap.appendChild(source);\n" +
			"hiding the textarea after appending it means both editors are visible briefly")
	}

	// The toggle button should start showing "Markdown" (switching TO markdown mode).
	if !strings.Contains(src, `toggle.textContent = "Markdown"`) {
		t.Error("11-prose-edit.js: expected the initial toggle label to be \"Markdown\"")
	}

	// When toggling from rich text to Markdown, source must become visible.
	if !strings.Contains(src, "source.hidden = false") &&
		!strings.Contains(src, "source.hidden=false") {
		t.Error("11-prose-edit.js: the toggle handler must set source.hidden = false when switching to Markdown mode\n" +
			"so the textarea becomes visible for editing raw markdown")
	}

	// When toggling back from Markdown to rich text, source must become hidden again.
	if !strings.Contains(src, "source.hidden = true") &&
		!strings.Contains(src, "source.hidden=true") {
		t.Error("11-prose-edit.js: the toggle handler must set source.hidden = true when switching to Rich text mode\n" +
			"so the textarea is hidden and only the editor div remains visible")
	}

	// The check in the toggle click handler should use source.hidden (not source.disabled).
	if strings.Contains(src, "if (source.disabled)") {
		t.Error("11-prose-edit.js: the toggle must check source.hidden to decide which mode we are in,\n" +
			"not source.disabled — the hidden attribute is what controls visibility")
	}

	// The editor div and toolbar bar should be hidden when toggling to Markdown.
	if !strings.Contains(src, "editor.hidden = ") &&
		!strings.Contains(src, "editor.hidden=") {
		t.Error("11-prose-edit.js: the toggle handler must hide the editor div when switching to Markdown mode")
	}

	if !strings.Contains(src, "bar.hidden = true") &&
		!strings.Contains(src, "bar.hidden=true") {
		t.Error("11-prose-edit.js: the toggle handler must hide the toolbar bar when switching to Markdown mode\n" +
			"so only one field is visible at a time — see backlog 0424")
	}

	// The hidden inputs should be disabled while in Markdown source mode (source is sending data).
	if !strings.Contains(src, "html.disabled = true") &&
		!strings.Contains(src, "html.disabled=true") {
		t.Error("11-prose-edit.js: the toggle handler must disable the html hidden input when in Markdown mode\n" +
			"so raw markdown is sent via prop- instead of html-")
	}

	if !strings.Contains(src, "html.disabled = false") &&
		!strings.Contains(src, "html.disabled=false") {
		t.Error("11-prose-edit.js: the toggle handler must re-enable the html hidden input when switching back to Rich text mode")
	}
}
