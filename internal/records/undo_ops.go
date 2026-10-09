package records

import (
	"errors"
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Undo for an entry that keeps its ops: the same ops the other way, last
// first, each thing back to what it was before. It needs no rule for the
// verb; only the sentence of the reversal is said from what it undid.

// batchVerbs are the entries that touched many records as one: their
// reversal is said as the records put back.
var batchVerbs = map[string]bool{"imported": true, "synced": true, "arranged": true, "wrote up": true, "organised": true, "suggested": true, "rescheduled": true}

// inverseOps works out the ops that take an entry back. The thing the
// entry names must still be as the entry left it: there, if it made or
// changed it; gone, if it removed it. Anything else in it that is already
// as it was is let be, and one that went since is put back. Only the
// thing named is looked at before: a batch of a thousand records is not
// read on every look at the log, only when it is undone.
func (b *Book) inverseOps(a *store.Record, ops []Op) (func() (Change, error), error) {
	named := primary(a, ops)
	if named >= 0 {
		if _, reason := b.undoOp(ops[named], true); reason != "" {
			return nil, errors.New(reason)
		}
	} else if len(ops) == 0 {
		return nil, errors.New("the entry does not say what it changed")
	}
	return func() (Change, error) {
		var back []Op
		for i := len(ops) - 1; i >= 0; i-- {
			undo, reason := b.undoOp(ops[i], i == named)
			if reason != "" && i == named {
				return Change{}, errors.New(reason)
			}
			if reason == "" {
				back = append(back, undo)
			}
		}
		if len(back) == 0 {
			return Change{}, errors.New("every record is already as it was")
		}
		done, err := b.Apply(back...)
		if err != nil {
			return Change{}, err
		}
		return b.reversal(a, ops, done), nil
	}, nil
}

// primary is which op is the thing an entry names, or -1 for a batch.
func primary(a *store.Record, ops []Op) int {
	action, _ := a.Fields["action"].(string)
	id, _ := a.Fields["target_id"].(string)
	if batchVerbs[action] {
		return -1
	}
	for i, op := range ops {
		if op.ID == id && id != "" || len(ops) == 1 {
			return i
		}
	}
	return -1
}

// undoOp is the op that puts one thing back as op found it, or why there
// is nothing to do.
func (b *Book) undoOp(op Op, strict bool) (Op, string) {
	if op.Type == SettingOp {
		if b.SetSetting == nil || op.Before == nil {
			return Op{}, "the entry does not say what it was"
		}
		if was, _ := op.Before["value"].(string); b.setting(op.ID) == was {
			return Op{}, "it is already as it was"
		}
		return Op{Type: SettingOp, ID: op.ID, After: op.Before}, ""
	}
	var cur map[string]any
	if rec, err := b.Store.Get(op.Type, op.ID); err == nil {
		cur = rec.Fields
	}
	switch {
	case op.Before == nil && cur == nil:
		return Op{}, "it is already gone"
	case op.Before == nil:
		return Op{Type: op.Type, ID: op.ID}, ""
	case cur == nil && strict && op.After != nil:
		return Op{}, "it is gone"
	case cur != nil && op.After == nil:
		return Op{}, "it is already back"
	case cur != nil && Same(cur, op.Before):
		return Op{}, "it is already as it was"
	}
	return Op{Type: op.Type, ID: op.ID, After: op.Before}, ""
}

// reversal says what undoing an entry did, the way the change itself
// would have been said: a batch as the records put back, a setting as
// set, the thing the entry named as removed, added or updated.
func (b *Book) reversal(a *store.Record, ops, done []Op) Change {
	target, _ := a.Fields["target"].(string)
	named := primary(a, ops)
	if named < 0 {
		return Change{Action: "synced", Component: target, Detail: fmt.Sprintf("%d records put back", len(done)), Ops: done}
	}
	key := ops[named]
	var c Change
	for _, op := range done {
		if op.ID != key.ID || op.Type != key.Type {
			continue
		}
		switch {
		case op.Type == SettingOp:
			v, _ := op.After["value"].(string)
			c = Change{Action: "set", Component: op.ID, Detail: orNone(v)}
		case op.After == nil:
			c = describe(b.Store, op.Type, &store.Record{ID: op.ID, Type: op.Type, Fields: op.Before})
			c.Action, c.Href = "removed", ""
			if op.Type != CanvasType && op.Type != BlockType {
				c.Action = "deleted"
			}
		case op.Before == nil:
			c = describe(b.Store, op.Type, &store.Record{ID: op.ID, Type: op.Type, Fields: op.After})
			c.Action = "added"
			if op.Type != CanvasType && op.Type != BlockType {
				c.Action = "created"
			}
		default:
			c = describe(b.Store, op.Type, &store.Record{ID: op.ID, Type: op.Type, Fields: op.After})
			c.Action = "updated"
		}
	}
	c.Ops = done
	return c
}
