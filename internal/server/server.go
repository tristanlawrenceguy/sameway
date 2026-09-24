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
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Server serves one workspace.
type Server struct {
	app   *app.App
	css   []byte
	js    []byte
	mux   *http.ServeMux
	turns turns
	fleet *Fleet
	// model is whether the assistant can reach its model, last looked;
	// see connect.go.
	model modelState
	// notify tells a ring beyond the page; see ring.go.
	notify func(title, text, url string)
}

// New builds the handler for an app.
func New(a *app.App) *Server {
	s := &Server{app: a, css: []byte(a.Registry.CSS()), js: []byte(a.Registry.JS()), mux: http.NewServeMux()}
	s.routes()
	a.Chat.Look = s.lookFor
	// Wrap the mux so unmatched routes get our HTML 404 page.
	s.mux = s.wrapNotFound(s.mux)
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

func (s *Server) routes() {
	m := s.mux
	m.HandleFunc("GET /{$}", s.canvasPage)
	m.HandleFunc("GET /c/{canvas}", s.canvasPage)
	m.HandleFunc("GET /chat", s.chatPage)
	m.HandleFunc("GET /canvas/{id}", s.focusPage)
	m.HandleFunc("POST /chat", s.chatSend)
	m.HandleFunc("POST /chat/stream", s.chatStream)
	m.HandleFunc("POST /chat/stop", s.chatStop)
	m.HandleFunc("GET /chat/live", s.chatLive)
	m.HandleFunc("POST /model/use", s.modelUse)
	m.HandleFunc("POST /t/{type}/add", s.addRecord)
	m.HandleFunc("POST /model/check", s.modelCheck)
	m.HandleFunc("POST /chat/clear", s.chatClear)
	m.HandleFunc("POST /chat/new", s.chatNew)
	m.HandleFunc("POST /chat/open", s.chatOpen)
	m.HandleFunc("POST /chat/delete", s.chatDelete)
	m.HandleFunc("POST /canvas/{id}/place", s.blockPlace)
	m.HandleFunc("POST /proposal/{id}/accept", s.proposalAccept)
	m.HandleFunc("POST /proposal/{id}/dismiss", s.proposalDismiss)
	m.HandleFunc("POST /activity/{id}/undo", s.undo)
	m.HandleFunc("POST /act/{id}", s.act)
	m.HandleFunc("POST /canvas/{id}/props", s.blockProps)
	m.HandleFunc("POST /canvas/{id}/delete", s.canvasDelete)
	m.HandleFunc("GET /activity", s.activityPage)
	m.HandleFunc("POST /clock/set", s.clockSet)
	m.HandleFunc("POST /habit/{id}/log", s.habitLog)
	m.HandleFunc("POST /clock/{id}/done", s.clockDone)
	m.HandleFunc("POST /clock/{id}/snooze", s.clockSnooze)
	m.HandleFunc("GET /clock/stream", s.clockStream)
	m.HandleFunc("GET /workspaces", s.workspacesPage)
	m.HandleFunc("POST /workspaces/start", s.workspacesStart)
	m.HandleFunc("GET /workspaces/new", s.workspacesNewPage)
	m.HandleFunc("POST /workspaces/new", s.workspacesNew)
	m.HandleFunc("GET /workspaces/copy", s.workspacesCopyPage)
	m.HandleFunc("POST /workspaces/copy", s.workspacesCopy)
	m.HandleFunc("GET /workspaces/delete", s.workspacesDeletePage)
	m.HandleFunc("POST /workspaces/delete", s.workspacesDelete)

	m.HandleFunc("GET /search", s.searchPage)
	m.HandleFunc("GET /design", s.designPage)
	m.HandleFunc("GET /design/sameway.css", s.stylesheet)
	m.HandleFunc("GET /design/sameway.js", s.script)
	m.HandleFunc("GET /design/base/{file}", s.baseFile)

	m.HandleFunc("GET /t/{type}", s.listPage)
	m.HandleFunc("GET /t/{type}/import", s.importPage)
	m.HandleFunc("POST /t/{type}/import", s.importUpload)
	m.HandleFunc("POST /t/{type}/import/{file}/run", s.importRun)
	m.HandleFunc("GET /t/{type}/{id}", s.detailPage)
	m.HandleFunc("POST /t/{type}/{id}/delete", s.deleteForm)
	m.HandleFunc("POST /t/{type}/{id}/props", s.recordProps)
	m.HandleFunc("POST /t/file/upload", s.upload)
	m.HandleFunc("GET /files/{id}", s.serveFile)

	m.HandleFunc("GET /api/describe", s.apiDescribe)
	m.HandleFunc("GET /api/search", s.apiSearch)
	m.HandleFunc("GET /api/describe/{part}", s.apiDescribePart)
	m.HandleFunc("GET /api/describe/{part}/{name}", s.apiDescribePart)
	m.HandleFunc("GET /api/look", s.apiLook)
	m.HandleFunc("POST /api/look", s.apiLook)
	m.HandleFunc("POST /api/prose", s.apiProse)
	m.HandleFunc("POST /api/types", s.apiAddType)
	m.HandleFunc("POST /api/types/{type}/fields", s.apiAddField)
	m.HandleFunc("POST /api/act/{id}", s.apiAct)
	m.HandleFunc("POST /hook/{token}", s.hook)
	m.HandleFunc("POST /api/chat", s.apiChat)
	m.HandleFunc("POST /api/chat/clear", s.apiChatClear)
	m.HandleFunc("POST /api/file/upload", s.apiFileUpload)
	m.HandleFunc("POST /api/import/{type}", s.apiImport)
	m.HandleFunc("GET /api/{type}", s.apiList)
	m.HandleFunc("POST /api/{type}", s.apiCreate)
	m.HandleFunc("GET /api/{type}/{id}", s.apiGet)
	m.HandleFunc("PUT /api/{type}/{id}", s.apiUpdate)
	m.HandleFunc("PATCH /api/{type}/{id}", s.apiUpdate)
	m.HandleFunc("DELETE /api/{type}/{id}", s.apiDelete)
	m.HandleFunc("/api/", s.apiNotFound)
}

func (s *Server) stylesheet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(s.css)
}

