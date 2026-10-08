package server

import (
	stdcmp "cmp"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// The workspaces page: a new blank workspace, a copy of this one, and
// deleting this one. How workspaces are found, started and reached is in fleet.go.

// sibling is where a new workspace goes: beside this one, in a folder
// named after it.
func (s *Server) sibling(name string) (string, error) {
	slug := strings.Trim(regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(name), "-"), "-")
	if slug == "" {
		return "", errors.New("a workspace needs a name")
	}
	dir := filepath.Join(filepath.Dir(s.app.Workspace.Dir), slug)
	if _, err := os.Stat(dir); err == nil {
		return "", fmt.Errorf("there is already a folder at %s", dir)
	}
	return dir, nil
}

// blank makes a new workspace named name beside this one: the starter
// shape, no content, and the model this workspace talks to.
func (s *Server) blank(name string) (string, error) {
	dir, err := s.sibling(name)
	if err != nil {
		return "", err
	}
	if err := workspace.Init(dir, examples.FS, examples.StarterRoot, false); err != nil {
		return "", err
	}
	for _, f := range []string{"data.db", "data.db-wal", "data.db-shm"} {
		os.Remove(filepath.Join(dir, f))
	}
	return dir, s.settle(dir, name)
}

// settle names a new workspace and gives it the model this one uses.
func (s *Server) settle(dir, name string) error {
	ws, err := workspace.Load(dir)
	if err != nil {
		return err
	}
	if err := ws.Set("name", name); err != nil {
		return err
	}
	cfg := s.app.Workspace.Config.LLM
	for key, value := range map[string]string{"llm.provider": cfg.Provider, "llm.model": cfg.Model, "llm.base_url": cfg.BaseURL, "llm.api_key_env": cfg.APIKeyEnv} {
		if value != "" {
			ws.Set(key, value)
		}
	}
	return workspace.Remember(dir, "")
}

// copy makes a workspace beside this one with everything this one has:
// its shape, its content, its files and a whole copy of its database.
func (s *Server) copy(name string) (string, error) {
	dir, err := s.sibling(name)
	if err != nil {
		return "", err
	}
	if err := s.app.CopyTo(dir, false); err != nil {
		return "", err
	}
	return dir, s.settle(dir, name)
}

func (s *Server) workspacesPage(w http.ResponseWriter, r *http.Request) {
	s.showWorkspaces(w, r, "")
}

