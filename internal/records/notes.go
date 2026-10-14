package records

import (
	"sync"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// What is worth saying about a change, once, for every way in. A task made
// that is called as an open one is, a block added over records another
// already shows: each was said where it was first seen, by the assistant's
// own tool, and an agent making the same through the API was told nothing,
// until each way in was taught each thing, one at a time. The notes of a
// change are worked out here, from the change alone, by what registered
// once; every way in says them as they are.

// Noticer is what is worth saying about a change just made: plain
// sentences for whoever made it, nothing when there is nothing to say.
type Noticer func(c Change) []string

type noticerKey struct{}

type noticers struct {
	mu   sync.Mutex
	keys []any
	list []Noticer
}

// Notice registers n under key for every change on st; a key registered
// already is left as it is, so registering twice says nothing twice.
func Notice(st *store.Store, key any, n Noticer) {
	k := st.Keep(noticerKey{}, func() any { return &noticers{} }).(*noticers)
	k.mu.Lock()
	defer k.mu.Unlock()
	for _, have := range k.keys {
		if have == key {
			return
		}
	}
	k.keys, k.list = append(k.keys, key), append(k.list, n)
}

// Notes is what the registered noticers say of c, in the order they were
// registered.
func Notes(st *store.Store, c Change) []string {
	k, _ := st.Kept(noticerKey{}).(*noticers)
	if k == nil {
		return nil
	}
	k.mu.Lock()
	list := append([]Noticer(nil), k.list...)
	k.mu.Unlock()
	var out []string
	for _, n := range list {
		out = append(out, n(c)...)
	}
	return out
}

// Made is each record c made, as its type and id: the ops that wrote what
// was not there before.
func (c Change) Made() []Op {
	var out []Op
	for _, op := range c.Ops {
		if op.Before == nil && op.After != nil && op.ID != "" && op.Type != SettingOp {
			out = append(out, op)
		}
	}
	return out
}

// WriteNoted is WriteAs for a way in that tells its caller what the
// change's notes say, and the log entry that takes it back, as the JSON
// API does.
func WriteNoted(st *store.Store, who Who, action, typ, id string, fields map[string]any) (*store.Record, []string, string, error) {
	rec, c, err := Write(st, action, typ, id, fields)
	if err != nil {
		return nil, nil, "", err
	}
	entry := logAs(st, who, c)
	return rec, Notes(st, c), entry, nil
}
