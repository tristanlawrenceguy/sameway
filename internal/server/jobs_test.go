package server_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
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
	every, sleepers := map[string]time.Duration{}, 0
	for _, j := range jobs {
		if j.Every > 0 || j.Next != nil {
			sleepers++
		}
		every[j.Name] = j.Every
		if j.Name == "Changes passed through the cloud folder" && !j.Wait {
			t.Error("the cloud folder's sync waits one round before its first, as its loop did")
		}
	}
	want := map[string]time.Duration{"Reminders ring": 5 * time.Second, "Meeting transcripts from Teams and Zoom": 5 * time.Second,
		"Calendars kept in step": time.Hour, "Daily copy in the cloud folder": time.Hour, "Changes passed through the cloud folder": 15 * time.Second,
		"Search by meaning kept current": 2 * time.Minute, "Email read from the mailbox": 5 * time.Minute}
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
	f.BlockUntil(sleepers)
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
		// Calendars and search by meaning depend on what this computer has.
		if st.LastErr != "" && !strings.Contains(st.Name, "Calendars") && !strings.Contains(st.Name, "meaning") {
			t.Errorf("%s: %s", st.Name, st.LastErr)
		}
	}
}

// The morning brief and the weekly review come at the times set on Help,
// on the runner's clock: across midnight, at a time changed while they
// wait, not at all with none set, and never once the server has stopped.
func TestTheBriefAndReviewComeAtTheirTimes(t *testing.T) {
	t.Parallel()
	f := runner.NewFake(time.Date(2026, 10, 10, 6, 0, 0, 0, time.Local)) // a Saturday
	a, h := newAppWith(t, app.Options{Clock: f.Now, After: f.After})
	srv := h.(*server.Server)
	set := func(key, value string) {
		if err := a.Workspace.Set(key, value); err != nil {
			t.Fatal(err)
		}
	}
	set("ui.clock", "12")
	set("brief.at", "07:30")
	set("review.on", "sunday")
	a.Store.Create("task", map[string]any{"title": "Pay rent", "due": "2026-10-10"}) // late from Sunday on
	told := make(chan string, 8)
	srv.OnRing(func(title, _, _ string) { told <- title })
	for _, j := range srv.BackgroundJobs() {
		if j.Name == "Morning brief" || j.Name == "Weekly review ready" {
			a.Jobs.Add(j)
		}
	}
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	a.Jobs.Start(ctx)
	step := func(d time.Duration, want string) {
		t.Helper()
		f.BlockUntil(2)
		f.Advance(d)
		if want == "" {
			return
		}
		select {
		case got := <-told:
			if !strings.HasPrefix(got, want) {
				t.Fatalf("at %v: %q, want %q", f.Now(), got, want)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("at %v nothing came, want %q", f.Now(), want)
		}
	}
	step(90*time.Minute, "Today: 1 task") // Saturday 7:30
	set("brief.at", "07:00")
	set("review.on", "monday")
	step(23*time.Hour+30*time.Minute, "Today: 1 overdue") // Sunday 7:00, the new time
	step(11*time.Hour, "")                                // Sunday 18:00, no longer the day
	f.BlockUntil(2)
	if sent := a.Store.Meta("review:sent"); sent != "" {
		t.Fatalf("the review came on the day it was moved from: %q", sent)
	}
	step(13*time.Hour, "Today: 1 overdue")    // Monday 7:00
	step(11*time.Hour, "Your week in review") // Monday 18:00
	f.BlockUntil(2)
	if page := get(t, h, "/help").Body.String(); !strings.Contains(page, "Morning brief: at the time set for it, if one is. Last ran today at 7am, and went well. Next tomorrow at 7am.") {
		t.Errorf("Help says when the brief comes next\n%.4000s", page)
	}

	set("brief.at", "")
	set("review.on", "")
	step(time.Minute, "")
	step(48*time.Hour, "")
	f.BlockUntil(2)
	for _, st := range a.Jobs.Status() {
		if !st.NextRun.IsZero() || st.LastErr != "" {
			t.Errorf("with no time set, %s waits for none: %+v", st.Name, st)
		}
	}
	set("brief.at", "07:00")
	stop()
	if !a.Jobs.Wait(time.Minute) {
		t.Fatal("the jobs did not stop")
	}
	f.Advance(48 * time.Hour)
	if sent := a.Store.Meta("brief:sent"); sent != "2026-10-12" {
		t.Fatalf("a brief after the server stopped: %q", sent)
	}
}

// Email is read on the runner every five minutes after the first wait,
// and what went wrong is kept for Help: a mailbox that cannot be reached,
// then nothing once it is disconnected.
func TestTheMailJobSaysWhatWentWrong(t *testing.T) {
	t.Parallel()
	f := runner.NewFake(time.Date(2026, 10, 10, 9, 0, 0, 0, time.Local))
	a, h := newAppWith(t, app.Options{Clock: f.Now, After: f.After})
	// Connected to a mailbox on a port where nothing answers.
	if err := llm.Keys(a.Workspace.Machine.Keys).Save("SAMEWAY_MAIL_PASSWORD me@example.com", "secret"); err != nil {
		t.Fatal(err)
	}
	a.Store.SetMeta("mail:account", `{"host":"127.0.0.1:1","user":"me@example.com"}`)
	for _, j := range h.(*server.Server).BackgroundJobs() {
		if j.Name == "Email read from the mailbox" {
			a.Jobs.Add(j)
		}
	}
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	a.Jobs.Start(ctx)
	status := func() runner.Status {
		f.BlockUntil(1)
		return a.Jobs.Status()[0]
	}
	if st := status(); st.Runs != 0 {
		t.Fatalf("the mailbox is read after the first five minutes, not at the start: %+v", st)
	}
	f.Advance(5 * time.Minute)
	if st := status(); st.Runs != 1 || st.LastErr == "" {
		t.Fatalf("a mailbox that cannot be reached is an error on Help: %+v", st)
	}
	a.Store.SetMeta("mail:account", "")
	f.Advance(5 * time.Minute)
	if st := status(); st.Runs != 2 || st.LastErr != "" {
		t.Fatalf("disconnected, there is nothing to go wrong: %+v", st)
	}
	stop()
	if !a.Jobs.Wait(time.Minute) {
		t.Fatal("the job did not stop")
	}
	f.Advance(time.Hour)
	if st := a.Jobs.Status()[0]; st.Runs != 2 {
		t.Fatalf("it ran after the server stopped: %+v", st)
	}
}