// showWorkspaces is the page: this workspace, the others, and what can
// be done, with a problem said at the top when there is one.
func (s *Server) showWorkspaces(w http.ResponseWriter, r *http.Request, problem string) {
	// An agent gets the problem the page would open with (agents.go).
	if problem != "" && pageAction(r) {
		not := map[string]string{"start": "Not started", "new": "Not created", "copy": "Not copied", "delete": "Not deleted"}[strings.TrimPrefix(r.URL.Path, "/workspaces/")]
		tellJSON(w, outcome{Failed: true, Title: stdcmp.Or(not, "Not done"), Text: problem}, "/workspaces")
		return
	}
	cur := s.app.Workspace
	var b strings.Builder
	if problem != "" {
		b.WriteString(string(s.component("alert", map[string]any{"kind": "warning", "message": problem})))
	}
	own := ""
	for _, k := range workspace.KnownWorkspaces() {
		if sameDir(k.Dir, cur.Dir) && k.Addr != "" {
			own = "http://" + k.Addr + "/"
		}
	}
	meta := cur.Dir
	if own != "" {
		meta += " · " + own
	}
	b.WriteString(`<section class="sw-stack" aria-labelledby="ws-this"><h2 id="ws-this">This workspace</h2>`)
	b.WriteString(string(s.component("card", map[string]any{"title": cur.Config.Name, "meta": meta, "level": 3})))
	b.WriteString(s.takeEverything(r)) // see export_docs.go
	b.WriteString(s.atLoginSection())  // at_login.go
	b.WriteString(s.quitSection())     // quit.go
	b.WriteString(`</section>`)
	b.WriteString(s.phoneSection(r)) // phone_lan_page.go

	others := s.others()
	b.WriteString(`<section class="sw-stack" aria-labelledby="ws-others"><h2 id="ws-others">Other workspaces</h2>`)
	if len(others) == 0 {
		// This one is a workspace, so not "none yet"; and the way to make
		// another is a link to its field, not a direction to look below.
		b.WriteString(string(s.component("empty", map[string]any{"message": "Only this one so far.", "action": map[string]any{"href": "#new-name", "label": "Create a workspace"}})))
	} else {
		b.WriteString(`<ul class="sw-plain sw-rows">`)
		for _, o := range others {
			fmt.Fprintf(&b, `<li class="sw-row sw-ws"><div class="sw-ws__who"><span class="sw-row__title">%s</span><span class="sw-muted sw-small">%s</span></div>`, template.HTMLEscapeString(o.Name), template.HTMLEscapeString(o.Dir))
			if o.Running {
				fmt.Fprintf(&b, `<a class="sw-button sw-button--secondary sw-pressable" href="%s" target="_blank" rel="opener">Open<span class="sw-visually-hidden"> %s</span> (new tab)</a>`, template.HTMLEscapeString(o.URL), template.HTMLEscapeString(o.Name))
			} else {
				fmt.Fprintf(&b, `<form method="post" action="/workspaces/start"><input type="hidden" name="dir" value="%s"><button type="submit" class="sw-button sw-button--secondary sw-pressable">Start<span class="sw-visually-hidden"> %s</span></button></form>`, template.HTMLEscapeString(o.Dir), template.HTMLEscapeString(o.Name))
			}
			b.WriteString(`</li>`)
		}
		b.WriteString(`</ul>`)
	}
	b.WriteString(`</section>`)

	b.WriteString(s.cloudSection()) // cloud_backup.go
	b.WriteString(s.trashSection())
	b.WriteString(s.newSection() + s.copySection() + s.deleteSection())

	s.page(w, r, "Workspaces", template.HTML(b.String()), pageOptions{Lede: "Each workspace runs on its own, so several can be open at once.", JSONURL: "/api/workspaces"})
}

func (s *Server) workspacesStart(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	dir := r.PostForm.Get("dir")
	known := false
	for _, k := range workspace.KnownWorkspaces() {
		known = known || sameDir(k.Dir, dir)
	}
	if !known {
		s.showWorkspaces(w, r, "That is not a workspace this machine knows.")
		return
	}
	url, err := s.start(dir)
	if err != nil {
		s.showWorkspaces(w, r, err.Error())
		return
	}
	http.Redirect(w, r, url, http.StatusSeeOther)
}

func (s *Server) workspacesNewPage(w http.ResponseWriter, r *http.Request) {
	s.page(w, r, "New workspace", template.HTML(s.newSection()), pageOptions{})
}

func (s *Server) workspacesCopyPage(w http.ResponseWriter, r *http.Request) {
	s.page(w, r, "Copy workspace", template.HTML(s.copySection()), pageOptions{})
}

func (s *Server) workspacesDeletePage(w http.ResponseWriter, r *http.Request) {
	s.page(w, r, "Delete workspace", template.HTML(s.deleteSection()), pageOptions{})
}

func (s *Server) workspacesNew(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	s.makeAndOpen(w, r, false, strings.TrimSpace(r.PostForm.Get("name")), "new")
}

func (s *Server) workspacesCopy(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	s.makeAndOpen(w, r, true, strings.TrimSpace(r.PostForm.Get("name")), "copy")
}

func (s *Server) makeAndOpen(w http.ResponseWriter, r *http.Request, copy bool, name string, op string) {
	dir, _, notStarted, err := s.makeWorkspace(name, copy) // home.go
	if err != nil {
		s.showWorkspaces(w, r, err.Error())
		return
	}
	if notStarted != nil {
		s.showWorkspaceCreated(w, r, name, dir, op, notStarted)
		return
	}
	http.Redirect(w, r, "/workspaces", http.StatusSeeOther)
}

