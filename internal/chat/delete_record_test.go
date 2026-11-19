package chat_test

import (
	"strings"
	"testing"
)

// Deleting the done tasks is one call and one change; one Undo puts every
// one back with what it had.
func TestDeleteRecordIsOneChangeAndOneUndo(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	keep, _ := svc.Store.Create("task", map[string]any{"title": "Call mum"})
	var ids []string
	for _, title := range []string{"Email Sarah", "Book the car"} {
		rec, _ := svc.Store.Create("task", map[string]any{"title": title, "done": true})
		ids = append(ids, rec.ID)
	}
	text := run(t, svc, "delete_record", map[string]any{"type": "task", "where": []string{"done=true"}})
	if !strings.Contains(text, "deleted 2 tasks, as one change") || !strings.Contains(text, "Book the car") {
		t.Errorf("the result names the two, as one change: %s", text)
	}
	for _, id := range ids {
		if _, err := svc.Store.Get("task", id); err == nil {
			t.Errorf("%s is deleted", id)
		}
	}
	if _, err := svc.Store.Get("task", keep.ID); err != nil {
		t.Errorf("the one not done is kept: %v", err)
	}
	made := entries(t, svc, "deleted")
	if len(made) != 1 {
		t.Fatalf("one entry in the log, got %d", len(made))
	}
	if _, _, err := svc.Undo(made[0].ID); err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		if rec, err := svc.Store.Get("task", id); err != nil || rec.Fields["done"] != true {
			t.Errorf("one Undo puts %s back as it was: %v %v", id, rec, err)
		}
	}
}

// One id that is not there and nothing is deleted.
func TestDeleteRecordAllOrNone(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	rec, _ := svc.Store.Create("note", map[string]any{"title": "Old shopping list"})
	text, isErr := svc.Call("delete_record", []byte(`{"type":"note","ids":["`+rec.ID+`","nosuchnote000000"]}`))
	if !isErr || !strings.Contains(text, "Nothing was deleted") {
		t.Errorf("refused: %v %s", isErr, text)
	}
	if _, err := svc.Store.Get("note", rec.ID); err != nil {
		t.Errorf("the note is kept: %v", err)
	}
	if made := entries(t, svc, "deleted"); len(made) != 0 {
		t.Errorf("nothing is logged, %d entries", len(made))
	}
}
