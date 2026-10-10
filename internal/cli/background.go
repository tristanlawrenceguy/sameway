package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/runner"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// shutdownWait is how long stopping waits for the background jobs to
// finish what they are doing. Windows gives a closed window about five
// seconds before it ends the program anyway.
const shutdownWait = 4 * time.Second

// startBackground starts everything that runs beside the pages for as long
// as the server does, with or without a page open: the same for `sameway
// serve` and `sameway open`. Steps worth a person's notice go to out;
// notes, the daily copies and new versions, to notes (a double-click's
// window leaves them out). The jobs run on a.Jobs (internal/runner), which
// says when each last ran and what went wrong.
func startBackground(ctx context.Context, a *app.App, h *server.Server, all http.Handler, out, notes io.Writer) {
	// Still on loops of their own, until the work on their pages is done.
	h.KeepMeaning(ctx) // search by meaning, when an embedding model is here
	h.KeepReview(ctx)  // the weekly review, on the day set
	h.KeepBrief(ctx)   // the morning brief, when one is set
	h.KeepMail(ctx)    // email sent to the workspace, when connected

	h.OnRing(notifier(a)) // a ring told beyond the page
	connectDevices(ctx, out, a)
	joinTailnet(ctx, out, a, all, h)
	a.Jobs.Add(backgroundJobs(a, h, notes)...)
	a.Jobs.Start(ctx)
}

// backgroundJobs are the jobs the runner does, each on its own schedule:
// the server's (server/jobs.go: rings, calendars, the cloud folder,
// writing recordings down), the chat's and the program's.
func backgroundJobs(a *app.App, h *server.Server, notes io.Writer) []runner.Job {
	jobs := []runner.Job{
		a.SchemaJob(app.SchemaEvery, h.Changed), // types changed by hand, so open pages follow
		a.Chat.ScheduleJob(),                    // actions with a schedule; chat/schedule.go
		a.Chat.AutomateJob(),                    // actions that run when something happens; chat/automate.go
		snapshotJob(notes, a),
	}
	jobs = append(h.BackgroundJobs(), jobs...)
	// Kept current: a double-clicked Sameway, or one opened at sign-in,
	// looks for a new version as serve does.
	jobs = append(jobs, updateJob(notes, a)...)
	return jobs
}

// serveUntilStopped serves on ln until the program is told to stop
// (Ctrl-C, the window closed) or srv is shut down, then ends the
// background jobs and waits a moment for them to finish.
func serveUntilStopped(ctx context.Context, stop context.CancelFunc, out io.Writer, a *app.App, srv *http.Server, ln net.Listener) error {
	go func() {
		<-ctx.Done()
		quick, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		srv.Shutdown(quick)
	}()
	err := srv.Serve(ln)
	stop()
	if !a.Jobs.Wait(shutdownWait) {
		var busy []string
		for _, s := range a.Jobs.Status() {
			if s.Running {
				busy = append(busy, s.Name)
			}
		}
		fmt.Fprintf(out, "Stopped without waiting any longer for: %s\n", strings.Join(busy, ", "))
	}
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// stopContext ends when the program is told to stop.
func stopContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}
