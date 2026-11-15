package chat

import (
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// The owner can answer a kind of request once for good: Always on one of
// someone's requests puts that kind of change on their person page (Made
// without asking, as "create_record note"), and from then on the same
// kind from them is made without asking, logged as theirs. Only the
// owner's press adds one, never a model; taking it off their page asks
// again. Someone with no person page, a visitor from the internet, is
// always asked.

// AlwaysKey is a kind of change as a person page keeps it.
func AlwaysKey(tool, typeName string) string { return tool + " " + typeName }

// always says whether the one this service speaks for may have this kind
// of change made without asking.
func (s *Service) always(tool, typeName string) bool {
	if s.who.Person == "" || typeName == "" {
		return false
	}
	p, err := s.Store.Get(records.PersonType, s.who.Person)
	if err != nil {
		return false
	}
	return hasAlways(p, AlwaysKey(tool, typeName))
}

func hasAlways(p *store.Record, key string) bool {
	list, _ := p.Fields["always"].([]any)
	for _, k := range list {
		if k == key {
			return true
		}
	}
	return false
}

// madeAlways makes what someone asked for under the owner's standing yes:
// run as the owner would, logged as theirs.
func (s *Service) madeAlways(call llm.ToolCall, what string) toolResult {
	o := *s
	o.who = records.Visitor{Access: records.Owner}
	r := o.runOp(call, false)
	if r.isErr {
		return r
	}
	if r.change != nil {
		r.change.Activity = s.byPerson(*r.change) // theirs, by name, as a change they made
		r.change = nil                            // logged here, as theirs
	}
	r.text = fmt.Sprintf("Made, without asking: the owner said yes for good to this kind of change from them (%s). ", what) + r.text
	return r
}

// Always answers a request yes, and the same kind from the same person
// from then on, on their person page. It is the owner's press alone.
func (s *Service) Always(id string) error {
	p, err := s.Store.Get(records.ProposalType, id)
	if err != nil {
		return err
	}
	v, ok := RequestAsker(p)
	action, _ := p.Fields["action"].(map[string]any)
	tool, _ := action["tool"].(string)
	typeName, _ := action["type"].(string)
	if !ok || v.Person == "" || typeName == "" {
		return fmt.Errorf("only a request from someone with a person page can be answered for good")
	}
	if err := s.Accept(id); err != nil {
		return err
	}
	person, err := s.Store.Get(records.PersonType, v.Person)
	if err != nil {
		return err
	}
	key := AlwaysKey(tool, typeName)
	if hasAlways(person, key) {
		return nil
	}
	list, _ := person.Fields["always"].([]any)
	_, err = s.Apply(records.Op{Type: records.PersonType, ID: person.ID, After: map[string]any{"always": append(list, key)}})
	return err
}
