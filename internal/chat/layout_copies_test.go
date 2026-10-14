package chat_test

import (
	"strings"
	"testing"
)

// A block that lists records as words, each with its link, is said in
// Layout now as a copy that goes stale, with a collection to show them
// instead; a block with a link or two in its words is not.
func TestLayoutNowSaysABlockThatCopiesRecords(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	var items []any
	for _, title := range []string{"Dentist", "Dry cleaning", "Seeds"} {
		rec, err := svc.Store.Create("note", map[string]any{"title": title})
		if err != nil {
			t.Fatal(err)
		}
		items = append(items, title+" — /t/note/"+rec.ID)
	}
	run(t, svc, "add_component", map[string]any{"component": "text", "props": map[string]any{"content": "See " + items[0].(string)}, "span": 12})
	text := run(t, svc, "add_component", map[string]any{"component": "list", "props": map[string]any{"label": "Unfinished notes", "items": items}, "span": 12})
	if !strings.Contains(text, "(list) copies 3 notes as words") || !strings.Contains(text, "a collection block of note shows them") {
		t.Errorf("the list of note links is said as a copy:\n%s", text)
	}
	if strings.Contains(text, "(text) copies") {
		t.Errorf("one link in a sentence is not a copy:\n%s", text)
	}
}
