package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

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
	where := fmt.Sprintf("added %s as block %s at position %d", name, rec.ID, position)
	if len(layout) > 0 {
		where += " with " + strings.Join(layout, ", ")
	}
	return toolResult{text: where, change: &Change{Action: "added", Component: name, ID: rec.ID, Detail: Summarise(name, props), Href: "/canvas/" + rec.ID}}
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
