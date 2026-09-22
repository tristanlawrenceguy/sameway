package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/update"
)

// updateCmd is a person asking for a new version themselves, whatever
// update.mode says: running the command is the asking. It needs no
// workspace, because the program is not part of one.
//
// A sameway built from source does not ask about releases at all, not
// even to look: it cannot say which of the two is newer, so the answer
// would mean nothing.
func (c *ctx) updateCmd() error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(c.Stderr)
	check := fs.Bool("check", false, "only say whether a new version is out, without installing it")
	if err := fs.Parse(c.args); err != nil {
		return err
	}
	if !update.Known(update.Version) {
		return fmt.Errorf("this sameway says %q rather than a version, so it cannot tell whether a release is newer than it is; releases keep themselves current, and `make build` on a tagged checkout stamps the version in", update.Version)
	}
	update.Tidy("")
	out, err := update.Updater{}.Run(context.Background(), !*check)
	if err != nil {
		return err
	}
	c.print(out, func() {
		says := out.Says
		if out.Newer && !out.Installed {
			says += update.AskFor
		}
		fmt.Fprintln(c.Stdout, says)
		if out.Notes != "" && out.Newer {
			fmt.Fprintf(c.Stdout, "\n%s\n", out.Notes)
		}
	})
	return nil
}

// watchUpdates keeps sameway current while the server runs, in the way
// update.mode says: auto installs a release on its own, manual only says
// one is there. Either way it goes in the activity log, where the person
// reads what has happened, and on the terminal the server was started in.
func watchUpdates(ctx context.Context, out io.Writer, a *app.App) {
	// An install before this start left the program it replaced behind,
	// because Windows will not delete one that is running.
	update.Tidy("")
	mode := func() string { return a.Workspace.Config.Update.Mode }
	update.Updater{}.Watch(ctx, mode, update.Every, func(o update.Outcome, err error) {
		if err != nil {
			fmt.Fprintf(out, "  update  %v\n", err)
			return
		}
		fmt.Fprintf(out, "  update  %s\n", o.Says)
		action := "found a new version:"
		if o.Installed {
			action = "updated to"
		}
		chat.Record(a.Store, "system", chat.Change{Action: action, Component: "sameway " + o.Latest})
	})
}
