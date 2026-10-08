package records

import (
	"errors"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Entries from before the log kept ops each kept their before in a shape
// of their own for their kind. They are read here and nowhere else, and
// nothing new is written this way except a canvas or a chat cleared and
// put back: a thing added, removed or updated, a setting set and an
// amount logged are read as the ops they stand for and undone like any
// other; a batch is put back as it always was.

// inverseLegacy reverses an entry without ops.
func (b *Book) inverseLegacy(a *store.Record) (func() (Change, error), error) {
	action, _ := a.Fields["action"].(string)
	target, _ := a.Fields["target"].(string)
	id, _ := a.Fields["target_id"].(string)
	before, _ := a.Fields["before"].(map[string]any)
	// A chat cleared or deleted is put back with its messages.
	if target == "conversation" && (action == "cleared" || action == "deleted") {
		return b.inverseChat(id, before)
	}
	switch action {
	case "cleared":
		return b.inverseCleared(before)
	case "restored":
		blocks := blocksIn(before)
		if len(blocks) == 0 || !b.anyPresent(blocks) {
			return nil, errors.New("they are already gone")
		}
		return func() (Change, error) { return b.clearBlocks(blocks), nil }, nil
	case "imported", "synced", "arranged", "wrote up", "organised", "suggested", "rescheduled":
		// A batch: records made, changed and removed by one import, each
		// with what it was before (nothing, for one it made).
		changes := BatchOf(before)
		if len(changes) == 0 {
			return nil, errors.New("the entry does not say what it changed")
		}
		return func() (Change, error) { return b.reverseBatch(target, changes) }, nil
	}
	ops, err := b.legacyOps(action, target, id, before)
	if err != nil {
		return nil, err
	}
	return b.inverseOps(a, ops)
}

// legacyOps is what an older entry for one thing wrote, as ops.
func (b *Book) legacyOps(action, target, id string, before map[string]any) ([]Op, error) {
	typ := b.targetType(target)
	var now map[string]any
	if rec, err := b.Store.Get(typ, id); err == nil && typ != "" && id != "" {
		now = rec.Fields
	}
	switch action {
	case "added", "created":
		if typ == "" || id == "" {
			return nil, errors.New("the entry does not say what was added")
		}
		return []Op{{Type: typ, ID: id, After: now}}, nil
	case "removed", "deleted":
		if typ == "" || id == "" || before == nil {
			return nil, errors.New("the entry does not say what it was")
		}
		// A tab removed kept the blocks that were on it.
		var ops []Op
		for _, k := range blocksIn(before) {
			ops = append(ops, Op{Type: BlockType, ID: k.id, Before: k.fields})
		}
		fields := map[string]any{}
		for k, v := range before {
			if k != "blocks" {
				fields[k] = v
			}
		}
		return append(ops, Op{Type: typ, ID: id, Before: fields}), nil
	case "updated", "done", "snoozed":
		// A reminder dismissed or put off is a change like any other.
		if typ == "" || id == "" || before == nil {
			return nil, errors.New("the entry does not say what it was")
		}
		if now == nil {
			return nil, errors.New("it is gone")
		}
		return []Op{{Type: typ, ID: id, Before: before, After: now}}, nil
	case "set":
		// A setting goes back to what it was, which may be nothing.
		if b.SetSetting == nil || before == nil {
			return nil, errors.New("the entry does not say what it was")
		}
		return []Op{{Type: SettingOp, ID: target, Before: before}}, nil
	case "logged":
		// An amount logged against a habit is the entry it made.
		entry, _ := before["entry"].(string)
		if entry == "" {
			return nil, errors.New("the entry does not say what was logged")
		}
		return []Op{{Type: EntryType, ID: entry}}, nil
	}
	return nil, errors.New("that kind of entry cannot be undone")
}

// inverseCleared puts back the blocks a canvas cleared had.
func (b *Book) inverseCleared(before map[string]any) (func() (Change, error), error) {
	blocks := blocksIn(before)
	if len(blocks) == 0 {
		return nil, errors.New("the entry does not say what was cleared")
	}
	if b.allPresent(blocks) {
		return nil, errors.New("they are already back")
	}
	return func() (Change, error) {
		n := 0
		for _, k := range blocks {
			if _, err := b.Store.Get(BlockType, k.id); err == nil {
				continue
			}
			if _, err := b.Store.Restore(BlockType, k.id, k.fields); err != nil {
				return Change{}, err
			}
			n++
		}
		return Change{Action: "restored", Detail: fmt.Sprintf("%d blocks", n), Before: before}, nil
	}, nil
}

// orNone is a setting's value as a sentence says it: nothing, for none.
func orNone(s string) string {
	if strings.TrimSpace(s) == "" {
		return "nothing"
	}
	return s
}
