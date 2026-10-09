package server_test

import (
	"os"
	"strings"
	"testing"
)

// TestBlockBarNeverCoversTheBlock: a canvas block's bar (Edit, Expand,
// Remove) is revealed by the pointer on the block. Floating over the
// block's corner, it lay on whatever control sat there, so a press meant
// for a search's button or Start voice mode in a narrow pane landed on
// Expand or Remove. It takes its own row under the content for everyone,
// not only where it is always shown.
func TestBlockBarNeverCoversTheBlock(t *testing.T) {
	data, err := os.ReadFile("../../design/base/05-quiet.css")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, ".sw-block > .sw-bar {") {
			continue // indented rules sit inside a media query
		}
		found = true
		if !strings.Contains(line, "position: static") {
			t.Errorf("a block's bar must be in the flow, not over the content: %s", line)
		}
	}
	if !found {
		t.Errorf("05-quiet.css has no top-level `.sw-block > .sw-bar { position: static ... }` rule; a block's bar would float over its controls")
	}
}
