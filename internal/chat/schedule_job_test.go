package chat_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/runner"
)

// On the runner, scheduled actions are looked at a minute after the start
// and every minute after, on the runner's clock; the worker that runs
// actions when something happens stops when the runner does.
func TestScheduleAndAutomateRunOnTheRunner(t *testing.T) {
	t.Parallel()
	calls := make(chan string, 4)
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls <- r.URL.Path
		io.WriteString(w, "sunny")
	}))
	defer remote.Close()
	svc := newFullService(t)
	svc.HTTP = remote.Client()
	svc.Store.Create(records.ActionType, map[string]any{"title": "Weather", "url": remote.URL + "/hourly", "every": "hour"})

	f := runner.NewFake(time.Date(2026, 9, 15, 7, 30, 0, 0, time.UTC))
	r := runner.New(f.Clock())
	sched := svc.ScheduleJob()
	if sched.Every != time.Minute || !sched.Wait {
		t.Fatalf("the schedule is looked at every minute, after the first: %+v", sched)
	}
	r.Add(sched, svc.AutomateJob())
	ctx, stop := context.WithCancel(context.Background())
	r.Start(ctx)
	f.BlockUntil(1)
	select {
	case got := <-calls:
		t.Fatalf("ran %s at the start; the schedule waits a minute", got)
	default:
	}
	f.Advance(time.Minute)
	if got := <-calls; got != "/hourly" {
		t.Fatalf("a minute on, the hourly action runs: %s", got)
	}
	f.BlockUntil(1)
	stop()
	if !r.Wait(time.Minute) {
		t.Fatal("the automate worker did not stop with the runner")
	}
}
