package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// TestModelLaysOutTheCanvas covers span and position: the model decides how
// wide a block is and where it sits, without touching its props.
func TestModelLaysOutTheCanvas(t *testing.T) {
	svc := newFullService(t)
	svc.Provider = &scripted{steps: []*llm.Response{
		call("add_component", map[string]any{"component": "heading", "props": map[string]any{"text": "Week"}, "span": 12}),
		call("add_component", map[string]any{"component": "card", "props": map[string]any{"title": "Monday"}, "span": 4}),
	}}
	if _, err := svc.Send(context.Background(), "lay out my week"); err != nil {
		t.Fatal(err)
	}
	blocks, _ := svc.Store.List(chat.BlockType, store.ListOptions{OrderBy: "position"})
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	if blocks[0].Fields["span"] != int64(12) || blocks[1].Fields["span"] != int64(4) {
		t.Errorf("spans not stored: %v %v", blocks[0].Fields["span"], blocks[1].Fields["span"])
	}

	// Widen and reorder without sending props.
	id := blocks[1].ID
	m := &scripted{steps: []*llm.Response{call("update_component", map[string]any{"id": id, "span": 8, "position": -1})}}
	svc.Provider = m
	if _, err := svc.Send(context.Background(), "make that wider and put it first"); err != nil {
		t.Fatal(err)
	}
	if res := lastToolResult(m.seen[1]); res.IsError || !strings.Contains(res.Content, "span 8") {
		t.Fatalf("layout-only update should succeed: %+v", res)
	}
	rec, _ := svc.Store.Get(chat.BlockType, id)
	if rec.Fields["span"] != int64(8) || rec.Fields["position"] != int64(-1) {
		t.Errorf("span and position not applied: %v", rec.Fields)
	}
	if title, _ := rec.Fields["props"].(map[string]any)["title"].(string); title != "Monday" {
		t.Errorf("a layout-only update must not disturb props, got %q", title)
	}
	blocks, _ = svc.Store.List(chat.BlockType, store.ListOptions{OrderBy: "position"})
	if blocks[0].ID != id {
		t.Errorf("the reordered block should now be first")
	}
}

func TestLayoutToolsRejectNonsense(t *testing.T) {
	svc := newFullService(t)
	svc.Provider = &scripted{steps: []*llm.Response{
		call("add_component", map[string]any{"component": "text", "props": map[string]any{"content": "hi"}, "span": 99}),
	}}
	svc.Send(context.Background(), "too wide")
	m := svc.Provider.(*scripted)
	if res := lastToolResult(m.seen[1]); !res.IsError || !strings.Contains(res.Content, "between 1 and 12") {
		t.Errorf("span 99 should be refused with a usable message: %+v", res)
	}
	if n, _ := svc.Store.Count(chat.BlockType); n != 0 {
		t.Errorf("nothing should have been saved")
	}

	svc.Provider = &scripted{steps: []*llm.Response{
		call("add_component", map[string]any{"component": "text", "props": map[string]any{"content": "hi"}}),
	}}
	svc.Send(context.Background(), "add text")
	blocks, _ := svc.Store.List(chat.BlockType, store.ListOptions{})
	m2 := &scripted{steps: []*llm.Response{call("update_component", map[string]any{"id": blocks[0].ID})}}
	svc.Provider = m2
	svc.Send(context.Background(), "change nothing")
	if res := lastToolResult(m2.seen[1]); !res.IsError || !strings.Contains(res.Content, "nothing to change") {
		t.Errorf("an empty update should say what is missing: %+v", res)
	}
}

// TestChatIsABlockLikeAnyOther checks the conversation can be placed,
// restyled, and removed, and that only one may exist.
func TestChatIsABlockLikeAnyOther(t *testing.T) {
	svc := newFullService(t)
	svc.Provider = &scripted{steps: []*llm.Response{
		call("add_component", map[string]any{"component": "chat", "props": map[string]any{}, "span": 12}),
	}}
	svc.Send(context.Background(), "put the chat on the canvas")
	blocks, _ := svc.Store.List(chat.BlockType, store.ListOptions{})
	if len(blocks) != 1 || blocks[0].Fields["component"] != chat.ComponentName {
		t.Fatalf("chat should be an ordinary block: %+v", blocks)
	}
	id := blocks[0].ID

	// A second one would duplicate every message id, so it is refused.
	m := &scripted{steps: []*llm.Response{call("add_component", map[string]any{"component": "chat", "props": map[string]any{}})}}
	svc.Provider = m
	svc.Send(context.Background(), "another chat")
	if res := lastToolResult(m.seen[1]); !res.IsError || !strings.Contains(res.Content, "already a chat block") {
		t.Errorf("a second chat block should be refused: %+v", res)
	}

	// Restyling is an ordinary props update.
	m2 := &scripted{steps: []*llm.Response{call("update_component", map[string]any{"id": id, "props": map[string]any{"layout": "bare", "label": "Helper"}})}}
	svc.Provider = m2
	svc.Send(context.Background(), "make it bare")
	rec, _ := svc.Store.Get(chat.BlockType, id)
	props, _ := rec.Fields["props"].(map[string]any)
	if props["layout"] != "bare" || props["label"] != "Helper" {
		t.Errorf("chat restyle not applied: %v", props)
	}

	// And it can be removed entirely.
	m3 := &scripted{steps: []*llm.Response{call("remove_component", map[string]any{"id": id})}}
	svc.Provider = m3
	svc.Send(context.Background(), "remove the chat")
	if n, _ := svc.Store.Count(chat.BlockType); n != 0 {
		t.Errorf("the chat block should be removable, %d left", n)
	}
}
