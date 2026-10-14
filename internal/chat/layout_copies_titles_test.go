package chat_test

import (
	"strings"
	"testing"
)

// A text block of three task titles, one to a line with no links, copies
// the tasks as a list of their links would; three lines that are no
// record's title are only words.
func TestLayoutNowSaysABlockThatCopiesRecordsByTitle(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	for _, title := range []string{"Pay rent", "Call the bank", "Book the car"} {
		svc.Store.Create("task", map[string]any{"title": title})
	}
	run(t, svc, "add_component", map[string]any{"component": "text", "props": map[string]any{"content": "- Walk more\n- Sleep early\n- Read a book"}, "span": 12})
	text := run(t, svc, "add_component", map[string]any{"component": "text", "props": map[string]any{"content": "Today:\n\n1. Pay rent\n2. [ ] call the bank\n3. [Book the car](/t/task)"}, "span": 12})
	if !strings.Contains(text, "(text) copies 3 tasks as words") {
		t.Errorf("the three task titles are said as a copy:\n%s", text)
	}
	if strings.Count(text, "copies") != 1 {
		t.Errorf("lines that are no record's title are only words:\n%s", text)
	}
}
