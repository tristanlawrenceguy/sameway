package chat

import (
	"errors"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// GiveAccess lets someone in by name, for an invite on the home Wi-Fi
// (the server's lan_invite.go): the person with that name, or a new one,
// may now look or edit. It is logged, and Undo takes it back.
func (s *Service) GiveAccess(name, access string) (*store.Record, error) {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return nil, errors.New("say who it is for")
	}
	if access != records.View && access != records.Edit {
		return nil, errors.New("choose whether they may look or edit")
	}
	if _, ok := s.Store.Types().Get(records.PersonType); !ok {
		return nil, errors.New("this workspace has no people")
	}
	p := s.PersonByName(name)
	if p == nil {
		rec, err := s.Store.Create(records.PersonType, map[string]any{"name": name, "access": access})
		if err != nil {
			return nil, err
		}
		records.Record(s.Store, "human", records.Change{Action: "let in", Component: records.PersonType, ID: rec.ID, Detail: name + " to " + access})
		return rec, nil
	}
	before := p.Fields["access"]
	rec, err := s.Store.Update(records.PersonType, p.ID, map[string]any{"access": access})
	if err != nil {
		return nil, err
	}
	if before != access {
		records.Record(s.Store, "human", records.Change{Action: "let in", Component: records.PersonType, ID: p.ID, Detail: name + " to " + access, Before: map[string]any{"access": before}})
	}
	return rec, nil
}

// PersonByName is the person of that name, in any case, if there is one.
func (s *Service) PersonByName(name string) *store.Record {
	people, err := s.Store.List(records.PersonType, store.ListOptions{})
	if err != nil {
		return nil
	}
	for _, p := range people {
		if n, _ := p.Fields["name"].(string); strings.EqualFold(strings.TrimSpace(n), name) {
			return p
		}
	}
	return nil
}
