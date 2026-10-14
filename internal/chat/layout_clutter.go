package chat

import (
	"encoding/json"
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// What clutters a tab, beside how it is laid out: a block that can no
// longer be shown, and a block the same as another. Neither was said: the
// model was told the rules (keep the page calm, update rather than add a
// near duplicate) and given the blocks as their props, so a tab kept two
// "could not be shown" buttons and the habit tracker twice, and nothing
// was ever taken away. Said beside each block in the canvas listing and in
// Layout now after every write, it is in front of the model when it acts.

// clutter is what is wrong with each block of a tab that should go or be
// fixed, by id: it cannot be shown, and why; or it is the same as an
// earlier one.
func (s *Service) clutter(recs []*store.Record) map[string]string {
	out, _ := s.blockState(recs)
	return out
}

// blockState is clutter, and what each block that can be shown shows now
// ("4 tasks, not done, by due"; "nothing yet"), from one resolving of each:
// the listing gives the model what the person sees, not only the props.
func (s *Service) blockState(recs []*store.Record) (bad, shows map[string]string) {
	out, shows := map[string]string{}, map[string]string{}
	seen := map[string]string{}
	for _, b := range recs {
		name, _ := b.Fields["component"].(string)
		props, _ := b.Fields["props"].(map[string]any)
		said, why := s.cannotShow(name, props)
		if said != "" {
			shows[b.ID] = said
		}
		if why != "" {
			out[b.ID] = "cannot be shown: " + why
			continue
		}
		raw, _ := json.Marshal(props)
		key := name + string(raw)
		if first, ok := seen[key]; ok {
			out[b.ID] = "the same as block " + first
			continue
		}
		seen[key] = b.ID
	}
	return out, shows
}

// cannotShow is what a block shows, or why it would only say on the page
// that it is set up wrong.
func (s *Service) cannotShow(name string, props map[string]any) (shows, why string) {
	if s.Registry == nil {
		return "", ""
	}
	c, ok := s.Registry.Get(name)
	if !ok {
		return "", "there is no component " + name
	}
	if _, err := c.Validate(props); err != nil {
		return "", PropsTrouble(err)
	}
	// As the page draws it, not as a new write is checked: a block stored
	// before that check still renders, and is not clutter.
	if s.Blocks == nil {
		return "", ""
	}
	return blocks.Renders(s.Blocks, name, props)
}

// clutterProblems are Layout now's lines for them.
func (s *Service) clutterProblems(recs []*store.Record) []string {
	bad := s.clutter(recs)
	var out []string
	for _, b := range recs {
		if why, ok := bad[b.ID]; ok {
			out = append(out, fmt.Sprintf("block %s (%s) %s; remove it with remove_component unless the person wants it", b.ID, b.Fields["component"], why))
		}
	}
	return out
}

// Clutter is clutter for a page that offers the person to tidy it.
func (s *Service) Clutter(recs []*store.Record) map[string]string { return s.clutter(recs) }
