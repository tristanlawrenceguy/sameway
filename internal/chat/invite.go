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

// letOut is let_in taking access away, by email or, for someone invited
// on the Wi-Fi with none, by name.
func (s *Service) letOut(a letInArgs, p *store.Record) toolResult {
	if p == nil && a.Email == "" {
		p = s.PersonByName(strings.Join(strings.Fields(a.Name), " "))
	}
	who := a.Email
	if who == "" {
		who = a.Name
	}
	if p == nil || p.Fields["access"] == nil || p.Fields["access"] == "" {
		return toolResult{text: who + " had no access"}
	}
	c, err := s.takeAccess(p)
	if err != nil {
		return fail("could not take their access away: %v", err)
	}
	return toolResult{text: c.Detail + " can now " + verb(""), change: &c}
}

// takeAccess takes a person's access away, as a change to log: their
// Tailscale login and their devices on the Wi-Fi stop opening the
// workspace at once, and Undo puts it back.
func (s *Service) takeAccess(p *store.Record) (records.Change, error) {
	name, _ := p.Fields["name"].(string)
	done, err := records.ApplyOps(s.Store, records.Op{Type: records.PersonType, ID: p.ID, After: map[string]any{"access": ""}})
	if err != nil {
		return records.Change{}, err
	}
	return records.Change{Action: "took access from", Component: records.PersonType, ID: p.ID, Detail: name, Ops: done}, nil
}

// TakeAccess is Take away access on Workspaces: the owner's, logged as
// theirs. It says who it was and the log entry, for Undo.
func (s *Service) TakeAccess(id string) (name, entry string, err error) {
	p, err := s.Store.Get(records.PersonType, id)
	if err != nil {
		return "", "", errors.New("that person is no longer in this workspace")
	}
	name, _ = p.Fields["name"].(string)
	if a, _ := p.Fields["access"].(string); a == "" {
		return name, "", errors.New(name + " could not open it anyway")
	}
	c, err := s.takeAccess(p)
	if err != nil {
		return name, "", err
	}
	return name, records.Record(s.Store, "human", c), nil
}
