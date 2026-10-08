package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Canvas is one tab as the pages and the prompt see it.
type Canvas struct {
	ID   string // "" for Home
	Name string
	Path string
}

// Canvases lists the tabs in order: Home first, then the records by position.
func (s *Service) Canvases() []Canvas {
	out := []Canvas{{Name: "Home", Path: records.HomePath}}
	if _, ok := s.Store.Types().Get(records.CanvasType); !ok {
		return out
	}
	recs, err := s.Store.List(records.CanvasType, store.ListOptions{OrderBy: "position"})
	if err != nil {
		return out
	}
	for _, rec := range recs {
		name, _ := rec.Fields["name"].(string)
		out = append(out, Canvas{ID: rec.ID, Name: name, Path: records.CanvasPath(rec.ID)})
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

// canvasOps are offered when the workspace has the canvas type, which every
// workspace made or opened since tabs existed does.
var canvasOps = []Op{
	{Title: "Make a tab",
		Core: true,
		Tool: llm.Tool{Name: "create_canvas", Description: "Add a tab: a new canvas beside Home with blocks of its own. Use it when the person asks for a separate page or tab, or when what they want does not belong with what is already on the canvas. Returns the canvas id, which add_component takes as canvas.",
			Schema: obj(map[string]any{
				"name": map[string]any{"type": "string", "description": "The tab's name, in the person's words, one or two of them: Work, Garden, Rome."},
			}, "name")}, Offered: has(records.CanvasType)},
	{Title: "Remove a tab", Traits: Traits{Destructive: true, Idempotent: true},
		Words: []string{"tab"},
		Tool: llm.Tool{Name: "remove_canvas", Description: "Remove a tab and every block on it. Ask first with propose_change; Home cannot be removed.",
			Schema: obj(map[string]any{
				"id": map[string]any{"type": "string", "description": "The canvas id, from the list of tabs."},
			}, "id")}, Offered: has(records.CanvasType)},
}

func (s *Service) createCanvas(name string) toolResult {
	name = strings.TrimSpace(name)
	if name == "" {
		return fail("a canvas needs a name: what the tab is called")
	}
	// Two tabs with one name are two links that say the same and go to
	// different places.
	for _, c := range s.Canvases() {
		if strings.EqualFold(c.Name, name) {
			return fail("there is already a tab called %q (canvas %q); use it, or give this one another name", c.Name, c.ID)
		}
	}
	position := len(s.Canvases())
	rec, err := s.Store.Create(records.CanvasType, map[string]any{"name": name, "position": position})
	if err != nil {
		return fail("could not add the canvas: %v", err)
	}
	return toolResult{
		text:   fmt.Sprintf("added canvas %s: %q, at %s. Blocks go on it with canvas: %q on add_component.", rec.ID, name, records.CanvasPath(rec.ID), rec.ID),
		change: &records.Change{Action: "added", Component: records.CanvasType, ID: rec.ID, Detail: name, Href: records.CanvasPath(rec.ID)},
	}
}

func (s *Service) removeCanvas(id string) toolResult {
	c, err := records.RemoveCanvas(s.Store, id)
	if err != nil {
		return fail("%v", err)
	}
	blocks, _ := c.Before["blocks"].([]any)
	return toolResult{
		text:   fmt.Sprintf("removed canvas %s (%q) and the %d blocks on it", id, c.Detail, len(blocks)),
		change: &c,
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
