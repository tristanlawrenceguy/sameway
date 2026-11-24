package automate

import (
	"cmp"
	"fmt"
	"net/url"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// An action that sends something out of the workspace (a webhook, a
// message to a device) and runs on its own (when a record changes, on a
// schedule, from outside) sends only what the owner accepted, once, with
// it in front of them: the assistant can make such an action, and words
// it read (an email, a shared record) must never be enough to have the
// workspace send things on its own. A command asks the same way
// (command.go); a person pressing their own button is not asked.

// Sends is what an action that reaches outside would send, as set up
// (before what set it off is filled in), or "" for one that stays inside
// or has nowhere to send, which is refused as it runs.
func Sends(rec *store.Record) string {
	text := func(k string) string { v, _ := rec.Fields[k].(string); return strings.TrimSpace(v) }
	switch text("kind") {
	case "arrangement", "message", "classify", "suggest", "command":
		return ""
	case "mqtt":
		if text("topic") == "" {
			return ""
		}
		return "mqtt " + text("topic") + "\n" + text("payload")
	}
	if text("url") == "" {
		return ""
	}
	return strings.ToUpper(cmp.Or(text("method"), "POST")) + " " + text("url") + "\n" + text("body")
}

// unaccepted asks the owner to accept what an action sends on its own,
// once, and says so; asked is false when it may run.
func (s *Service) unaccepted(rec *store.Record) (Result, bool) {
	what := Sends(rec)
	if what == "" {
		return Result{}, false
	}
	if accepted, _ := rec.Fields["accepted"].(string); accepted == what {
		return Result{}, false
	}
	title, _ := rec.Fields["title"].(string)
	if s.asking(rec.ID) {
		return Result{Text: title + " has not been accepted to send on its own yet, so nothing was sent; the question is waiting for the owner"}, true
	}
	var q Question
	if rec.Fields["kind"] == "mqtt" {
		topic, _ := rec.Fields["topic"].(string)
		payload, _ := rec.Fields["payload"].(string)
		q = Question{"Let your “" + title + "” button signal a device on its own?",
			fmt.Sprintf("Each time it is set off, without asking again, it tells %s: %q. The device may act on it straight away.", topic, trim.Clip(payload, 120)),
			"Yes, let it", "No, don't"}
	} else {
		to, _ := rec.Fields["url"].(string)
		body, _ := rec.Fields["body"].(string)
		detail := "Each time it is set off, without asking again, it contacts " + to + "."
		if body != "" {
			detail = fmt.Sprintf("Each time it is set off, without asking again, it sends this to %s: %q.", to, trim.Clip(body, 160))
		}
		if strings.Contains(to+body, "{{") {
			detail += " Words in {{ }} are filled in from what set it off."
		}
		q = Question{"Let your “" + title + "” button send to " + hostOf(to) + " on its own?",
			detail + " Once sent, it can't be unsent.", "Yes, let it", "No, don't"}
	}
	// Waiting for a yes is not a failure, as a command's is not: a hook
	// that set it off is told it waits (waiting_for), and the schedule
	// logs no error each hour it is still waiting.
	r := s.Chat.Ask(q, map[string]any{"tool": "accept_action", "id": rec.ID})
	if !r.IsErr {
		r.Text = title + " has not been accepted to send on its own yet, so nothing was sent; " + r.Text
	}
	return r, true
}

// asking says whether the owner is already asked to accept this action,
// so one set off every hour asks once, not every hour.
func (s *Service) asking(id string) bool {
	if _, ok := s.Store.Types().Get(records.ProposalType); !ok {
		return false
	}
	recs, _ := s.Store.List(records.ProposalType, store.ListOptions{})
	for _, p := range recs {
		call, _ := p.Fields["action"].(map[string]any)
		if p.Fields["state"] == "pending" && call["tool"] == "accept_action" && call["id"] == id {
			return true
		}
	}
	return false
}

func hostOf(to string) string {
	if u, err := url.Parse(to); err == nil && u.Host != "" {
		return u.Host
	}
	return to
}
