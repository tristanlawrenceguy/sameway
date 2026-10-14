package chat_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A brain dump of tasks is one call, one entry in the log and one Undo:
// the result names each task by its id and page, and a duplicate is still
// said, as create_record says it.
func TestCreateRecordsIsOneChangeAndOneUndo(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	old, _ := svc.Store.Create("task", map[string]any{"title": "Call mum"})
	text := run(t, svc, "create_records", map[string]any{"type": "task", "records": []map[string]any{
		{"title": "Email Sarah"}, {"title": "Book the car", "due": "2026-10-16"}, {"title": "Call mum"},
	}})
	tasks, _ := svc.Store.List("task", store.ListOptions{})
	if len(tasks) != 4 {
		t.Fatalf("three made beside the one there: %d", len(tasks))
	}
	for _, rec := range tasks {
		if rec.ID != old.ID && !strings.Contains(text, rec.ID+` "`+rec.Fields["title"].(string)+`" at /t/task/`+rec.ID) {
			t.Errorf("the result names %s and its page: %s", rec.ID, text)
		}
	}
	if !strings.Contains(text, "called the same") || !strings.Contains(text, old.ID) {
		t.Errorf("the second Call mum is said: %s", text)
	}
	made := entries(t, svc, "created")
	if len(made) != 1 {
		t.Fatalf("one entry in the log, got %d", len(made))
	}
	if s, _ := made[0].Fields["summary"].(string); s != "Assistant created 3 tasks: Email Sarah, Book the car, Call mum" {
		t.Errorf("summary: %q", s)
	}
	if _, _, err := svc.Undo(made[0].ID); err != nil {
		t.Fatal(err)
	}
	if left, _ := svc.Store.List("task", store.ListOptions{}); len(left) != 1 || left[0].ID != old.ID {
		t.Errorf("one Undo takes the three back and leaves the one before: %d left", len(left))
	}
}

// One wrong entry and nothing is made; the error names it by its place
// and its title, and why.
func TestCreateRecordsAllOrNone(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	raw := `{"type":"task","records":[{"title":"Email Sarah"},{"title":"Book the car","status":"later"},{"title":"Call mum"}]}`
	text, isErr := svc.Call("create_records", []byte(raw))
	if !isErr || !strings.Contains(text, "Nothing was made") || !strings.Contains(text, `records[1] ("Book the car")`) || !strings.Contains(text, "status") {
		t.Errorf("refused, naming the one and why: %v %s", isErr, text)
	}
	if tasks, _ := svc.Store.List("task", store.ListOptions{}); len(tasks) != 0 {
		t.Errorf("nothing is made, %d were", len(tasks))
	}
	if all, _ := svc.Store.List(records.ActivityType, store.ListOptions{}); len(all) != 0 {
		t.Errorf("nothing is logged, %d entries", len(all))
	}
}
