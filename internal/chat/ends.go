package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// How a turn ends. Most end when the model answers in words. There is no
// count of rounds: a turn may use as many tools as the work takes. It
// ends early only when it is plainly getting nowhere, which a person
// would notice too: the same call made again with the same arguments,
// or tools that fail three rounds running. Or when the person stops it.
// Either way the reply says what was done, and what was done stays.

// progress watches a turn for going round in circles.
type progress struct {
	made    map[string]bool
	lastErr string
	failing int
}

// check says why a round should not run, or nothing: a call made
// earlier in the turn, with the same arguments, would only repeat it.
func (p *progress) check(calls []llm.ToolCall) string {
	if p.made == nil {
		p.made = map[string]bool{}
	}
	for _, c := range calls {
		key := c.Name + string(c.Args)
		if p.made[key] {
			return "That same call, with the same arguments, was made earlier in this turn; making it again would not help."
		}
		p.made[key] = true
	}
	return ""
}

// round notes how a round of calls went, and says why to stop when the
// tools have failed the same way three rounds in a row. Different
// failures are a model finding its way, and each error tells it more;
// the same failure again is a wall.
func (p *progress) round(results []llm.ToolResult) string {
	var errs []string
	for _, r := range results {
		if !r.IsError {
			p.failing, p.lastErr = 0, ""
			return ""
		}
		errs = append(errs, r.Content)
	}
	if len(errs) == 0 {
		return ""
	}
	if joined := strings.Join(errs, "\n"); joined == p.lastErr {
		p.failing++
	} else {
		p.failing, p.lastErr = 1, joined
	}
	if p.failing >= 3 {
		return "The tools have failed the same way three rounds in a row."
	}
	return ""
}

// wrapUp ends a turn that is getting nowhere: the model is offered no
// tools and asked to say where things stand, and that is the reply,
// with everything the turn changed.
func (s *Service) wrapUp(ctx context.Context, req llm.Request, why, said string, changes []Change, tools []map[string]any, on func(Event)) (*store.Record, error) {
	req.Tools = nil
	req.Messages = append(req.Messages, llm.Message{Role: llm.RoleUser, Content: why + " Stop here and say, in a few words, what you did and what is still to do."})
	resp, err := s.complete(ctx, req, on)
	if ctx.Err() != nil {
		return s.reply(said, stoppedText, changes, tools)
	}
	if err != nil || strings.TrimSpace(resp.Text) == "" {
		return s.failAfter(said, fmt.Errorf("%s The model then had no final answer.", why), changes, tools)
	}
	return s.reply(said, strings.TrimSpace(resp.Text), changes, tools)
}

// stoppedText is the reply when the person stopped the turn.
const stoppedText = "Stopped, as you asked."

// reply records the words the assistant ends a turn with, and what the
// turn did. When the tools ran in another program, the log knows what
// they did.
func (s *Service) reply(said, text string, changes []Change, tools []map[string]any) (*store.Record, error) {
	if s.runsToolsOutside() {
		changes = s.changesAfter(said)
	}
	return s.message(map[string]any{"role": "assistant", "content": text, "changes": changes, "tools": tools})
}

// failAfter is a turn that went wrong after it had already changed
// things. The failure is said, and so is what it did before, as the
// receipt any reply carries, each change with its Undo: a failure must
// not hide changes that happened.
func (s *Service) failAfter(said string, err error, changes []Change, tools []map[string]any) (*store.Record, error) {
	if s.runsToolsOutside() {
		changes = s.changesAfter(said)
	}
	if len(changes) == 0 {
		return s.fail(err)
	}
	Record(s.Store, "system", Change{Action: "failed", Detail: truncate(err.Error(), 200)})
	rec, storeErr := s.message(map[string]any{"role": "error", "content": err.Error() + " What it had done before that is below, and each can be undone.", "changes": changes, "tools": tools})
	if storeErr != nil {
		return nil, storeErr
	}
	return rec, err
}
