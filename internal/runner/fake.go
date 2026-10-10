package runner

import (
	"sync"
	"time"
)

// Fake is a clock that moves only when told, for tests: no test of the
// background work sleeps for real.
type Fake struct {
	mu      sync.Mutex
	cond    *sync.Cond
	now     time.Time
	waiting []fakeWait
}

type fakeWait struct {
	at time.Time
	ch chan time.Time
}

// NewFake is a clock stopped at now.
func NewFake(now time.Time) *Fake {
	f := &Fake{now: now}
	f.cond = sync.NewCond(&f.mu)
	return f
}

// Clock is the runner's clock on f.
func (f *Fake) Clock() Clock { return Clock{Now: f.Now, After: f.After} }

// Now is the time f says.
func (f *Fake) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

// After fires once f has been moved on by d.
func (f *Fake) After(d time.Duration) <-chan time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	ch := make(chan time.Time, 1)
	if d <= 0 {
		ch <- f.now
		return ch
	}
	f.waiting = append(f.waiting, fakeWait{f.now.Add(d), ch})
	f.cond.Broadcast()
	return ch
}

// Advance moves the time on by d, firing every wait that is now over.
func (f *Fake) Advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = f.now.Add(d)
	kept := f.waiting[:0]
	for _, w := range f.waiting {
		if w.at.After(f.now) {
			kept = append(kept, w)
			continue
		}
		w.ch <- f.now
	}
	f.waiting = kept
}

// BlockUntil returns once n waits are pending: every loop under test is
// asleep, so an Advance reaches them all.
func (f *Fake) BlockUntil(n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for len(f.waiting) < n {
		f.cond.Wait()
	}
}
