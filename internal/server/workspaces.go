package server

import (
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// The workspaces page and what it does: a new blank workspace, a copy of
// this one, and deleting this one. How workspaces are found, started and
// reached is in fleet.go.

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
	src := s.app.Workspace.Dir
	err = filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if rel == "." {
			return os.MkdirAll(dir, 0o755)
		}
		if strings.HasPrefix(rel, "data.db") {
			return nil
		}
		target := filepath.Join(dir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		return "", err
	}
	if err := s.app.Store.Backup(filepath.Join(dir, "data.db")); err != nil {
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
	cur := s.app.Workspace
	var b strings.Builder
	if problem != "" {
		b.WriteString(string(s.component("alert", map[string]any{"kind": "warning", "title": "That did not go through", "message": problem})))
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
	b.WriteString(`</section>`)

	b.WriteString(`<section class="sw-stack" aria-labelledby="ws-others"><h2 id="ws-others">Other workspaces</h2>`)
	others := s.others()
	if len(others) == 0 {
		b.WriteString(`<p class="sw-empty">None yet. Every workspace opened on this machine appears here.</p>`)
	} else {
		b.WriteString(`<ul class="sw-plain sw-rows">`)
		for _, o := range others {
			fmt.Fprintf(&b, `<li class="sw-row sw-ws"><div class="sw-ws__who"><span class="sw-row__title">%s</span><span class="sw-muted sw-small">%s</span></div>`, template.HTMLEscapeString(o.Name), template.HTMLEscapeString(o.Dir))
			if o.Running {
				fmt.Fprintf(&b, `<a class="sw-button sw-button--secondary sw-pressable" href="%s" target="_blank" rel="opener">Open<span class="sw-visually-hidden"> %s</span></a>`, template.HTMLEscapeString(o.URL), template.HTMLEscapeString(o.Name))
			} else {
				fmt.Fprintf(&b, `<form method="post" action="/workspaces/start"><input type="hidden" name="dir" value="%s"><button type="submit" class="sw-button sw-button--secondary sw-pressable">Start and open<span class="sw-visually-hidden"> %s</span></button></form>`, template.HTMLEscapeString(o.Dir), template.HTMLEscapeString(o.Name))
			}
			b.WriteString(`</li>`)
		}
		b.WriteString(`</ul>`)
	}
	b.WriteString(`</section>`)

	b.WriteString(`<section class="sw-stack" aria-labelledby="ws-new"><h2 id="ws-new">New workspace</h2><p class="sw-muted">A blank workspace beside this one, with the same model, in a window of its own.</p><form method="post" action="/workspaces/new" class="sw-stack">`)
	b.WriteString(string(s.component("text-field", map[string]any{"label": "Name", "name": "name", "required": true, "id": "new-name"})))
	b.WriteString(string(s.component("button", map[string]any{"label": "Create and open", "type": "submit"})))
	b.WriteString(`</form></section>`)

	b.WriteString(`<section class="sw-stack" aria-labelledby="ws-copy"><h2 id="ws-copy">Copy this workspace</h2><p class="sw-muted">Everything here, as a second workspace beside this one.</p><form method="post" action="/workspaces/copy" class="sw-stack">`)
	b.WriteString(string(s.component("text-field", map[string]any{"label": "Name for the copy", "name": "name", "required": true, "id": "copy-name", "value": cur.Config.Name + " copy"})))
	b.WriteString(string(s.component("button", map[string]any{"label": "Copy and open", "type": "submit", "variant": "secondary"})))
	b.WriteString(`</form></section>`)

	b.WriteString(`<section class="sw-stack" aria-labelledby="ws-delete"><h2 id="ws-delete">Delete this workspace</h2><p class="sw-muted">The folder and everything in it go, and this server stops. Type the name to be sure.</p><form method="post" action="/workspaces/delete" class="sw-stack">`)
	b.WriteString(string(s.component("text-field", map[string]any{"label": "Type " + cur.Config.Name + " to delete it", "name": "confirm", "required": true, "id": "delete-confirm"})))
	b.WriteString(string(s.component("button", map[string]any{"label": "Delete this workspace", "type": "submit", "variant": "danger"})))
	b.WriteString(`</form></section>`)

	s.page(w, r, "Workspaces", template.HTML(b.String()), pageOptions{Lede: "Each workspace runs on its own, so several can be open at once."})
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
	var b strings.Builder
	b.WriteString(`<section class="sw-stack" aria-labelledby="ws-new"><h2 id="ws-new">New workspace</h2><p class="sw-muted">A blank workspace beside this one, with the same model, in a window of its own.</p><form method="post" action="/workspaces/new" class="sw-stack">`)
	b.WriteString(string(s.component("text-field", map[string]any{"label": "Name", "name": "name", "required": true, "id": "new-name"})))
	b.WriteString(string(s.component("button", map[string]any{"label": "Create and open", "type": "submit"})))
	b.WriteString(`</form></section>`)

	s.page(w, r, "New workspace", template.HTML(b.String()), pageOptions{})
}

func (s *Server) workspacesCopyPage(w http.ResponseWriter, r *http.Request) {
	var b strings.Builder
	b.WriteString(`<section class="sw-stack" aria-labelledby="ws-copy"><h2 id="ws-copy">Copy this workspace</h2><p class="sw-muted">Everything here, as a second workspace beside this one.</p><form method="post" action="/workspaces/copy" class="sw-stack">`)
	b.WriteString(string(s.component("text-field", map[string]any{"label": "Name for the copy", "name": "name", "required": true, "id": "copy-name", "value": s.app.Workspace.Config.Name + " copy"})))
	b.WriteString(string(s.component("button", map[string]any{"label": "Copy and open", "type": "submit", "variant": "secondary"})))
	b.WriteString(`</form></section>`)

	s.page(w, r, "Copy workspace", template.HTML(b.String()), pageOptions{})
}

func (s *Server) workspacesDeletePage(w http.ResponseWriter, r *http.Request) {
	cur := s.app.Workspace
	var b strings.Builder
	b.WriteString(`<section class="sw-stack" aria-labelledby="ws-delete"><h2 id="ws-delete">Delete this workspace</h2><p class="sw-muted">The folder and everything in it go, and this server stops. Type the name to be sure.</p><form method="post" action="/workspaces/delete" class="sw-stack">`)
	b.WriteString(string(s.component("text-field", map[string]any{"label": "Type " + cur.Config.Name + " to delete it", "name": "confirm", "required": true, "id": "delete-confirm"})))
	b.WriteString(string(s.component("button", map[string]any{"label": "Delete this workspace", "type": "submit", "variant": "danger"})))
	b.WriteString(`</form></section>`)

	s.page(w, r, "Delete workspace", template.HTML(b.String()), pageOptions{})
}

func (s *Server) workspacesNew(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	s.makeAndOpen(w, r, s.blank, strings.TrimSpace(r.PostForm.Get("name")))
}

func (s *Server) workspacesCopy(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	s.makeAndOpen(w, r, s.copy, strings.TrimSpace(r.PostForm.Get("name")))
}

func (s *Server) makeAndOpen(w http.ResponseWriter, r *http.Request, make func(string) (string, error), name string) {
	dir, err := make(name)
	if err != nil {
		s.showWorkspaces(w, r, err.Error())
		return
	}
	_, err = s.start(dir)
	if err != nil {
		s.showWorkspaces(w, r, "The workspace is at "+dir+", but could not be started from here: "+err.Error())
		return
	}
	http.Redirect(w, r, "/workspaces", http.StatusSeeOther)
}

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
	for try := 0; try < 10; try++ {
		if err = os.RemoveAll(cur.Dir); err == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	workspace.Forget(cur.Dir)
	http.Redirect(w, r, url, http.StatusSeeOther)
	if s.fleet != nil && s.fleet.Exit != nil {
		s.fleet.Exit()
	}
}
