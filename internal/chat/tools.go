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

// ComponentName is the component that renders the conversation. It is a
// block like any other, so it can be moved, restyled, or removed; the
// server fills it with the live transcript when it renders the canvas.
const ComponentName = "chat"

// BlockFields drops keys the workspace's block schema does not define, so
// server code can write provenance and layout fields without checking
// whether an older workspace has them.
func (s *Service) BlockFields(in map[string]any) map[string]any {
	return s.fields(BlockType, in)
}

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
				"span":      map[string]any{"type": "integer", "description": "Width in columns of twelve. 12 is full width, 6 half, 4 a third. Defaults to 6."},
				"frame":     map[string]any{"type": "string", "enum": []string{"card", "bare"}, "description": "card gives the block a surface, bare sits flush on the page. Defaults to card."},
				"tone":      map[string]any{"type": "string", "enum": []string{"none", "accent", "success", "warning", "danger", "info"}, "description": "Tints the block's surface. Defaults to none."},
				"region":    map[string]any{"type": "string", "enum": []string{"main", "side"}, "description": "main is the body of the page; side is the collapsible pane beside it, for things the person glances at. Defaults to main."},
			}, "component", "props")},
		{Name: "update_component", Description: "Change a block already on the canvas: its props, its width, or its place in the order. Props replace the old ones completely, so send them all.",
			Schema: obj(map[string]any{
				"id":       map[string]any{"type": "string", "description": "Block id from the canvas listing."},
				"props":    map[string]any{"type": "object", "description": "The complete new props. Leave out to keep the current ones."},
				"span":     map[string]any{"type": "integer", "description": "New width in columns of twelve."},
				"position": map[string]any{"type": "integer", "description": "New sort order; lower comes first."},
				"frame":    map[string]any{"type": "string", "enum": []string{"card", "bare"}},
				"tone":     map[string]any{"type": "string", "enum": []string{"none", "accent", "success", "warning", "danger", "info"}},
				"region":   map[string]any{"type": "string", "enum": []string{"main", "side"}},
			}, "id")},
		{Name: "remove_component", Description: "Remove one block from the canvas by id.",
			Schema: obj(map[string]any{"id": map[string]any{"type": "string"}}, "id")},
		{Name: "propose_change", Description: "Ask before making a change instead of making it. Use this whenever a change takes something away, and whenever you are guessing at what the person wants. Nothing happens until they answer. Carries one add_component, update_component, or remove_component call.",
			Schema: obj(map[string]any{
				"summary":   map[string]any{"type": "string", "description": "The question, in plain words, ending in a question mark. Say what would change and why you are asking."},
				"tool":      map[string]any{"type": "string", "enum": []string{"add_component", "update_component", "remove_component"}, "description": "The change to make if they say yes."},
				"id":        map[string]any{"type": "string", "description": "Block id, for update or remove."},
				"component": map[string]any{"type": "string", "description": "Component name, for add."},
				"props":     map[string]any{"type": "object"},
				"span":      map[string]any{"type": "integer"},
				"frame":     map[string]any{"type": "string", "enum": []string{"card", "bare"}},
				"tone":      map[string]any{"type": "string", "enum": []string{"none", "accent", "success", "warning", "danger", "info"}},
				"region":    map[string]any{"type": "string", "enum": []string{"main", "side"}},
				"position":  map[string]any{"type": "integer"},
			}, "summary", "tool")},
		{Name: "clear_canvas", Description: "Remove every block from the canvas except the chat, which stays so the person can keep talking. Only when the person asks to start over. To remove the chat too, call remove_component on it.",
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
		Span      *int           `json:"span"`
		Position  *int           `json:"position"`
		Frame     string         `json:"frame"`
		Tone      string         `json:"tone"`
		Region    string         `json:"region"`
		Summary   string         `json:"summary"`
		Tool      string         `json:"tool"`
	}
	if len(call.Args) > 0 {
		if err := json.Unmarshal(call.Args, &args); err != nil {
			return fail("arguments were not valid JSON: %v", err)
		}
	}
	switch call.Name {
	case "propose_change":
		var action map[string]any
		json.Unmarshal(call.Args, &action)
		delete(action, "summary")
		return s.propose(args.Summary, action)
	case "add_component":
		return s.addComponent(args.Component, args.Props, look{args.Span, args.Position, args.Frame, args.Tone, args.Region})
	case "update_component":
		return s.updateComponent(args.ID, args.Props, look{args.Span, args.Position, args.Frame, args.Tone, args.Region})
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
		// Starting over means clearing the content, not deleting the
		// conversation the person is typing into.
		blocks, err := s.Store.List(BlockType, store.ListOptions{})
		if err != nil {
			return fail("could not read the canvas: %v", err)
		}
		n := 0
		for _, b := range blocks {
			if b.Fields["component"] == ComponentName {
				continue
			}
			if err := s.Store.Delete(BlockType, b.ID); err != nil {
				return fail("could not clear the canvas: %v", err)
			}
			n++
		}
		if n == 0 {
			return toolResult{text: "the canvas was already empty"}
		}
		return toolResult{text: fmt.Sprintf("cleared %d blocks; the chat stayed", n), change: &Change{Action: "cleared", Detail: fmt.Sprintf("%d blocks", n)}}
	}
	return fail("unknown tool %s", call.Name)
}

