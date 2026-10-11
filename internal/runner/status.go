package runner

import (
	"sync"
	"time"
)

// Status is what is known about one job: when it last ran, how that went,
// and when it runs next. It is kept in memory; a job that already keeps
// its last error somewhere of its own (a setting, the activity log) still
// does.
type Status struct {
	Name  string        `json:"name"`
	Every time.Duration `json:"every,omitempty"`
	// Runs is how many times it has run since the program started.
	Runs int `json:"runs"`
	// Running is true while a run is under way, and for the whole life of
	// a worker that runs once.
	Running bool      `json:"running"`
	LastRun time.Time `json:"last_run,omitzero"`
	// LastErr is what went wrong in the last run, "" when it went well.
	LastErr string    `json:"last_error,omitempty"`
	NextRun time.Time `json:"next_run,omitzero"`
}

// entry is a job and what is known about it.
type entry struct {
	job Job
	mu  sync.Mutex
	st  Status
}

func (e *entry) begin(now time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.st.Running, e.st.LastRun, e.st.NextRun = true, now, time.Time{}
	e.st.Runs++
}

// end records how a run went and says whether its error is news.
func (e *entry) end(err error) bool {
	said := ""
	if err != nil {
		said = err.Error()
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	news := said != e.st.LastErr
	e.st.Running, e.st.LastErr = false, said
	return news
}

func (e *entry) setNext(t time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.st.NextRun = t
}

func (e *entry) status() Status {
	e.mu.Lock()
	defer e.mu.Unlock()
	s := e.st
	s.Name, s.Every = e.job.Name, e.job.Every
	return s
}

// Status is every job, in the order they were added.
func (r *Runner) Status() []Status {
	r.mu.Lock()
	jobs := append([]*entry(nil), r.jobs...)
	r.mu.Unlock()
	out := make([]Status, 0, len(jobs))
	for _, e := range jobs {
		out = append(out, e.status())
	}
	return out
}

// Names are the jobs' names, in the order they were added.
func (r *Runner) Names() []string {
	var names []string
	for _, s := range r.Status() {
		names = append(names, s.Name)
	}
	return names
}
