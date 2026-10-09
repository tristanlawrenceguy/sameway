// Package server is the HTTP layer: HTML pages for people, JSON for agents,
// both generated from the same schema and components.
package server

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/render"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/server/media"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// Server serves one workspace.
type Server struct {
	app     *app.App
	css, js []byte
	mux     *http.ServeMux
	turns   turns
	fleet   *Fleet
	model   modelState                    // whether the assistant can reach its model, last looked; connect.go
	notify  func(title, text, url string) // tells a ring beyond the page; ring.go
	changes atomic.Int64                  // changes arrived from other computers; see sync.go
	present presence                      // who else is here just now; see presence.go
	media   *media.Service                // files, recordings and meetings; internal/server/media
	logged  signal                        // a change logged here, for those waiting on /api/changes; changes.go
}

// New builds the handler for an app.
func New(a *app.App) *Server {
	s := &Server{app: a, css: []byte(a.Registry.CSS()), js: []byte(a.Registry.JS()), mux: http.NewServeMux()}
	s.media = media.New(face{s})
	s.routes()
	s.hooks() // what the rest of the app asks of the pages; see hooks.go
	s.mux = s.wrapNotFound(s.mux)
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r, ok := s.keyed(w, r); ok && s.allowed(w, r) && !s.tried(w, r) { // agent_keys.go, dry_run.go
		s.fresh()
		s.serve(w, r) // a page's actions are an agent's too; see agents.go
	}
}

func (s *Server) stylesheet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(s.css)
}

// page renders a body inside the site layout with the shared navigation.
func (s *Server) page(w http.ResponseWriter, r *http.Request, title string, body template.HTML, opts pageOptions) {
	p := render.Page{
		Site:       s.app.Workspace.Config.Name,
		Title:      title,
		Said:       opts.Said,
		Controls:   s.controlsFor(r),
		Pace:       s.app.Workspace.Config.UI.Pace,
		Hours24:    s.h24(),
		Lang:       s.app.Workspace.Config.UI.Language,
		Text:       s.app.Workspace.Config.UI.Text,
		Spacing:    s.app.Workspace.Config.UI.Spacing,
		Body:       body,
		JSONURL:    opts.JSONURL,
		Focus:      opts.Focus,
		FocusLabel: opts.FocusLabel,
		QuietTitle: opts.QuietTitle,
		Kicker:     opts.Kicker,
		Lede:       opts.Lede,
		Dot:        opts.Dot,
		Shell:      opts.Shell,
		Left:       opts.Left, Right: opts.Right,
		Header: opts.Header, Footer: opts.Footer,
		Present:      s.presentFor(r),
		ExtraScripts: opts.ExtraScripts,
		Outcome:      s.told(w, r) + s.sinceNotice(r) + s.updateNotice(r), // restart.go
		EditControls: s.editControls(body, opts.Left, opts.Right, opts.Header, opts.Footer),
	}
	for _, t := range s.app.Types.Types {
		// Entries are listed on their habit's page, not as a list of their own.
		if t.Internal || t.Name == EntryType || !s.listedFor(r, t) {
			continue
		}
		href := "/t/" + t.Name
		p.Nav = append(p.Nav, render.NavItem{HTML: s.navLink(href, schema.Plural(t.Name), strings.HasPrefix(r.URL.Path, href)), Dot: s.dotOf(t.Name)})
	}
	more := []struct{ href, label string }{{"/search", "Search"}, {"/chat", "Chat"}, {"/activity", "Activity"}, {"/workspaces", "Workspaces"}, {"/help", "Help"}}
	if s.app.Workspace.Config.UI.Developer == "shown" {
		more = append(more, struct{ href, label string }{"/design", "Design system"})
	}
	for _, l := range more {
		href := l.href
		if href == "/search" {
			href = s.searchFrom(r) // from a kind's pages, a search of that kind
		}
		p.More = append(p.More, s.navLink(href, l.label, r.URL.Path == l.href))
	}
	p.Developer = s.app.Workspace.Config.UI.Developer == "shown"
	out, err := render.RenderPage(p)
	if err != nil {
		s.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if opts.Status != 0 {
		w.WriteHeader(opts.Status)
	}
	w.Write(out)
}

// notFoundPage is a 404 in the same accessible layout as every other page.
func (s *Server) notFoundPage(w http.ResponseWriter, r *http.Request) {
	body := template.HTML(`<p>The page you are looking for does not exist.</p>
` + s.navLink("/", "Home", false))
	s.page(w, r, "404 · Page not found", body, pageOptions{Status: http.StatusNotFound})
}

// detailPageExtraScripts are additional <script> tags rendered in the head on
// content-type record detail pages, enabling inline editing via 09-edit.js.
var detailPageExtraScripts = []template.HTML{
	`<script defer src="/design/base/09-edit.js"></script>`,
}

func (s *Server) navLink(href, label string, current bool) template.HTML {
	h, err := s.app.Registry.Render("link", map[string]any{"href": href, "label": label, "current": current})
	if err != nil {
		return template.HTML(template.HTMLEscapeString(label))
	}
	return h
}

// component renders a component or, if that fails, a visible error so a
// broken block never silently disappears. The person reads what could not
// be shown and what to do in plain words; why, in the schema's terms, goes
// to the log.
func (s *Server) component(name string, props map[string]any) template.HTML {
	h, err := s.app.Registry.Render(name, props)
	if err != nil {
		log.Printf("render %s: %v", name, err)
		msg := fmt.Sprintf("This %s could not be shown. Something it was given is missing or not right; ask the assistant to fix it.", strings.ReplaceAll(name, "-", " "))
		h, err = s.app.Registry.Render("alert", map[string]any{"kind": "danger", "message": msg})
		if err != nil {
			return template.HTML("<p>" + template.HTMLEscapeString(msg) + "</p>")
		}
	}
	return h
}

// fail reports an unexpected error as a page.
func (s *Server) fail(w http.ResponseWriter, err error) {
	log.Printf("error: %v", err)
	status := http.StatusInternalServerError
	if errors.Is(err, store.ErrNotFound) {
		status = http.StatusNotFound
	}
	http.Error(w, err.Error(), status)
}

// listed says whether a list belongs in the sidebar: one with something in
// it, or one the person made themselves, which they will want to see even
// before its first record. An empty list the system provides, such as
// files in a workspace with no files, is not in the way. ui.lists: all
// shows every one.
func (s *Server) listed(t *schema.Type) bool {
	if t.Hidden {
		return false
	}
	if s.app.Workspace.Config.UI.Lists == "all" || !t.Provided {
		return true
	}
	n, err := s.app.Store.Count(t.Name)
	return err != nil || n > 0
}

// linkTitle is the name of the record at /t/<type>/<id>, or nothing.
func (s *Server) linkTitle(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/t/"), "/")
	if len(parts) != 2 {
		return ""
	}
	t, ok := s.app.Types.Get(parts[0])
	if !ok {
		return ""
	}
	rec, err := s.app.Store.Get(t.Name, parts[1])
	if err != nil {
		return ""
	}
	return trim.Title(s.title(t, rec))
}
