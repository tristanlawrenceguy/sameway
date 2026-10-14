package chat

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// What clutters a tab, beside how it is laid out: a block that can no
// longer be shown, and a block the same as another. Neither was said: the
// model was told the rules (keep the page calm, update rather than add a
// near duplicate) and given the blocks as their props, so a tab kept two
// "could not be shown" buttons and the habit tracker twice, and nothing
// was ever taken away. Said once, in Layout now's to look at, which the
// model reads at the start of a turn and after every write, it is in front
// of the model when it acts.

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
			// What it shows, without the advice an add gives with it.
			shows[b.ID], _, _ = strings.Cut(said, " (it fills in")
		}
		if why != "" {
			// Why, without the whole of what the component takes: a
			// refusal gives that when the block is changed.
			why, _, _ = strings.Cut(why, "; "+name+" takes: ")
			out[b.ID] = "cannot be shown: " + strings.TrimSuffix(why, ".")
			continue
		}
		raw, _ := json.Marshal(props)
		key := name + string(raw)
		if first, ok := seen[key]; ok {
			out[b.ID] = "is the same as block " + first
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
		return "", storedTrouble.Replace(PropsTrouble(err))
	}
	// As the page draws it, not as a new write is checked: a block stored
	// before that check still renders, and is not clutter.
	if s.Blocks == nil {
		return "", ""
	}
	return blocks.Renders(s.Blocks, name, props)
}

// storedTrouble says a write's refusal of props as one line about a block
// already stored: nothing was being saved, and a numbered list of what to
// look at reads wrong broken over lines.
var storedTrouble = strings.NewReplacer(" props do not fit, so nothing was saved:\n- ", " props do not fit: ", "\n- ", "; ", "\n", "; ")

// clutterProblems are Layout now's lines for them.
func clutterProblems(recs []*store.Record, bad map[string]string) []string {
	var out []string
	for _, b := range recs {
		if why, ok := bad[b.ID]; ok {
			out = append(out, fmt.Sprintf("block %s (%s) %s: propose taking it away", b.ID, b.Fields["component"], why))
		}
	}
	return out
}

// Clutter is clutter for a page that offers the person to tidy it.
func (s *Service) Clutter(recs []*store.Record) map[string]string { return s.clutter(recs) }
