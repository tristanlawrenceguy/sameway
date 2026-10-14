package chat_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// clutter is a page with three blocks to tidy away and one to keep.
func clutter(t *testing.T, svc *chat.Service) (keep string, gone []string) {
	keep = idOf(run(t, svc, "add_component", map[string]any{"component": "heading", "props": map[string]any{"text": "Garden"}}))
	for _, props := range []map[string]any{{"content": "Weather note"}, {"content": "Old reminder"}, {"title": "Leftover"}} {
		component := "text"
		if props["title"] != nil {
			component = "card"
		}
		gone = append(gone, idOf(run(t, svc, "add_component", map[string]any{"component": component, "props": props})))
	}
	return keep, gone
}

func blockCount(t *testing.T, svc *chat.Service) int {
	n, _ := svc.Store.Count(records.BlockType)
	return n
}

// Several blocks go in one call, as one entry that one Undo takes back.
func TestRemoveSeveralIsOneChangeAndOneUndo(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	keep, gone := clutter(t, svc)
	text := run(t, svc, "remove_component", map[string]any{"ids": gone})
	if !strings.Contains(text, "removed 3 blocks") || !strings.Contains(text, "Layout now") {
		t.Errorf("the result should say what went and how the page reads now:\n%s", text)
	}
	if n := blockCount(t, svc); n != 1 {
		t.Fatalf("three of four blocks should be gone, %d left", n)
	}
	if _, err := svc.Store.Get(records.BlockType, keep); err != nil {
		t.Fatalf("the block not named should stay: %v", err)
	}
	removed := entries(t, svc, "removed")
	if len(removed) != 1 {
		t.Fatalf("removing several should be one entry, got %d", len(removed))
	}
	if s, _ := removed[0].Fields["summary"].(string); s != "Assistant removed 3 blocks: Weather note, Old reminder, Leftover card" {
		t.Errorf("summary: %q", s)
	}
	run(t, svc, "undo_change", map[string]any{"id": removed[0].ID})
	for _, id := range gone {
		if _, err := svc.Store.Get(records.BlockType, id); err != nil {
			t.Errorf("one Undo should put %s back: %v", id, err)
		}
	}
	if label := chat.Describe(llm.ToolCall{Name: "remove_component", Args: []byte(`{"ids":["a","b","c"]}`)}); label != "Removing 3 blocks" {
		t.Errorf("label: %q", label)
	}
}

// An id that is not a block removes nothing, and the error lists the
// tab's blocks so the next call can name the right ones.
func TestRemoveSeveralIsAllOrNone(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	keep, gone := clutter(t, svc)
	refused(t, svc, "remove_component", map[string]any{"ids": append(gone, "nope")}, "nothing was removed: no block nope")
	refused(t, svc, "remove_component", map[string]any{"ids": []string{"nope", "gone"}}, keep+" (heading)")
	refused(t, svc, "remove_component", map[string]any{"ids": append(gone, "nope")}, gone[0]+" (text)")
	if n := blockCount(t, svc); n != 4 {
		t.Errorf("a refused removal should leave every block, %d left", n)
	}
}

// A question can carry several blocks, and the person's Yes removes them all.
func TestProposeRemovingSeveral(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	_, gone := clutter(t, svc)
	run(t, svc, "propose_change", map[string]any{"summary": "Tidy the page?", "tool": "remove_component", "ids": gone})
	asked := svc.Proposals()
	if len(asked) != 1 {
		t.Fatalf("one question should wait, got %d", len(asked))
	}
	if err := svc.Accept(asked[0].ID); err != nil {
		t.Fatal(err)
	}
	if n := blockCount(t, svc); n != 1 {
		t.Errorf("Yes should remove all three, %d left", n)
	}
}

// One id still removes one block, said as it always was.
func TestRemoveOneStillWorks(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	_, gone := clutter(t, svc)
	if text := run(t, svc, "remove_component", map[string]any{"id": gone[0]}); !strings.Contains(text, "removed block "+gone[0]) {
		t.Errorf("one removal: %s", text)
	}
	run(t, svc, "remove_component", map[string]any{"ids": []string{gone[1]}})
	if n := blockCount(t, svc); n != 2 {
		t.Errorf("two single removals should leave two blocks, %d left", n)
	}
	if label := chat.Describe(llm.ToolCall{Name: "remove_component", Args: []byte(`{"id":"a"}`)}); label != "Removing a block" {
		t.Errorf("label: %q", label)
	}
}
