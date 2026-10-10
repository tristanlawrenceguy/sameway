package cli

import (
	"context"
	"io"
	"slices"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/runner"
	"github.com/tristanlawrenceguy/sameway/internal/server/servertest"
)

// The background jobs run on the app's runner and its clock: the daily
// copy is made at the start, looked for again every hour, and the runner
// stops and waits for them when the server does.
func TestBackgroundJobsRunOnTheAppsRunner(t *testing.T) {
	t.Parallel()
	f := runner.NewFake(time.Date(2026, 10, 10, 23, 30, 0, 0, time.Local))
	a, _ := servertest.NewWith(t, app.Options{Clock: f.Now, After: f.After})
	jobs := backgroundJobs(a, io.Discard)
	names := []string{}
	for _, j := range jobs {
		names = append(names, j.Name)
	}
	if !slices.Contains(names, "Daily copy of the data") {
		t.Fatalf("jobs = %v", names)
	}
	a.Jobs.Add(jobs...)
	ctx, stop := context.WithCancel(context.Background())
	a.Jobs.Start(ctx)
	f.BlockUntil(len(jobs))
	if got := a.Workspace.Snapshots(); len(got) != 1 {
		t.Fatalf("the first copy is made at the start: %v", got)
	}
	f.Advance(time.Hour) // past midnight: a new day, a new copy
	f.BlockUntil(len(jobs))
	if got := a.Workspace.Snapshots(); len(got) != 2 {
		t.Fatalf("a new day gets its copy within the hour: %v", got)
	}
	st := a.Jobs.Status()[0]
	if st.Runs != 2 || st.LastErr != "" || !st.NextRun.Equal(f.Now().Add(time.Hour)) {
		t.Fatalf("status = %+v", st)
	}
	stop()
	if !a.Jobs.Wait(time.Minute) {
		t.Fatal("the jobs did not stop when the server did")
	}
}
