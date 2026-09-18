package chat

import (
	"context"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A provider that runs the tools itself, over MCP in another process,
// says so; the receipt under its reply then comes from the activity log,
// where every tool call lands whoever made it.
type toolsOutside interface{ ToolsOutside() bool }

func (s *Service) runsToolsOutside() bool {
	p, ok := s.Provider.(toolsOutside)
	return ok && p.ToolsOutside()
}

// changesAfter is what the assistant did during a turn that ran its
// tools elsewhere: every log entry after the turn's own "said" entry that
// is not the person's, as the receipt the reply carries, each undoable.
// The log is ordered as it was written, so the said entry is the fence.
func (s *Service) changesAfter(saidID string) []Change {
	recs, err := s.Store.List(ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 100})
	if err != nil {
		return nil
	}
	var out []Change
	for _, rec := range recs {
		if rec.ID == saidID {
			break
		}
		actor, _ := rec.Fields["actor"].(string)
		action, _ := rec.Fields["action"].(string)
		if actor == "human" || action == "said" {
			continue
		}
		target, _ := rec.Fields["target"].(string)
		id, _ := rec.Fields["target_id"].(string)
		detail, _ := rec.Fields["detail"].(string)
		c := Change{Action: action, Component: target, ID: id, Detail: detail, Activity: rec.ID}
		if id != "" {
			if _, isType := s.Store.Types().Get(target); isType {
				c.Href = "/t/" + target + "/" + id
			} else {
				c.Href = "/canvas/" + id
			}
		}
		out = append([]Change{c}, out...)
	}
	return out
}

// watch tells on each change as it lands in the log while the tools of a
// turn run in another program, so a page can show that turn as it
// happens too: each change as a step, then the change itself. The
// function returned ends the watch and tells whatever landed last.
func (s *Service) watch(ctx context.Context, saidID string, on func(Event)) (stop func()) {
	seen := map[string]bool{}
	tell := func() {
		for _, c := range s.changesAfter(saidID) {
			if seen[c.Activity] {
				continue
			}
			seen[c.Activity] = true
			c := c
			on(Event{Kind: "tool", Tool: c.Action, Label: describeChange(c)})
			on(Event{Kind: "change", Change: &c})
		}
	}
	done, finished := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(finished)
		tick := time.NewTicker(300 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-done:
				tell()
				return
			case <-ctx.Done():
				return
			case <-tick.C:
				tell()
			}
		}
	}()
	return func() { close(done); <-finished }
}
