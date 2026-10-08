package server

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// A whole copy of the workspace, once a day, in the cloud folder the
// person already has (workspace.CloudFolders): its database copied safely
// while it runs, its settings, its kinds of record and its files, in one
// zip, the last seven kept. Start from a copy, on Workspaces, brings one
// back: on a new computer, after signing in to the same cloud.

// cloudCopiesKept is how many daily copies the cloud folder keeps.
const cloudCopiesKept = 7

// cloudDir is where this workspace's copies go in a cloud folder.
func (s *Server) cloudDir(folder string) string {
	return filepath.Join(folder, "Sameway copies", safeName(s.app.Workspace.Config.Name))
}

func safeName(name string) string {
	name = strings.Map(func(r rune) rune {
		if strings.ContainsRune(`<>:"/\|?*`, r) || r < 32 {
			return '-'
		}
		return r
	}, strings.TrimSpace(name))
	if name == "" {
		return "Sameway"
	}
	return name
}

// cloudCopies are this workspace's copies in the folder, newest first.
func (s *Server) cloudCopies(folder string) []string {
	found, _ := filepath.Glob(filepath.Join(s.cloudDir(folder), "* ????-??-??.zip"))
	sort.Sort(sort.Reverse(sort.StringSlice(found)))
	return found
}

// KeepCloudCopy puts today's copy in the cloud folder, once a day, while
// ctx lasts.
func (s *Server) KeepCloudCopy(ctx context.Context) {
	go func() {
		tick := time.NewTicker(time.Hour)
		defer tick.Stop()
		for {
			if folder := s.app.Workspace.Config.Backup.Folder; folder != "" {
				if err := s.cloudCopy(folder, time.Now()); err != nil {
					s.app.Store.SetMeta("backup:err", err.Error())
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
		}
	}()
}

// cloudCopy makes the day's copy unless there is one, and lets go of the
// oldest past the seven kept.
func (s *Server) cloudCopy(folder string, now time.Time) error {
	dir := s.cloudDir(folder)
	name := filepath.Join(dir, safeName(s.app.Workspace.Config.Name)+" "+now.Format("2006-01-02")+".zip")
	if _, err := os.Stat(name); err == nil {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	part := name + ".part"
	if err := s.writeCopy(part); err != nil {
		os.Remove(part)
		return err
	}
	if err := os.Rename(part, name); err != nil {
		return err
	}
	s.app.Store.SetMeta("backup:err", "")
	for _, old := range s.cloudCopies(folder)[min(cloudCopiesKept, len(s.cloudCopies(folder))):] {
		os.Remove(old)
	}
	return nil
}

// writeCopy writes the whole workspace as a zip: every file in its folder
// but the live database, and the database as a safe copy.
func (s *Server) writeCopy(path string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	z := zip.NewWriter(out)
	root := s.app.Workspace.Dir
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		if d.IsDir() {
			if rel == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(rel, "data.db") {
			return nil
		}
		return addToZip(z, p, filepath.ToSlash(rel))
	})
	if err != nil {
		return err
	}
	db, err := os.CreateTemp("", "sameway-copy-*.db")
	if err != nil {
		return err
	}
	db.Close()
	defer os.Remove(db.Name())
	os.Remove(db.Name())
	if err := s.app.Store.Backup(db.Name()); err != nil {
		return err
	}
	if err := addToZip(z, db.Name(), "data.db"); err != nil {
		return err
	}
	return z.Close()
}

func addToZip(z *zip.Writer, from, name string) error {
	f, err := os.Open(from)
	if err != nil {
		return err
	}
	defer f.Close()
	w, err := z.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, f)
	return err
}

// cloudSection is A copy in your cloud on Workspaces, for the owner.
func (s *Server) cloudSection() string {
	esc := template.HTMLEscapeString
	var b strings.Builder
	b.WriteString(`<section class="sw-stack" aria-labelledby="ws-cloud"><h2 id="ws-cloud">A copy in your cloud</h2>`)
	if folder := s.app.Workspace.Config.Backup.Folder; folder != "" {
		state := "The first copy is made within the hour."
		if copies := s.cloudCopies(folder); len(copies) > 0 {
			state = fmt.Sprintf("The newest of %d copies is %s.", len(copies), strings.TrimSuffix(filepath.Base(copies[0]), ".zip"))
		}
		if e := s.app.Store.Meta("backup:err"); e != "" {
			state = "The last copy could not be made: " + e
		}
		b.WriteString(`<p>A whole copy of this workspace goes to ` + esc(s.cloudDir(folder)) + ` once a day, the last seven kept. ` + esc(state) + `</p>`)
		b.WriteString(`<form method="post" action="/backup/cloud"><input type="hidden" name="folder" value="">` + string(s.component("button", map[string]any{"label": "Stop putting copies there", "type": "submit", "variant": "secondary"})) + `</form>`)
	} else if clouds := workspace.CloudFolders(); len(clouds) > 0 {
		b.WriteString(`<p>The daily copies stay on this computer. A copy in your cloud folder is kept off it too, and is there on your next computer.</p>`)
		for _, c := range clouds {
			b.WriteString(`<form method="post" action="/backup/cloud"><input type="hidden" name="folder" value="` + esc(c.Dir) + `">` +
				string(s.component("button", map[string]any{"label": "Keep a daily copy in " + c.Name, "type": "submit", "variant": "secondary"})) + `</form>`)
		}
	} else {
		b.WriteString(`<p>The daily copies stay on this computer. With OneDrive, Dropbox, iCloud Drive or Google Drive installed, a copy can go there too.</p>`)
	}
	b.WriteString(`<h3>Start from a copy</h3><form method="post" action="/workspaces/from-copy" enctype="multipart/form-data" class="sw-stack"><label for="from-copy">A copy Sameway made</label><input type="file" id="from-copy" name="file" accept=".zip" required>` +
		string(s.component("button", map[string]any{"label": "Bring it back", "type": "submit", "variant": "secondary"})) + `</form></section>`)
	return b.String()
}

func (s *Server) cloudSet(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	folder := r.PostForm.Get("folder")
	if s.app.Chat.SetSetting == nil {
		s.failed(w, r, "Not changed", errors.New("this workspace has no settings file"), "/workspaces")
		return
	}
	if err := s.app.Chat.SetSetting("backup.folder", folder); err != nil {
		s.failed(w, r, "Not changed", err, "/workspaces")
		return
	}
	if folder == "" {
		s.tell(w, r, outcome{Title: "Copies stay on this computer", Text: "The copies already in the cloud folder are left there."}, "/workspaces#ws-cloud")
		return
	}
	err := s.cloudCopy(folder, time.Now())
	if err != nil {
		s.failed(w, r, "No copy made yet", err, "/workspaces#ws-cloud")
		return
	}
	s.tell(w, r, outcome{Title: "A copy is in your cloud folder", Text: "Another goes there every day, the last seven kept, in " + s.cloudDir(folder) + "."}, "/workspaces#ws-cloud")
}
