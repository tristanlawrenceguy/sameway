package chat

import (
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// An agent's blocks on the person's page. Every agent that tried
// something put its block on Home, the page the person opens first, and
// Home grew to twenty-two rows of lists and trackers nobody asked for. An
// agent adding a block to a page that is already long is told how long,
// and that a tab of its own keeps what it tries apart; nothing is refused.

// crowdedFrom is how many blocks make a page long.
const crowdedFrom = 12

// crowdedSaid is the note for a block an agent made on a long page, "".
func (s *Service) crowdedSaid(b *store.Record) string {
	if b.Fields["created_by"] != records.ActorAgent {
		return ""
	}
	canvas, _ := b.Fields["canvas"].(string)
	n := 0
	for _, other := range s.canvasBlocks(canvas) {
		if other.Fields["created_by"] != "system" {
			n++
		}
	}
	if n < crowdedFrom {
		return ""
	}
	where := "Home, the page the person opens first,"
	for _, c := range s.Canvases() {
		if c.ID == canvas && canvas != "" {
			where = "the " + c.Name + " tab"
		}
	}
	return fmt.Sprintf("%s now holds %d blocks. If this is something you are trying rather than what the person asked for there, a tab of its own (POST /api/canvas, then canvas on the block) keeps it apart from their page; remove_component takes it back.", where, n)
}