// page renders a body inside the site layout with the shared navigation.
func (s *Server) page(w http.ResponseWriter, r *http.Request, title string, body template.HTML, opts pageOptions) {
	p := render.Page{
		Site:         s.app.Workspace.Config.Name,
		Title:        title,
		Controls:     s.app.Workspace.Config.UI.Controls,
		Pace:         s.app.Workspace.Config.UI.Pace,
		Body:         body,
		JSONURL:      opts.JSONURL,
		Focus:        opts.Focus,
		FocusLabel:   opts.FocusLabel,
		QuietTitle:   opts.QuietTitle,
		Kicker:       opts.Kicker,
		Lede:         opts.Lede,
		Dot:          opts.Dot,
		Shell:        opts.Shell,
		Left:         opts.Left,
		Right:        opts.Right,
		Header:       opts.Header,
		Footer:       opts.Footer,
		ExtraScripts: opts.ExtraScripts,
		Outcome:      s.told(w, r),
	}
	for _, t := range s.app.Types.Types {
		if t.Internal || !s.listed(t) {
			continue
		}
		href := "/t/" + t.Name
		p.Nav = append(p.Nav, render.NavItem{HTML: s.navLink(href, plural(t.Name), strings.HasPrefix(r.URL.Path, href)), Dot: s.dotOf(t.Name)})
	}
	more := []struct{ href, label string }{{"/chat", "Chat"}, {"/activity", "Activity"}, {"/workspaces", "Workspaces"}}
	if s.app.Workspace.Config.UI.Developer == "shown" {
		more = append(more, struct{ href, label string }{"/design", "Design system"})
	}
	for _, l := range more {
		p.More = append(p.More, s.navLink(l.href, l.label, r.URL.Path == l.href))
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

// notFoundPage renders a full HTML 404 page with heading, title, and
// navigation so that anyone landing on an unknown path still gets the same
// accessible layout as every other page.
func (s *Server) notFoundPage(w http.ResponseWriter, r *http.Request) {
	body := template.HTML(`<p>The page you are looking for does not exist.</p>
` + s.navLink("/", "Home", false))
	s.page(w, r, "404 · Page not found", body, pageOptions{Status: http.StatusNotFound})
}

// detailPageExtraScripts are additional <script> tags rendered in the head on
// content-type record detail pages, enabling inline editing via 08-edit.js.
var detailPageExtraScripts = []template.HTML{
	`<script defer src="/design/base/08-edit.js"></script>`,
}

type pageOptions struct {
	QuietTitle   bool
	Shell        string
	Kicker       template.HTML
	Lede         template.HTML
	Dot          int
	Left         template.HTML
	Right        template.HTML
	Header       template.HTML
	Footer       template.HTML
	JSONURL      string
	Focus        string
	FocusLabel   string
	Status       int
	ExtraScripts []template.HTML
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
	return schema.Plural(strings.ReplaceAll(name, "_", " "))
}

// listed says whether a list belongs in the sidebar: one with something in
// it, or one the person made themselves, which they will want to see even
// before its first record. An empty list the system provides, such as
// files in a workspace with no files, is not in the way. ui.lists: all
// shows every one.
func (s *Server) listed(t *schema.Type) bool {
	if s.app.Workspace.Config.UI.Lists == "all" || !t.Provided {
		return true
	}
	n, err := s.app.Store.Count(t.Name)
	return err != nil || n > 0
}
