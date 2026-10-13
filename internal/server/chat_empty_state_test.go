package server_test

import (
	"regexp"
	"strings"
	"testing"
)

// An empty conversation says what to do next in one short line, not a
// second sentence about where things go (task 0155).
func TestChatBlockEmptyStateIsOneLine(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	a.Store.Create("note", map[string]any{"title": "Not brand new"}) // a brand new one is welcomed instead
	page := get(t, h, "/chat").Body.String()
	empty := regexp.MustCompile(`(?s)<[^>]*data-component="empty".*?</p>`).FindString(page)
	if !strings.Contains(empty, "Ask for anything.") {
		t.Errorf("an empty conversation says Ask for anything.; its empty state: %q\nbody: %s", empty, truncate(page))
	}
	if strings.Contains(page, "What you ask for appears on the canvas") {
		t.Error("one line, not a second sentence about the canvas")
	}
}
