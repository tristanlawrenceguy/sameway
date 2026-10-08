package server

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// Start from a copy: on a new computer, a copy Sameway put in the cloud
// folder (cloud_backup.go) becomes a workspace beside this one, with
// everything it had, and is opened.

// copyDate is the day a copy's name ends with.
var copyDate = regexp.MustCompile(`\s+\d{4}-\d{2}-\d{2}$`)

func (s *Server) fromCopy(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<30)
	f, hdr, err := r.FormFile("file")
	if err != nil {
		s.showWorkspaces(w, r, "Choose the copy: a zip Sameway made, from your cloud folder's Sameway copies.")
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		s.showWorkspaces(w, r, err.Error())
		return
	}
	name := copyDate.ReplaceAllString(strings.TrimSuffix(filepath.Base(hdr.Filename), filepath.Ext(hdr.Filename)), "")
	dir, err := s.sibling(name)
	if err != nil {
		dir, err = s.sibling(name + " (from a copy)")
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

// unpackCopy puts a copy's files in dir; a copy is a zip with a
// workspace.yaml and a data.db, and nothing outside its folder.
func unpackCopy(data []byte, dir string) error {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return errors.New("this is not a copy Sameway made: it is not a zip")
	}
	have := map[string]bool{}
	for _, f := range zr.File {
		have[f.Name] = true
	}
	if !have[workspace.ConfigFile] || !have["data.db"] {
		return errors.New("this zip is not a copy Sameway made: it has no workspace.yaml and data.db; Take everything's zip is brought in with sameway import")
	}
	for _, f := range zr.File {
		target := filepath.Join(dir, filepath.FromSlash(f.Name))
		if !strings.HasPrefix(target, filepath.Clean(dir)+string(filepath.Separator)) {
			return errors.New("this copy has a file outside its folder, and was not brought back")
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0o755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(target)
		if err == nil {
			_, err = io.Copy(out, rc)
			out.Close()
		}
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
