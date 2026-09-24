package chat

import (
	"context"
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

// Tools is what the model can call: the canvas tools, which write ordinary
// block records, and the record tools generated from the workspace's schema.
// It is the one list; /api/describe and the CLI publish it from here, so a
// tool added or changed shows up on every surface at once.
func (s *Service) Tools() []llm.Tool {
	return append([]llm.Tool{
		{Name: "add_component", Description: "Add a component to the canvas the person is looking at. Props must match the component's props schema from the catalogue. Returns the new block id.",
			Schema: obj(map[string]any{
				"component": map[string]any{"type": "string", "description": "Component name from the catalogue."},
				"props":     map[string]any{"type": "object", "description": "Props matching the component's schema."},
				"span":      map[string]any{"type": "integer", "description": "Width in columns of twelve. 12 is full width, 6 half, 4 a third. Defaults to 6."},
				"frame":     map[string]any{"type": "string", "enum": []string{"card", "bare"}, "description": "card gives the block a surface, bare sits flush on the page. Defaults to card."},
				"tone":      map[string]any{"type": "string", "enum": []string{"none", "accent", "success", "warning", "danger", "info"}, "description": "Tints the block's surface. Defaults to none."},
				"region":    map[string]any{"type": "string", "enum": []string{"main", "left", "right", "header", "footer"}, "description": "main is the body of the page. left and right are full height panes beside it: left for history and navigation, right for what the person glances at. header is the bar at the top, for what they reach for on every page; footer the bar at the bottom. Defaults to main."},
				"size":      map[string]any{"type": "string", "enum": []string{"full", "compact", "icon"}, "description": "full is the whole thing (the default). compact fits more on a page. icon is a glyph with its name for screen readers that opens the full thing; for people who know what it is."},
				"canvas":    map[string]any{"type": "string", "description": "Which tab the block goes on, as a canvas id from the list of tabs; empty string is Home. Defaults to the tab the person is looking at."},
			}, "component", "props")},
		{Name: "update_component", Description: "Change a block already on the canvas: its props, its width, or its place in the order. Props replace the old ones completely, so send them all.",
			Schema: obj(map[string]any{
				"id":       map[string]any{"type": "string", "description": "Block id from the canvas listing."},
				"props":    map[string]any{"type": "object", "description": "The complete new props. Leave out to keep the current ones."},
				"span":     map[string]any{"type": "integer", "description": "New width in columns of twelve."},
				"position": map[string]any{"type": "integer", "description": "New sort order; lower comes first."},
				"frame":    map[string]any{"type": "string", "enum": []string{"card", "bare"}},
				"tone":     map[string]any{"type": "string", "enum": []string{"none", "accent", "success", "warning", "danger", "info"}},
				"region":   map[string]any{"type": "string", "enum": []string{"main", "left", "right", "header", "footer"}},
				"size":     map[string]any{"type": "string", "enum": []string{"full", "compact", "icon"}},
				"canvas":   map[string]any{"type": "string", "description": "Move the block to another tab: a canvas id, or empty string for Home."},
			}, "id")},
		{Name: "remove_component", Description: "Remove one block from the canvas by id.",
			Schema: obj(map[string]any{"id": map[string]any{"type": "string"}}, "id")},
		{Name: "propose_change", Description: "Ask before making a change instead of making it. Use this whenever a change takes something away, and whenever you are guessing at what the person wants. Nothing happens until they answer. Carries one add_component, update_component, or remove_component call.",
			Schema: obj(map[string]any{
				"summary":   map[string]any{"type": "string", "description": "The question, in plain words, ending in a question mark. Say what would change and why you are asking."},
				"tool":      map[string]any{"type": "string", "enum": []string{"add_component", "update_component", "remove_component", "remove_canvas"}, "description": "The change to make if they say yes."},
				"id":        map[string]any{"type": "string", "description": "Block id, for update or remove."},
				"component": map[string]any{"type": "string", "description": "Component name, for add."},
				"props":     map[string]any{"type": "object"},
				"span":      map[string]any{"type": "integer"},
				"frame":     map[string]any{"type": "string", "enum": []string{"card", "bare"}},
				"tone":      map[string]any{"type": "string", "enum": []string{"none", "accent", "success", "warning", "danger", "info"}},
				"region":    map[string]any{"type": "string", "enum": []string{"main", "left", "right", "header", "footer"}},
				"size":      map[string]any{"type": "string", "enum": []string{"full", "compact", "icon"}},
				"canvas":    map[string]any{"type": "string"},
				"position":  map[string]any{"type": "integer"},
			}, "summary", "tool")},
		{Name: "clear_canvas", Description: "Remove every block from the canvas except the chat, which stays so the person can keep talking. Only when the person asks to start over. To remove the chat too, call remove_component on it.",
			Schema: obj(map[string]any{})},
		undoTool,
		searchTool,
		actionTool,
		updateTool,
		s.arrangementTool(),
		settingTool,
	}, append(append(append(s.recordTools(), s.canvasTools()...), shapeTools()...), s.lookTools()...)...)
}

// runTool executes one tool call.
func (s *Service) runTool(call llm.ToolCall) toolResult {
	var args struct {
		File        string         `json:"file"`
		Mapping     map[string]any `json:"mapping"`
		Component   string         `json:"component"`
		ID          string         `json:"id"`
		Props       map[string]any `json:"props"`
		Span        *int           `json:"span"`
		Position    *int           `json:"position"`
		Frame       string         `json:"frame"`
		Tone        string         `json:"tone"`
		Region      string         `json:"region"`
		Size        string         `json:"size"`
		Canvas      *string        `json:"canvas"`
		Name        string         `json:"name"`
		Summary     string         `json:"summary"`
		Tool        string         `json:"tool"`
		Type        string         `json:"type"`
		Fields      map[string]any `json:"fields"`
		Query       string         `json:"query"`
		Where       []string       `json:"where"`
		Order       string         `json:"order"`
		Limit       int            `json:"limit"`
		Key         string         `json:"key"`
		Install     bool           `json:"install"`
		Value       string         `json:"value"`
		Kind        string         `json:"kind"`
		Description string         `json:"description"`
		Values      []string       `json:"values"`
		To          string         `json:"to"`
		Required    bool           `json:"required"`
		Default     any            `json:"default"`
		Properties  []fieldDef     `json:"properties"`
		Fills       map[string]any `json:"fills"`
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
	case "look_at_page":
		return s.lookAtPage(call.Args)
	case "create_canvas":
		return s.createCanvas(args.Name)
	case "remove_canvas":
		return s.removeCanvas(args.ID)
	case "create_record":
		return s.createRecord(args.Type, args.Fields)
	case "import_records":
		return s.importRecords(args.Type, args.File, args.Mapping)
	case "update_record":
		return s.updateRecord(args.Type, args.ID, args.Fields)
	case "find_records":
		return s.findRecords(args.Type, args.Query, args.Where, args.Order, args.Limit)
	case "get_record":
		return s.getRecord(args.Type, args.ID)
	case "add_field":
		return s.addField(args.Type, fieldDef{Name: args.Name, Kind: args.Kind, Description: args.Description, Values: args.Values, To: args.To, Required: args.Required, Default: args.Default})
	case "add_type":
		return s.addType(args.Name, args.Description, args.Properties)
	case "add_component":
		return s.addComponent(args.Component, args.Props, look{Span: args.Span, Position: args.Position, Frame: args.Frame, Tone: args.Tone, Region: args.Region, Size: args.Size, Canvas: deref(args.Canvas), SetCanvas: args.Canvas != nil})
	case "update_component":
		return s.updateComponent(args.ID, args.Props, look{Span: args.Span, Position: args.Position, Frame: args.Frame, Tone: args.Tone, Region: args.Region, Size: args.Size, Canvas: deref(args.Canvas), SetCanvas: args.Canvas != nil})
	case "remove_component":
		return s.removeBlock(args.ID)
	case "undo_change":
		return s.Undo(args.ID)
	case "add_arrangement":
		return s.addArrangement(args.Name, args.Fills)
	case "search":
		return s.search(args.Query)
	case "run_action":
		// What cannot be taken back is asked first; see consent.go.
		if r, ask := s.askFirst("run_action", args.ID, "", ""); ask {
			return r
		}
		return s.Run(context.Background(), args.ID, s.current)
	case "accept_action":
		return s.acceptAction(context.Background(), args.ID)
	case "update_sameway":
		return s.updateSameway(args.Install)
	case "set_setting":
		if r, ask := s.askFirst("set_setting", "", args.Key, args.Value); ask {
			return r
		}
		return s.setSetting(args.Key, args.Value)
	case "clear_canvas":
		// Starting over means clearing the content, not deleting the
		// conversation the person is typing into.
		blocks, err := s.Store.List(BlockType, store.ListOptions{})
		if err != nil {
			return fail("could not read the canvas: %v", err)
		}
		var gone []*store.Record
		for _, b := range blocks {
			if b.Fields["component"] == ComponentName {
				continue
			}
			// Only the tab the person is looking at: the others keep theirs.
			if on, _ := b.Fields["canvas"].(string); on != s.current {
				continue
			}
			if err := s.Store.Delete(BlockType, b.ID); err != nil {
				return fail("could not clear the canvas: %v", err)
			}
			gone = append(gone, b)
		}
		if len(gone) == 0 {
			return toolResult{text: "the canvas was already empty"}
		}
		// What was cleared goes in the log, so it can be put back whole.
		return toolResult{text: fmt.Sprintf("cleared %d blocks; the chat stayed", len(gone)), change: &Change{Action: "cleared", Detail: fmt.Sprintf("%d blocks", len(gone)), Before: map[string]any{"blocks": keep(gone)}}}
	}
	return fail("unknown tool %s", call.Name)
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
	// On the tab the person is looking at, unless the call says otherwise.
	fields := map[string]any{"component": name, "props": props, "position": position, "actor": "assistant", "created_by": "assistant", "canvas": s.current}
	if l.SetCanvas && !s.HasCanvas(l.Canvas) {
		return fail("no canvas with id %q; the tabs and their ids are listed in the prompt, and \"\" is Home", l.Canvas)
	}
	layout, err := l.apply(fields)
	if err != nil {
		return fail("%v", err)
	}
	rec, err := s.Store.Create(BlockType, s.fields(BlockType, fields))
	if err != nil {
		return fail("could not save the block: %v", err)
	}
	// Say where it went, so the model confirms what really happened
	// rather than what it asked for.
	where := fmt.Sprintf("added %s as block %s at position %d", name, rec.ID, position)
	if len(layout) > 0 {
		where += " with " + strings.Join(layout, ", ")
	}
	return toolResult{
		text:   where,
		change: &Change{Action: "added", Component: name, ID: rec.ID, Detail: Summarise(name, props), Href: "/canvas/" + rec.ID},
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
	if l.SetCanvas && !s.HasCanvas(l.Canvas) {
		return fail("no canvas with id %q; the tabs and their ids are listed in the prompt, and \"\" is Home", l.Canvas)
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
	return toolResult{text: "updated " + strings.Join(what, " and ") + " on block " + id, change: &Change{Action: "updated", Component: name, ID: id, Detail: Summarise(name, props), Href: "/canvas/" + id, Before: rec.Fields}}
}
