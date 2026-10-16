package chat_test

import (
	"encoding/json"
	"strings"
	"testing"
)

// An event moved to another day names its tasks still on the old day, and
// the one change that moves them; one already done, or on another day, is
// not named.
func TestAMovedEventNamesWhatStayedBehind(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	ev, _ := svc.Store.Create("event", map[string]any{"title": "Cleanup Day", "starts": "2026-10-22T09:00:00Z"})
	a, _ := svc.Store.Create("task", map[string]any{"title": "Bring gloves", "due": "2026-10-22", "event": ev.ID})
	svc.Store.Create("task", map[string]any{"title": "Book the van", "due": "2026-10-22", "event": ev.ID, "done": true})
	svc.Store.Create("task", map[string]any{"title": "Thank everyone", "due": "2026-10-26", "event": ev.ID})
	raw, _ := json.Marshal(map[string]any{"type": "event", "id": ev.ID, "fields": map[string]any{"starts": "2026-10-25T09:00:00Z"}})
	text, isErr := svc.Call("update_record", raw)
	if isErr || !strings.Contains(text, "1 task in Cleanup Day is still on its old day") || !strings.Contains(text, "Bring gloves") || !strings.Contains(text, a.ID) {
		t.Errorf("the task left behind is named, with how to move it: %v %s", isErr, text)
	}
	if strings.Contains(text, "Book the van") || strings.Contains(text, "Thank everyone") {
		t.Errorf("one done, or on another day, is not named: %s", text)
	}
}
