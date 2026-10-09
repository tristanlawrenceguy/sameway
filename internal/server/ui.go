package server

import (
	"html/template"

	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// part renders a component from its builder in internal/ui, the typed way
// to give the props a page uses most. A component with no builder is
// still rendered from a map with component.
func (s *Server) part(p ui.Part) template.HTML {
	return s.component(p.Component(), p.Props())
}

// form renders a form that does one action: its hidden fields, the page
// and place to come back to, and its button, written once in ui.Form.
func (s *Server) form(f ui.Form) template.HTML {
	return f.HTML(s.part)
}
