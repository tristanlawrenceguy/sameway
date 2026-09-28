package chat_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// One call lays out a whole page the way it was thought through: each
// block in its region at its width, in order, with the person's words
// filled in, each logged and undoable on its own.
func TestAnArrangementIsAPageInOneCall(t *testing.T) {
	svc := newFullService(t)
	raw, _ := json.Marshal(map[string]any{"name": "week", "fills": map[string]any{
		"todo": map[string]any{"items": []string{"Water the plants", "Call Sam"}},
	}})
	text, isErr := svc.Call("add_arrangement", raw)
	if isErr {
		t.Fatal(text)
	}
	if !strings.Contains(text, "4 blocks") || !strings.Contains(text, "todo: added list") {
		t.Errorf("the result should list every block, got %q", text)
	}
	blocks, _ := svc.Store.List(chat.BlockType, store.ListOptions{OrderBy: "position"})
	if len(blocks) != 4 {
		t.Fatalf("the week arrangement has four blocks, got %d", len(blocks))
	}
	got := map[string]*store.Record{}
	for _, b := range blocks {
		got[b.Fields["component"].(string)] = b
	}
	if got["calendar"].Fields["region"] != "right" {
		t.Errorf("the calendar sits in the right pane, got %v", got["calendar"].Fields["region"])
	}
	if got["heading"].Fields["span"] != int64(12) || got["heading"].Fields["frame"] != "bare" {
		t.Errorf("the heading is full width and bare, got %v", got["heading"].Fields)
	}
	items, _ := got["list"].Fields["props"].(map[string]any)["items"].([]any)
	if len(items) != 2 || items[0] != "Water the plants" {
		t.Errorf("the fill should replace the starting items, got %v", items)
	}
	if blocks[0].Fields["component"] != "heading" || blocks[3].Fields["component"] != "text" {
		t.Errorf("blocks keep the arrangement's order: %v %v", blocks[0].Fields["component"], blocks[3].Fields["component"])
	}

	// Every block is its own entry in the log, so one can be undone alone.
	log, _ := svc.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at"})
	if len(log) != 4 {
		t.Fatalf("four blocks, four log entries, got %d", len(log))
	}
	if text, isErr := svc.Call("undo_change", json.RawMessage(`{}`)); isErr {
		t.Fatal(text)
	}
	if after, _ := svc.Store.List(chat.BlockType, store.ListOptions{}); len(after) != 3 {
		t.Errorf("undoing the newest change removes one block, got %d left", len(after))
	}

	if text, isErr := svc.Call("add_arrangement", json.RawMessage(`{"name":"nope"}`)); !isErr || !strings.Contains(text, "week") {
		t.Errorf("an unknown arrangement should be refused with the list, got %q", text)
	}
}
