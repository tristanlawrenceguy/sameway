package chat_test

import (
	"strings"
	"testing"
)

// A list found says how each record's words start, so which notes are
// about the garden is told from the list, not from reading each in full.
func TestFoundRecordsSayHowTheirWordsStart(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	svc.Store.Create("note", map[string]any{"title": "Compost bins", "body": "Turn the heap monthly, and keep it damp."})
	svc.Store.Create("note", map[string]any{"title": "Tax return"})
	text := run(t, svc, "find_records", map[string]any{"type": "note"})
	if !strings.Contains(text, "Compost bins") || !strings.Contains(text, "Turn the heap monthly, and keep it damp.") {
		t.Errorf("the words' start is on its line: %s", text)
	}
}

// One tool makes one record or several, and changes one or several.
func TestOneToolForOneOrSeveral(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	for _, tl := range svc.Tools() {
		if tl.Name == "create_records" || tl.Name == "update_records" {
			t.Errorf("%s is not offered: create_record and update_record do both", tl.Name)
		}
	}
	text := run(t, svc, "create_record", map[string]any{"type": "task", "records": []map[string]any{{"title": "Call mum"}, {"title": "Book the car"}}})
	if !strings.Contains(text, "created 2 tasks") {
		t.Fatalf("several are made in one call: %s", text)
	}
	text = run(t, svc, "update_record", map[string]any{"type": "task", "where": []string{"done=false"}, "fields": map[string]any{"done": true}})
	if !strings.Contains(text, "updated 2 tasks") {
		t.Errorf("and changed in one call: %s", text)
	}
}
