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
	return []llm.Tool{
		{
			Name:        "add_component",
			Description: "Add a component to the canvas the person is looking at. Props must match the component's props schema from the catalogue. Returns the new block id.",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"component": map[string]any{"type": "string", "description": "Component name from the catalogue."},
					"props":     map[string]any{"type": "object", "description": "Props matching the component's schema."},
				},
				"required":             []string{"component", "props"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "update_component",
			Description: "Replace the props of a block already on the canvas. Send the complete new props, not a partial patch.",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":    map[string]any{"type": "string", "description": "Block id from the canvas listing."},
					"props": map[string]any{"type": "object"},
				},
				"required":             []string{"id", "props"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "remove_component",
			Description: "Remove one block from the canvas by id.",
			Schema: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{"id": map[string]any{"type": "string"}},
				"required":             []string{"id"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "clear_canvas",
			Description: "Remove every block from the canvas. Only when the person asks to start over.",
			Schema:      map[string]any{"type": "object", "properties": map[string]any{}, "additionalProperties": false},
		},
	}
}

// runTool executes one tool call and returns text for the model.
func (s *Service) runTool(call llm.ToolCall) (string, bool) {
	var args struct {
		Component string         `json:"component"`
		ID        string         `json:"id"`
		Props     map[string]any `json:"props"`
	}
	if len(call.Args) > 0 {
		if err := json.Unmarshal(call.Args, &args); err != nil {
			return "arguments were not valid JSON: " + err.Error(), true
		}
	}
	switch call.Name {
	case "add_component":
		return s.addComponent(args.Component, args.Props)
	case "update_component":
		return s.updateComponent(args.ID, args.Props)
	case "remove_component":
		if err := s.Store.Delete(BlockType, args.ID); err != nil {
			return "could not remove block " + args.ID + ": " + err.Error(), true
		}
		return "removed block " + args.ID, false
	case "clear_canvas":
		if err := s.Store.DeleteAll(BlockType); err != nil {
			return "could not clear the canvas: " + err.Error(), true
		}
		return "canvas cleared", false
	}
	return "unknown tool " + call.Name, true
}

func (s *Service) addComponent(name string, props map[string]any) (string, bool) {
	// Models often capitalise names ("List"); be forgiving about case and space.
	name = strings.ToLower(strings.TrimSpace(name))
	c, ok := s.Registry.Get(name)
	if !ok {
		return fmt.Sprintf("unknown component %q. Available: %s", name, strings.Join(s.Registry.Names(), ", ")), true
	}
	if _, err := c.Validate(props); err != nil {
		return err.Error() + ". Fix the props and call add_component again.", true
	}
	position := 0
	if existing, err := s.Store.List(BlockType, store.ListOptions{OrderBy: "position", Desc: true, Limit: 1}); err == nil && len(existing) > 0 {
		if p, ok := existing[0].Fields["position"].(int64); ok {
			position = int(p) + 1
		}
	}
	rec, err := s.Store.Create(BlockType, map[string]any{"component": name, "props": props, "position": position})
	if err != nil {
		return "could not save the block: " + err.Error(), true
	}
	return fmt.Sprintf("added %s as block %s at position %d", name, rec.ID, position), false
}

func (s *Service) updateComponent(id string, props map[string]any) (string, bool) {
	rec, err := s.Store.Get(BlockType, id)
	if err != nil {
		return "no block with id " + id + " on the canvas", true
	}
	name, _ := rec.Fields["component"].(string)
	c, ok := s.Registry.Get(name)
	if !ok {
		return "block " + id + " uses unknown component " + name, true
	}
	if _, err := c.Validate(props); err != nil {
		return err.Error() + ". Fix the props and call update_component again.", true
	}
	if _, err := s.Store.Update(BlockType, id, map[string]any{"props": props}); err != nil {
		return "could not update block " + id + ": " + err.Error(), true
	}
	return "updated block " + id, false
}
