package cli

import (
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

// importCmd reads content/ back into the database, after a git pull. Every
// record it changes goes in the activity log with what it was before, so
// an import is undone the way any other change is.
func (c *ctx) importCmd() error {
	a, err := c.load()
	if err != nil {
		return err
	}
	defer a.Close()
	rep, err := a.Mirror.Import(a.Store, func(action string, rec *store.Record, before map[string]any) {
		t, _ := a.Types.Get(rec.Type)
		ch := chat.Change{Action: action, Component: rec.Type, ID: rec.ID, Before: before}
		if t != nil {
			ch.Detail = summary(rec, t.Title)
			if !t.Internal && action != "deleted" {
				ch.Href = "/t/" + rec.Type + "/" + rec.ID
			}
		}
		chat.Record(a.Store, "human", ch)
	})
	if err != nil {
		return err
	}
	c.print(rep, func() {
		fmt.Fprintf(c.Stdout, "created %d, updated %d, deleted %d from %s\n", rep.Created, rep.Updated, rep.Deleted, a.Mirror.Dir)
		if len(rep.Problems) > 0 {
			fmt.Fprintf(c.Stdout, "skipped:\n  %s\n", strings.Join(rep.Problems, "\n  "))
		}
	})
	return nil
}
