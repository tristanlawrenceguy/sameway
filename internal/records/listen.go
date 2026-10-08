package records

import (
	"sync"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A change logged is news to whatever follows the workspace: an agent
// waiting on GET /api/changes is answered at once rather than at its next
// look. Each listener hears a change once, whole, after it is logged,
// however many records it wrote; a single record written, whichever way
// it came, is heard by store.Listen instead (automations listen there, as
// they must hear a record imported or read back from the content folder
// too, which no change logs).

type feed struct {
	mu  sync.Mutex
	fns []func(actor string, c Change)
}

// feeds are each open store's listeners.
var feeds sync.Map // *store.Store → *feed

// Listen adds fn to what is told of each change logged in st, once per
// change, with who made it; c.Activity is its entry in the log.
func Listen(st *store.Store, fn func(actor string, c Change)) {
	v, _ := feeds.LoadOrStore(st, &feed{})
	f := v.(*feed)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fns = append(f.fns, fn)
}

// tell tells st's listeners of one change logged.
func tell(st *store.Store, actor string, c Change) {
	v, ok := feeds.Load(st)
	if !ok {
		return
	}
	f := v.(*feed)
	f.mu.Lock()
	fns := f.fns
	f.mu.Unlock()
	for _, fn := range fns {
		fn(actor, c)
	}
}

// Forget lets go of st's listeners, when it closes.
func Forget(st *store.Store) { feeds.Delete(st) }
