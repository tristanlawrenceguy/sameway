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

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/render"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Server serves one workspace.
type Server struct {
	app *app.App
	css []byte
	js  []byte
	mux *http.ServeMux
}

// New builds the handler for an app.
func New(a *app.App) *Server {
	s := &Server{app: a, css: []byte(a.Registry.CSS()), js: []byte(a.Registry.JS()), mux: http.NewServeMux()}
	s.routes()
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

func (s *Server) routes() {
	m := s.mux
	m.HandleFunc("GET /{$}", s.canvasPage)
	m.HandleFunc("GET /chat", s.chatPage)
	m.HandleFunc("POST /chat", s.chatSend)
	m.HandleFunc("POST /chat/clear", s.chatClear)
	m.HandleFunc("POST /canvas/{id}/delete", s.canvasDelete)
	m.HandleFunc("GET /activity", s.activityPage)
	m.HandleFunc("GET /design", s.designPage)
	m.HandleFunc("GET /design/sameway.css", s.stylesheet)
	m.HandleFunc("GET /design/sameway.js", s.script)

	m.HandleFunc("GET /t/{type}", s.listPage)
	m.HandleFunc("GET /t/{type}/new", s.newPage)
	m.HandleFunc("POST /t/{type}", s.createForm)
	m.HandleFunc("GET /t/{type}/{id}", s.detailPage)
	m.HandleFunc("GET /t/{type}/{id}/edit", s.editPage)
	m.HandleFunc("POST /t/{type}/{id}", s.updateForm)
	m.HandleFunc("POST /t/{type}/{id}/delete", s.deleteForm)

	m.HandleFunc("GET /api/describe", s.apiDescribe)
	m.HandleFunc("POST /api/chat", s.apiChat)
	m.HandleFunc("GET /api/{type}", s.apiList)
	m.HandleFunc("POST /api/{type}", s.apiCreate)
	m.HandleFunc("GET /api/{type}/{id}", s.apiGet)
	m.HandleFunc("PUT /api/{type}/{id}", s.apiUpdate)
	m.HandleFunc("PATCH /api/{type}/{id}", s.apiUpdate)
	m.HandleFunc("DELETE /api/{type}/{id}", s.apiDelete)
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
		Controls:   s.app.Workspace.Config.UI.Controls,
		Body:       body,
		JSONURL:    opts.JSONURL,
		Focus:      opts.Focus,
		FocusLabel: opts.FocusLabel,
	}
	p.Nav = append(p.Nav, s.navLink("/", "Canvas", r.URL.Path == "/"))
	p.Nav = append(p.Nav, s.navLink("/chat", "Chat", r.URL.Path == "/chat"))
	for _, t := range s.app.Types.Types {
		if t.Internal {
			continue
		}
		href := "/t/" + t.Name
		p.Nav = append(p.Nav, s.navLink(href, plural(t.Name), strings.HasPrefix(r.URL.Path, href)))
	}
	p.Nav = append(p.Nav, s.navLink("/activity", "Activity", r.URL.Path == "/activity"))
	p.Nav = append(p.Nav, s.navLink("/design", "Design", r.URL.Path == "/design"))
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

type pageOptions struct {
	JSONURL    string
	Focus      string
	FocusLabel string
	Status     int
}

func (s *Server) navLink(href, label string, current bool) template.HTML {
	h, err := s.app.Registry.Render("link", map[string]any{"href": href, "label": label, "current": current})
	if err != nil {
		return template.HTML(template.HTMLEscapeString(label))
	}
	return h
}

// component renders a component or, if that fails, a visible error so a
// broken block never silently disappears.
func (s *Server) component(name string, props map[string]any) template.HTML {
	h, err := s.app.Registry.Render(name, props)
	if err != nil {
		msg := fmt.Sprintf("Could not render %s: %v", name, err)
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

func plural(name string) string {
	label := strings.ReplaceAll(name, "_", " ")
	if strings.HasSuffix(label, "s") {
		return label
	}
	return label + "s"
}
