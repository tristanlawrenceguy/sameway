package server

import (
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A record can be for someone: a task's For, or any field that points at a
// person. It shows as theirs, in their colour; and when it comes from
// another computer made out for the owner of this one, they hear of it
// the way a reminder rings, once.

// personChip is someone a record is for, with their colour.
func (s *Server) personChip(label, id, name string) string {
	login := ""
	if p, err := s.app.Store.Get(records.PersonType, id); err == nil {
		login, _ = p.Fields["email"].(string)
	}
	props := map[string]any{"name": name, "label": label}
	if c := records.PersonColour(login); c > 0 {
		props["colour"] = c
	}
	return string(s.component("person", props))
}

// forYou hears of a record written because another computer sent it, and
// tells the owner of this one when it is newly theirs.
func (s *Server) forYou(typeName, id string, rec *store.Record) {
	me := strings.ToLower(s.app.Chat.Owner.Login)
	t, ok := s.app.Types.Get(typeName)
	if rec == nil || me == "" || !ok || s.notify == nil {
		return
	}
	for _, f := range t.Fields {
		pid, _ := rec.Fields[f.Name].(string)
		if f.Type != "ref" || f.To != records.PersonType || pid == "" {
			continue
		}
		p, err := s.app.Store.Get(records.PersonType, pid)
		if err != nil {
			continue
		}
		if email, _ := p.Fields["email"].(string); !strings.EqualFold(strings.TrimSpace(email), me) {
			continue
		}
		key := "told:" + typeName + ":" + id + ":" + f.Name
		if s.app.Store.Meta(key) == pid {
			continue
		}
		s.app.Store.SetMeta(key, pid)
		go s.notify("For you: "+s.title(t, rec), "A "+typeName+" was made out for you on another computer.", s.linkTo("/t/"+typeName+"/"+id))
	}
}