// look is how a block sits on the canvas: its width, its order, and how
// much surface it brings. All optional, all independent of its props.
type look struct {
	Span, Position      *int
	Frame, Tone, Region string
}

// apply writes the layout fields that were given, and describes them.
func (l look) apply(fields map[string]any) ([]string, error) {
	var what []string
	if l.Span != nil {
		if *l.Span < 1 || *l.Span > 12 {
			return nil, fmt.Errorf("span must be between 1 and 12, got %d", *l.Span)
		}
		fields["span"] = *l.Span
		what = append(what, fmt.Sprintf("span %d", *l.Span))
	}
	if l.Position != nil {
		fields["position"] = *l.Position
		what = append(what, fmt.Sprintf("position %d", *l.Position))
	}
	if l.Frame != "" {
		if l.Frame != "card" && l.Frame != "bare" {
			return nil, fmt.Errorf("frame must be card or bare, got %q", l.Frame)
		}
		fields["frame"] = l.Frame
		what = append(what, "frame "+l.Frame)
	}
	if l.Tone != "" {
		fields["tone"] = l.Tone
		what = append(what, "tone "+l.Tone)
	}
	if l.Region != "" {
		if l.Region != "main" && l.Region != "side" {
			return nil, fmt.Errorf("region must be main or side, got %q", l.Region)
		}
		fields["region"] = l.Region
		what = append(what, "region "+l.Region)
	}
	return what, nil
}

func (s *Service) addComponent(name string, props map[string]any, l look) toolResult {
	// Models often capitalise names ("List"); be forgiving about case and space.
	name = strings.ToLower(strings.TrimSpace(name))
	c, ok := s.Registry.Get(name)
	if !ok {
		return fail("unknown component %q. Available: %s", name, strings.Join(s.Registry.Names(), ", "))
	}
	if _, err := c.Validate(props); err != nil {
		return fail("%v. Fix the props and call add_component again.", err)
	}
	// One conversation only: a second would duplicate every message id.
	if name == ComponentName {
		if existing, err := s.Store.List(BlockType, store.ListOptions{}); err == nil {
			for _, b := range existing {
				if b.Fields["component"] == ComponentName {
					return fail("there is already a chat block on the canvas (id %s); move or restyle that one with update_component instead", b.ID)
				}
			}
		}
	}
	position := 0
	if existing, err := s.Store.List(BlockType, store.ListOptions{OrderBy: "position", Desc: true, Limit: 1}); err == nil && len(existing) > 0 {
		if p, ok := existing[0].Fields["position"].(int64); ok {
			position = int(p) + 1
		}
	}
	fields := map[string]any{"component": name, "props": props, "position": position, "actor": "assistant", "created_by": "assistant"}
	if _, err := l.apply(fields); err != nil {
		return fail("%v", err)
	}
	rec, err := s.Store.Create(BlockType, s.fields(BlockType, fields))
	if err != nil {
		return fail("could not save the block: %v", err)
	}
	return toolResult{
		text:   fmt.Sprintf("added %s as block %s at position %d", name, rec.ID, position),
		change: &Change{Action: "added", Component: name, ID: rec.ID, Detail: Summarise(name, props)},
	}
}

func (s *Service) updateComponent(id string, props map[string]any, l look) toolResult {
	rec, err := s.Store.Get(BlockType, id)
	if err != nil {
		return fail("no block with id %s on the canvas", id)
	}
	name, _ := rec.Fields["component"].(string)
	c, ok := s.Registry.Get(name)
	if !ok {
		return fail("block %s uses unknown component %s", id, name)
	}
	fields := map[string]any{"actor": "assistant"}
	var what []string
	if props != nil {
		if _, err := c.Validate(props); err != nil {
			return fail("%v. Fix the props and call update_component again.", err)
		}
		fields["props"] = props
		what = append(what, "props")
	} else {
		props, _ = rec.Fields["props"].(map[string]any)
	}
	layout, err := l.apply(fields)
	if err != nil {
		return fail("%v", err)
	}
	what = append(what, layout...)
	if len(what) == 0 {
		return fail("nothing to change: pass props, span, position, frame, tone, or region")
	}
	if _, err := s.Store.Update(BlockType, id, s.fields(BlockType, fields)); err != nil {
		return fail("could not update block %s: %v", id, err)
	}
	return toolResult{text: "updated " + strings.Join(what, " and ") + " on block " + id, change: &Change{Action: "updated", Component: name, ID: id, Detail: Summarise(name, props)}}
}
