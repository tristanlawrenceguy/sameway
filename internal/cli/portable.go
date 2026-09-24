package cli

import (
	"flag"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// exportCmd rewrites content/ from the database. The folder is kept
// current as records change, so this is for a workspace that predates
// that, or whose database was changed behind its back.
func (c *ctx) exportCmd() error {
	a, err := c.load()
	if err != nil {
		return err
	}
	defer a.Close()
	rep, err := a.Mirror.Export(a.Store)
	if err != nil {
		return err
	}
	c.print(rep, func() {
		fmt.Fprintf(c.Stdout, "wrote %d files under %s, removed %d\n", rep.Written, a.Mirror.Dir, rep.Removed)
	})
	return nil
}

// importCmd reads content/ back into the database, after a git pull. The
// folder wins: records without a file go, and records that differ take
// the file's words. So the whole import is one entry in the activity log,
// with what every record it touched was before, and one Undo takes it all
// back. --dry-run says what it would do and does none of it.
func (c *ctx) importCmd() error {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(c.Stderr)
	dry := fs.Bool("dry-run", false, "say what the import would make, change and remove, and change nothing")
	if err := fs.Parse(c.args); err != nil {
		return err
	}
	a, err := c.load()
	if err != nil {
		return err
	}
	defer a.Close()
	m := a.Mirror
	m.DryRun = *dry
	var batch []chat.BatchItem
	rep, err := m.Import(a.Store, func(action string, rec *store.Record, before map[string]any) {
		batch = append(batch, chat.BatchItem{Type: rec.Type, ID: rec.ID, Before: before})
	})
	if err != nil {
		return err
	}
	if len(batch) > 0 {
		chat.Record(a.Store, "human", chat.Change{Action: "synced", Component: "content",
			Detail: fmt.Sprintf("from %s: %d made, %d changed, %d removed", a.Mirror.Dir, rep.Created, rep.Updated, rep.Deleted),
			Before: chat.Batch(batch), Via: chat.ThroughCLI})
	}
	c.print(rep, func() {
		if *dry {
			fmt.Fprintf(c.Stdout, "would make %d, change %d, remove %d from %s; nothing was changed\n", rep.Created, rep.Updated, rep.Deleted, a.Mirror.Dir)
			for _, line := range rep.Changes {
				fmt.Fprintln(c.Stdout, "  "+line)
			}
		} else {
			fmt.Fprintf(c.Stdout, "created %d, updated %d, deleted %d from %s; one Undo on the activity page takes it all back\n", rep.Created, rep.Updated, rep.Deleted, a.Mirror.Dir)
		}
		if len(rep.Problems) > 0 {
			fmt.Fprintf(c.Stdout, "skipped:\n  %s\n", strings.Join(rep.Problems, "\n  "))
		}
	})
	return nil
}
