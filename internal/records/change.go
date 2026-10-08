package records

import (
	"encoding/json"
	"errors"
	"fmt"

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
// id when it is gone.
func (b *Book) Apply(ops ...Op) ([]Op, error) {
	done := make([]Op, 0, len(ops))
	for _, op := range ops {
		made, err := b.applyOne(op)
		if err != nil {
			for i := len(done) - 1; i >= 0; i-- {
				back := done[i]
				back.Before, back.After = back.After, back.Before
				b.applyOne(back) // best effort: it was there a moment ago
			}
			return nil, err
		}
		done = append(done, made)
	}
	return done, nil
}

// ApplyOps is Apply for a store with no settings to change.
func ApplyOps(st *store.Store, ops ...Op) ([]Op, error) {
	return (&Book{Store: st}).Apply(ops...)
}

func (b *Book) applyOne(op Op) (Op, error) {
	if op.Type == SettingOp {
		if b.SetSetting == nil {
			return Op{}, errors.New("there are no settings to change here")
		}
		was, now := b.setting(op.ID), fmt.Sprint(op.After["value"])
		if op.After["value"] == nil {
			now = ""
		}
		if err := b.SetSetting(op.ID, now); err != nil {
			return Op{}, err
		}
		return Op{Type: SettingOp, ID: op.ID, Before: map[string]any{"value": was}, After: map[string]any{"value": now}}, nil
	}
	var cur *store.Record
	if op.ID != "" {
		cur, _ = b.Store.Get(op.Type, op.ID)
	}
	var rec *store.Record
	var err error
	switch {
	case op.After == nil && cur == nil:
		return Op{Type: op.Type, ID: op.ID}, nil
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
		return Op{}, err
	}
	out := Op{Type: op.Type, ID: op.ID}
	if cur != nil {
		out.Before = cur.Fields
	}
	if rec != nil {
		out.ID, out.After = rec.ID, rec.Fields
	}
	return out, nil
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

// opsField is ops as the log's json field keeps them.
func opsField(ops []Op) any {
	raw, _ := json.Marshal(ops)
	var out []any
	json.Unmarshal(raw, &out)
	return out
}

// EntryOps is what an entry in the log changed, as ops: its own, or, for
// an entry from before the log kept ops, the batch its before keeps, each
// record with what it was (its after is not known).
func EntryOps(e *store.Record) []Op {
	if raw, ok := e.Fields["ops"].([]any); ok && len(raw) > 0 {
		b, _ := json.Marshal(raw)
		var ops []Op
		if json.Unmarshal(b, &ops) == nil {
			return ops
		}
	}
	before, _ := e.Fields["before"].(map[string]any)
	var ops []Op
	for _, c := range BatchOf(before) {
		ops = append(ops, Op{Type: c.Type, ID: c.ID, Before: c.Before})
	}
	return ops
}

// hasOps says whether an entry keeps its ops, as every new one does.
func hasOps(e *store.Record) bool {
	raw, _ := e.Fields["ops"].([]any)
	return len(raw) > 0
}
