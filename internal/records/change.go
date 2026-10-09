package records

import (
	"errors"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A change is what it did to each thing it touched: for each record, what
// it was and what it became; for a setting, its value before and after.
// That one shape is enough to write a change, log it and take it back,
// whatever the change was called, so undo is the same ops the other way
// rather than a rule for every verb. The log used to keep a before in a
// shape of its own for each kind of entry; those entries are still read
// (undo.go), but new ones keep their ops.

// Op is one thing a change wrote: a record by its type and id, or a
// setting (Type SettingOp, ID its key, values as {"value": ...}). Before
// is nil for a record the change made, After nil for one it removed.
type Op struct {
	Type   string         `json:"type"`
	ID     string         `json:"id"`
	Before map[string]any `json:"before,omitempty"`
	After  map[string]any `json:"after,omitempty"`
}

// SettingOp is the type of an op that changed a line of workspace.yaml.
const SettingOp = "setting"

// Apply writes ops in order, all or none: when one cannot be written,
// those already written are put back as they were. It returns each op as
// it was made, with what the thing was just before and is now, ready for
// the log. An op with no id makes a record; one with no After removes it;
// otherwise After is written over what is there, or put back under that
// id when it is gone. A tab removed takes the blocks on it with it.
func (b *Book) Apply(ops ...Op) ([]Op, error) {
	done, _, err := b.apply(ops)
	return done, err
}

// ApplyOps is Apply for a store with no settings to change.
func ApplyOps(st *store.Store, ops ...Op) ([]Op, error) {
	return (&Book{Store: st}).Apply(ops...)
}

// apply is Apply, with each record as it now is (nil where it went), for
// a writer that answers with the record.
func (b *Book) apply(ops []Op) ([]Op, []*store.Record, error) {
	var done []Op
	var recs []*store.Record
	for _, op := range b.withBlocks(ops) {
		made, rec, err := b.applyOne(op)
		if err != nil {
			for i := len(done) - 1; i >= 0; i-- {
				back := done[i]
				back.Before, back.After = back.After, back.Before
				b.applyOne(back) // best effort: it was there a moment ago
			}
			return nil, nil, err
		}
		done, recs = append(done, made), append(recs, rec)
	}
	return done, recs, nil
}

// withBlocks puts the removal of a tab's blocks before the tab's own.
func (b *Book) withBlocks(ops []Op) []Op {
	var out []Op
	for _, op := range ops {
		if op.Type == CanvasType && op.ID != "" && op.After == nil {
			blocks, _ := b.Store.List(BlockType, store.ListOptions{})
			for _, k := range OnCanvas(blocks, op.ID) {
				out = append(out, Op{Type: BlockType, ID: k.ID})
			}
		}
		out = append(out, op)
	}
	return out
}

func (b *Book) applyOne(op Op) (Op, *store.Record, error) {
	if op.Type == SettingOp {
		return b.applySetting(op)
	}
	var cur *store.Record
	if op.ID != "" {
		cur, _ = b.Store.Get(op.Type, op.ID)
	}
	var rec *store.Record
	var err error
	switch {
	case op.After == nil && cur == nil:
		return Op{Type: op.Type, ID: op.ID}, nil, nil
	case op.After == nil:
		err = b.Store.Delete(op.Type, op.ID)
	case op.ID == "":
		rec, err = b.Store.Create(op.Type, op.After)
	case cur == nil:
		rec, err = b.Store.Restore(op.Type, op.ID, op.After)
	default:
		rec, err = b.Store.Update(op.Type, op.ID, op.After)
	}
	if err != nil {
		return Op{}, nil, err
	}
	out := Op{Type: op.Type, ID: op.ID}
	if cur != nil {
		out.Before = cur.Fields
	}
	if rec != nil {
		out.ID, out.After = rec.ID, rec.Fields
	}
	return out, rec, nil
}

// applySetting changes one line of workspace.yaml.
func (b *Book) applySetting(op Op) (Op, *store.Record, error) {
	if b.SetSetting == nil {
		return Op{}, nil, errors.New("there are no settings to change here")
	}
	now, _ := op.After["value"].(string)
	was := b.setting(op.ID)
	if err := b.SetSetting(op.ID, now); err != nil {
		return Op{}, nil, err
	}
	return Op{Type: SettingOp, ID: op.ID, Before: map[string]any{"value": was}, After: map[string]any{"value": now}}, nil, nil
}

// OpsOf is a batch as ops, with each record as it is now for its after:
// for a writer that wrote the batch itself and logs it after.
func OpsOf(st *store.Store, batch []BatchItem) []Op {
	ops := make([]Op, 0, len(batch))
	for _, c := range batch {
		op := Op{Type: c.Type, ID: c.ID, Before: c.Before}
		if rec, err := st.Get(c.Type, c.ID); err == nil {
			op.After = rec.Fields
		}
		ops = append(ops, op)
	}
	return ops
}

// Made is the ops of records a change made, by their ids, such as an
// import from a file: undoing it takes them away together.
func Made(st *store.Store, typ string, ids []string) []Op {
	batch := make([]BatchItem, 0, len(ids))
	for _, id := range ids {
		batch = append(batch, BatchItem{Type: typ, ID: id})
	}
	return OpsOf(st, batch)
}
