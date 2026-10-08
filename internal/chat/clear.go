package chat

import (
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// clearCanvas empties the tab the person is looking at, all but the chat.
func (s *Service) clearCanvas() toolResult {
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
	return toolResult{text: fmt.Sprintf("cleared %d blocks; the chat stayed", len(gone)), change: &Change{Action: "cleared", Detail: fmt.Sprintf("%d blocks", len(gone)), Before: map[string]any{"blocks": records.Keep(gone)}}}
}
