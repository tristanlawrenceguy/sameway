package server

import (
	"context"
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/cloudsync"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// One workspace on a laptop and a desktop, or on two people's computers
// at home, without Tailscale: kept in step through the cloud folder both
// already have (internal/cloudsync). The first computer turns it on under
// Workspaces; the other opens it from the same place, from the copy the
// first left there, and from then on each passes its changes through the
// folder. Which folder is kept with the workspace's database, so the copy
// the second computer starts from knows it too.

// syncMark is the file in a workspace's folder naming what it is kept in
// step as.
const syncMark = ".sameway-sync"

// cloudFolders are the cloud folders on this computer; a test names its own.
var cloudFolders = workspace.CloudFolders

// syncFolder is the folder this workspace is kept in step through, if any.
func (s *Server) syncFolder() string {
	name, id := s.app.Store.Meta("cloudsync:cloud"), s.app.Store.Meta("cloudsync:id")
	if name == "" || id == "" {
		return ""
	}
	for _, c := range cloudFolders() {
		if c.Name == name {
			return filepath.Join(c.Dir, cloudsync.Root, id)
		}
	}
	return ""
}

// joinable is a workspace another computer keeps in step through a cloud
// folder here, which this computer does not have yet.
type joinable struct{ Cloud, ID, Name string }

func (s *Server) joinables() []joinable {
	// A workspace kept in step has a mark in its folder with its id, so
	// one any workspace here already holds is not offered again.
	have := map[string]bool{s.app.Store.Meta("cloudsync:id"): true}
	for _, k := range workspace.KnownWorkspaces() {
		if id, err := os.ReadFile(filepath.Join(k.Dir, syncMark)); err == nil {
			have[strings.TrimSpace(string(id))] = true
		}
	}
	var out []joinable
	for _, c := range cloudFolders() {
		dirs, _ := os.ReadDir(filepath.Join(c.Dir, cloudsync.Root))
		for _, d := range dirs {
			dir := filepath.Join(c.Dir, cloudsync.Root, d.Name())
			if have[d.Name()] || !d.IsDir() {
				continue
			}
			if _, err := os.Stat(filepath.Join(dir, cloudsync.Start)); err != nil {
				continue
			}
			name, _ := os.ReadFile(filepath.Join(dir, "name.txt"))
			out = append(out, joinable{c.Name, d.Name(), strings.TrimSpace(string(name))})
		}
	}
	return out
}

// cloudSyncSection is On your other computers on Workspaces.
func (s *Server) cloudSyncSection() string {
	esc := template.HTMLEscapeString
	var b strings.Builder
	b.WriteString(`<section class="sw-stack" aria-labelledby="ws-sync"><h2 id="ws-sync">On your other computers</h2>`)
	if folder := s.syncFolder(); folder != "" {
		others := len(cloudsync.Others(s.app.Store, folder))
		state := "No other computer has opened it yet: on the other computer, open Workspaces in Sameway and choose it under On your other computers."
		if others > 0 {
			state = "It is open on " + plainCount(others, "other computer") + "; a change on one reaches the others as soon as " + esc(s.app.Store.Meta("cloudsync:cloud")) + " has copied it, usually within a minute."
		}
		if e := s.app.Store.Meta("cloudsync:err"); e != "" {
			state = "Not in step just now: " + e
		}
		b.WriteString(`<p>This workspace is kept in step through ` + esc(s.app.Store.Meta("cloudsync:cloud")) + `. ` + state + `</p>`)
		b.WriteString(`<form method="post" action="/cloud-sync/off">` + string(s.component("button", map[string]any{"label": "Stop keeping it in step", "type": "submit", "variant": "secondary"})) + `</form>`)
	} else if clouds := cloudFolders(); len(clouds) > 0 {
		b.WriteString(`<p>Use this workspace on another computer too, such as a laptop and a desktop, or a partner's computer that shares your cloud folder: each keeps a whole copy, and changes pass between them through the cloud folder.</p>`)
		for _, c := range clouds {
			b.WriteString(`<form method="post" action="/cloud-sync"><input type="hidden" name="cloud" value="` + esc(c.Name) + `">` +
				string(s.component("button", map[string]any{"label": "Keep it in step through " + c.Name, "type": "submit", "variant": "secondary"})) + `</form>`)
		}
	} else {
		b.WriteString(`<p>With OneDrive, Dropbox, iCloud Drive or Google Drive on this computer, this workspace can be kept in step with your other computers through it.</p>`)
	}
	for _, j := range s.joinables() {
		name := j.Name
		if name == "" {
			name = "A workspace"
		}
		b.WriteString(`<form method="post" action="/workspaces/join"><input type="hidden" name="cloud" value="` + esc(j.Cloud) + `"><input type="hidden" name="id" value="` + esc(j.ID) + `">` +
			string(s.component("button", map[string]any{"label": "Open " + name + " from " + j.Cloud, "type": "submit"})) + `</form>`)
	}
	b.WriteString(`</section>`)
	return b.String()
}

func plainCount(n int, what string) string {
	if n == 1 {
		return "1 " + what
	}
	return strconv.Itoa(n) + " " + what + "s"
}

func (s *Server) cloudSyncOn(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	cloud := r.PostForm.Get("cloud")
	var dir string
	for _, c := range cloudFolders() {
		if c.Name == cloud {
			dir = c.Dir
		}
	}
	if dir == "" {
		s.failed(w, r, "Not kept in step", errors.New("that cloud folder is not on this computer"), "/workspaces#ws-sync")
		return
	}
	name := s.app.Workspace.Config.Name
	id := strings.ReplaceAll(strings.ToLower(safeName(name)), " ", "-") + "-" + token(3)
	s.app.Store.SetMeta("cloudsync:cloud", cloud)
	s.app.Store.SetMeta("cloudsync:id", id)
	folder := s.syncFolder()
	os.WriteFile(filepath.Join(s.app.Workspace.Dir, syncMark), []byte(id), 0o644)
	if err := s.syncStart(folder); err != nil {
		s.app.Store.SetMeta("cloudsync:id", "")
		s.failed(w, r, "Not kept in step", err, "/workspaces#ws-sync")
		return
	}
	s.syncRound()
	s.tell(w, r, outcome{Title: "Kept in step through " + cloud, Text: "On your other computer, open Workspaces in Sameway, with " + cloud + " signed in to the same account, and choose Open " + name + " from " + cloud + "."}, "/workspaces#ws-sync")
}

// syncStart leaves a whole copy to start another computer from, and the
// workspace's name.
func (s *Server) syncStart(folder string) error {
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(folder, "name.txt"), []byte(s.app.Workspace.Config.Name), 0o644); err != nil {
		return err
	}
	tmp := filepath.Join(folder, cloudsync.Start+".part")
	if err := s.writeCopy(tmp); err != nil {
		os.Remove(tmp)
		return err
	}
	s.app.Store.SetMeta("cloudsync:started", time.Now().Format(time.RFC3339))
	return os.Rename(tmp, filepath.Join(folder, cloudsync.Start))
}

