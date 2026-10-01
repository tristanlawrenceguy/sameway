package app

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// CopyTo makes dir a workspace with everything this one has: its shape,
// its content, its files and a whole copy of its database. With link, a
// file is linked rather than copied where the disk allows (Sameway never
// writes over a file it keeps, so a link is as good as a copy), and
// content/, which is written from the database, is left to be written.
func (a *App) CopyTo(dir string, link bool) error {
	src := a.Workspace.Dir
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
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
		if link && d.IsDir() && rel == "content" {
			return filepath.SkipDir
		}
		target := filepath.Join(dir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if link && os.Link(p, target) == nil {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		return err
	}
	return a.Store.Backup(filepath.Join(dir, "data.db"))
}

// Sandbox is a throwaway copy of the workspace, for a change to be tried
// in: what it answers is what the real one would, and nothing here
// changes. done closes it and takes it away.
func (a *App) Sandbox() (sb *App, done func(), err error) {
	dir, err := os.MkdirTemp("", "sameway-try-")
	if err != nil {
		return nil, nil, err
	}
	gone := func() { os.RemoveAll(dir) }
	if err := a.CopyTo(dir, true); err != nil {
		gone()
		return nil, nil, err
	}
	// Loading names the folder commands run in, for the one program; it
	// stays this workspace's.
	defer func(w string) { chat.Workdir = w }(chat.Workdir)
	sb, err = Load(dir, false)
	if err != nil {
		gone()
		return nil, nil, err
	}
	return sb, func() { sb.Close(); gone() }, nil
}
