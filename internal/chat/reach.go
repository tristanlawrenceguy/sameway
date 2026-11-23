package chat

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// The assistant working for someone has their permissions, never the
// owner's: it reaches the kinds of record they may, and does what their
// access lets them. The kinds that are the owner's alone (schema Owners:
// what others' Sameways asked, the log, the assistant's own questions)
// are out of reach for anyone else, as their pages are, and whoever may
// only look is given only the tools that read. Every tool goes through
// here for its types, so none can forget.

// reachable says whether the one this service speaks for may reach a
// kind of record.
func (s *Service) reachable(t *schema.Type) bool {
	if s.who.Access == records.Public {
		return s.Published != nil && s.Published()[t.Name]
	}
	return s.owner() || !t.Owners
}

// contentTypes are the content types the one this service speaks for may
// reach.
func (s *Service) contentTypes() []*schema.Type {
	var out []*schema.Type
	for _, t := range records.ContentTypes(s.Store) {
		if s.reachable(t) {
			out = append(out, t)
		}
	}
	return out
}

// typeNames are their names, in order.
func (s *Service) typeNames() []string {
	var names []string
	for _, t := range s.contentTypes() {
		names = append(names, t.Name)
	}
	sort.Strings(names)
	return names
}

// contentType is the content type a name means, when the one this
// service speaks for may reach it, or what they have instead.
func (s *Service) contentType(name string) (*schema.Type, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" { // said as a type named "" it told a model nothing
		return nil, errors.New("type is missing: name the content type, one of " + schema.Names(s.typeNames()))
	}
	for _, t := range s.contentTypes() {
		if t.Name == name {
			return t, nil
		}
	}
	return nil, errors.New(records.NoType(name, s.typeNames()))
}

// types is the schema as the one this service speaks for may search it.
func (s *Service) types() *schema.Set {
	if s.owner() {
		return s.Store.Types()
	}
	return s.Store.Types().Only(s.reachable)
}

// mayUse says whether the one this service speaks for may have a tool
// used: the owner's alone for the owner, and only what reads for someone
// who may only look.
func (s *Service) mayUse(name string) bool {
	if s.owner() {
		return true
	}
	op, _ := OpFor(name)
	if op.Access == ForOwner {
		return false
	}
	return s.who.Access == records.Edit || s.who.Access == records.Host || op.Access == ForViewers
}

// forWhom is whom a turn works for, as a model that runs its tools itself
// hands it to their MCP server (llm.Request.For): "" for the owner.
func (s *Service) forWhom() string {
	if s.owner() {
		return ""
	}
	raw, _ := json.Marshal(s.who)
	return string(raw)
}
