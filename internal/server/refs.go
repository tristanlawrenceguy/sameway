package server

import (
	"fmt"
	"html/template"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
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
		return fmt.Sprintf(`<dd data-prop="%s" data-source="%s">%s</dd>`, f.Name, template.HTMLEscapeString(id), template.HTMLEscapeString(title))
	}
	return fmt.Sprintf(`<dd data-prop="%s" data-source="%s"><a class="sw-link" href="/t/%s/%s">%s</a></dd>`, f.Name, template.HTMLEscapeString(id), f.To, id, template.HTMLEscapeString(title))
}
