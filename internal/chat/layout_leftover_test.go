package chat_test

import (
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"
)

// A block the assistant added over a week ago that has shown nothing
// since is said in Layout now, to ask the person about; the same block
// a day old, or one that shows something, is not.
func TestLayoutNowSaysWhatMayBeLeftOver(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	svc.Blocks = &blocks.Workspace{Store: svc.Store} // as the app has it
	svc.Store.Create("task", map[string]any{"title": "Pay rent"})
	run(t, svc, "add_component", map[string]any{"component": "collection", "props": map[string]any{"type": "note", "label": "Recipes"}, "span": 12})
	run(t, svc, "add_component", map[string]any{"component": "collection", "props": map[string]any{"type": "task", "label": "Tasks"}, "span": 12})
	if text := svc.LayoutNow(""); strings.Contains(text, "left over") {
		t.Errorf("a block added today is not said:\n%s", text)
	}
	later := time.Now().Add(9*24*time.Hour + time.Hour)
	svc.Book.Clock = func() time.Time { return later }
	text := svc.LayoutNow("")
	if !strings.Contains(text, `("Recipes" collection) has shown nothing since you added it 9 days ago (there are no notes)`) || !strings.Contains(text, "propose_change") {
		t.Errorf("the empty list from 9 days ago is said, to ask about:\n%s", text)
	}
	if strings.Contains(text, `("Tasks" collection) has shown`) {
		t.Errorf("a list that shows something is not said:\n%s", text)
	}
}
