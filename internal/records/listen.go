package records

import (
	"sync"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A change is news to whatever follows the workspace: an agent waiting on
// GET /api/changes is answered at once rather than at its next look, and
// an automation runs when a record comes to be a certain way. Each
// listener hears a change once, whole, with who made it and the ops it
// wrote, however many records that was:
//
//   - a change logged, when it is logged (Record), with its entry in the
//     log as c.Activity;
//   - the system's own bookkeeping, written by ApplyOps and never logged
//     (a reminder ringing, a file read, a device's state), when it is
//     written, as the system's, with no entry.
//
// Book.Apply tells nothing itself: whoever applies ops that way logs them
// after, and is heard then. Only this process's changes are heard: what
// another computer sends is left to that computer, and what another
// process writes to the same workspace (the MCP server a model runs its
// tools through) is read from the log instead (chat/outside.go). This is
// the one way to hear a change; the store has none of its own.

type feed struct {
	mu        sync.Mutex
	next      int
	listeners []listener
}

type listener struct {
	id int
	fn func(actor string, c Change)
}

// feeds are each open store's listeners.
var feeds sync.Map // *store.Store → *feed

// Listen adds fn to what is told of each change in st, once per change,
// with who made it, until stop is called.
func Listen(st *store.Store, fn func(actor string, c Change)) (stop func()) {
	v, _ := feeds.LoadOrStore(st, &feed{})
	f := v.(*feed)
	f.mu.Lock()
	defer f.mu.Unlock()
	id := f.next
	f.next++
	f.listeners = append(f.listeners, listener{id, fn})
	return func() {
		f.mu.Lock()
		defer f.mu.Unlock()
		kept := make([]listener, 0, len(f.listeners))
		for _, l := range f.listeners {
			if l.id != id {
				kept = append(kept, l)
			}
		}
		f.listeners = kept
	}
}

// tell tells st's listeners of one change, in the order they listened.
func tell(st *store.Store, actor string, c Change) {
	v, ok := feeds.Load(st)
	if !ok {
		return
	}
	f := v.(*feed)
	f.mu.Lock()
	ls := f.listeners
	f.mu.Unlock()
	for _, l := range ls {
		l.fn(actor, c)
	}
}

// Forget lets go of st's listeners, when it closes.
func Forget(st *store.Store) { feeds.Delete(st) }
