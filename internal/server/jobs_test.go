package server_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/runner"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// The server's background work runs on the app's runner, each job at the
// interval its own loop had: reminders ring on the runner's clock, the
// day's copy goes to the cloud folder, and the cloud folder's sync waits
// one round before its first.
func TestTheServersJobsRunOnTheRunner(t *testing.T) {
	t.Parallel()
	f := runner.NewFake(time.Date(2026, 10, 10, 9, 0, 0, 0, time.Local))
	a, h := newAppWith(t, app.Options{Clock: f.Now, After: f.After})
	srv := h.(*server.Server)
	jobs := srv.BackgroundJobs()
	every := map[string]time.Duration{}
	for _, j := range jobs {
		every[j.Name] = j.Every
		if j.Name == "Changes passed through the cloud folder" && !j.Wait {
			t.Error("the cloud folder's sync waits one round before its first, as its loop did")
		}
	}
	want := map[string]time.Duration{"Reminders ring": 5 * time.Second, "Meeting transcripts from Teams and Zoom": 5 * time.Second,
		"Calendars kept in step": time.Hour, "Daily copy in the cloud folder": time.Hour, "Changes passed through the cloud folder": 15 * time.Second}
	for name, d := range want {
		if every[name] != d {
			t.Errorf("%s every %v, want %v", name, every[name], d)
		}
	}

	cloud := t.TempDir()
	if err := a.Workspace.Set("backup.folder", cloud); err != nil {
		t.Fatal(err)
	}
	rang := make(chan string, 4)
	srv.OnRing(func(title, _, _ string) { rang <- title })
	a.Store.Create(server.ReminderType, map[string]any{"title": "Tea", "at": when.Store(f.Now().Add(-time.Minute), false)})
	a.Store.Create(server.ReminderType, map[string]any{"title": "Stretch", "at": when.Store(f.Now().Add(3*time.Second), false)})
	a.Jobs.Add(jobs...)
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	a.Jobs.Start(ctx)
	if got := <-rang; got != "Tea" {
		t.Fatalf("what is due rings at the start: %q", got)
	}
	f.BlockUntil(len(jobs))
	f.Advance(5 * time.Second)
	if got := <-rang; got != "Stretch" {
		t.Fatalf("five seconds on, the next rings: %q", got)
	}
	copies, _ := filepath.Glob(filepath.Join(cloud, "Sameway copies", "*", "* 2026-10-10.zip"))
	if len(copies) != 1 {
		t.Fatalf("the day's copy is in the cloud folder, dated by the runner's clock: %v", copies)
	}
	stop()
	if !a.Jobs.Wait(time.Minute) {
		t.Fatal("the jobs did not stop")
	}
	for _, st := range a.Jobs.Status() {
		if st.LastErr != "" && !strings.Contains(st.Name, "Calendars") {
			t.Errorf("%s: %s", st.Name, st.LastErr)
		}
	}
}
