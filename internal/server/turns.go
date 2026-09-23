package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// The turns under way, each kept as it goes so that any page can follow
// it, however late it comes, and so that it can be stopped. See stream.go.

// A liveTurn is a turn under way: what it has done so far, kept so a page
// that comes to it late hears all of it, and the way to stop it.
type liveTurn struct {
	id      string
	started time.Time
	cancel  context.CancelFunc

	mu        sync.Mutex
	events    []chat.Event
	over      bool
	changed   chan struct{}
	listeners int
}

// listen counts a page that follows the turn, coming (1) or going (-1).
func (t *liveTurn) listen(n int) {
	t.mu.Lock()
	t.listeners += n
	t.mu.Unlock()
}

// followed says whether any page is following the turn now.
func (t *liveTurn) followed() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.listeners > 0
}

// add records what the turn just did and wakes whoever is following it.
// Events come from the turn and, when the tools run elsewhere, from the
// watch on the log: one at a time.
func (t *liveTurn) add(e chat.Event) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, e)
	close(t.changed)
	t.changed = make(chan struct{})
}

// since is what the turn has done from the i-th event on, whether it is
// over, and what closes when there is more.
func (t *liveTurn) since(i int) ([]chat.Event, bool, <-chan struct{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.events[i:], t.over, t.changed
}

// turns are the live turns under way.
type turns struct {
	mu   sync.Mutex
	n    int
	live map[string]*liveTurn
}

// start begins a turn and the context it runs under.
func (ts *turns) start(parent context.Context) (*liveTurn, context.Context) {
	ctx, cancel := context.WithCancel(parent)
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if ts.live == nil {
		ts.live = map[string]*liveTurn{}
	}
	ts.n++
	t := &liveTurn{id: fmt.Sprintf("turn-%d", ts.n), started: time.Now(), cancel: cancel, changed: make(chan struct{})}
	ts.live[t.id] = t
	return t, ctx
}

// end is a turn over: whoever follows it hears the rest and stops.
func (ts *turns) end(t *liveTurn) {
	t.cancel()
	ts.mu.Lock()
	delete(ts.live, t.id)
	ts.mu.Unlock()
	t.mu.Lock()
	t.over = true
	close(t.changed)
	t.changed = make(chan struct{})
	t.mu.Unlock()
}

// find is the turn named, or the latest under way when none is named.
func (ts *turns) find(id string) *liveTurn {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if id != "" {
		return ts.live[id]
	}
	var latest *liveTurn
	for _, t := range ts.live {
		if latest == nil || t.started.After(latest.started) {
			latest = t
		}
	}
	return latest
}

// stop ends the turn named, or every turn under way when none is.
func (ts *turns) stop(id string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	for k, t := range ts.live {
		if id == "" || k == id {
			t.cancel()
		}
	}
}
