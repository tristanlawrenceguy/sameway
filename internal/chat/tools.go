package chat

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// BlockType is the content type that holds canvas items.
const BlockType = "block"

// Tools the model can call. Every tool operates on block records, so the
// canvas is ordinary content that the CLI and API can also read and edit.
func (s *Service) tools() []llm.Tool {
	// Strict servers (llama.cpp builds a grammar from this) reject
	// "required": null, so the key is only present when there is a list.
	obj := func(props map[string]any, required ...string) map[string]any {
		s := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
		if len(required) > 0 {
			s["required"] = required
		}
		return s
	}
	return []llm.Tool{
		{Name: "add_component", Description: "Add a component to the canvas the person is looking at. Props must match the component's props schema from the catalogue. Returns the new block id.",
			Schema: obj(map[string]any{
				"component": map[string]any{"type": "string", "description": "Component name from the catalogue."},
				"props":     map[string]any{"type": "object", "description": "Props matching the component's schema."},
			}, "component", "props")},
		{Name: "update_component", Description: "Replace the props of a block already on the canvas. Send the complete new props, not a partial patch.",
			Schema: obj(map[string]any{
				"id":    map[string]any{"type": "string", "description": "Block id from the canvas listing."},
				"props": map[string]any{"type": "object"},
			}, "id", "props")},
		{Name: "remove_component", Description: "Remove one block from the canvas by id.",
			Schema: obj(map[string]any{"id": map[string]any{"type": "string"}}, "id")},
		{Name: "clear_canvas", Description: "Remove every block from the canvas. Only when the person asks to start over.",
			Schema: obj(map[string]any{})},
	}
}

// toolResult is what one tool call produced: text for the model, an error
// flag, and the change to record if any.
type toolResult struct {
	text   string
	isErr  bool
	change *Change
}

func fail(format string, args ...any) toolResult {
	return toolResult{text: fmt.Sprintf(format, args...), isErr: true}
}

// runTool executes one tool call.
func (s *Service) runTool(call llm.ToolCall) toolResult {
	var args struct {
		Component string         `json:"component"`
		ID        string         `json:"id"`
		Props     map[string]any `json:"props"`
	}
	if len(call.Args) > 0 {
		if err := json.Unmarshal(call.Args, &args); err != nil {
			return fail("arguments were not valid JSON: %v", err)
		}
	}
	switch call.Name {
	case "add_component":
		return s.addComponent(args.Component, args.Props)
	case "update_component":
		return s.updateComponent(args.ID, args.Props)
	case "remove_component":
		rec, err := s.Store.Get(BlockType, args.ID)
		if err != nil {
			return fail("could not remove block %s: %v", args.ID, err)
		}
		if err := s.Store.Delete(BlockType, args.ID); err != nil {
			return fail("could not remove block %s: %v", args.ID, err)
		}
		name, _ := rec.Fields["component"].(string)
		props, _ := rec.Fields["props"].(map[string]any)
		return toolResult{text: "removed block " + args.ID, change: &Change{Action: "removed", Component: name, ID: args.ID, Detail: Summarise(name, props)}}
	case "clear_canvas":
		n, _ := s.Store.Count(BlockType)
		if err := s.Store.DeleteAll(BlockType); err != nil {
			return fail("could not clear the canvas: %v", err)
		}
		return toolResult{text: "canvas cleared", change: &Change{Action: "cleared", Detail: fmt.Sprintf("%d blocks", n)}}
	}
	return fail("unknown tool %s", call.Name)
}

func (s *Service) addComponent(name string, props map[string]any) toolResult {
	// Models often capitalise names ("List"); be forgiving about case and space.
	name = strings.ToLower(strings.TrimSpace(name))
	c, ok := s.Registry.Get(name)
	if !ok {
		return fail("unknown component %q. Available: %s", name, strings.Join(s.Registry.Names(), ", "))
	}
	if _, err := c.Validate(props); err != nil {
		return fail("%v. Fix the props and call add_component again.", err)
	}
	position := 0
	if existing, err := s.Store.List(BlockType, store.ListOptions{OrderBy: "position", Desc: true, Limit: 1}); err == nil && len(existing) > 0 {
		if p, ok := existing[0].Fields["position"].(int64); ok {
			position = int(p) + 1
		}
	}
	rec, err := s.Store.Create(BlockType, s.fields(BlockType, map[string]any{"component": name, "props": props, "position": position, "actor": "assistant", "created_by": "assistant"}))
	if err != nil {
		return fail("could not save the block: %v", err)
	}
	return toolResult{
		text:   fmt.Sprintf("added %s as block %s at position %d", name, rec.ID, position),
		change: &Change{Action: "added", Component: name, ID: rec.ID, Detail: Summarise(name, props)},
	}
}

func (s *Service) updateComponent(id string, props map[string]any) toolResult {
	rec, err := s.Store.Get(BlockType, id)
	if err != nil {
		return fail("no block with id %s on the canvas", id)
	}
	name, _ := rec.Fields["component"].(string)
	c, ok := s.Registry.Get(name)
	if !ok {
		return fail("block %s uses unknown component %s", id, name)
	}
	if _, err := c.Validate(props); err != nil {
		return fail("%v. Fix the props and call update_component again.", err)
	}
	if _, err := s.Store.Update(BlockType, id, s.fields(BlockType, map[string]any{"props": props, "actor": "assistant"})); err != nil {
		return fail("could not update block %s: %v", id, err)
	}
	return toolResult{text: "updated block " + id, change: &Change{Action: "updated", Component: name, ID: id, Detail: Summarise(name, props)}}
}
