package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// A block that can no longer be shown, and a block the same as another,
// are said beside them in the canvas the model is given, and in Layout now
// after a write, with what to do: they were what cluttered a page the
// model had only been told rules about.
func TestClutterIsSaidWhereTheModelLooks(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	text := map[string]any{"content": "The weather is nice today."}
	first, _ := svc.Store.Create(records.BlockType, map[string]any{"component": "text", "props": text, "position": 0, "canvas": ""})
	twin, _ := svc.Store.Create(records.BlockType, map[string]any{"component": "text", "props": text, "position": 1, "canvas": ""})
	broken, _ := svc.Store.Create(records.BlockType, map[string]any{"component": "button", "props": map[string]any{}, "position": 2, "canvas": ""})

	m := &scripted{steps: []*llm.Response{{Text: "ok"}}}
	svc.Provider = m
	if _, err := svc.Send(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
	system := m.seen[0].System
	if !strings.Contains(system, "[the same as block "+first.ID+"]") || !strings.Contains(system, broken.ID+" button") || !strings.Contains(system, "[cannot be shown: ") {
		t.Errorf("the canvas listing says which block is clutter:\n%s", system[strings.Index(system, "Current canvas"):])
	}

	text2, _ := svc.Call("add_component", []byte(`{"component":"text","props":{"content":"A new line."}}`))
	for _, want := range []string{"block " + twin.ID + " (text) the same as block " + first.ID, "block " + broken.ID + " (button) cannot be shown", "remove_component"} {
		if !strings.Contains(text2, want) {
			t.Errorf("Layout now after a write says %q:\n%s", want, text2)
		}
	}
}
