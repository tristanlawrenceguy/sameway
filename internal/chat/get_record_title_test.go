package chat_test

import (
	"encoding/json"
	"testing"
)

// get_record says a record's title, the one find_records lists it by and
// its page is headed with: an entry's habit and how much, a note's title
// (crew findings 0562 and 0564).
func TestGetRecordSaysItsTitle(t *testing.T) {
	svc := newFullService(t)
	habit, err := svc.Store.Create("habit", map[string]any{"name": "Read", "unit": "minutes", "cadence": "day"})
	if err != nil {
		t.Fatal(err)
	}
	entry, err := svc.Store.Create("entry", map[string]any{"habit": habit.ID, "at": "2026-09-28T09:00", "amount": 25.0})
	if err != nil {
		t.Fatal(err)
	}
	note, err := svc.Store.Create("note", map[string]any{"title": "Call the dentist"})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ typ, id, want string }{{"entry", entry.ID, "Read: 25 minutes"}, {"note", note.ID, "Call the dentist"}} {
		args, _ := json.Marshal(map[string]any{"type": c.typ, "id": c.id})
		text, isErr := svc.Call("get_record", args)
		var out struct {
			Title  string         `json:"title"`
			Fields map[string]any `json:"fields"`
		}
		json.Unmarshal([]byte(text), &out)
		if isErr || out.Title != c.want || out.Fields == nil {
			t.Errorf("get_record on a %s says its title %q beside its fields: %s", c.typ, c.want, text)
		}
	}
}
