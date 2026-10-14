package chat_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// arrange_canvas takes blocks away in the same change as it lays the tab
// out: one entry, one Undo that puts them back where they were; a block
// both listed and removed is refused with nothing changed.
func TestArrangeCanTakeBlocksAway(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	heading, due, tracker, overdue := weekPage(t, svc)
	raw := `{"blocks":[{"id":"` + heading + `","span":12},{"id":"` + overdue + `","span":12},{"id":"` + due + `","span":12}],"remove":["` + tracker + `","` + due + `"]}`
	if text, isErr := svc.Call("arrange_canvas", []byte(raw)); !isErr || !strings.Contains(text, "both listed and in remove") {
		t.Errorf("listed and removed is refused: %v %s", isErr, text)
	}
	raw = `{"blocks":[{"id":"` + heading + `","span":12},{"id":"` + overdue + `","span":12},{"id":"` + due + `","span":12}],"remove":["` + tracker + `"]}`
	text, isErr := svc.Call("arrange_canvas", []byte(raw))
	if isErr || !strings.Contains(text, "and 1 block taken away, in one change") {
		t.Fatalf("arranged, one taken away: %v %s", isErr, text)
	}
	if _, err := svc.Store.Get(records.BlockType, tracker); err == nil {
		t.Error("the tracker is gone")
	}
	made := entries(t, svc, "arranged")
	if len(made) != 1 {
		t.Fatalf("one entry in the log, got %d", len(made))
	}
	if _, _, err := svc.Undo(made[0].ID); err != nil {
		t.Fatal(err)
	}
	if blocks, _ := svc.Store.List(records.BlockType, store.ListOptions{}); len(blocks) != 4 {
		t.Errorf("one Undo puts the tracker back: %d blocks", len(blocks))
	}
}
