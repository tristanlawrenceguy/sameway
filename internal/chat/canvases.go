package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// CanvasType is the content type that holds the tabs. The first canvas,
// Home, needs no record: it is the blocks whose canvas is empty. Every
// record of this type is one more tab beside it, with blocks of its own.
const CanvasType = "canvas"

// Canvas is one tab as the pages and the prompt see it.
type Canvas struct {
	ID   string // "" for Home
	Name string
	Path string
}

// HomePath is where the first canvas lives.
const HomePath = "/"

// CanvasPath is the page for a canvas: Home, or a tab by id.
func CanvasPath(id string) string {
	if id == "" {
		return HomePath
	}
	return "/c/" + id
}

// Canvases lists the tabs in order: Home first, then the records by position.
func (s *Service) Canvases() []Canvas {
	out := []Canvas{{Name: "Home", Path: HomePath}}
	if _, ok := s.Store.Types().Get(CanvasType); !ok {
		return out
	}
	recs, err := s.Store.List(CanvasType, store.ListOptions{OrderBy: "position"})
	if err != nil {
		return out
	}
	for _, rec := range recs {
		name, _ := rec.Fields["name"].(string)
		out = append(out, Canvas{ID: rec.ID, Name: name, Path: CanvasPath(rec.ID)})
	}
	return out
}

// HasCanvas says whether id names a tab: Home, or a record that exists.
func (s *Service) HasCanvas(id string) bool {
	for _, c := range s.Canvases() {
		if c.ID == id {
			return true
		}
	}
	return false
}

// OnCanvas keeps the blocks that belong to one tab.
func OnCanvas(blocks []*store.Record, id string) []*store.Record {
	var out []*store.Record
	for _, b := range blocks {
		on, _ := b.Fields["canvas"].(string)
		if on == id {
			out = append(out, b)
		}
	}
	return out
}

// canvasTools are offered when the workspace has the canvas type, which every
// workspace made or opened since tabs existed does.
func (s *Service) canvasTools() []llm.Tool {
	if _, ok := s.Store.Types().Get(CanvasType); !ok {
		return nil
	}
	obj := func(props map[string]any, required ...string) map[string]any {
		o := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
		if len(required) > 0 {
			o["required"] = required
		}
		return o
	}
	return []llm.Tool{
		{Name: "create_canvas", Description: "Add a tab: a new canvas beside Home with blocks of its own. Use it when the person asks for a separate page or tab, or when what they want does not belong with what is already on the canvas. Returns the canvas id, which add_component takes as canvas.",
			Schema: obj(map[string]any{
				"name": map[string]any{"type": "string", "description": "The tab's name, in the person's words: Work, Garden, Trip to Rome."},
			}, "name")},
		{Name: "remove_canvas", Description: "Remove a tab and every block on it. Ask first with propose_change; Home cannot be removed.",
			Schema: obj(map[string]any{
				"id": map[string]any{"type": "string", "description": "The canvas id, from the list of tabs."},
			}, "id")},
	}
}

func (s *Service) createCanvas(name string) toolResult {
	name = strings.TrimSpace(name)
	if name == "" {
		return fail("a canvas needs a name: what the tab is called")
	}
	position := len(s.Canvases())
	rec, err := s.Store.Create(CanvasType, map[string]any{"name": name, "position": position})
	if err != nil {
		return fail("could not add the canvas: %v", err)
	}
	return toolResult{
		text:   fmt.Sprintf("added canvas %s: %q, at %s. Blocks go on it with canvas: %q on add_component.", rec.ID, name, CanvasPath(rec.ID), rec.ID),
		change: &Change{Action: "added", Component: CanvasType, ID: rec.ID, Detail: name, Href: CanvasPath(rec.ID)},
	}
}

func (s *Service) removeCanvas(id string) toolResult {
	if id == "" {
		return fail("Home is the first canvas and stays; remove the blocks on it instead")
	}
	rec, err := s.Store.Get(CanvasType, id)
	if err != nil {
		return fail("no canvas with id %s; the tabs are listed in the prompt", id)
	}
	name, _ := rec.Fields["name"].(string)
	blocks, _ := s.Store.List(BlockType, store.ListOptions{})
	n := 0
	for _, b := range OnCanvas(blocks, id) {
		if s.Store.Delete(BlockType, b.ID) == nil {
			n++
		}
	}
	if err := s.Store.Delete(CanvasType, id); err != nil {
		return fail("could not remove the canvas: %v", err)
	}
	return toolResult{
		text:   fmt.Sprintf("removed canvas %s (%q) and the %d blocks on it", id, name, n),
		change: &Change{Action: "removed", Component: CanvasType, ID: id, Detail: name},
	}
}

// canvasDigest tells the model which tabs exist and which one the person is
// looking at, so blocks land where they are wanted.
func (s *Service) canvasDigest(current string) string {
	canvases := s.Canvases()
	if len(canvases) == 1 && current == "" {
		return ""
	}
	var parts []string
	for _, c := range canvases {
		label := fmt.Sprintf("%s (canvas id %q, at %s)", c.Name, c.ID, c.Path)
		if c.ID == current {
			label += " <- the person is looking at this one"
		}
		parts = append(parts, label)
	}
	return "\nTabs, each a canvas with its own blocks: " + strings.Join(parts, "; ") +
		". Blocks you add go on the one the person is looking at unless you pass canvas; the canvas listing below is that one.\n"
}
