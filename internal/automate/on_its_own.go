package automate

import (
	"cmp"
	"fmt"
	"net/http"
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

// Shown is what a question about an action puts in front of the owner:
// what it sends, or the program it runs. The question carries it, so a
// yes runs only what it showed (chat/actions.go), never what the action
// was changed to after.
func Shown(rec *store.Record) string {
	if line, _ := rec.Fields["command"].(string); rec.Fields["kind"] == "command" {
		return strings.TrimSpace(line)
	}
	return Sends(rec)
}

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
	case "watch":
		if text("url") == "" {
			return ""
		}
		return "GET " + text("url") + "\n"
	}
	if text("url") == "" {
		return ""
	}
	return strings.ToUpper(cmp.Or(text("method"), "POST")) + " " + text("url") + "\n" + text("body")
}

// Asks is the question an action that reaches outside puts to the owner:
// before the assistant presses it now, or (own) before it may run on its
// own from then on. It says where it reaches and what it sends, from the
// action itself, never in a model's words.
func Asks(rec *store.Record, own bool) Question {
	text := func(k string) string { v, _ := rec.Fields[k].(string); return v }
	title := "“" + text("title") + "”"
	var now, verb, does, yes, no string
	switch text("kind") {
	case "command":
		now, yes, no = "Run a program on this computer?", "Run it", "Don't run it"
		does = "runs " + strings.TrimSpace(text("command")) + " as you, with access to your files. What it does can't be undone from Sameway."
	case "watch":
		now, verb, yes, no = "Read "+hostOf(text("url"))+" for you?", "read "+hostOf(text("url")), "Read it", "Don't"
		does = fmt.Sprintf("reads %s, as anyone visiting would, and tells you when it shows: %s. Nothing of yours is sent.", text("url"), trim.Clip(text("watch_for"), 160))
	case "mqtt":
		now, verb, yes, no = "Send a signal to one of your devices?", "signal a device", "Send it", "Don't send"
		does = fmt.Sprintf("tells %s: %q. The device may act on it straight away.", text("topic"), trim.Clip(text("payload"), 120))
	default:
		now, verb, yes, no = "Send something from Sameway to "+hostOf(text("url"))+"?", "send to "+hostOf(text("url")), "Send it", "Don't send"
		does = "contacts " + text("url") + "."
		if text("body") != "" && text("method") != http.MethodGet {
			does = fmt.Sprintf("sends this to %s: %q.", text("url"), trim.Clip(text("body"), 160))
		}
		does += " Once sent, it can't be unsent."
	}
	if !own {
		return Question{now, "The assistant wants to press your " + title + " button now. It " + does, yes, no}
	}
	if strings.Contains(does, "{{") {
		does += " Words in {{ }} are filled in from what set it off."
	}
	return Question{"Let your " + title + " button " + verb + " on its own?", "Each time it is set off, without asking again, it " + does, "Yes, let it", "No, don't"}
}

// unaccepted asks the owner to accept what an action sends on its own,
// once, and says so; asked is false when it may run. Waiting for a yes is
// not a failure, as a command's is not: a hook that set it off is told it
// waits (waiting_for), and the schedule logs no error each hour.
func (s *Service) unaccepted(rec *store.Record) (Result, bool) {
	what := Sends(rec)
	if what == "" || rec.Fields["accepted"] == what {
		return Result{}, false
	}
	title, _ := rec.Fields["title"].(string)
	waits := title + " has not been accepted to send on its own yet, so nothing was sent; "
	if s.asking("accept_action", rec.ID, "") {
		return Result{Text: waits + "the question is waiting for the owner"}, true
	}
	r := s.Chat.Ask(Asks(rec, true), map[string]any{"tool": "accept_action", "id": rec.ID, "shown": what})
	if !r.IsErr {
		r.Text = waits + r.Text
	}
	return r, true
}

// asking says whether the owner is already asked about this action (and
// record, for one held), so one set off every hour asks once, not every
// hour.
func (s *Service) asking(tool, id, record string) bool {
	if _, ok := s.Store.Types().Get(records.ProposalType); !ok {
		return false
	}
	recs, _ := s.Store.List(records.ProposalType, store.ListOptions{})
	for _, p := range recs {
		call, _ := p.Fields["action"].(map[string]any)
		if p.Fields["state"] == "pending" && call["tool"] == tool && call["id"] == id && (record == "" || call["record"] == record) {
			return true
		}
	}
	return false
}

// hostOf is the name of the place an address points at, as a person
// knows it: example.com for https://example.com/hooks/1.
func hostOf(to string) string {
	if u, err := url.Parse(strings.TrimSpace(to)); err == nil && u.Host != "" {
		return u.Hostname()
	}
	return to
}
