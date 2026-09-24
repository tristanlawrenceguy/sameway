package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// workspacesDelete removes this workspace, once confirmed, handing off to a
// replacement that a fleet (sameway open) started or can start.
func (s *Server) workspacesDelete(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	cur := s.app.Workspace
	if strings.TrimSpace(r.PostForm.Get("confirm")) != cur.Config.Name {
		s.showWorkspaces(w, r, "To delete this workspace, type its name exactly: "+cur.Config.Name)
		return
	}
	if s.fleet == nil {
		s.showWorkspaces(w, r, "This server was not opened by sameway open, so it has no way to hand off to another workspace; delete this one from sameway open instead.")
		return
	}
	next := ""
	for _, o := range s.others() {
		next = o.Dir
		if o.Running {
			break
		}
	}
	var err error
	if next == "" {
		if next, err = s.blank("New workspace"); err != nil {
			s.showWorkspaces(w, r, "Could not make a replacement workspace for after the deletion: "+err.Error())
			return
		}
	}
	url, err := s.start(next)
	if err != nil {
		s.showWorkspaces(w, r, "Could not start the replacement workspace after deleting this one: "+err.Error())
		return
	}
	s.app.Close()
	// Into the trash, not gone: the Workspaces page can put it back.
	trashed, err := workspace.Trash(cur.Dir, cur.Config.Name)
	if err != nil {
		s.showWorkspaces(w, r, err.Error())
		return
	}
	workspace.Forget(cur.Dir)

	ws, _ := workspace.Load(next)
	nextName := ""
	if ws != nil {
		nextName = ws.Config.Name
	}
	var b strings.Builder
	b.WriteString(string(s.component("alert", map[string]any{"kind": "success", "title": cur.Config.Name + " deleted",
		"message": "It is in Sameway's trash with everything that was in its folder (" + trashed.Now + "). Restore it from Workspaces."})))
	b.WriteString(fmt.Sprintf(`<p class="sw-muted">Opening <a href="%s">%s</a>…</p>`, template.HTMLEscapeString(url), template.HTMLEscapeString(nextName)))

	w.Header().Set("Refresh", fmt.Sprintf("3; url=%s", template.HTMLEscapeString(url)))
	s.page(w, r, "Workspace deleted", template.HTML(b.String()), pageOptions{})
	if s.fleet != nil && s.fleet.Exit != nil {
		s.fleet.Exit()
	}
}
