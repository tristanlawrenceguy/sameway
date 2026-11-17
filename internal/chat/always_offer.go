package chat

import (
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// The owner's own answers say what they would say yes to for good: once
// they have said yes to the same kind of request from the same person a
// few times, they are asked, once, whether to say it for good (Always,
// request_always.go), on that person's page. Their press decides; no
// model adds a standing yes.

// alwaysAfter is how many yeses to one kind from one person come before
// the owner is asked.
const alwaysAfter = 3

// offerAlways asks the owner, once, whether the kind of request they just
// said yes to, from whoever asked, is yes for good.
func (s *Service) offerAlways(p *store.Record) {
	v, ok := RequestAsker(p)
	action, _ := p.Fields["action"].(map[string]any)
	tool, _ := action["tool"].(string)
	typeName, _ := action["type"].(string)
	if p.Fields["by_action"] != ByRequest || !ok || v.Person == "" || typeName == "" {
		return
	}
	person, err := s.Store.Get(records.PersonType, v.Person)
	key := AlwaysKey(tool, typeName)
	if err != nil || hasAlways(person, key) {
		return
	}
	recs, _ := s.Store.List(records.ProposalType, store.ListOptions{})
	yeses := 0
	for _, r := range recs {
		a, _ := r.Fields["action"].(map[string]any)
		if w, ok := RequestAsker(r); ok && w.Person == v.Person && RequestState(r) == "yes" && a["tool"] == tool && a["type"] == typeName {
			yeses++
		}
	}
	name, _ := person.Fields["name"].(string)
	verb := "made"
	if tool == "update_record" {
		verb = "changed"
	}
	ask := fmt.Sprintf("Always say yes to %s %s by %s?", schema.Plural(typeName), verb, name)
	for _, r := range recs {
		if r.Fields["summary"] == ask {
			return // asked once
		}
	}
	if yeses < alwaysAfter {
		return
	}
	list, _ := person.Fields["always"].([]any)
	s.ask(question{ask,
		fmt.Sprintf("You have said yes to %d of their requests for this. From now on they would be %s without asking you, logged as theirs; taking it off their page asks you again.", yeses, verb),
		"Yes, always", "Keep asking me"},
		map[string]any{"tool": "update_record", "type": records.PersonType, "id": person.ID, "fields": map[string]any{"always": append(append([]any{}, list...), key)}})
}
