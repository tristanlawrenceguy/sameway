package chat

import (
	"errors"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Undo is the activity log run backwards. Every entry that changed
// something carries what the thing was before, so reversing it needs no
// second bookkeeping: an added thing is removed, a removed thing is put
// back with everything it had, an update goes back to what it was. The
// reversal is itself an entry with its own before, which is why undoing an
// undo just works, and why there is no redo.

var undoTool = llm.Tool{
	Name:        "undo_change",
	Description: "Reverse one change from the activity log, yours or the person's: an added thing is removed, a removed thing is put back with everything it had, an update goes back to what it was. Undoing an undo puts it back again. Without an id, the newest change that can still be undone.",
	Schema: map[string]any{"type": "object", "properties": map[string]any{
		"id": map[string]any{"type": "string", "description": "The activity entry's id, from the recent changes listed in the prompt."},
	}, "additionalProperties": false},
}

// Undoable says whether an entry can be reversed now: it is the kind of
// entry that can be, and the thing is still as the entry left it.
func (s *Service) Undoable(a *store.Record) bool {
	_, err := s.inverse(a)
	return err == nil
}

// Undo reverses one entry, or the newest reversible one when id is empty.
// The change it returns is the reversal, ready to be recorded.
func (s *Service) Undo(id string) toolResult {
	a, err := s.entry(id)
	if err != nil {
		return fail("%v", err)
	}
	summary, _ := a.Fields["summary"].(string)
	apply, err := s.inverse(a)
	if err != nil {
		return fail("cannot undo %q: %v", summary, err)
	}
	c, err := apply()
	if err != nil {
		return fail("could not undo %q: %v", summary, err)
	}
	c.Undoes, c.Undone = a.ID, summary
	return toolResult{text: "undone: " + summary, change: &c}
}

// UndoAs is Undo for a person, recorded as theirs.
func (s *Service) UndoAs(actor, id string) error {
	r := s.Undo(id)
	if r.isErr {
		return errors.New(r.text)
	}
	if r.change != nil {
		Record(s.Store, actor, *r.change)
	}
	return nil
}

func (s *Service) entry(id string) (*store.Record, error) {
	if id != "" {
		a, err := s.Store.Get(ActivityType, id)
		if err != nil {
			return nil, fmt.Errorf("no activity entry %s", id)
		}
		return a, nil
	}
	for _, a := range s.recentEntries(50) {
		if s.Undoable(a) {
			return a, nil
		}
	}
	return nil, errors.New("nothing to undo")
}

func (s *Service) recentEntries(n int) []*store.Record {
	recs, _ := s.Store.List(ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: n})
	return recs
}

