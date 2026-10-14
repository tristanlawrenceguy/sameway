package records_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// WriteAll writes several, all or none: one refused puts back those
// written before it, made or changed, and says which it was.
func TestWriteAllIsAllOrNone(t *testing.T) {
	t.Parallel()
	b := newBook(t)
	recs, done, _, err := records.WriteAll(b.Store, "created", "task", []records.Writing{
		{Fields: map[string]any{"title": "Email Sarah"}}, {Fields: map[string]any{"title": "Call mum"}},
	})
	if err != nil || len(recs) != 2 || len(records.Joined(done)) != 2 {
		t.Fatalf("both made, two ops: %v %d", err, len(done))
	}
	_, _, at, err := records.WriteAll(b.Store, "created", "task", []records.Writing{
		{Fields: map[string]any{"title": "Book the car"}}, {Fields: map[string]any{"title": "Pay", "status": "later"}},
	})
	if err == nil || at != 1 {
		t.Fatalf("the second is refused, and said: %d %v", at, err)
	}
	if all, _ := b.Store.List("task", store.ListOptions{}); len(all) != 2 {
		t.Errorf("the one made before it is taken back: %d tasks", len(all))
	}
	_, _, at, err = records.WriteAll(b.Store, "updated", "task", []records.Writing{
		{ID: recs[0].ID, Fields: map[string]any{"title": "Email Sam"}}, {ID: recs[1].ID, Fields: map[string]any{"status": "later"}},
	})
	if err == nil || at != 1 {
		t.Fatalf("the second change is refused: %d %v", at, err)
	}
	if got, _ := b.Store.Get("task", recs[0].ID); got.Fields["title"] != "Email Sarah" {
		t.Errorf("the one changed before it is as it was: %v", got.Fields["title"])
	}
}
