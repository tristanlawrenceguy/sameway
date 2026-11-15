package chat

import (
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// alwaysLabel is what the Always press on a request says, "Always for
// Vi's notes", or "" when the request cannot be answered for good: one
// from someone with no person page, such as a visitor from the internet.
func (s *Service) alwaysLabel(p *store.Record) string {
	if p.Fields["by_action"] != ByRequest {
		return ""
	}
	v, ok := RequestAsker(p)
	action, _ := p.Fields["action"].(map[string]any)
	typeName, _ := action["type"].(string)
	if !ok || v.Person == "" || typeName == "" {
		return ""
	}
	person, err := s.Store.Get(records.PersonType, v.Person)
	if err != nil {
		return ""
	}
	name, _ := person.Fields["name"].(string)
	if action["tool"] == "update_record" {
		return "Always let " + name + " change " + schema.Plural(typeName)
	}
	return "Always for " + name + "'s " + schema.Plural(typeName)
}
