package server_test

import (
	"strings"
	"testing"
)

// A list's address that asks what the list does not read says so, and
// links to the list as asked; one it reads says nothing of the kind.
func TestAListSaysWhatItDidNotRead(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	a.Store.Create("task", map[string]any{"title": "Old errand", "due": "2020-01-01"})
	page := get(t, h, "/t/task?done=false&overdue=true").Body.String()
	if !strings.Contains(page, "This list does not read done=false or overdue in its address, so it shows every task.") {
		t.Errorf("the page says what it did not read: %s", truncate(page))
	}
	if !strings.Contains(page, `href="/t/task?where=done%3Dfalse&amp;where=due%3Ctoday"`) {
		t.Errorf("and links to the list as asked: %s", truncate(page))
	}
	if page := get(t, h, "/t/task?where=done%3Dfalse").Body.String(); strings.Contains(page, "does not read") {
		t.Error("a list that read all its address says nothing of it")
	}
}