func (s *Server) cloudSyncOff(w http.ResponseWriter, r *http.Request) {
	s.app.Store.SetMeta("cloudsync:cloud", "")
	s.app.Store.SetMeta("cloudsync:id", "")
	os.Remove(filepath.Join(s.app.Workspace.Dir, syncMark))
	s.tell(w, r, outcome{Title: "No longer kept in step", Text: "This copy stays as it is; the others go on without it."}, "/workspaces#ws-sync")
}

// cloudJoin opens a workspace kept in step elsewhere as a copy of its own
// here, from the copy left in the cloud folder; its own sync brings it up
// to date.
func (s *Server) cloudJoin(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var j *joinable
	for _, c := range s.joinables() {
		if c.Cloud == r.PostForm.Get("cloud") && c.ID == r.PostForm.Get("id") {
			c := c
			j = &c
		}
	}
	if j == nil {
		s.showWorkspaces(w, r, "That workspace is no longer in the cloud folder.")
		return
	}
	var folder string
	for _, c := range cloudFolders() {
		if c.Name == j.Cloud {
			folder = filepath.Join(c.Dir, cloudsync.Root, j.ID)
		}
	}
	data, err := os.ReadFile(filepath.Join(folder, cloudsync.Start))
	if err != nil {
		s.showWorkspaces(w, r, "The copy to start from has not reached this computer yet; try again in a minute.")
		return
	}
	name := j.Name
	if name == "" {
		name = "Workspace"
	}
	dir, err := s.sibling(name)
	if err != nil {
		dir, err = s.sibling(name + " (" + j.Cloud + ")")
	}
	if err != nil {
		s.showWorkspaces(w, r, err.Error())
		return
	}
	if err := unpackCopy(data, dir); err != nil {
		os.RemoveAll(dir)
		s.showWorkspaces(w, r, err.Error())
		return
	}
	workspace.Remember(dir, "")
	if _, notStarted := s.start(dir); notStarted != nil {
		s.showWorkspaceCreated(w, r, name, dir, "copy", notStarted)
		return
	}
	http.Redirect(w, r, "/workspaces", http.StatusSeeOther)
}

// KeepCloudSync passes changes through the cloud folder every fifteen
// seconds, and leaves a fresh copy to start from once a day.
func (s *Server) KeepCloudSync(ctx context.Context) {
	go func() {
		tick := time.NewTicker(15 * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				s.syncRound()
			}
		}
	}()
}

// syncRound reads what the others wrote, then writes what this copy did.
func (s *Server) syncRound() {
	folder := s.syncFolder()
	if folder == "" {
		return
	}
	files := s.app.Workspace.FilesDir()
	n, err := cloudsync.In(s.app.Store, folder, files)
	if n > 0 {
		s.Changed()
	}
	if err == nil {
		_, err = cloudsync.Out(s.app.Store, folder, files)
	}
	said := ""
	if err != nil {
		said = err.Error()
		log.Printf("cloud sync: %v", err)
	}
	s.app.Store.SetMeta("cloudsync:err", said)
	if at, _ := time.Parse(time.RFC3339, s.app.Store.Meta("cloudsync:started")); time.Since(at) > 24*time.Hour {
		s.syncStart(folder)
	}
}
