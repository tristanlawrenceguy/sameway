package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// A block that can no longer be shown, and a block the same as another,
// are said in the model's prompt and in Layout now after a write, with
// what to do: they were what cluttered a page the model had only been told
// rules about. Each is said once, on one line, in the words Layout now
// gives everywhere: the prompt carries Layout now itself.
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
	listing := system[strings.Index(system, "Current canvas"):]
	if !strings.Contains(system, svc.LayoutNow("")) {
		t.Errorf("the prompt carries Layout now as it is said after a write:\n%s", listing)
	}
	for _, want := range []string{"block " + twin.ID + " (text) is the same as block " + first.ID + ": propose taking it away", "block " + broken.ID + " (button) cannot be shown: button props do not fit: label is required and missing: propose taking it away"} {
		if n := strings.Count(system, want); n != 1 {
			t.Errorf("%q is said once, not %d times:\n%s", want, n, listing)
		}
	}
	if strings.Contains(listing, "nothing was saved") || strings.Count(listing, "propose_change") != 1 {
		t.Errorf("a stored block is not a refused write, and how to take one away is said once:\n%s", listing)
	}

	text2, _ := svc.Call("add_component", []byte(`{"component":"text","props":{"content":"A new line."}}`))
	for _, want := range []string{"block " + twin.ID + " (text) is the same as block " + first.ID, "block " + broken.ID + " (button) cannot be shown", "remove_component"} {
		if !strings.Contains(text2, want) {
			t.Errorf("Layout now after a write says %q:\n%s", want, text2)
		}
	}
}
