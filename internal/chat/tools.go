package chat

import (
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

// lookProps are the fields of a block beside its props, as a tool's schema
// takes them: where it sits and how it looks. said gives the choices their
// words, for the tool that adds a block; the tools that change one name
// the same choices and need no more.
func lookProps(said bool, more map[string]any) map[string]any {
	choice := func(words string, values ...string) map[string]any {
		p := map[string]any{"type": "string", "enum": values}
		if said {
			p["description"] = words
		}
		return p
	}
	props := map[string]any{
		"span":     map[string]any{"type": "integer", "description": "Width in columns of twelve: 12 full, 6 half, 4 a third."},
		"frame":    choice("card has a surface (the default); bare sits flush on the page.", frames...),
		"tone":     choice("Tints the surface. Defaults to none.", tones...),
		"region":   choice("main is the body of the page (the default). left and right are full height panes beside it: left for history and navigation, right for what the person glances at. header and footer are the bars at the top and bottom, for what they reach for on every page.", regions...),
		"size":     choice("full is the default. compact fits more on a page. icon is a glyph that opens the full thing, for people who know what it is.", sizes...),
		"canvas":   map[string]any{"type": "string", "description": "The tab, as a canvas id from the list of tabs; empty string is Home."},
		"position": map[string]any{"type": "integer", "description": "Sort order; lower comes first."},
	}
	for k, v := range more {
		props[k] = v
	}
	return props
}

// blockOps place and change the blocks on the canvas, and put a change to
// the person before making it.
var blockOps = []Op{
	{Title: "Add a block",
		Core:  true,
		Doing: func(a callArgs) string { return "Adding" + an(thing(a.Component, a.Props)) },
		Run: func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.addComponent(a.Component, a.Props, a.look())
		},
		Tool: llm.Tool{Name: "add_component", Description: "Add a component to the canvas: half width, last, on the tab the person is looking at, unless you say. Returns the new block id and what it shows; read it: \"nothing yet\" means it shows no records now. Props that do not fit, or a block that could not be shown, add nothing, and the error says why and what to do instead.",
			Schema: obj(lookProps(true, map[string]any{
				"component": map[string]any{"type": "string", "description": "Component name, as the prompt lists it."},
				"props":     map[string]any{"type": "object", "description": "Props matching the component's schema."},
			}), "component", "props")}},
	{Title: "Change a block", Traits: Traits{Idempotent: true},
		Doing: saying("Changing a block"),
		Run: func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.updateComponent(a.ID, a.Props, a.look())
		},
		Tool: llm.Tool{Name: "update_component", Description: "Change a block already on the canvas: its props, its width and look, its place in the order, or its tab. Props replace the old ones completely, so send them all.",
			Schema: obj(lookProps(false, map[string]any{
				"id":    map[string]any{"type": "string", "description": "Block id from the canvas listing."},
				"props": map[string]any{"type": "object", "description": "The complete new props. Leave out to keep the current ones."},
			}), "id")}},
	{Title: "Remove a block", Traits: Traits{Destructive: true, Idempotent: true},
		Doing: removing,
		Run:   func(s *Service, a toolArgs, call llm.ToolCall) toolResult { return s.removeCall(a) },
		Tool: llm.Tool{Name: "remove_component", Description: "Remove blocks from the canvas: one by id, or several by ids in one change that one Undo takes back. If any id is not a block, nothing is removed.",
			Schema: obj(map[string]any{
				"id":  map[string]any{"type": "string", "description": "One block id from the canvas listing."},
				"ids": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Several block ids."},
			})}},
	arrangeOp,
	{Title: "Ask the person before a change",
		Core:  true,
		Doing: saying("Asking you about a change"),
		Run: func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
			return s.proposeByModel(a.Summary, call.Args)
		},
		Tool: llm.Tool{Name: "propose_change", Description: "Ask before making a change instead of making it. Use this whenever a change takes something away, and whenever you are guessing at what the person wants. Nothing happens until they answer. Carries one call of the tool named, with that tool's arguments.",
			Schema: obj(lookProps(false, map[string]any{
				"summary":   map[string]any{"type": "string", "description": "The question, in plain words, ending in a question mark. Say what would change and why you are asking."},
				"tool":      map[string]any{"type": "string", "enum": []string{"add_component", "update_component", "remove_component", "remove_canvas"}, "description": "The change to make if they say yes."},
				"id":        map[string]any{"type": "string", "description": "The block, or the tab for remove_canvas."},
				"ids":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"component": map[string]any{"type": "string"},
				"props":     map[string]any{"type": "object"},
			}), "summary", "tool")}},
	{Title: "Clear the page", Traits: Traits{Destructive: true, Idempotent: true},
		Doing: saying("Clearing the page"),
		Run:   func(s *Service, a toolArgs, call llm.ToolCall) toolResult { return s.clearCanvas() },
		Tool: llm.Tool{Name: "clear_canvas", Description: "Remove every block from the tab the person is looking at except the chat, which only remove_component takes. Only when the person asks to start over.",
			Schema: obj(map[string]any{})}},
}

func (s *Service) addComponent(name string, props map[string]any, l look) toolResult {
	// Models often capitalise names ("List"); be forgiving about case and space.
	name = strings.ToLower(strings.TrimSpace(name))
	c, ok := s.Registry.Get(name)
	if !ok {
		return fail("%s", s.noComponent(name))
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
	layout, err := s.place(l, fields)
	if err != nil {
		return fail("%v", err)
	}
	if why := s.skipIfWritten(fields["canvas"].(string), withFields(&store.Record{ID: "new"}, fields)); why != "" {
		return fail("not added: %s", why)
	}
	done, err := s.Apply(records.Op{Type: records.BlockType, After: s.fields(records.BlockType, fields)})
	if err != nil {
		return fail("could not save the block: %v", err)
	}
	rec := done[0]
	// Say where it went, so the model confirms what really happened
	// rather than what it asked for.
	where := fmt.Sprintf("added %s as block %s at position %d", name, rec.ID, position)
	if len(layout) > 0 {
		where += " with " + strings.Join(layout, ", ")
	}
	return toolResult{
		text:   showing(where, shows),
		change: &records.Change{Action: "added", Component: name, ID: rec.ID, Detail: records.Summarise(name, props), Href: "/canvas/" + rec.ID, Ops: done},
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
	layout, err := s.place(l, fields)
	if err != nil {
		return fail("%v", err)
	}
	what = append(what, layout...)
	if len(what) == 0 {
		return fail("nothing to change: pass props, span, position, frame, tone, region, size or canvas")
	}
	after := withFields(rec, fields)
	on, _ := after.Fields["canvas"].(string)
	if why := s.skipIfWritten(on, after); why != "" {
		return fail("not changed: %s", why)
	}
	done, err := s.Apply(records.Op{Type: records.BlockType, ID: id, After: s.fields(records.BlockType, fields)})
	if err != nil {
		return fail("could not update block %s: %v", id, err)
	}
	return toolResult{text: showing("updated "+strings.Join(what, " and ")+" on block "+id, shows), change: &records.Change{Action: "updated", Component: name, ID: id, Detail: records.Summarise(name, props), Href: "/canvas/" + id, Ops: done}}
}
