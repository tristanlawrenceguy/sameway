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

// Undoing a batch after the person changed one of them again keeps what
// they did: the others go back, theirs stays, and the result says so. A
// repeated id is one record, changed once. The log names the first few.
func TestUpdateRecordsUndoKeepsLaterEdits(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	var ids []string
	for _, title := range []string{"Email Sarah", "Book the car", "Call mum", "Water plants", "Pay rent"} {
		rec, _ := svc.Store.Create("task", map[string]any{"title": title, "due": "2026-09-28"})
		ids = append(ids, rec.ID)
	}
	run(t, svc, "update_records", map[string]any{"type": "task", "ids": append(ids, ids[0]), "fields": map[string]any{"due": "2026-10-19"}})
	made := entries(t, svc, "updated")
	if s, _ := made[0].Fields["summary"].(string); !strings.HasPrefix(s, "Assistant updated 5 tasks: ") || !strings.HasSuffix(s, " and 1 more") {
		t.Errorf("five, the first four by name: %q", s)
	}
	svc.Store.Update("task", ids[1], map[string]any{"due": "2026-10-22"})
	text := run(t, svc, "undo_change", map[string]any{"id": made[0].ID})
	if !strings.Contains(text, "Kept as changed since: due of Book the car") {
		t.Errorf("the result says what was kept: %s", text)
	}
	for i, id := range ids {
		want := "2026-09-28"
		if i == 1 {
			want = "2026-10-22"
		}
		if rec, _ := svc.Store.Get("task", id); !strings.HasPrefix(rec.Fields["due"].(string), want) {
			t.Errorf("%s: due %v, want %s", rec.Fields["title"], rec.Fields["due"], want)
		}
	}
}
