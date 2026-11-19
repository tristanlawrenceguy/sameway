package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// A message can carry choices for whoever gets it to take: confirm a
// time, take another, say no. Each choice is a change the owner says yes
// to with the message itself, in Sameway's words (requests.go), so the one
// who presses its link only picks which; it runs once, as the owner, and
// the others close. Each is kept as a question already answered for them,
// with its own address, so being chosen is something an action can wait
// for, and the owner is told.

// ByOffer marks a choice offered in a message.
const ByOffer = "offer"

var errUnreachable = errors.New("choices need this Sameway reachable from the internet, so whoever gets the message can press them: publish something, such as the booking link, first; or send it without choices")

// mostChoices is how many a message may carry.
const mostChoices = 4

// A choice is one button: what it says, and the change it makes.
type choice struct {
	Label  string         `json:"label"`
	Tool   string         `json:"tool"`
	Type   string         `json:"type"`
	ID     string         `json:"id,omitempty"`
	Fields map[string]any `json:"fields"`
	what   string
}

// OfferPath is where a choice is taken.
func OfferPath(token string) string { return "/offer/" + token }

// choicesOf reads the choices of a call: each one a change a yes could
// make, said in words.
func (s *Service) choicesOf(list []choice) ([]choice, error) {
	if len(list) > mostChoices {
		return nil, fmt.Errorf("a message carries at most %d choices", mostChoices)
	}
	for i := range list {
		c := &list[i]
		c.Label = trim.Line(c.Label, 40)
		if c.Label == "" || !requestable(c.Tool) {
			return nil, errors.New("each choice needs a label and a change: tool create_record or update_record, with type, fields, and id to change one")
		}
		if c.what = s.requestWords(c.Tool, c.Type, c.ID, c.Fields); c.what == "" {
			return nil, fmt.Errorf("choice %q names nothing that could be made or changed here", c.Label)
		}
		if err := s.requestFits(c.Tool, c.Type, c.ID, c.Fields); err != nil {
			return nil, fmt.Errorf("choice %q could not be made: %v", c.Label, err)
		}
	}
	return list, nil
}

// offer keeps the choices of one message and gives the lines that carry
// them: each label with its address on the owner's published Sameway.
func (s *Service) offer(to string, list []choice) (string, error) {
	address := s.address()
	if address == "" {
		return "", errUnreachable
	}
	group := requestToken()
	var lines []string
	for i, c := range list {
		token := fmt.Sprintf("%s%d", group, i)
		action := map[string]any{"tool": c.Tool, "type": c.Type, "fields": c.Fields}
		if c.ID != "" {
			action["id"] = c.ID
		}
		if _, err := s.Apply(records.Op{Type: records.ProposalType, After: map[string]any{
			"summary": trim.Line(to+" may choose "+c.Label+": "+c.what, 300), "action": action, "state": "offered",
			"by_action": ByOffer, "asked": c.what, "yes": c.Label, "token": token, "detail": group}}); err != nil {
			return "", err
		}
		lines = append(lines, c.Label+": "+address+OfferPath(token))
	}
	return strings.Join(lines, "\n"), nil
}

// address is where the internet reaches this Sameway, or "".
func (s *Service) address() string {
	if s.Address == nil {
		return ""
	}
	return strings.TrimSuffix(s.Address(), "/")
}

// OfferByToken is the choice with that key, if there is one.
func (s *Service) OfferByToken(token string) *store.Record {
	if token == "" {
		return nil
	}
	recs, _ := s.Store.List(records.ProposalType, store.ListOptions{})
	for _, p := range recs {
		if p.Fields["by_action"] == ByOffer && p.Fields["token"] == token {
			return p
		}
	}
	return nil
}

// Take makes the choice with that key, as the owner said yes to, and
// closes the others offered with it. Once one is taken it is said which.
func (s *Service) Take(token string) error {
	answering.Lock()
	defer answering.Unlock()
	p := s.OfferByToken(token)
	if p == nil {
		return errors.New("there is no such choice")
	}
	if p.Fields["state"] != "offered" {
		return errors.New("this was already answered")
	}
	action, _ := p.Fields["action"].(map[string]any)
	tool, _ := action["tool"].(string)
	args, _ := json.Marshal(action)
	result := s.runAgreed(llm.ToolCall{Name: tool, Args: args})
	if result.isErr {
		return errors.New(result.text)
	}
	if result.change != nil {
		records.Record(s.Store, "assistant", *result.change)
	}
	group, _ := p.Fields["detail"].(string)
	recs, _ := s.Store.List(records.ProposalType, store.ListOptions{})
	var ops []records.Op
	for _, o := range recs {
		if o.Fields["by_action"] == ByOffer && o.Fields["detail"] == group && o.Fields["state"] == "offered" && o.ID != p.ID {
			ops = append(ops, records.Op{Type: records.ProposalType, ID: o.ID, After: map[string]any{"state": "dismissed"}})
		}
	}
	done := map[string]any{"state": "accepted"}
	if c := result.change; c != nil && c.ID != "" && c.Component != "" {
		done["made"] = c.Component + "/" + c.ID
	}
	ops = append(ops, records.Op{Type: records.ProposalType, ID: p.ID, After: done})
	records.ApplyOps(s.Store, ops...) // heard, so an action may wait for a choice
	summary, _ := p.Fields["summary"].(string)
	said := "Chosen: " + strings.Replace(summary, " may choose ", " chose ", 1)
	s.Say(said)
	if s.Automate != nil && s.Automate.Tell != nil {
		s.Automate.Tell("They chose", said, "/today")
	}
	return nil
}
