package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// Publishing is what anyone on the internet may read, with no login, at
// the workspace's own address, through Tailscale Funnel: the tabs and the
// content types the owner asked to publish (workspace.yaml publish:), and
// nothing else. The internet reaches only Public, never the workspace
// itself: a request for anything not published is not found, nothing is
// ever written, and the pages carry no controls, no conversation and no
// log. On the tailnet, the same address is the whole workspace as before.

// Published is what is public just now.
type Published struct {
	Tabs  map[string]string // canvas id -> its title; "" is Home
	Types map[string]bool
	AI    bool
}

// Any says whether anything is published.
func (p Published) Any() bool { return len(p.Tabs) > 0 || len(p.Types) > 0 }

// Published reads what is public from workspace.yaml, as it is now.
func (s *Server) Published() Published {
	cfg := s.app.Workspace.Config.Publish
	out := Published{Tabs: map[string]string{}, Types: map[string]bool{}, AI: cfg.AI == "on"}
	for _, name := range splitList(cfg.Types) {
		if t, ok := s.app.Types.Get(strings.ToLower(name)); ok && !t.Internal {
			out.Types[t.Name] = true
		}
	}
	for _, name := range splitList(cfg.Tabs) {
		for _, c := range s.app.Chat.Canvases() {
			if strings.EqualFold(c.Name, name) || c.ID == name {
				out.Tabs[c.ID] = c.Name
			}
		}
	}
	return out
}

func splitList(list string) []string {
	var out []string
	for _, p := range strings.Split(list, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// isPublic says whether a request came from the internet.
func isPublic(r *http.Request) bool { return chat.VisitorOf(r.Context()).Access == chat.Public }

// Public is the workspace as the internet has it.
func (s *Server) Public(mcp http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(chat.WithVisitor(r.Context(), chat.Visitor{Access: chat.Public}))
		pub := s.Published()
		switch {
		case r.URL.Path == "/mcp" && pub.AI && mcp != nil:
			mcp.ServeHTTP(w, r)
			return
		case r.Method != http.MethodGet && r.Method != http.MethodHead:
			http.Error(w, "this is a published page: it can be read, not changed", http.StatusMethodNotAllowed)
			return
		case strings.HasPrefix(r.URL.Path, "/design/"):
			s.ServeHTTP(w, r)
			return
		}
		if s.publicAllows(pub, r.URL.Path) {
			s.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == "/" {
			s.publicIndex(w, r, pub)
			return
		}
		s.page(w, r, "Not published", template.HTML(`<p>This page is not published.</p>`), pageOptions{Status: http.StatusNotFound})
	})
}

// publicAllows says whether a path is a published page.
func (s *Server) publicAllows(pub Published, path string) bool {
	if path == "/" {
		_, home := pub.Tabs[""]
		return home
	}
	if id, ok := strings.CutPrefix(path, "/c/"); ok {
		_, yes := pub.Tabs[id]
		return yes
	}
	if rest, ok := strings.CutPrefix(path, "/t/"); ok {
		typ, _, _ := strings.Cut(rest, "/")
		return pub.Types[typ]
	}
	return false
}

// publicIndex is the front page of what is published.
func (s *Server) publicIndex(w http.ResponseWriter, r *http.Request, pub Published) {
	if !pub.Any() {
		s.page(w, r, "Nothing published", template.HTML(`<p>Nothing here is published.</p>`), pageOptions{Status: http.StatusNotFound})
		return
	}
	var b strings.Builder
	b.WriteString(`<ul class="sw-plain sw-stack">`)
	for id, title := range pub.Tabs {
		fmt.Fprintf(&b, `<li>%s</li>`, s.navLink(chat.CanvasPath(id), title, false))
	}
	for _, t := range s.app.Types.Types {
		if pub.Types[t.Name] {
			fmt.Fprintf(&b, `<li>%s</li>`, s.navLink("/t/"+t.Name, plural(t.Name), false))
		}
	}
	b.WriteString(`</ul>`)
	s.page(w, r, s.app.Workspace.Config.Name, template.HTML(b.String()), pageOptions{})
}

// listedFor is whether a type is in the navigation of the one asking: for
// the internet, only what is published.
func (s *Server) listedFor(r *http.Request, t *schema.Type) bool {
	if isPublic(r) {
		return s.Published().Types[t.Name]
	}
	return s.listed(t)
}

// controlsFor is how a page shows its controls: none for the internet,
// which may only read.
func (s *Server) controlsFor(r *http.Request) string {
	if isPublic(r) {
		return "none"
	}
	return s.app.Workspace.Config.UI.Controls
}
