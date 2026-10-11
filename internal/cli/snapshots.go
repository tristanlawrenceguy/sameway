package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/runner"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// snapshotJob copies the workspace's database once a day for as long as
// the server runs, keeping the last seven: see workspace/snapshots.go. It
// looks every hour, so a day that starts while it runs gets its copy.
func snapshotJob(out io.Writer, a *app.App) runner.Job {
	return runner.Job{Name: "Daily copy of the data", Every: time.Hour, Run: func(_ context.Context, now time.Time) error {
		path, err := a.Workspace.DailySnapshot(now, a.Store.Backup)
		if err != nil {
			return err
		}
		if path != "" {
			fmt.Fprintf(out, "  saved   a copy of today's data in %s\n", filepath.Dir(path))
		}
		return nil
	}}
}

// snapshotsCmd lists the copies of this workspace's data, newest first.
func (c *ctx) snapshotsCmd() error {
	ws, err := c.loadWorkspace()
	if err != nil {
		return err
	}
	list := ws.Snapshots()
	c.print(list, func() {
		if len(list) == 0 {
			fmt.Fprintf(c.Stdout, "No copies yet: one is made each day the workspace runs, in %s\n", ws.SnapshotDir())
			return
		}
		for _, s := range list {
			fmt.Fprintf(c.Stdout, "%s  %s  %d KB\n", s.At.Local().Format("Mon 2 Jan 15:04"), s.Path, s.Size/1024)
		}
		fmt.Fprintln(c.Stdout, "\nTo go back to one: sameway restore <path>, with the workspace stopped.")
	})
	return nil
}

// restoreCmd puts a copy back as the workspace's data. The data as it is
// now is kept first, so the restore can be undone the same way.
func (c *ctx) restoreCmd() error {
	if len(c.args) != 1 {
		return errors.New("usage: sameway restore <path of a copy>   (sameway snapshots lists them)")
	}
	ws, err := c.loadWorkspace()
	if err != nil {
		return err
	}
	kept, err := ws.Restore(c.args[0])
	if err != nil {
		return err
	}
	c.print(map[string]any{"restored": c.args[0], "kept": kept}, func() {
		fmt.Fprintf(c.Stdout, "Restored %s.\n", c.args[0])
		if kept != "" {
			fmt.Fprintf(c.Stdout, "The data from before is kept at %s; restore that to undo this.\n", kept)
		}
	})
	return nil
}

// loadWorkspace finds the workspace without opening its database, which a
// restore replaces and so must not hold.
func (c *ctx) loadWorkspace() (*workspace.Workspace, error) {
	dir := c.workspaceDir
	if dir == "" {
		start := c.Dir
		if start == "" {
			start, _ = os.Getwd()
		}
		found, err := workspace.Find(start)
		if err != nil {
			return nil, err
		}
		dir = found
	}
	return workspace.Load(dir)
}
