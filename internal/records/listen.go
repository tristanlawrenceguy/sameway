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
// too, which no change logs). A store's listeners are kept on it
// (store.Keep), so they go when it closes.

type feed struct {
	mu  sync.Mutex
	fns []func(actor string, c Change)
}

// feedKey is where a store keeps its feed.
type feedKey struct{}

// Listen adds fn to what is told of each change logged in st, once per
// change, with who made it; c.Activity is its entry in the log.
func Listen(st *store.Store, fn func(actor string, c Change)) {
	f := st.Keep(feedKey{}, func() any { return &feed{} }).(*feed)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fns = append(f.fns, fn)
}

// tell tells st's listeners of one change logged.
func tell(st *store.Store, actor string, c Change) {
	f, ok := st.Kept(feedKey{}).(*feed)
	if !ok {
		return
	}
	f.mu.Lock()
	fns := f.fns
	f.mu.Unlock()
	for _, fn := range fns {
		fn(actor, c)
	}
}
