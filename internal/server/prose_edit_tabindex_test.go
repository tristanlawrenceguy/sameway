package server_test

import (
	"os"
	"strings"
	"testing"
)

// The formatting toolbar is one Tab stop, after the words it formats: the
// first button is reachable by Tab, the arrow keys move between the rest,
// and the Body field still comes first. Once every button was out of the
// Tab order, which kept Body first but left the toolbar unreachable by
// keyboard at all (WCAG 2.1.1).
func TestProseToolbarButtonsAreNonTabbable(t *testing.T) {
	tools, err := os.ReadFile("../../design/base/12-prose-tools.js")
	if err != nil {
		t.Fatal(err)
	}
	src := string(tools)
	if !strings.Contains(src, "b.tabIndex = buttons.length ? -1 : 0") {
		t.Error("the toolbar's first button is its one Tab stop")
	}
	if !strings.Contains(src, "next.tabIndex = 0") || !strings.Contains(src, "b.tabIndex = -1; });") {
		t.Error("the arrow keys move the one stop between the buttons")
	}
	edit, err := os.ReadFile("../../design/base/11-prose-edit.js")
	if err != nil {
		t.Fatal(err)
	}
	field := string(edit)
	if i, j := strings.Index(field, "wrap.appendChild(editor)"), strings.Index(field, "wrap.appendChild(foot)"); i < 0 || j < 0 || i > j {
		t.Error("the toolbar comes after the words in the page, so Tab reaches Body first")
	}
}
