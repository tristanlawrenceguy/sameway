package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// A list filtered to nothing says nothing matched and the way back to all
// of them, not that there are none yet, when there are.
func TestAFilteredListWithNoMatchesIsNotAFirstUse(t *testing.T) {
	_, h := newApp(t)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Order compost", "done": true}), http.StatusCreated)
	body := get(t, h, "/t/task?where=done%3Dfalse").Body.String()
	if strings.Contains(body, "No tasks yet") {
		t.Errorf("tasks exist; the list must not say there are none yet\n%s", truncate(body))
	}
	if !strings.Contains(body, "No matching tasks") || !strings.Contains(body, "Nothing is not done. Try fewer conditions, or") || !strings.Contains(body, `href="/t/task">see all tasks</a>`) {
		t.Errorf("the list says nothing matched, what was looked for, and the way to all of them\n%s", truncate(body))
	}
	if first := get(t, h, "/t/note").Body.String(); !strings.Contains(first, "No notes yet") {
		t.Errorf("an unfiltered empty list is still the first-use words")
	}
}
