package chat

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

// keep is how a list of blocks is written into an entry's before, so a
// cleared canvas or a removed tab can be put back whole.
func keep(blocks []*store.Record) []any {
	out := make([]any, 0, len(blocks))
	for _, b := range blocks {
		out = append(out, map[string]any{"id": b.ID, "fields": b.Fields})
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

func (s *Service) allPresent(blocks []kept) bool {
	for _, b := range blocks {
		if _, err := s.Store.Get(BlockType, b.id); err != nil {
			return false
		}
	}
	return true
}

func (s *Service) anyPresent(blocks []kept) bool {
	for _, b := range blocks {
		if _, err := s.Store.Get(BlockType, b.id); err == nil {
			return true
		}
	}
	return false
}

// clearBlocks removes the given blocks and records what they were.
func (s *Service) clearBlocks(blocks []kept) Change {
	var gone []*store.Record
	for _, b := range blocks {
		rec, err := s.Store.Get(BlockType, b.id)
		if err != nil {
			continue
		}
		if s.Store.Delete(BlockType, b.id) == nil {
			gone = append(gone, rec)
		}
	}
	return Change{Action: "cleared", Detail: fmt.Sprintf("%d blocks", len(gone)), Before: map[string]any{"blocks": keep(gone)}}
}

// same compares two field maps the way they are stored: as JSON, so an
// integer read back as a float still matches.
func same(a, b map[string]any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

// removeBlock takes one block off the canvas and logs what it was.
func (s *Service) removeBlock(id string) toolResult {
	rec, err := s.Store.Get(BlockType, id)
	if err != nil {
		return fail("could not remove block %s: %v", id, err)
	}
	if err := s.Store.Delete(BlockType, id); err != nil {
		return fail("could not remove block %s: %v", id, err)
	}
	c := s.describe(BlockType, rec)
	c.Action, c.Href, c.Before = "removed", "", rec.Fields
	return toolResult{text: "removed block " + id, change: &c}
}