// showWorkspaceCreated renders the workspaces page with a success alert
// confirming what was created/copied and a warning about auto-start failure.
func (s *Server) showWorkspaceCreated(w http.ResponseWriter, r *http.Request, name, dir string, op string, err error) {
	var b strings.Builder
	opPast := "created"
	if op == "copy" {
		opPast = "copied"
	}
	if pageAction(r) {
		tellJSON(w, outcome{Title: "Workspace " + name + " " + opPast, Text: "It is at " + dir + ", but could not be started from here: " + err.Error()}, "/workspaces")
		return
	}
	b.WriteString(string(s.component("alert", map[string]any{
		"kind":    "success",
		"message": "Workspace " + name + " " + opPast + ".",
	})))
	b.WriteString(string(s.component("alert", map[string]any{
		"kind":    "warning",
		"title":   "Auto-start failed",
		"message": "The workspace is at " + dir + ", but could not be started from here: " + err.Error(),
	})))
	s.page(w, r, "Workspaces", template.HTML(b.String()), pageOptions{})
}

// newSection is the form that makes a blank workspace, on the workspaces page and on its own.
func (s *Server) newSection() string {
	var b strings.Builder
	b.WriteString(`<section class="sw-stack" aria-labelledby="ws-example"><h2 id="ws-example">Try it with an example</h2><p class="sw-muted">A workspace of its own, beside this one, with a week of tasks, events, a habit and notes in it, to see what Sameway does before yours is full. Delete it here when you are done.</p><form method="post" action="/workspaces/example">` +
		string(s.component("button", map[string]any{"label": "Open an example", "type": "submit", "variant": "secondary"})) + `</form></section>`)
	b.WriteString(`<section class="sw-stack" aria-labelledby="ws-new"><h2 id="ws-new">New workspace</h2><p class="sw-muted">A blank workspace beside this one, with the same model, in a window of its own.</p><form method="post" action="/workspaces/new" class="sw-stack">`)
	b.WriteString(string(s.component("text-field", map[string]any{"label": "Name", "name": "name", "required": true, "id": "new-name", "autocomplete": "off"})))
	b.WriteString(string(s.component("button", map[string]any{"label": "Create", "type": "submit"})))
	b.WriteString(`</form></section>`)
	return b.String()
}

// copySection is the form that copies this workspace, on the workspaces page and on its own.
func (s *Server) copySection() string {
	var b strings.Builder
	b.WriteString(`<section class="sw-stack" aria-labelledby="ws-copy"><h2 id="ws-copy">Copy this workspace</h2><p class="sw-muted">Everything here, as a second workspace beside this one.</p><form method="post" action="/workspaces/copy" class="sw-stack">`)
	b.WriteString(string(s.component("text-field", map[string]any{"label": "Name for the copy", "name": "name", "required": true, "id": "copy-name", "autocomplete": "off", "value": s.app.Workspace.Config.Name + " copy"})))
	b.WriteString(string(s.component("button", map[string]any{"label": "Copy", "type": "submit", "variant": "secondary"})))
	b.WriteString(`</form></section>`)
	return b.String()
}

// deleteSection is the form that deletes this workspace, on the workspaces page and on its own.
func (s *Server) deleteSection() string {
	var b strings.Builder
	b.WriteString(`<section class="sw-stack" aria-labelledby="ws-delete"><h2 id="ws-delete">Delete this workspace</h2><p class="sw-muted">The folder, with everything in it, moves to Sameway's trash, and this server stops. You can restore it from this page. Type the name to be sure.</p><form method="post" action="/workspaces/delete" class="sw-stack">`)
	b.WriteString(string(s.component("text-field", map[string]any{"label": "Type " + s.app.Workspace.Config.Name + " to delete it", "name": "confirm", "required": true, "autocomplete": "off", "spellcheck": false, "id": "delete-confirm"})))
	b.WriteString(string(s.component("button", map[string]any{"label": "Delete", "context": "workspace", "type": "submit", "variant": "danger"})))
	b.WriteString(`</form></section>`)
	return b.String()
}
