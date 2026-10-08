package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A file is kept by streaming it to the workspace's files folder, never
// by holding it whole, so a recording or a video of gigabytes is kept as
// easily as a note. Its text is read afterwards, from the file on disk,
// when it is a document small enough to read.

// maxFile bounds one file: a workspace is a person's folder, and
// recordings and videos are large.
const maxFile = 4 << 30

// maxRead is the largest document whose text is read; anything bigger is
// kept as it is.
const maxRead = 64 << 20

// keepFile makes a file's record and streams src into the files folder
// under it, and logs it as added by who: one place for the page's upload,
// the API and the command line, which each logged it their own way (the
// API as a person, though an agent sent it). The record says converting
// until the file is read.
func (s *Server) keepFile(who records.Who, src io.Reader, name, title, description string) (*store.Record, string, error) {
	if _, ok := s.app.Types.Get(FileType); !ok {
		return nil, "", errors.New("this workspace has no file type; run sameway init --force to add it")
	}
	name = filepath.Base(name)
	if name == "" || name == "." {
		name = "upload"
	}
	if title = strings.TrimSpace(title); title == "" {
		title = strings.TrimSuffix(name, filepath.Ext(name))
	}
	fields := map[string]any{"title": title, "name": name, "kind": convert.Kind(name), "status": "converting"}
	if d := strings.TrimSpace(description); d != "" {
		fields["description"] = d
	}
	rec, err := s.app.Store.Create(FileType, fields)
	if err != nil {
		return nil, "", err
	}
	stored := rec.ID + strings.ToLower(filepath.Ext(name))
	dir := s.app.Workspace.FilesDir()
	path := filepath.Join(dir, stored)
	n, err := func() (int64, error) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return 0, err
		}
		f, err := os.Create(path)
		if err != nil {
			return 0, err
		}
		n, err := io.Copy(f, io.LimitReader(src, maxFile+1))
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		return n, err
	}()
	switch {
	case err == nil && n > maxFile:
		err = errors.New("the file must be smaller than 4 GB")
	case err == nil && n == 0:
		err = errors.New("the selected file is empty")
	}
	if err != nil {
		os.Remove(path)
		s.app.Store.Delete(FileType, rec.ID)
		if strings.Contains(err.Error(), "4 GB") || strings.Contains(err.Error(), "empty") {
			return nil, "", err
		}
		return nil, "", fmt.Errorf("could not keep the file: %w", err)
	}
	rec, err = s.app.Store.Update(FileType, rec.ID, map[string]any{"path": stored, "size": n})
	if err != nil {
		return rec, path, err
	}
	records.Record(s.app.Store, who.Actor, records.Change{Action: "added", Component: FileType, ID: rec.ID, Detail: title,
		Href: "/t/" + FileType + "/" + rec.ID, By: who.By, Via: who.Via, ByLogin: who.ByLogin})
	return rec, path, nil
}

// readKept reads a kept file into its record: by the workspace's converter
// when it names one (in the background, or at once when wait says the
// caller cannot stay for it), else here, from the file on disk.
func (s *Server) readKept(id, name, path string, wait bool) {
	if converter := s.app.Workspace.Config.Files.Convert[convert.Ext(name)]; converter != "" {
		if wait {
			s.convertLater(id, converter, name, path)
		} else {
			go s.convertLater(id, converter, name, path)
		}
		return
	}
	kind := convert.Kind(name)
	if kind == "image" || kind == "audio" || kind == "video" {
		s.readNow(id, name, nil)
		s.writeLater(id) // written down here, when this server does that
		return
	}
	if st, err := os.Stat(path); err == nil && st.Size() > maxRead {
		s.app.Store.Update(FileType, id, map[string]any{"status": "ready", "note": "It is kept as it is: its text is read only from files up to 64 MB."})
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		s.app.Store.Update(FileType, id, map[string]any{"status": "failed", "note": err.Error()})
		return
	}
	s.readNow(id, name, data)
	if kind == "captions" {
		s.pairCaptions(id, data)
	}
}

// AddFile keeps a file from src as a person's own, logged as added
// through the command line, and reads it before it returns: the command
// line's way to add a file already on this computer without a browser.
func (s *Server) AddFile(ctx context.Context, src io.Reader, name, title string) (*store.Record, error) {
	rec, path, err := s.keepFile(records.Who{Actor: "human", Via: records.ThroughCLI}, src, name, title, "")
	if err != nil {
		return nil, err
	}
	s.readKept(rec.ID, name, path, true)
	return s.app.Store.Get(FileType, rec.ID)
}
