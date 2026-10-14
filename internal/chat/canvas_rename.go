package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// A tab kept the name it was made with. One made for a test, or for what
// the person no longer does, could only stay as it was called or be taken
// away with what is on it, so a model that saw what a tab now holds could
// not make its name say so. Renaming is one change, undone in one.

var renameCanvasOp = Op{Title: "Rename a tab", Traits: Traits{Idempotent: true},
	Words: []string{"tab", "rename", "call it"},
	Doing: func(a callArgs) string { return "Renaming a tab" + called(a.Name) },
	Run:   func(s *Service, a toolArgs, call llm.ToolCall) toolResult { return s.renameCanvas(a.ID, a.Name) },
	Tool: llm.Tool{Name: "rename_canvas", Description: "Give a tab another name, in the person's words, when they ask or when what it holds is no longer what its name says. Home keeps its name.",
		Schema: obj(map[string]any{
			"id":   map[string]any{"type": "string", "description": "The canvas id, from the list of tabs."},
			"name": map[string]any{"type": "string", "description": "The new name, one or two words: Work, Garden, Rome."},
		}, "id", "name")}, Offered: has(records.CanvasType)}

func (s *Service) renameCanvas(id, name string) toolResult {
	name = strings.TrimSpace(name)
	if id == "" {
		return fail("Home keeps its name; rename one of the other tabs")
	}
	if name == "" {
		return fail("pass name: what the tab is to be called")
	}
	rec, err := s.Store.Get(records.CanvasType, id)
	if err != nil {
		return fail("no canvas with id %s; the tabs are listed in the prompt", id)
	}
	for _, c := range s.Canvases() {
		if c.ID != id && strings.EqualFold(c.Name, name) {
			return fail("there is already a tab called %q (canvas %q); give this one another name", c.Name, c.ID)
		}
	}
	was, _ := rec.Fields["name"].(string)
	after := map[string]any{}
	for k, v := range rec.Fields {
		after[k] = v
	}
	after["name"] = name
	done, err := s.Apply(records.Op{Type: records.CanvasType, ID: id, After: after})
	if err != nil {
		return fail("could not rename the canvas: %v", err)
	}
	return toolResult{
		text:   fmt.Sprintf("renamed tab %q to %q, at %s.", was, name, records.CanvasPath(id)) + longTabSaid(name), // canvas_names.go
		change: &records.Change{Action: "updated", Component: records.CanvasType, ID: id, Detail: name, Href: records.CanvasPath(id), Ops: done},
	}
}
