package records

import (
	"encoding/json"
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// The block side of undo: how a list of blocks is kept in an entry and put
// back, and the one removal every path shares.

type kept struct {
	id     string
	fields map[string]any
}

// Keep is how a list of records is written into an entry's before, so a
// cleared canvas, a removed tab or a cleared chat can be put back whole.
func Keep(blocks []*store.Record) []any {
	out := make([]any, 0, len(blocks))
	for _, k := range blocks {
		out = append(out, map[string]any{"id": k.ID, "fields": k.Fields})
	}
	return out
}

func blocksIn(before map[string]any) []kept {
	list, _ := before["blocks"].([]any)
	var out []kept
	for _, item := range list {
		m, _ := item.(map[string]any)
		id, _ := m["id"].(string)
		fields, _ := m["fields"].(map[string]any)
		if id != "" && fields != nil {
			out = append(out, kept{id: id, fields: fields})
		}
	}
	return out
}

func (b *Book) allPresent(blocks []kept) bool {
	for _, k := range blocks {
		if _, err := b.Store.Get(BlockType, k.id); err != nil {
			return false
		}
	}
	return true
}

func (b *Book) anyPresent(blocks []kept) bool {
	for _, k := range blocks {
		if _, err := b.Store.Get(BlockType, k.id); err == nil {
			return true
		}
	}
	return false
}

// clearBlocks removes the given blocks and records what they were.
func (b *Book) clearBlocks(blocks []kept) Change {
	var gone []*store.Record
	for _, k := range blocks {
		rec, err := b.Store.Get(BlockType, k.id)
		if err != nil {
			continue
		}
		if b.Store.Delete(BlockType, k.id) == nil {
			gone = append(gone, rec)
		}
	}
	return Change{Action: "cleared", Detail: fmt.Sprintf("%d blocks", len(gone)), Before: map[string]any{"blocks": Keep(gone)}}
}

// Same compares two field maps the way they are stored: as JSON, so an
// integer read back as a float still matches.
func Same(a, b map[string]any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

// RemoveBlock takes one block off the canvas and says it as a change that
// keeps what it was.
func RemoveBlock(st *store.Store, id string) (Change, error) {
	rec, err := st.Get(BlockType, id)
	if err != nil {
		return Change{}, fmt.Errorf("could not remove block %s: %v", id, err)
	}
	done, err := ApplyOps(st, Op{Type: BlockType, ID: id})
	if err != nil {
		return Change{}, fmt.Errorf("could not remove block %s: %v", id, err)
	}
	c := describe(st, BlockType, rec)
	c.Action, c.Href, c.Ops = "removed", "", done
	return c, nil
}
