package cli

import (
	"errors"
	"flag"
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// agentCmd lets an agent in with a key of its own, lists those let in,
// and takes a key away. The key is shown here once, in the terminal of
// the one who made it, and kept nowhere: not in the workspace, not in a
// conversation a model reads.
func (c *ctx) agentCmd() error {
	fs := flag.NewFlagSet("agent", flag.ContinueOnError)
	fs.SetOutput(c.Stderr)
	access := fs.String("access", records.Edit, "what it may do: view, edit, or owner (everything the owner may)")
	positional, err := parseMixed(fs, c.args)
	if err != nil {
		return err
	}
	usage := errors.New("usage: sameway agent add <name> [--access view|edit|owner]   sameway agent list   sameway agent remove <name>")
	if len(positional) == 0 {
		return usage
	}
	a, err := c.load()
	if err != nil {
		return err
	}
	defer a.Close()
	me := records.Who{Actor: "human", Via: records.ThroughCLI}
	switch verb := positional[0]; {
	case verb == "add" && len(positional) == 2:
		key, rec, err := records.LetAgentIn(a.Store, me, positional[1], *access)
		if err != nil {
			return err
		}
		c.print(map[string]any{"name": rec.Fields["name"], "access": rec.Fields["access"], "key": key}, func() {
			fmt.Fprintf(c.Stdout, "%s may now %s this workspace. Its key, shown this once:\n\n  %s\n\n"+
				"The agent sends it as the header Authorization: Bearer <key>, to the API and to /mcp.\n"+
				"Take it away with: sameway agent remove %q\n", rec.Fields["name"], map[string]string{records.View: "look at", records.Edit: "change", records.Owner: "do everything in"}[*access], key, rec.Fields["name"])
		})
	case verb == "list" && len(positional) == 1:
		var out []map[string]any
		for _, x := range records.Agents(a.Store) {
			out = append(out, map[string]any{"name": x.Fields["name"], "access": x.Fields["access"], "last_used": x.Fields["last_used"]})
		}
		c.print(out, func() {
			if len(out) == 0 {
				fmt.Fprintln(c.Stdout, "No agent has a key. Let one in with: sameway agent add <name>")
			}
			for _, x := range out {
				used := "never used"
				if x["last_used"] != nil {
					used = fmt.Sprintf("last used %v", x["last_used"])
				}
				fmt.Fprintf(c.Stdout, "%v\t%v\t%s\n", x["name"], x["access"], used)
			}
		})
	case verb == "remove" && len(positional) == 2:
		ch, err := records.TakeAgentAway(a.Store, positional[1])
		if err != nil {
			return err
		}
		records.Record(a.Store, "human", records.Change{Action: ch.Action, Component: ch.Component, ID: ch.ID, Detail: ch.Detail, Before: ch.Before, Via: records.ThroughCLI})
		c.print(map[string]any{"removed": positional[1]}, func() {
			fmt.Fprintf(c.Stdout, "%s's key no longer works. Undo it from the activity page to let it back in.\n", positional[1])
		})
	default:
		return usage
	}
	return nil
}
