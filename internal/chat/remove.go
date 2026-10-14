package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Removing several blocks at once: tidying a page is one change, so one
// Undo puts it all back, and a model that stops half way has not left
// the page half tidied.

// removeCall is remove_component: one id as it always was, or ids for
// several in one change.
func (s *Service) removeCall(a toolArgs) toolResult {
	ids := a.IDs
	if a.ID != "" {
		ids = append([]string{a.ID}, ids...)
	}
	ids = unique(ids)
	switch len(ids) {
	case 0:
		return fail("remove_component needs id, or ids for several blocks")
	case 1:
		return s.removeBlock(ids[0])
	}
	return s.removeBlocks(ids)
}

// removeBlocks takes several blocks off the canvas as one change. All or
// none: an id that is not a block removes nothing, and the error lists
// the tab's blocks so the next call can name the right ones.
func (s *Service) removeBlocks(ids []string) toolResult {
	var found []*store.Record
	var missing []string
	for _, id := range ids {
		if rec, err := s.Store.Get(records.BlockType, id); err == nil {
			found = append(found, rec)
		} else {
			missing = append(missing, id)
		}
	}
	canvas := s.current
	if len(found) > 0 {
		canvas, _ = found[0].Fields["canvas"].(string)
	}
	if len(missing) > 0 {
		return fail("nothing was removed: no block %s. The blocks on this tab are: %s", strings.Join(missing, ", "), s.blockList(canvas))
	}
	ops := make([]records.Op, 0, len(found))
	names := make([]string, 0, len(found))
	for _, rec := range found {
		ops = append(ops, records.Op{Type: records.BlockType, ID: rec.ID})
		names = append(names, blockName(rec))
	}
	done, err := s.Apply(ops...)
	if err != nil {
		return fail("nothing was removed: %v", err)
	}
	c := records.Change{Action: "removed", Detail: fmt.Sprintf("%d blocks: %s", len(done), listed(names, 4)),
		Href: records.CanvasPath(canvas), Ops: done}
	return toolResult{text: fmt.Sprintf("removed %d blocks (%s), in one change that one Undo takes back", len(done), strings.Join(ids, ", ")), change: &c}
}

// blockList is a tab's blocks as "id (component)", for a call to retry.
func (s *Service) blockList(canvas string) string {
	var out []string
	for _, b := range s.canvasBlocks(canvas) {
		component, _ := b.Fields["component"].(string)
		out = append(out, b.ID+" ("+component+")")
	}
	if len(out) == 0 {
		return "none"
	}
	return strings.Join(out, ", ")
}

// blockName is a block as the log says it: "Weather note", "Save button".
// Text and headings are their own words; anything else says what it is.
func blockName(rec *store.Record) string {
	component, _ := rec.Fields["component"].(string)
	props, _ := rec.Fields["props"].(map[string]any)
	words := records.Summarise(component, props)
	noun := blocks.Noun(component)
	switch {
	case words == "":
		return noun
	case component == "text" || component == "heading":
		return words
	}
	return words + " " + noun
}

// listed is the first few names and how many more: "a, b, c and 2 more".
func listed(names []string, most int) string {
	if len(names) <= most {
		return strings.Join(names, ", ")
	}
	return fmt.Sprintf("%s and %d more", strings.Join(names[:most], ", "), len(names)-most)
}

// unique drops repeats and blanks, keeping the order given.
func unique(ids []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}
