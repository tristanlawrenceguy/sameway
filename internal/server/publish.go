package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// Publishing is what anyone on the internet may read, with no login, at
// the workspace's own address, through Tailscale Funnel: the tabs and the
// content types the owner asked to publish (workspace.yaml publish:), and
// nothing else. People and AI services have it the same way, as Sameway
// has everything: pages for the one, MCP for the other. The internet reaches only Public, never the workspace
// itself: a request for anything not published is not found, nothing is
// ever written, and the pages carry no controls, no conversation and no
// log. On the tailnet, the same address is the whole workspace as before.

// Published is what is public just now.
type Published struct {
	Tabs  map[string]string // canvas id -> its title; "" is Home
	Types map[string]bool
}

// Any says whether anything is published.
func (p Published) Any() bool { return len(p.Tabs) > 0 || len(p.Types) > 0 }

// Published reads what is public from workspace.yaml, as it is now.
func (s *Server) Published() Published {
	cfg := s.app.Workspace.Config.Publish
	out := Published{Tabs: map[string]string{}, Types: map[string]bool{}}
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
func isPublic(r *http.Request) bool { return records.VisitorOf(r.Context()).Access == records.Public }

// Public is the workspace as the internet has it.
func (s *Server) Public(mcp http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(records.WithVisitor(r.Context(), records.Visitor{Access: records.Public}))
		pub := s.Published()
		switch {
		// What is published is published to people and to AI services
		// the same way: pages for the one, MCP for the other.
		case strings.EqualFold(r.URL.Path, "/mcp") && pub.Any() && mcp != nil:
			mcp.ServeHTTP(w, r)
			return
		case r.Method != http.MethodGet && r.Method != http.MethodHead:
			http.Error(w, "this is a published page: it can be read, not changed", http.StatusMethodNotAllowed)
			return
		case strings.HasPrefix(r.URL.Path, "/design/"):
			s.ServeHTTP(w, r)
			return
		// The file and what is read from it (its captions, its words), but
		// not its sound copied out for writing it down: that writes into the
		// workspace's folder, and is the owner's to ask for.
		case strings.HasPrefix(r.URL.Path, "/files/") && !strings.Contains(r.URL.Path, "/sound") && s.publicFile(pub, strings.TrimPrefix(r.URL.Path, "/files/")):
			s.ServeHTTP(w, r) // a picture on a published page
			return
		}
		// Every page is sent as a reader has it; see public_clean.go.
		s.cleaned(w, r, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case s.publicAllows(pub, r.URL.Path):
				s.ServeHTTP(w, r)
			case r.URL.Path == "/":
				s.publicIndex(w, r, pub)
			default:
				s.page(w, r, "Not published", template.HTML(`<p>This page is not published.</p>`), pageOptions{Status: http.StatusNotFound})
			}
		})
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
	// A published list or record goes out as a file too; nothing else does.
	if rest, ok := strings.CutPrefix(path, "/export/"); ok {
		typ, _, _ := strings.Cut(rest, "/")
		typ, _, _ = strings.Cut(typ, ".")
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
		fmt.Fprintf(&b, `<li>%s</li>`, s.navLink(records.CanvasPath(id), title, false))
	}
	for _, t := range s.app.Types.Types {
		if pub.Types[t.Name] {
			fmt.Fprintf(&b, `<li>%s</li>`, s.navLink("/t/"+t.Name, schema.Plural(t.Name), false))
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
