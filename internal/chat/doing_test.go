package chat_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// rounds is a model that answers each round with what next gives it.
type rounds struct {
	n    int
	next func(n int) *llm.Response
}

func (r *rounds) Name() string { return "rounds" }
func (r *rounds) Complete(context.Context, llm.Request) (*llm.Response, error) {
	r.n++
	return r.next(r.n), nil
}

func toolOf(id, name string, args map[string]any) llm.ToolCall {
	raw, _ := json.Marshal(args)
	return llm.ToolCall{ID: id, Name: name, Args: raw}
}

// While a turn runs the status line says what the assistant is doing in
// words its calls give: what is made and of what, what is looked up, which
// block is changed. Never the tool's name, and never "a block" when more
// is known. A call that changes the canvas says where, so the page can
// mark the place while it runs.
func TestATurnSaysWhatItIsDoingInWords(t *testing.T) {
	svc, _ := newService(t)
	svc.Provider = &rounds{next: func(n int) *llm.Response {
		switch n {
		case 1:
			return &llm.Response{ToolCalls: []llm.ToolCall{
				toolOf("a", "add_component", map[string]any{"component": "collection", "props": map[string]any{"type": "task"}, "span": 4}),
				toolOf("b", "add_component", map[string]any{"component": "card", "props": map[string]any{"title": "Shopping"}, "region": "right"}),
				toolOf("c", "find_records", map[string]any{"type": "task"}),
				toolOf("d", "search", map[string]any{"query": "milk"}),
				toolOf("e", "arrange_canvas", map[string]any{}),
				toolOf("f", "create_record", map[string]any{"type": "task", "fields": map[string]any{"title": "Buy milk"}}),
			}}
		case 2:
			blocks, _ := svc.Store.List(chat.BlockType, store.ListOptions{})
			for _, b := range blocks {
				if b.Fields["component"] == "collection" {
					return &llm.Response{ToolCalls: []llm.ToolCall{toolOf("g", "update_component", map[string]any{"id": b.ID, "span": 6})}}
				}
			}
			t.Fatal("the list was not made")
		}
		return &llm.Response{Text: "Done."}
	}}
	var steps []chat.Event
	if _, err := svc.SendLive(context.Background(), "", "set up my tasks", "", func(e chat.Event) {
		if e.Kind == "tool" {
			steps = append(steps, e)
		}
	}); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"Adding a list of tasks",
		"Adding a card called Shopping",
		"Looking up your tasks",
		"Searching for milk",
		"Arranging the page",
		"Adding a task called Buy milk",
		"Changing the list of tasks",
	}
	if len(steps) != len(want) {
		t.Fatalf("want %d steps, got %+v", len(want), steps)
	}
	for i, w := range want {
		if steps[i].Label != w {
			t.Errorf("step %d: want %q, got %q", i, w, steps[i].Label)
		}
	}
	if a := steps[0].Aim; a.Region != "main" || a.Span != 4 || a.Block != "" {
		t.Errorf("a new block says it lands in main, four wide: %+v", a)
	}
	if a := steps[1].Aim; a.Region != "right" {
		t.Errorf("a block for a pane says so, so the main canvas holds no place for it: %+v", a)
	}
	if steps[2].Aim != (chat.Target{}) {
		t.Errorf("looking something up changes no place: %+v", steps[2].Aim)
	}
	if steps[6].Aim.Block == "" {
		t.Error("a block being changed is named, so the page can outline it")
	}
}
