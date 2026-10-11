package runner

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

var start = time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)

// ran is a job that says each time it runs on a channel.
func ran(name string, every time.Duration) (Job, chan time.Time) {
	ch := make(chan time.Time, 16)
	return Job{Name: name, Every: every, Run: func(_ context.Context, now time.Time) error {
		ch <- now
		return nil
	}}, ch
}

func none(t *testing.T, ch chan time.Time) {
	t.Helper()
	select {
	case at := <-ch:
		t.Fatalf("ran at %v, before its time", at)
	default:
	}
}

func quiet(r *Runner) *Runner {
	r.Log = func(string, ...any) {}
	return r
}

func TestRunsOnSchedule(t *testing.T) {
	t.Parallel()
	f := NewFake(start)
	r := quiet(New(f.Clock()))
	job, ch := ran("Ring", time.Minute)
	later, lch := ran("Later", time.Minute)
	later.Wait = true
	r.Add(job, later)
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	r.Start(ctx)
	if at := <-ch; !at.Equal(start) {
		t.Fatalf("the first run is at the start, not %v", at)
	}
	f.BlockUntil(2)
	none(t, lch)
	f.Advance(59 * time.Second)
	none(t, ch)
	f.Advance(time.Second)
	if at := <-ch; !at.Equal(start.Add(time.Minute)) {
		t.Fatalf("the second run is a minute on, not %v", at)
	}
	if at := <-lch; !at.Equal(start.Add(time.Minute)) {
		t.Fatalf("a job that waits first runs after one interval, not %v", at)
	}
	f.BlockUntil(2)
	st := r.Status()
	if st[0].Name != "Ring" || st[0].Runs != 2 || !st[0].NextRun.Equal(start.Add(2*time.Minute)) || !st[0].LastRun.Equal(start.Add(time.Minute)) {
		t.Fatalf("status = %+v", st[0])
	}
}

func TestNextChoosesTheTime(t *testing.T) {
	t.Parallel()
	f := NewFake(start)
	r := quiet(New(f.Clock()))
	job, ch := ran("Daily", 0)
	job.Next = func(now time.Time) time.Time { return now.Truncate(24 * time.Hour).Add(24 * time.Hour) }
	r.Add(job)
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	r.Start(ctx)
	<-ch
	f.BlockUntil(1)
	f.Advance(15 * time.Hour)
	if at := <-ch; !at.Equal(time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("the next run is at midnight, not %v", at)
	}
}

func TestStopsOnCancelAndWaits(t *testing.T) {
	t.Parallel()
	f := NewFake(start)
	r := quiet(New(f.Clock()))
	job, ch := ran("Ring", time.Minute)
	release := make(chan struct{})
	busy := make(chan struct{})
	r.Add(job, Job{Name: "Slow", Every: time.Minute, Run: func(context.Context, time.Time) error {
		close(busy)
		<-release
		return nil
	}})
	ctx, stop := context.WithCancel(context.Background())
	r.Start(ctx)
	<-ch
	<-busy
	f.BlockUntil(1) // Ring asleep, Slow still running
	stop()
	done := make(chan bool)
	go func() { done <- r.Wait(time.Minute) }()
	f.BlockUntil(2) // Ring's wait, left by its loop, and Wait's own limit
	select {
	case <-done:
		t.Fatal("shutdown finished while a job was still running")
	default:
	}
	close(release)
	if !<-done {
		t.Fatal("shutdown gave up although every job finished")
	}
	f.Advance(time.Hour)
	none(t, ch)
}

func TestWaitGivesUpAfterTimeout(t *testing.T) {
	t.Parallel()
	f := NewFake(start)
	r := quiet(New(f.Clock()))
	stuck, began := make(chan struct{}), make(chan struct{})
	defer close(stuck)
	r.Add(Job{Name: "Stuck", Run: func(context.Context, time.Time) error { close(began); <-stuck; return nil }})
	ctx, stop := context.WithCancel(context.Background())
	r.Start(ctx)
	<-began
	stop()
	done := make(chan bool)
	go func() { done <- r.Wait(5 * time.Second) }()
	f.BlockUntil(1)
	f.Advance(5 * time.Second)
	if <-done {
		t.Fatal("Wait said every job finished, but one is stuck")
	}
	if st := r.Status()[0]; !st.Running {
		t.Fatalf("a worker that runs once is running for its whole life: %+v", st)
	}
}

func TestRecordsErrorsAndRecoversPanics(t *testing.T) {
	t.Parallel()
	f := NewFake(start)
	r := New(f.Clock())
	var mu sync.Mutex
	var logged []string
	r.Log = func(format string, args ...any) {
		mu.Lock()
		logged = append(logged, format)
		mu.Unlock()
	}
	runs := make(chan int, 8)
	n := 0
	r.Add(Job{Name: "Copy", Every: time.Hour, Run: func(context.Context, time.Time) error {
		n++
		runs <- n
		switch n {
		case 1, 2:
			return errors.New("the folder is not there")
		case 3:
			panic("nil map")
		}
		return nil
	}})
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	r.Start(ctx)
	want := []string{"the folder is not there", "the folder is not there", "it stopped on a fault in the program: nil map", ""}
	for i, w := range want {
		<-runs
		f.BlockUntil(1)
		if got := r.Status()[0].LastErr; got != w {
			t.Fatalf("after run %d the last error is %q, want %q", i+1, got, w)
		}
		f.Advance(time.Hour)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(logged) != 2 || !strings.Contains(logged[1], "%v\n%s") {
		t.Fatalf("logged the same error once and the fault with its stack, got %q", logged)
	}
}

func TestAddAfterStartRunsAtOnce(t *testing.T) {
	t.Parallel()
	f := NewFake(start)
	r := quiet(New(f.Clock()))
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	r.Start(ctx)
	job, ch := ran("Late", time.Minute)
	r.Add(job)
	<-ch
	if got := r.Names(); len(got) != 1 || got[0] != "Late" {
		t.Fatalf("names = %v", got)
	}
}
