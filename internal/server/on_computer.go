package server

import (
	"errors"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// A file on this computer that is not in Sameway can be found from a
// phone and brought in, so it opens there: a search also looks by name in
// the owner's Documents, Downloads and Desktop, and each one found can be
// kept as a file. Only those folders: nothing else on the computer is
// offered, or let in when asked for by its path.

// HomeFolders are the folders a search looks in on this computer: the
// owner's Documents, Downloads and Desktop, those that are there.
func HomeFolders() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	var out []string
	for _, name := range []string{"Documents", "Downloads", "Desktop"} {
		dir := filepath.Join(home, name)
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			out = append(out, dir)
		}
	}
	return out
}

// FindOnComputer has a search also look in these folders on this
// computer. None, the default, looks nowhere: a test does not read the
// home folder of whoever runs it.
func (s *Server) FindOnComputer(dirs ...string) { s.computer = dirs }

// A disk of years is not read whole for one search: so many are shown, a
// look stops after a while, and folders deeper than this are passed over.
const (
	foundMost  = 20
	lookAtMost = 2 * time.Second
	deepest    = 8
)

type foundFile struct {
	path    string
	changed time.Time
	size    int64
}

// findOnComputer is the files whose names have every word, newest first.
// Hidden files and folders are passed over. The folders are read a level
// at a time, all of them together: a file at the top of Downloads is
// found before a deep tree in Documents uses up the time.
func findOnComputer(dirs []string, q string) []foundFile {
	words := strings.Fields(strings.ToLower(q))
	if len(words) == 0 {
		return nil
	}
	type folder struct {
		path  string
		depth int
	}
	var todo []folder
	for _, dir := range dirs {
		todo = append(todo, folder{dir, 0})
	}
	until := time.Now().Add(lookAtMost) // wall clock: the time a look really takes
	var out []foundFile
	for len(todo) > 0 && len(out) < foundMost && time.Now().Before(until) { // wall clock: as above
		in := todo[0]
		todo = todo[1:]
		entries, _ := os.ReadDir(in.path)
		for _, d := range entries {
			path := filepath.Join(in.path, d.Name())
			switch {
			case strings.HasPrefix(d.Name(), "."):
			case d.IsDir():
				if in.depth < deepest && !hiddenEntry(d) {
					todo = append(todo, folder{path, in.depth + 1})
				}
			case d.Type().IsRegular() && hasWords(strings.ToLower(d.Name()), words) && len(out) < foundMost:
				if fi, err := d.Info(); err == nil && !markedHidden(fi) {
					out = append(out, foundFile{path, fi.ModTime(), fi.Size()})
				}
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].changed.After(out[j].changed) })
	return out
}

// hiddenEntry is whether a file or folder is marked hidden.
func hiddenEntry(d fs.DirEntry) bool {
	fi, err := d.Info()
	return err != nil || markedHidden(fi)
}

// hidden is whether a path in a folder passes through anything hidden,
// by its name or as marked, which a search never offers.
func hidden(dir, rel string) bool {
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		dir = filepath.Join(dir, part)
		fi, err := os.Lstat(dir)
		if strings.HasPrefix(part, ".") || err != nil || markedHidden(fi) {
			return true
		}
	}
	return false
}

func hasWords(name string, words []string) bool {
	for _, w := range words {
		if !strings.Contains(name, w) {
			return false
		}
	}
	return true
}

// onComputer is the search page's On this computer, after Sameway's own
// results: the owner's files found by name, each ready to bring in. Only
// the owner's, and nothing when nothing is found.
func (s *Server) onComputer(r *http.Request, q, only string) string {
	if len(s.computer) == 0 || only != "" || !s.chatFor(r).IsOwner() {
		return ""
	}
	found := findOnComputer(s.computer, q)
	if len(found) == 0 {
		return ""
	}
	esc := template.HTMLEscapeString
	var b strings.Builder
	b.WriteString(`<section class="sw-stack" aria-labelledby="on-computer"><h2 id="on-computer">On this computer</h2>`)
	b.WriteString(`<p class="sw-muted">Not in Sameway yet: found by name in your Documents, Downloads and Desktop. Bring one in to open it here and on your phone.</p><ul class="sw-stack">`)
	for _, f := range found {
		name := filepath.Base(f.path)
		b.WriteString(`<li class="sw-stack"><p><strong>` + esc(name) + `</strong><br><span class="sw-muted sw-small">In ` + esc(filepath.Dir(f.path)) + `, changed ` + esc(s.savedOn(f.changed)) + `</span></p>`)
		b.WriteString(string(s.form(ui.Form{Action: "/computer/bring", Hidden: ui.Hidden("path", f.path),
			Button: &ui.Button{Label: "Bring it in", Context: name, Variant: ui.Secondary}})) + `</li>`)
	}
	b.WriteString(`</ul></section>`)
	return b.String()
}

// FindFiles is the assistant's find_on_computer (chat/home_tools.go): the
// same files the search page finds, said in words, each with the search
// page where the person presses Bring it in. Bringing in stays theirs.
func (s *Server) FindFiles(words string) (string, error) {
	if len(s.computer) == 0 {
		return "nothing was looked for: this workspace does not look on this computer. Files in your Documents, Downloads and Desktop are found when Sameway runs on it, opened with sameway open or sameway serve", nil
	}
	found := findOnComputer(s.computer, words)
	if len(found) == 0 {
		return "no file in your Documents, Downloads or Desktop has " + strconv.Quote(words) + " in its name", nil
	}
	link := "/search?q=" + url.QueryEscape(words)
	var b strings.Builder
	b.WriteString("found by name in your Documents, Downloads and Desktop, not in Sameway yet; to have one here the person presses Bring it in beside it at " + link + ":\n")
	for _, f := range found {
		b.WriteString("- " + filepath.Base(f.path) + ", in " + filepath.Dir(f.path) + ", " + sizeWords(f.size) + ", changed " + s.savedOn(f.changed) + "\n")
	}
	return b.String(), nil
}

var errNotOffered = errors.New("only a file in your Documents, Downloads or Desktop can be brought in this way")

// bringIn keeps a file found on this computer, and opens it. The path
// comes from the page, so it is asked again whether it is a file in one
// of the folders, through any link on the way.
func (s *Server) bringIn(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	path := r.PostForm.Get("path")
	if !s.offered(path) {
		s.failed(w, r, "Not brought in", errNotOffered, "/search")
		return
	}
	f, err := os.Open(path)
	if err != nil {
		s.failed(w, r, "Not brought in", err, "/search")
		return
	}
	name := filepath.Base(path)
	rec, stored, err := s.media.KeepFile(s.who(r), f, name, "", "")
	f.Close()
	if err != nil {
		s.failed(w, r, "Not brought in", err, "/search")
		return
	}
	s.media.ReadKept(rec.ID, name, stored, false)
	s.tellAt(w, r, outcome{Title: "Brought in", Text: name + " is in your files."}, "/t/"+records.FileType+"/"+rec.ID)
}

// offered is whether path is a file a search here could have found: a
// regular file inside one of the folders, once links are followed.
func (s *Server) offered(path string) bool {
	if path == "" || !filepath.IsAbs(path) {
		return false
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	if fi, err := os.Lstat(real); err != nil || !fi.Mode().IsRegular() {
		return false
	}
	for _, dir := range s.computer {
		if d, err := filepath.EvalSymlinks(dir); err == nil {
			if rel, err := filepath.Rel(d, real); err == nil && rel != "." && filepath.IsLocal(rel) && !hidden(d, rel) {
				return true
			}
		}
	}
	return false
}
