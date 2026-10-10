package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/render"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// mostNested is how many of a list's records the sidebar names under it;
// past that, a last link says how many there are and leads to them all.
const mostNested = 20

// navList is one list's item in the sidebar. On the list's own page it is
// the current page; on a page inside it, such as one of its records, it
// is where the person is (aria-current=true), so a screen reader does not
// call the list the page open now. A record nested under it (ui.nest)
// that is open takes that mark instead, so it is said once.
func (s *Server) navList(r *http.Request, t *schema.Type) render.NavItem {
	href := "/t/" + t.Name
	sub, here := s.nested(r, t)
	in := !here && strings.HasPrefix(r.URL.Path, href+"/")
	link := s.part(ui.Link{Href: href, Label: schema.Plural(t.Name), Current: r.URL.Path == href, Within: in})
	return render.NavItem{HTML: link, Dot: s.dotOf(t.Name), Sub: sub}
}

// nested is a list's own records, each a link under the list's item in
// the sidebar, when the workspace asks for that list (ui.nest): a habit
// one click away. An archived one is left out, as the list leaves it,
// newest first, as the list shows them. Past mostNested a last link leads to them
// all, so a cut list never passes for the whole. here says one of them
// holds the page open now.
func (s *Server) nested(r *http.Request, t *schema.Type) (out []template.HTML, here bool) {
	if isPublic(r) || !hasKey(s.app.Workspace.Config.UI.Nest, t.Name) {
		return nil, false
	}
	recs, err := s.app.Store.List(t.Name, store.ListOptions{})
	if err != nil {
		return nil, false
	}
	var live []*store.Record
	for _, rec := range recs {
		if on, _ := rec.Fields["archived"].(bool); !on {
			live = append(live, rec)
		}
	}
	for i, rec := range live {
		if i == mostNested {
			all := fmt.Sprintf("See all %d %s", len(live), schema.Words(schema.Plural(t.Name)))
			out = append(out, s.part(ui.Link{Href: "/t/" + t.Name, Label: all}))
			break
		}
		href := "/t/" + t.Name + "/" + rec.ID
		cur, in := r.URL.Path == href, strings.HasPrefix(r.URL.Path, href+"/")
		here = here || cur || in
		out = append(out, s.part(ui.Link{Href: href, Label: trim.Title(s.title(t, rec)), Current: cur, Within: in}))
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
