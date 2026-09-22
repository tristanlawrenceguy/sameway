package server_test

import (
	"os"
	"strings"
	"testing"
)

// TestProseToolbarButtonsAreNonTabbable verifies acceptance item 3 of task
// 0401: all prose toolbar buttons must have tabIndex=-1 so Tab navigation
// passes over them entirely, making the Body textarea reachable without
// cycling through 17 formatting buttons first.
func TestProseToolbarButtonsAreNonTabbable(t *testing.T) {
	data, err := os.ReadFile("../../design/base/12-prose-tools.js")
	if err != nil {
		t.Fatalf("read 12-prose-tools.js: %v", err)
	}

	src := string(data)

	// The roving tabindex pattern (arrow-key navigation inside the toolbar)
	// dynamically sets tabIndex=0 on focused buttons — that is fine and
	// expected. What must NOT exist is a static assignment giving the first
	// button tabIndex=0 so Tab from outside reaches it before Body.
	if strings.Contains(src, "b.tabIndex = i === 0 ? 0 : -1") ||
		strings.Contains(src, `b.tabIndex=i===0?0:-1`) {
		t.Errorf("toolbar buttons must all get tabIndex=-1 so Tab passes over them;\nthe pattern 'i === 0 ? 0 : -1' makes the first button a tab stop before Body\nline: b.tabIndex = i === 0 ? 0 : -1")
	}

	// Every button created in TOOLS.forEach must receive tabIndex=-1.
	// The only place buttons are created is inside TOOLS.forEach, and each
	// one must get b.tabIndex = -1 unconditionally.
	if !strings.Contains(src, "b.tabIndex = -1") {
		t.Error("every toolbar button must have b.tabIndex = -1; without it Tab can land on a formatting button before Body")
	}

	// The roving tabindex for ArrowLeft/ArrowRight (lines ~149-159) is fine:
	// it sets tabIndex=0 dynamically when the user is inside the toolbar.
	// We only care that the initial assignment is -1 for all buttons.
	if !strings.Contains(src, "next.tabIndex = 0") && !strings.Contains(src, "items[at].tabIndex = -1") {
		t.Error("the arrow-key roving tabindex pattern must still exist; it dynamically sets tabIndex=0 on the focused button when ArrowLeft/ArrowRight is pressed inside the toolbar")
	}
}
