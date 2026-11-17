package chat

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// Everything with one person, in order: the conversations by email with
// them, what they asked for and what the owner answered, and the events
// they are in with how they answered an invitation. It is the record a
// customer list would keep, kept already by what happened, for the owner
// to ask about: what Sam and I have done together, who waits on me, how
// long I take to answer.

var historyOp = Op{Title: "See the history with someone", Traits: Traits{ReadOnly: true, Idempotent: true},
	Access: ForOwner,
	Words:  []string{"history", "with them", "relationship", "last time", "how long", "customer", "client"},
	Doing:  saying("Looking back"),
	Run:    func(s *Service, _ toolArgs, call llm.ToolCall) toolResult { return s.historyWith(call.Args) },
	Tool: llm.Tool{Name: "history",
		Description: "Everything with one person, oldest first: emails with them, what they asked for and your answer, events they are in and their answers. who: their name or email.",
		Schema:      obj(map[string]any{"who": map[string]any{"type": "string"}}, "who")}}

type happened struct {
	at   time.Time
	line string
}

func (s *Service) historyWith(raw json.RawMessage) toolResult {
	var a struct{ Who string }
	json.Unmarshal(raw, &a)
	p := s.PersonByEmail(strings.TrimSpace(a.Who))
	if p == nil {
		p = s.PersonByName(strings.TrimSpace(a.Who))
	}
	address := strings.TrimSpace(a.Who)
	name := address
	if p != nil {
		address, _ = p.Fields["email"].(string)
		name, _ = p.Fields["name"].(string)
	}
	if !strings.Contains(address, "@") && p == nil {
		return fail("say who by their name, as on their person page, or their email")
	}
	var all []happened
	all = append(all, s.emailsWith(address)...)
	all = append(all, s.requestsBy(p, address)...)
	all = append(all, s.eventsWith(p, address)...)
	if len(all) == 0 {
		return toolResult{text: "Nothing yet with " + name + "."}
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].at.Before(all[j].at) })
	loc := s.Now().Location()
	lines := []string{fmt.Sprintf("With %s, oldest first (%s):", name, records.Untrusted)}
	for _, m := range all {
		lines = append(lines, m.at.In(loc).Format("Mon 2 Jan 2006 15:04")+"  "+m.line)
	}
	return toolResult{text: strings.Join(lines, "\n")}
}

// sameAs says whether an address as an email names it is this one.
func sameAs(written, address string) bool {
	a, err := mail.ParseAddress(written)
	return err == nil && address != "" && strings.EqualFold(a.Address, address)
}

// emailsWith are the conversations with an address: every email in a
// thread they wrote in.
func (s *Service) emailsWith(address string) []happened {
	emails, _ := s.Store.List(records.EmailType, store.ListOptions{OrderBy: "created_at"})
	threads := map[string]bool{}
	for _, e := range emails {
		if from, _ := e.Fields["from"].(string); sameAs(from, address) {
			root, _ := e.Fields["thread"].(string)
			if root == "" {
				root = e.ID
			}
			threads[root] = true
		}
	}
	var out []happened
	for _, e := range emails {
		root, _ := e.Fields["thread"].(string)
		if !threads[e.ID] && !threads[root] {
			continue
		}
		who := "They wrote"
		if mine, _ := e.Fields["from_me"].(bool); mine {
			who = "You wrote"
		}
		subject, _ := e.Fields["subject"].(string)
		line := fmt.Sprintf("%s: %q (/t/email/%s)", who, trim.Line(subject, 120), e.ID)
		if turn, _ := e.Fields["turn"].(string); turn != "" {
			line += ", " + turn
		}
		out = append(out, happened{emailWhen(e), line})
	}
	return out
}

// requestsBy are what someone asked for, and what became of it.
func (s *Service) requestsBy(p *store.Record, address string) []happened {
	recs, _ := s.Store.List(records.ProposalType, store.ListOptions{})
	var out []happened
	for _, r := range recs {
		v, ok := RequestAsker(r)
		if r.Fields["by_action"] != ByRequest || !ok || !(p != nil && v.Person == p.ID || strings.Contains(strings.ToLower(v.Name), strings.ToLower(address)) && address != "") {
			continue
		}
		asked, _ := r.Fields["asked"].(string)
		out = append(out, happened{r.CreatedAt, fmt.Sprintf("They asked to %s: %s", trim.Line(asked, 200), map[string]string{"yes": "you said yes", "no": "you said no", "waiting": "it waits for you"}[RequestState(r)])})
		if RequestState(r) != "waiting" {
			out = append(out, happened{r.UpdatedAt, "You answered their request"})
		}
	}
	return out
}

// eventsWith are the events someone is in, and their answers.
func (s *Service) eventsWith(p *store.Record, address string) []happened {
	t, ok := s.Store.Types().Get(records.EventType)
	if !ok {
		return nil
	}
	recs, _ := s.Store.List(records.EventType, store.ListOptions{})
	var out []happened
	for _, e := range recs {
		in := p != nil && slices.Contains(records.StringList(e.Fields["people"]), p.ID)
		answer := ""
		for _, l := range records.StringList(e.Fields["answers"]) {
			if sameAs(answerWho(l), address) {
				in, answer = true, strings.TrimPrefix(l[len(answerWho(l)):], ": ")
			}
		}
		if !in {
			continue
		}
		line := fmt.Sprintf("Event: %q (/t/event/%s)", records.Name(s.Store, t, e), e.ID)
		if answer != "" {
			line += ", their answer: " + answer
		}
		out = append(out, happened{e.CreatedAt, line})
	}
	return out
}

// emailWhen is when an email was sent, or came when it does not say.
func emailWhen(e *store.Record) time.Time {
	if v, _ := e.Fields["received"].(string); v != "" {
		if at, err := time.Parse(time.RFC3339, v); err == nil {
			return at
		}
	}
	return e.CreatedAt
}
