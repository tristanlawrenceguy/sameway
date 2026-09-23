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

// refCell is a ref on a record's page: the target's title as a link to
// its page, with the id kept on the element for the inline editor.
func (s *Server) refCell(f schema.Field, id string) string {
	title := s.refTitle(f, id)
	if _, err := s.app.Store.Get(f.To, id); err != nil {
		return fmt.Sprintf(`<dd data-prop="%s" data-source="%s"%s>%s</dd>`, f.Name, template.HTMLEscapeString(id), s.choices(f, id), template.HTMLEscapeString(title))
	}
	return fmt.Sprintf(`<dd data-prop="%s" data-source="%s"%s><a class="sw-link" href="/t/%s/%s">%s</a></dd>`, f.Name, template.HTMLEscapeString(id), s.choices(f, id), f.To, id, template.HTMLEscapeString(title))
}

// maxChoices is how many records a ref offers to choose from when edited;
// past it the field is edited by id, as before.
const maxChoices = 500

// choices is what a field with a fixed set of values can be changed to,
// on the element for the inline editor, which offers them as a list: an
// enum's values, or a ref's records by their titles, so nobody is asked
// to type an id. Empty when there is no such set.
func (s *Server) choices(f schema.Field, current string) string {
	type choice struct {
		Value string `json:"value"`
		Label string `json:"label"`
	}
	var list []choice
	switch f.Type {
	case "enum":
		for _, v := range f.Values {
			list = append(list, choice{v, v})
		}
	case "ref":
		t, ok := s.app.Types.Get(f.To)
		if !ok {
			return ""
		}
		recs, err := s.app.Store.List(f.To, store.ListOptions{OrderBy: "created_at", Limit: maxChoices + 1})
		if err != nil || len(recs) > maxChoices {
			return ""
		}
		found := false
		for _, rec := range recs {
			list = append(list, choice{rec.ID, s.title(t, rec)})
			found = found || rec.ID == current
		}
		if current != "" && !found {
			list = append(list, choice{current, s.refTitle(f, current)})
		}
	}
	if len(list) == 0 {
		return ""
	}
	b, _ := json.Marshal(list)
	return fmt.Sprintf(` data-options="%s"`, template.HTMLEscapeString(string(b)))
}
