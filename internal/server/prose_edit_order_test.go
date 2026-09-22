package server_test

import (
	"os"
	"strings"
	"testing"
)

// TestProseEditDOMOrderReachesBodyFirst verifies acceptance items 1 and 2 of
// task 0401: the inline edit form must place the Body editor before the
// formatting toolbar and Markdown toggle button in DOM order, so Tab reaches
// Body as one of the first tab stops.
func TestProseEditDOMOrderReachesBodyFirst(t *testing.T) {
	data, err := os.ReadFile("../../design/base/11-prose-edit.js")
	if err != nil {
		t.Fatalf("read 11-prose-edit.js: %v", err)
	}

	src := string(data)

	// The DOM construction must append the editor (the contentEditable div)
	// before the toolbar bar and the Markdown switcher, so Tab reaches Body
	// first. We check that wrap.appendChild(editor) appears BEFORE
	// wrap.appendChild(bar) in the source code.
	editorIdx := strings.Index(src, "wrap.appendChild(editor)")
	barIdx := strings.Index(src, "wrap.appendChild(bar)")

	if editorIdx < 0 {
		t.Fatal("swProseField must append the editor div to the wrap; without it Body is not editable")
	}
	if barIdx < 0 {
		t.Fatal("swProseField must append the toolbar bar to the wrap; without it the formatting controls are missing")
	}

	if editorIdx > barIdx {
		t.Errorf("the editor div (Body field) must be appended before the toolbar bar in DOM order;\nthis makes Body reachable as one of the first Tab stops.\nCurrent order: appendChild(bar) at line %d, appendChild(editor) at line %d",
			strings.Count(src[:barIdx], "\n")+1, strings.Count(src[:editorIdx], "\n")+1)
	}

	// The Markdown toggle button (switcher) must also come after the editor,
	// so Tab reaches Body before reaching it.
	switcherIdx := strings.Index(src, "wrap.appendChild(switcher)")
	if switcherIdx < 0 {
		t.Fatal("swProseField must append the Markdown switcher div to the wrap")
	}

	if editorIdx > switcherIdx {
		t.Errorf("the editor div (Body field) must be appended before the Markdown toggle button in DOM order;\nthis keeps Body reachable as one of the first Tab stops.\nCurrent order: appendChild(switcher) at line %d, appendChild(editor) at line %d",
			strings.Count(src[:switcherIdx], "\n")+1, strings.Count(src[:editorIdx], "\n")+1)
	}

	// The toolbar bar must come after the Markdown switcher in DOM order
	// (so it is last among these three), putting formatting controls at the
	// end of tab order for this field.
	if barIdx < switcherIdx {
		t.Errorf("the toolbar bar should be appended after the Markdown toggle button;\nthis keeps formatting controls out of the way of Body access.\nCurrent order: appendChild(bar) at line %d, appendChild(switcher) at line %d",
			strings.Count(src[:barIdx], "\n")+1, strings.Count(src[:switcherIdx], "\n")+1)
	}

	// Source textarea must come before the switcher (it is a field the person
	// sees when toggling to Markdown).
	sourceIdx := strings.Index(src, "wrap.appendChild(source)")
	if sourceIdx < 0 {
		t.Fatal("swProseField must append the source textarea to the wrap")
	}

	if editorIdx > sourceIdx {
		t.Errorf("the editor div must be appended before the source textarea;\nBody should be the first tabbable element in this field.\nCurrent order: appendChild(source) at line %d, appendChild(editor) at line %d",
			strings.Count(src[:sourceIdx], "\n")+1, strings.Count(src[:editorIdx], "\n")+1)
	}

	if sourceIdx > switcherIdx {
		t.Errorf("the source textarea must be appended before the Markdown toggle button;\na field should come before its toggle.\nCurrent order: appendChild(switcher) at line %d, appendChild(source) at line %d",
			strings.Count(src[:switcherIdx], "\n")+1, strings.Count(src[:sourceIdx], "\n")+1)
	}

	// Hidden inputs (html and lvl) should come last — they are not tabbable.
	htmlIdx := strings.Index(src, "wrap.appendChild(html)")
	lvlIdx := strings.Index(src, "wrap.appendChild(lvl)")

	if htmlIdx < 0 {
		t.Fatal("swProseField must append the hidden html input to the wrap")
	}
	if lvlIdx < 0 {
		t.Fatal("swProseField must append the hidden lvl input to the wrap")
	}

	if barIdx > htmlIdx || switcherIdx > htmlIdx {
		t.Errorf("hidden inputs should come after visible controls;\nthey are not tabbable and belong at the end of DOM order.\nCurrent order: appendChild(html) at line %d",
			strings.Count(src[:htmlIdx], "\n")+1)
	}

	if barIdx > lvlIdx || switcherIdx > lvlIdx {
		t.Errorf("hidden inputs should come after visible controls;\nthey are not tabbable and belong at the end of DOM order.\nCurrent order: appendChild(lvl) at line %d",
			strings.Count(src[:lvlIdx], "\n")+1)
	}
}
