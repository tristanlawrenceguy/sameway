package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Tabs are canvases. The assistant makes one, blocks go on the tab the
// person is looking at unless the call says otherwise, the prompt says
// which tab that is, and clearing a tab leaves the others alone.
func TestTabsAreCanvases(t *testing.T) {
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{
		call("create_canvas", map[string]any{"name": "Garden"}),
		call("add_component", map[string]any{"component": "heading", "props": map[string]any{"text": "Home things"}}),
	}}
	svc.Provider = m
	if _, err := svc.Send(context.Background(), "make a garden tab"); err != nil {
		t.Fatal(err)
	}
	canvases := svc.Canvases()
	if len(canvases) != 2 || canvases[0].Name != "Home" || canvases[1].Name != "Garden" || canvases[1].Path != "/c/"+canvases[1].ID {
		t.Fatalf("expected Home then Garden, got %+v", canvases)
	}
	garden := canvases[1].ID
	if res := lastToolResult(m.seen[1]); res.IsError || !strings.Contains(res.Content, garden) {
		t.Errorf("create_canvas should return the id to build on: %+v", res)
	}
	if !strings.Contains(m.seen[2].System, "Garden (canvas id \""+garden+"\"") || !strings.Contains(m.seen[2].System, "Home (canvas id \"\", at /) <- the person is looking at this one") {
		t.Errorf("the prompt should list the tabs and mark the current one:\n%s", tail(m.seen[2].System))
	}

	// On the Garden tab, new blocks land there, and the listing is that tab's.
	m = &scripted{steps: []*llm.Response{
		call("add_component", map[string]any{"component": "heading", "props": map[string]any{"text": "Beds to dig"}}),
		call("add_component", map[string]any{"component": "card", "props": map[string]any{"title": "Seeds"}, "canvas": ""}),
	}}
	svc.Provider = m
	if _, err := svc.SendOn(context.Background(), garden, "plan the beds"); err != nil {
		t.Fatal(err)
	}
	blocks, _ := svc.Store.List(chat.BlockType, store.ListOptions{OrderBy: "position"})
	on := map[string]string{}
	for _, b := range blocks {
		props, _ := b.Fields["props"].(map[string]any)
		name, _ := props["text"].(string)
		if name == "" {
			name, _ = props["title"].(string)
		}
		on[name], _ = b.Fields["canvas"].(string)
	}
	if on["Home things"] != "" || on["Beds to dig"] != garden || on["Seeds"] != "" {
		t.Errorf("blocks should land on the tab the person is on, or the one named: %v", on)
	}
	if sys := m.seen[2].System; !strings.Contains(sys, "Beds to dig") || strings.Contains(sys, "Home things") {
		t.Errorf("the canvas listing should be the current tab's only:\n%s", tail(sys))
	}
	if res := lastToolResult(m.seen[2]); res.IsError || !strings.Contains(res.Content, "canvas Home") {
		t.Errorf("placing a block on another tab is confirmed: %+v", res)
	}

	// Clearing the Garden tab leaves Home alone; removing it takes its blocks.
	m = &scripted{steps: []*llm.Response{
		call("clear_canvas", map[string]any{}),
		call("remove_canvas", map[string]any{"id": garden}),
	}}
	svc.Provider = m
	svc.SendOn(context.Background(), garden, "start this tab over, then drop it")
	if res := lastToolResult(m.seen[1]); !strings.Contains(res.Content, "cleared 1 block") {
		t.Errorf("clear_canvas should clear only this tab: %+v", res)
	}
	blocks, _ = svc.Store.List(chat.BlockType, store.ListOptions{})
	if len(blocks) != 2 {
		t.Errorf("Home's two blocks should survive, got %d", len(blocks))
	}
	if len(svc.Canvases()) != 1 {
		t.Error("the Garden tab should be gone")
	}
}

func TestCanvasToolsRefuseWhatDoesNotExist(t *testing.T) {
	svc := newFullService(t)
	m := &scripted{steps: []*llm.Response{
		call("add_component", map[string]any{"component": "heading", "props": map[string]any{"text": "x"}, "canvas": "nope"}),
		call("remove_canvas", map[string]any{"id": ""}),
		call("create_canvas", map[string]any{"name": "  "}),
	}}
	svc.Provider = m
	svc.Send(context.Background(), "bad ones")
	for i, want := range []string{"no canvas with id", "Home is the first canvas", "needs a name"} {
		if res := lastToolResult(m.seen[i+1]); !res.IsError || !strings.Contains(res.Content, want) {
			t.Errorf("call %d: want an error mentioning %q, got %+v", i+1, want, res)
		}
	}
	if _, err := svc.SendOn(context.Background(), "nope", "hello"); err == nil {
		t.Error("a message on a tab that does not exist is an error")
	}
}
