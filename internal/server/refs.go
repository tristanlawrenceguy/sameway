package server

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

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
	return titleOf(t, rec)
}

// refCell is a ref on a record's page: the target's title as a link to
// its page, with the id kept on the element for the inline editor.
func (s *Server) refCell(f schema.Field, id string) string {
	title := s.refTitle(f, id)
	if _, err := s.app.Store.Get(f.To, id); err != nil {
		return fmt.Sprintf(`<dd data-prop="%s" data-source="%s">%s</dd>`, f.Name, template.HTMLEscapeString(id), template.HTMLEscapeString(title))
	}
	return fmt.Sprintf(`<dd data-prop="%s" data-source="%s"><a class="sw-link" href="/t/%s/%s">%s</a></dd>`, f.Name, template.HTMLEscapeString(id), f.To, id, template.HTMLEscapeString(title))
}

// backlinks is what points at a record, on its page: for every type with
// a ref to this one, the records that hold this id, as the same
// collection a block would show. A project's page lists its tasks by
// itself, and the list leads on to the list page with the same query.
func (s *Server) backlinks(t *schema.Type, rec *store.Record) string {
	var b strings.Builder
	for _, u := range s.app.Types.Types {
		for _, f := range u.Fields {
			if f.Type != "ref" || f.To != t.Name {
				continue
			}
			props := s.resolveCollection(map[string]any{
				"type": u.Name, "where": []string{f.Name + "=" + rec.ID}, "order": "-updated_at", "limit": 50,
				"label": capitalize(plural(u.Name)), "level": 2, "id": "backlinks-" + u.Name + "-" + f.Name,
			})
			b.WriteString(`<div class="sw-backlinks">` + string(s.component(collectionComponent, props)) + `</div>`)
		}
	}
	return b.String()
}
