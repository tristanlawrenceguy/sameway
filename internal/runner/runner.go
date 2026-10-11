// Package runner does a workspace's background work: each job on its own
// schedule, for as long as the server runs, and no longer. One Runner per
// app holds them all, so there is one loop to read instead of one per job,
// one place that says when each last ran and what went wrong, and one
// shutdown that waits for them to finish.
package runner

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"time"
)

// Job is one piece of background work.
type Job struct {
	// Name says what it does, in a person's words: "Reminders ring".
	Name string
	// Every is the time between runs. A job with neither Every nor Next
	// runs once, at the start: a worker that lasts until ctx ends.
	Every time.Duration
	// Next, when given, is when to run after a run that started at now,
	// instead of Every.
	Next func(now time.Time) time.Time
	// Wait has the first run come after one interval instead of at once.
	Wait bool
	// Run does the work. ctx is the runner's, which ends at shutdown;
	// now is the injected clock's time.
	Run func(ctx context.Context, now time.Time) error
}

// periodic says whether the job runs more than once.
func (j Job) periodic() bool { return j.Every > 0 || j.Next != nil }

// after is when the job runs next after a run that started at now.
func (j Job) after(now time.Time) time.Time {
	if j.Next != nil {
		return j.Next(now)
	}
	return now.Add(j.Every)
}

// Clock is the time it is and the way to wait. A zero field is this
// computer's: time.Now and time.After. A test gives a Fake's.
type Clock struct {
	Now   func() time.Time
	After func(time.Duration) <-chan time.Time
}

func (c Clock) now() time.Time {
	if c.Now == nil {
		return time.Now()
	}
	return c.Now()
}

func (c Clock) after(d time.Duration) <-chan time.Time {
	if c.After == nil {
		return time.After(d)
	}
	return c.After(d)
}

// Runner runs jobs from Start until its context ends.
type Runner struct {
	clock Clock
	// Log is told a job's error when it is not the same as last time, so
	// a job failing every few seconds says so once. nil is log.Printf.
	Log func(format string, args ...any)

	mu   sync.Mutex
	jobs []*entry
	ctx  context.Context // set by Start
	wg   sync.WaitGroup
}

// New is a runner on the clock c.
func New(c Clock) *Runner { return &Runner{clock: c} }

// Add adds jobs. Added after Start, a job starts at once.
func (r *Runner) Add(jobs ...Job) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, j := range jobs {
		e := &entry{job: j}
		r.jobs = append(r.jobs, e)
		if r.ctx != nil {
			r.start(e)
		}
	}
}

// Start runs every job until ctx ends. A second Start does nothing.
func (r *Runner) Start(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.ctx != nil {
		return
	}
	r.ctx = ctx
	for _, e := range r.jobs {
		r.start(e)
	}
}

// start runs one job's loop; r.mu is held.
func (r *Runner) start(e *entry) {
	r.wg.Add(1)
	go r.loop(r.ctx, e)
}

// Wait waits for every job to finish once the context has ended, for up
// to timeout. It says whether they all did.
func (r *Runner) Wait(timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-r.clock.after(timeout):
		return false
	}
}

func (r *Runner) loop(ctx context.Context, e *entry) {
	defer r.wg.Done()
	if e.job.Wait && e.job.periodic() && !r.sleep(ctx, e, e.job.after(r.clock.now())) {
		return
	}
	for {
		start := r.clock.now()
		r.once(ctx, e, start)
		if !e.job.periodic() || !r.sleep(ctx, e, e.job.after(start)) {
			return
		}
	}
}

// sleep waits until next, or says false when ctx ends first.
func (r *Runner) sleep(ctx context.Context, e *entry, next time.Time) bool {
	e.setNext(next)
	if ctx.Err() != nil {
		return false
	}
	select {
	case <-ctx.Done():
		return false
	case <-r.clock.after(max(next.Sub(r.clock.now()), 0)):
		return true
	}
}

// once runs a job one time, recovering a panic as its error.
func (r *Runner) once(ctx context.Context, e *entry, now time.Time) {
	e.begin(now)
	var err error
	defer func() {
		p := recover()
		if p != nil {
			err = fmt.Errorf("it stopped on a fault in the program: %v", p)
		}
		news := e.end(err)
		switch {
		case p != nil:
			r.logf("%s: %v\n%s", e.job.Name, p, debug.Stack())
		case news && err != nil:
			r.logf("%s: %v", e.job.Name, err)
		}
	}()
	err = e.job.Run(ctx, now)
}

func (r *Runner) logf(format string, args ...any) {
	if r.Log != nil {
		r.Log(format, args...)
		return
	}
	log.Printf(format, args...)
}