// undoDigest lists the changes the model could reverse, newest first, so
// "undo that" has an id to point at.
func (s *Service) undoDigest() string {
	var lines []string
	for _, a := range s.recentEntries(20) {
		if s.Undoable(a) {
			lines = append(lines, fmt.Sprintf("%s: %s", a.ID, a.Fields["summary"]))
		}
		if len(lines) == 5 {
			break
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return "\nRecent changes that can be undone, newest first (id: what happened); undo_change takes the id:\n" + strings.Join(lines, "\n") + "\n"
}

// inverse works out what reversing an entry means, without doing it, and
// says why when it cannot be done.
func (s *Service) inverse(a *store.Record) (func() (Change, error), error) {
	action, _ := a.Fields["action"].(string)
	target, _ := a.Fields["target"].(string)
	id, _ := a.Fields["target_id"].(string)
	before, _ := a.Fields["before"].(map[string]any)
	typ := s.targetType(target)
	switch action {
	case "added", "created":
		if typ == "" || id == "" {
			return nil, errors.New("the entry does not say what was added")
		}
		if _, err := s.Store.Get(typ, id); err != nil {
			return nil, errors.New("it is already gone")
		}
		return func() (Change, error) { return s.take(typ, id) }, nil
	case "removed", "deleted":
		if typ == "" || id == "" || before == nil {
			return nil, errors.New("the entry does not say what it was")
		}
		if _, err := s.Store.Get(typ, id); err == nil {
			return nil, errors.New("it is already back")
		}
		return func() (Change, error) { return s.restore(typ, id, before) }, nil
	case "updated":
		if typ == "" || id == "" || before == nil {
			return nil, errors.New("the entry does not say what it was")
		}
		cur, err := s.Store.Get(typ, id)
		if err != nil {
			return nil, errors.New("it is gone")
		}
		if same(cur.Fields, before) {
			return nil, errors.New("it is already as it was")
		}
		return func() (Change, error) {
			rec, err := s.Store.Update(typ, id, before)
			if err != nil {
				return Change{}, err
			}
			c := s.describe(typ, rec)
			c.Action, c.Before = "updated", cur.Fields
			return c, nil
		}, nil
	case "cleared":
		blocks := blocksIn(before)
		if len(blocks) == 0 {
			return nil, errors.New("the entry does not say what was cleared")
		}
		if s.allPresent(blocks) {
			return nil, errors.New("they are already back")
		}
		return func() (Change, error) {
			n := 0
			for _, b := range blocks {
				if _, err := s.Store.Get(BlockType, b.id); err == nil {
					continue
				}
				if _, err := s.Store.Restore(BlockType, b.id, b.fields); err != nil {
					return Change{}, err
				}
				n++
			}
			return Change{Action: "restored", Detail: fmt.Sprintf("%d blocks", n), Before: before}, nil
		}, nil
	case "restored":
		blocks := blocksIn(before)
		if len(blocks) == 0 || !s.anyPresent(blocks) {
			return nil, errors.New("they are already gone")
		}
		return func() (Change, error) { return s.clearBlocks(blocks), nil }, nil
	}
	return nil, errors.New("that kind of entry cannot be undone")
}

// targetType is the content type an entry's target names: a tab, a record
// type, or else a block, since a block's target is its component.
func (s *Service) targetType(target string) string {
	switch {
	case target == "":
		return ""
	case target == CanvasType:
		return CanvasType
	}
	if _, ok := s.Store.Types().Get(target); ok && target != BlockType {
		return target
	}
	return BlockType
}

// take removes a thing that was added, through the same code the tools use,
// so the removal is logged with everything needed to put it back.
func (s *Service) take(typ, id string) (Change, error) {
	var r toolResult
	switch typ {
	case CanvasType:
		r = s.removeCanvas(id)
	case BlockType:
		r = s.removeBlock(id)
	default:
		r = s.deleteRecord(typ, id)
	}
	if r.isErr || r.change == nil {
		return Change{}, errors.New(r.text)
	}
	return *r.change, nil
}

// restore puts back a thing that was removed, with its blocks when it was a
// tab, and describes it the way an addition is described.
func (s *Service) restore(typ, id string, before map[string]any) (Change, error) {
	fields := map[string]any{}
	for k, v := range before {
		if k != "blocks" {
			fields[k] = v
		}
	}
	rec, err := s.Store.Restore(typ, id, fields)
	if err != nil {
		return Change{}, err
	}
	for _, b := range blocksIn(before) {
		if _, err := s.Store.Get(BlockType, b.id); err != nil {
			s.Store.Restore(BlockType, b.id, b.fields)
		}
	}
	c := s.describe(typ, rec)
	c.Action = "added"
	if typ != CanvasType && typ != BlockType {
		c.Action = "created"
	}
	return c, nil
}

// describe names a thing the way its own tool would in a receipt.
func (s *Service) describe(typ string, rec *store.Record) Change {
	switch typ {
	case CanvasType:
		name, _ := rec.Fields["name"].(string)
		return Change{Component: CanvasType, ID: rec.ID, Detail: name, Href: CanvasPath(rec.ID)}
	case BlockType:
		name, _ := rec.Fields["component"].(string)
		props, _ := rec.Fields["props"].(map[string]any)
		return Change{Component: name, ID: rec.ID, Detail: Summarise(name, props), Href: "/canvas/" + rec.ID}
	}
	t, _ := s.Store.Types().Get(typ)
	return Change{Component: typ, ID: rec.ID, Detail: recordTitle(t, rec), Href: "/t/" + typ + "/" + rec.ID}
}
