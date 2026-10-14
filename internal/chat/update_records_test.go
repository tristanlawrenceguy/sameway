package chat_test

import (
	"strings"
	"testing"
)

// Moving the late tasks is one call, one entry in the log and one Undo,
// which puts every one back as it was.
func TestUpdateRecordsIsOneChangeAndOneUndo(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	var ids []string
	for _, title := range []string{"Email Sarah", "Book the car", "Call mum"} {
		rec, _ := svc.Store.Create("task", map[string]any{"title": title, "due": "2026-09-28"})
		ids = append(ids, rec.ID)
	}
	text := run(t, svc, "update_records", map[string]any{"type": "task", "ids": ids, "fields": map[string]any{"due": "2026-10-19"}})
	if !strings.Contains(text, "updated 3 tasks, as one change") || !strings.Contains(text, "Book the car") {
		t.Errorf("the result says the three, as one change: %s", text)
	}
	for _, id := range ids {
		if rec, _ := svc.Store.Get("task", id); !strings.HasPrefix(rec.Fields["due"].(string), "2026-10-19") {
			t.Errorf("%s moved: %v", id, rec.Fields["due"])
		}
	}
	made := entries(t, svc, "updated")
	if len(made) != 1 {
		t.Fatalf("one entry in the log, got %d", len(made))
	}
	if _, _, err := svc.Undo(made[0].ID); err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		if rec, _ := svc.Store.Get("task", id); !strings.HasPrefix(rec.Fields["due"].(string), "2026-09-28") {
			t.Errorf("one Undo puts %s back: %v", id, rec.Fields["due"])
		}
	}
}

// One record that cannot be changed and none is: those written before it
// are put back, and nothing is logged.
func TestUpdateRecordsAllOrNone(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	rec, _ := svc.Store.Create("task", map[string]any{"title": "Email Sarah", "due": "2026-09-28"})
	raw := `{"type":"task","ids":["` + rec.ID + `","nosuchtask000000"],"fields":{"due":"2026-10-19"}}`
	text, isErr := svc.Call("update_records", []byte(raw))
	if !isErr || !strings.Contains(text, "Nothing was changed: ids[1]") {
		t.Errorf("refused, naming the one: %v %s", isErr, text)
	}
	if got, _ := svc.Store.Get("task", rec.ID); !strings.HasPrefix(got.Fields["due"].(string), "2026-09-28") {
		t.Errorf("the first is put back: %v", got.Fields["due"])
	}
	if made := entries(t, svc, "updated"); len(made) != 0 {
		t.Errorf("nothing is logged, %d entries", len(made))
	}
}
