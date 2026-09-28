package server_test

import (
	"os"
	"strings"
	"testing"
)

// TestProseEditDOMOrderReachesBodyFirst: the prose editor puts the words
// first, so Tab reaches Body before anything that formats it (task 0401).
// Then the Markdown field it switches to, then one line under the words
// with the toolbar at its left and the switch to Markdown at its right,
// in that order, and the hidden inputs last.
func TestProseEditDOMOrderReachesBodyFirst(t *testing.T) {
	data, err := os.ReadFile("../../design/base/11-prose-edit.js")
	if err != nil {
		t.Fatalf("read 11-prose-edit.js: %v", err)
	}
	src := string(data)
	order := []string{
		"wrap.appendChild(editor)", "wrap.appendChild(source)", "wrap.appendChild(foot)",
		"wrap.appendChild(html)", "wrap.appendChild(lvl)",
	}
	last := -1
	for _, step := range order {
		at := strings.Index(src, step)
		if at < 0 {
			t.Fatalf("swProseField must do %s", step)
		}
		if at < last {
			t.Errorf("%s comes too early: the order is %s", step, strings.Join(order, ", "))
		}
		last = at
	}
	bar, sw := strings.Index(src, "foot.appendChild(bar)"), strings.Index(src, "foot.appendChild(switcher)")
	if bar < 0 || sw < 0 || bar > sw {
		t.Error("the line under the words holds the toolbar, then the switch to Markdown")
	}
}
