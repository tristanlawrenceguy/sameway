package server

import (
	"sort"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// The canvas is read into regions here, and the blocks that changed in
// the last turn are put in arrival order; canvas.go renders them.

// canvasBlocks reads the canvas in display order, or nothing if it cannot.
func (s *Server) canvasBlocks() []*store.Record {
	blocks, err := s.app.Store.List(chat.BlockType, store.ListOptions{OrderBy: "position"})
	if err != nil {
		return nil
	}
	return blocks
}

// regions sorts blocks into the five places on the page.
type regions struct{ main, left, right, header, footer []*store.Record }

func split(blocks []*store.Record) regions {
	var r regions
	for _, blk := range blocks {
		switch str(blk.Fields["region"], "main") {
		case "left":
			r.left = append(r.left, blk)
		case "right", "side": // side was the earlier name for right
			r.right = append(r.right, blk)
		case "header":
			r.header = append(r.header, blk)
		case "footer":
			r.footer = append(r.footer, blk)
		default:
			r.main = append(r.main, blk)
		}
	}
	return r
}

func layoutName(solo bool) string {
	if solo {
		return "solo"
	}
	return "wide"
}

// arrivals orders the blocks that changed in the last turn by when they
// changed, so the page can show them one after another as they were made:
// where each will be, then what it is, then what it says. The conversation
// is the person's own tool and never arrives; it is simply there.
func arrivals(blocks []*store.Record, convo *conversation) map[string]int {
	type change struct {
		id string
		at time.Time
	}
	var changes []change
	for _, b := range blocks {
		// What the system put there to begin with is not news either.
		if b.Fields["component"] == chat.ComponentName || b.Fields["created_by"] == "system" {
			continue
		}
		switch {
		case inLastTurn(b.CreatedAt, convo):
			changes = append(changes, change{b.ID, b.CreatedAt})
		case inLastTurn(b.UpdatedAt, convo):
			changes = append(changes, change{b.ID, b.UpdatedAt})
		}
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].at.Before(changes[j].at) })
	order := map[string]int{}
	for i, c := range changes {
		order[c.id] = i + 1
	}
	return order
}
