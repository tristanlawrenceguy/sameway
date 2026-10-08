package records

import (
	"errors"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Undo is the activity log run backwards. Every entry that changed
// something carries what the thing was before, so reversing it needs no
// second bookkeeping: an added thing is removed, a removed thing is put
// back with everything it had, an update goes back to what it was. The
// reversal is itself an entry with its own before, which is why undoing an
// undo just works, and why there is no redo.

// Undoable says whether an entry can be reversed now: it is the kind of
// entry that can be, and the thing is still as the entry left it.
func (b *Book) Undoable(a *store.Record) bool {
	_, err := b.inverse(a)
	return err == nil
}

// Undo reverses one entry, or the newest reversible one when id is empty.
// The change it returns is the reversal, ready to be recorded, and said
// is what was done in a few words.
func (b *Book) Undo(id string) (said string, c Change, err error) {
	a, err := b.entry(id)
	if err != nil {
		return "", Change{}, err
	}
	summary, _ := a.Fields["summary"].(string)
	apply, err := b.inverse(a)
	if err != nil {
		return "", Change{}, fmt.Errorf("cannot undo %q: %v", summary, err)
	}
	c, err = apply()
	if err != nil {
		return "", Change{}, fmt.Errorf("could not undo %q: %v", summary, err)
	}
	c.Undoes, c.Undone = a.ID, summary
	// Undoing an undo puts the original back: the sentence names the
	// original, not a chain of undids.
	if undoes, _ := a.Fields["undoes"].(string); undoes != "" {
		if _, original, ok := strings.Cut(summary, ": "); ok {
			c.Undone, c.Redid = original, true
			return "put back: " + original, c, nil
		}
	}
	return "undone: " + summary, c, nil
}

// UndoAs is Undo for a person, recorded as theirs.
func (b *Book) UndoAs(actor, id string) error {
	_, c, err := b.Undo(id)
	if err != nil {
		return err
	}
	Record(b.Store, actor, c)
	return nil
}

func (b *Book) entry(id string) (*store.Record, error) {
	if id != "" {
		a, err := b.Store.Get(ActivityType, id)
		if err != nil {
			return nil, fmt.Errorf("no activity entry %s", id)
		}
		return a, nil
	}
	for _, a := range b.Recent(50) {
		if b.Undoable(a) {
			return a, nil
		}
	}
	return nil, errors.New("nothing to undo")
}

// Recent is the newest n entries in the log, newest first.
func (b *Book) Recent(n int) []*store.Record {
	recs, _ := b.Store.List(ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: n})
	return recs
}

// inverse works out what reversing an entry means, without doing it, and
// says why when it cannot be done.
func (b *Book) inverse(a *store.Record) (func() (Change, error), error) {
	action, _ := a.Fields["action"].(string)
	target, _ := a.Fields["target"].(string)
	id, _ := a.Fields["target_id"].(string)
	before, _ := a.Fields["before"].(map[string]any)
	typ := b.targetType(target)
	// A chat cleared or deleted is put back with its messages.
	if target == "conversation" && (action == "cleared" || action == "deleted") {
		return b.inverseChat(id, before)
	}
	switch action {
	case "added", "created":
		if typ == "" || id == "" {
			return nil, errors.New("the entry does not say what was added")
		}
		if _, err := b.Store.Get(typ, id); err != nil {
			return nil, errors.New("it is already gone")
		}
		return func() (Change, error) { return b.take(typ, id) }, nil
	case "removed", "deleted":
		if typ == "" || id == "" || before == nil {
			return nil, errors.New("the entry does not say what it was")
		}
		if _, err := b.Store.Get(typ, id); err == nil {
			return nil, errors.New("it is already back")
		}
		return func() (Change, error) { return b.restore(typ, id, before) }, nil
	case "updated", "done", "snoozed":
		// A reminder dismissed or put off is a change like any other.
		if typ == "" || id == "" || before == nil {
			return nil, errors.New("the entry does not say what it was")
		}
		cur, err := b.Store.Get(typ, id)
		if err != nil {
			return nil, errors.New("it is gone")
		}
		if Same(cur.Fields, before) {
			return nil, errors.New("it is already as it was")
		}
		return func() (Change, error) {
			rec, err := b.Store.Update(typ, id, before)
			if err != nil {
				return Change{}, err
			}
			c := describe(b.Store, typ, rec)
			c.Action, c.Before = "updated", cur.Fields
			return c, nil
		}, nil
	case "cleared":
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
	case "restored":
		blocks := blocksIn(before)
		if len(blocks) == 0 || !b.anyPresent(blocks) {
			return nil, errors.New("they are already gone")
		}
		return func() (Change, error) { return b.clearBlocks(blocks), nil }, nil
	}
	return b.inverseMore(a)
}

// targetType is the content type an entry's target names: a tab, a record
// type, or else a block, since a block's target is its component.
func (b *Book) targetType(target string) string {
	switch {
	case target == "":
		return ""
	case target == CanvasType:
		return CanvasType
	}
	if _, ok := b.Store.Types().Get(target); ok && target != BlockType {
		return target
	}
	return BlockType
}

// take removes a thing that was added, through the same code the tools use,
// so the removal is logged with everything needed to put it back.
func (b *Book) take(typ, id string) (Change, error) {
	switch typ {
	case CanvasType:
		return RemoveCanvas(b.Store, id)
	case BlockType:
		return RemoveBlock(b.Store, id)
	}
	return DeleteRecord(b.Store, typ, id)
}

// restore puts back a thing that was removed, with its blocks when it was a
// tab, and describes it the way an addition is described.
func (b *Book) restore(typ, id string, before map[string]any) (Change, error) {
	fields := map[string]any{}
	for k, v := range before {
		if k != "blocks" {
			fields[k] = v
		}
	}
	rec, err := b.Store.Restore(typ, id, fields)
	if err != nil {
		return Change{}, err
	}
	for _, k := range blocksIn(before) {
		if _, err := b.Store.Get(BlockType, k.id); err != nil {
			b.Store.Restore(BlockType, k.id, k.fields)
		}
	}
	c := describe(b.Store, typ, rec)
	c.Action = "added"
	if typ != CanvasType && typ != BlockType {
		c.Action = "created"
	}
	return c, nil
}

// describe names a thing the way its own tool would in a receipt.
func describe(st *store.Store, typ string, rec *store.Record) Change {
	switch typ {
	case CanvasType:
		name, _ := rec.Fields["name"].(string)
		return Change{Component: CanvasType, ID: rec.ID, Detail: name, Href: CanvasPath(rec.ID)}
	case BlockType:
		name, _ := rec.Fields["component"].(string)
		props, _ := rec.Fields["props"].(map[string]any)
		return Change{Component: name, ID: rec.ID, Detail: Summarise(name, props), Href: "/canvas/" + rec.ID}
	}
	t, _ := st.Types().Get(typ)
	return Change{Component: typ, ID: rec.ID, Detail: Title(st, t, rec), Href: "/t/" + typ + "/" + rec.ID}
}
