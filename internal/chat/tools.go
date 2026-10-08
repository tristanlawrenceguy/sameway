package chat

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// BlockFields drops keys the workspace's block schema does not define, so
// server code can write provenance and layout fields without checking
// whether an older workspace has them.
func (s *Service) BlockFields(in map[string]any) map[string]any {
	return s.fields(records.BlockType, in)
}

// blockOps place and change the blocks on the canvas, and put a change to
// the person before making it.
var blockOps = []Op{
	{Title: "Add a block",
		Tool: llm.Tool{Name: "add_component", Description: "Add a component to the canvas the person is looking at. Props must match the component's props schema; a refusal gives the schema and an example. Returns the new block id and what it shows; read it: \"nothing yet\" means it shows no records now. A block that could not be shown (a type, field, date field, condition or tag the workspace does not have, one field asked for two values, a chart by a date with no period) is not added, and the error says why and what to do instead.",
			Schema: obj(map[string]any{
				"component": map[string]any{"type": "string", "description": "Component name, as the prompt lists it."},
				"props":     map[string]any{"type": "object", "description": "Props matching the component's schema."},
				"span":      map[string]any{"type": "integer", "description": "Width in columns of twelve. 12 is full width, 6 half, 4 a third. Defaults to 6."},
				"frame":     map[string]any{"type": "string", "enum": []string{"card", "bare"}, "description": "card gives the block a surface, bare sits flush on the page. Defaults to card."},
				"tone":      map[string]any{"type": "string", "enum": []string{"none", "accent", "success", "warning", "danger", "info"}, "description": "Tints the block's surface. Defaults to none."},
				"region":    map[string]any{"type": "string", "enum": []string{"main", "left", "right", "header", "footer"}, "description": "main is the body of the page. left and right are full height panes beside it: left for history and navigation, right for what the person glances at. header is the bar at the top, for what they reach for on every page; footer the bar at the bottom. Defaults to main."},
				"size":      map[string]any{"type": "string", "enum": []string{"full", "compact", "icon"}, "description": "full is the whole thing (the default). compact fits more on a page. icon is a glyph with its name for screen readers that opens the full thing; for people who know what it is."},
				"canvas":    map[string]any{"type": "string", "description": "Which tab the block goes on, as a canvas id from the list of tabs; empty string is Home. Defaults to the tab the person is looking at."},
			}, "component", "props")}},
	{Title: "Change a block", Traits: Traits{Idempotent: true},
		Tool: llm.Tool{Name: "update_component", Description: "Change a block already on the canvas: its props, its width, or its place in the order. Props replace the old ones completely, so send them all.",
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
			}, "id")}},
	{Title: "Remove a block", Traits: Traits{Destructive: true, Idempotent: true},
		Tool: llm.Tool{Name: "remove_component", Description: "Remove one block from the canvas by id.",
			Schema: obj(map[string]any{"id": map[string]any{"type": "string"}}, "id")}},
	arrangeOp,
	{Title: "Ask the person before a change",
		Tool: llm.Tool{Name: "propose_change", Description: "Ask before making a change instead of making it. Use this whenever a change takes something away, and whenever you are guessing at what the person wants. Nothing happens until they answer. Carries one add_component, update_component, or remove_component call.",
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
			}, "summary", "tool")}},
	{Title: "Clear the page", Traits: Traits{Destructive: true, Idempotent: true},
		Tool: llm.Tool{Name: "clear_canvas", Description: "Remove every block from the canvas except the chat, which stays so the person can keep talking. Only when the person asks to start over. To remove the chat too, call remove_component on it.",
			Schema: obj(map[string]any{})}},
}

// runTool executes one tool call, by the handler its name has in
// toolHandlers (tool_handlers.go).
func (s *Service) runTool(call llm.ToolCall) toolResult {
	var args toolArgs
	call.Args = loosen(call.Name, call.Args) // loose_args.go
	if len(call.Args) > 0 {
		if err := json.Unmarshal(call.Args, &args); err != nil {
			return fail("%s", ArgsTrouble(err))
		}
	}
	if run, ok := toolHandlers()[call.Name]; ok {
		return run(s, args, call)
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
	shows, bad := s.writable("add_component", c, props)
	if bad != nil {
		return *bad
	}
	// One conversation only: a second would duplicate every message id.
	if name == records.ComponentName {
		if existing, err := s.Store.List(records.BlockType, store.ListOptions{}); err == nil {
			for _, b := range existing {
				if b.Fields["component"] == records.ComponentName {
					return fail("there is already a chat block on the canvas (id %s); move or restyle that one with update_component instead", b.ID)
				}
			}
		}
	}
	position := 0
	if existing, err := s.Store.List(records.BlockType, store.ListOptions{OrderBy: "position", Desc: true, Limit: 1}); err == nil && len(existing) > 0 {
		if p, ok := existing[0].Fields["position"].(int64); ok {
			position = int(p) + 1
		}
	}
	// On the tab the person is looking at, unless the call says otherwise.
	fields := s.marked(map[string]any{"component": name, "props": props, "position": position, "created_by": s.actor(), "canvas": s.current})
	if l.SetCanvas && !s.HasCanvas(l.Canvas) {
		return fail("no canvas with id %q; the tabs and their ids are listed in the prompt, and \"\" is Home", l.Canvas)
	}
	layout, err := l.apply(fields)
	if err != nil {
		return fail("%v", err)
	}
	if why := s.skipIfWritten(fields["canvas"].(string), withFields(&store.Record{ID: "new"}, fields)); why != "" {
		return fail("not added: %s", why)
	}
	rec, err := s.Store.Create(records.BlockType, s.fields(records.BlockType, fields))
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
		text:   showing(where, shows),
		change: &records.Change{Action: "added", Component: name, ID: rec.ID, Detail: records.Summarise(name, props), Href: "/canvas/" + rec.ID},
	}
}

func (s *Service) updateComponent(id string, props map[string]any, l look) toolResult {
	rec, err := s.Store.Get(records.BlockType, id)
	if err != nil {
		return fail("no block with id %s on the canvas", id)
	}
	name, _ := rec.Fields["component"].(string)
	c, ok := s.Registry.Get(name)
	if !ok {
		return fail("block %s uses unknown component %s", id, name)
	}
	fields := s.marked(map[string]any{})
	var what []string
	var shows string
	if props != nil {
		var bad *toolResult
		if shows, bad = s.writable("update_component", c, props); bad != nil {
			return *bad
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
	if on, _ := withFields(rec, fields).Fields["canvas"].(string); s.skipIfWritten(on, withFields(rec, fields)) != "" {
		return fail("not changed: %s", s.skipIfWritten(on, withFields(rec, fields)))
	}
	if _, err := s.Store.Update(records.BlockType, id, s.fields(records.BlockType, fields)); err != nil {
		return fail("could not update block %s: %v", id, err)
	}
	return toolResult{text: showing("updated "+strings.Join(what, " and ")+" on block "+id, shows), change: &records.Change{Action: "updated", Component: name, ID: id, Detail: records.Summarise(name, props), Href: "/canvas/" + id, Before: rec.Fields}}
}
