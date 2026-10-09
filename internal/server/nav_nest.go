package server

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// mostNested is how many of a list's records the sidebar names under it;
// the list's own page has the rest.
const mostNested = 20

// nested is a list's own records, each a link under the list's item in
// the sidebar, when the workspace asks for that list (ui.nest): a habit
// one click away. An archived one is left out, as the list leaves it.
// here says one of them is the page open now.
func (s *Server) nested(r *http.Request, t *schema.Type) (out []template.HTML, here bool) {
	if isPublic(r) || !hasKey(s.app.Workspace.Config.UI.Nest, t.Name) {
		return nil, false
	}
	recs, err := s.app.Store.List(t.Name, store.ListOptions{Limit: mostNested})
	if err != nil {
		return nil, false
	}
	for _, rec := range recs {
		if on, _ := rec.Fields["archived"].(bool); on {
			continue
		}
		href := "/t/" + t.Name + "/" + rec.ID
		here = here || r.URL.Path == href
		out = append(out, s.navLink(href, trim.Title(s.title(t, rec)), r.URL.Path == href))
	}
	return out, here
}

// hasKey says whether a comma-separated setting names key.
func hasKey(list, key string) bool {
	for _, k := range strings.Split(list, ",") {
		if strings.TrimSpace(k) == key {
			return true
		}
	}
	return false
}
