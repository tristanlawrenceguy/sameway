package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// A deleted workspace is in Sameway's trash, not gone: the Workspaces page
// lists what is there, and one press puts it back where it was.

// trashSection lists the deleted workspaces, newest first, each with the
// way to put it back. Nothing when the trash is empty.
func (s *Server) trashSection() string {
	list := workspace.TrashedWorkspaces()
	if len(list) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<section class="sw-stack" aria-labelledby="ws-trash"><h2 id="ws-trash">Recently deleted</h2><p class="sw-muted">Deleted workspaces are kept here, with everything that was in their folder, until you remove them from ` + template.HTMLEscapeString(workspace.TrashDir()) + `.</p><ul class="sw-plain sw-rows">`)
	for _, t := range list {
		fmt.Fprintf(&b, `<li class="sw-row sw-ws"><div class="sw-ws__who"><span class="sw-row__title">%s</span><span class="sw-muted sw-small">was %s · deleted %s</span></div>`,
			template.HTMLEscapeString(t.Name), template.HTMLEscapeString(t.From), template.HTMLEscapeString(t.At.Local().Format("Mon 2 Jan, 15:04")))
		fmt.Fprintf(&b, `<form method="post" action="/workspaces/restore"><input type="hidden" name="now" value="%s">%s</form></li>`,
			template.HTMLEscapeString(t.Now), s.component("button", map[string]any{"label": "Restore", "context": t.Name, "type": "submit", "variant": "secondary"}))
	}
	b.WriteString(`</ul></section>`)
	return b.String()
}

// workspacesRestore puts a deleted workspace back where it was, among the
// known ones, ready to start.
func (s *Server) workspacesRestore(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	t, err := workspace.Untrash(r.PostForm.Get("now"))
	if err != nil {
		s.failed(w, r, "Not restored", err, "/workspaces")
		return
	}
	s.tellAt(w, r, outcome{Title: "Restored", Text: t.Name + " is back at " + t.From + ". Start it from the list."}, "/workspaces")
}
