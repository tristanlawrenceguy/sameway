package server

import (
	"encoding/json"
	"fmt"
	"html/template"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A ref on a record's page: the one it points at, named and led to.
//
// What points back the other way is not here. Every connection a record
// has, in both directions and beyond refs, is worked out in
// internal/relate and shown by related.go as a line of counts.

// refTitle is what a ref field shows: the title of the record it points
// at, or a word for one that has gone.
func (s *Server) refTitle(f schema.Field, id string) string {
	if id == "" {
		return ""
	}
	t, ok := s.app.Types.Get(f.To)
	if !ok {
		return id
	}
	rec, err := s.app.Store.Get(f.To, id)
	if err != nil {
		return "a " + f.To + " that is no longer here"
	}
	return s.title(t, rec)
}

// refItem is a ref on a record's page, as one of its fields: the target's
// title as a link to its page, with the id and the other choices kept for
// the inline editor.
func (s *Server) refItem(f schema.Field, id string) map[string]any {
	item := map[string]any{"label": fieldLabel(f), "value": s.refTitle(f, id), "prop": f.Name, "source": id}
	if list := s.choiceList(f, id); len(list) > 0 {
		item["options"] = list
	}
	if _, err := s.app.Store.Get(f.To, id); err == nil {
		item["href"] = "/t/" + f.To + "/" + id
	}
	return item
}

// maxChoices is how many records a ref offers to choose from when edited;
// past it the field is edited by id, as before.
const maxChoices = 500

// choices is what a field with a fixed set of values can be changed to,
// on the element for the inline editor, which offers them as a list: an
// enum's values, or a ref's records by their titles, so nobody is asked
// to type an id. Empty when there is no such set.
func (s *Server) choices(f schema.Field, current string) string {
	list := s.choiceList(f, current)
	if len(list) == 0 {
		return ""
	}
	b, _ := json.Marshal(list)
	return fmt.Sprintf(` data-options="%s"`, template.HTMLEscapeString(string(b)))
}

// choiceList is the same choices as a list of {value, label}, for a
// component to carry; empty when there are none to offer.
func (s *Server) choiceList(f schema.Field, current string) []any {
	// A struct rather than a map, so the value comes first when written.
	type choice struct {
		Value string `json:"value"`
		Label string `json:"label"`
	}
	var list []any
	add := func(value, label string) { list = append(list, choice{value, label}) }
	switch f.Type {
	case "enum":
		for _, v := range f.Values {
			add(v, f.ValueLabel(v))
		}
	case "ref":
		t, ok := s.app.Types.Get(f.To)
		if !ok {
			return nil
		}
		recs, err := s.app.Store.List(f.To, store.ListOptions{OrderBy: "created_at", Limit: maxChoices + 1})
		if err != nil || len(recs) > maxChoices {
			return nil
		}
		found := false
		for _, rec := range recs {
			add(rec.ID, s.title(t, rec))
			found = found || rec.ID == current
		}
		if current != "" && !found {
			add(current, s.refTitle(f, current))
		}
	}
	return list
}
