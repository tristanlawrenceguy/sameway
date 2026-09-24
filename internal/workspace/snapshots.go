package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// A workspace's database is copied once a day while it runs, and the last
// seven copies are kept: whatever else goes wrong, yesterday is there to
// go back to. The copies live beside Sameway's list of workspaces, not in
// the workspace, so they are never shared with it through git and they
// outlast the folder itself.

// KeepSnapshots is how many daily copies are kept.
const KeepSnapshots = 7

// Snapshot is one copy of a workspace's database.
type Snapshot struct {
	Path string    `json:"path"`
	At   time.Time `json:"at"`
	Size int64     `json:"size"`
}

// SnapshotDir is where this workspace's copies are kept: named for the
// folder, and for its full path, so two folders of one name stay apart.
func (w *Workspace) SnapshotDir() string {
	abs, _ := filepath.Abs(w.Dir)
	sum := sha256.Sum256([]byte(strings.ToLower(abs)))
	return filepath.Join(filepath.Dir(KnownPath()), "snapshots", filepath.Base(abs)+"-"+hex.EncodeToString(sum[:4]))
}

// Snapshots lists the copies, newest first.
func (w *Workspace) Snapshots() []Snapshot {
	entries, _ := os.ReadDir(w.SnapshotDir())
	var out []Snapshot
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".db") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, Snapshot{Path: filepath.Join(w.SnapshotDir(), e.Name()), At: info.ModTime(), Size: info.Size()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].At.After(out[j].At) })
	return out
}

// DailySnapshot makes today's copy with backup, unless there is one, and
// lets go of the oldest past the seven kept. It says the path it made.
func (w *Workspace) DailySnapshot(now time.Time, backup func(path string) error) (string, error) {
	path := filepath.Join(w.SnapshotDir(), "data-"+now.Format("2006-01-02")+".db")
	if _, err := os.Stat(path); err == nil {
		return "", nil
	}
	if err := os.MkdirAll(w.SnapshotDir(), 0o755); err != nil {
		return "", err
	}
	if err := backup(path); err != nil {
		return "", err
	}
	var daily []Snapshot
	for _, s := range w.Snapshots() {
		if strings.HasPrefix(filepath.Base(s.Path), "data-2") {
			daily = append(daily, s)
		}
	}
	for i := KeepSnapshots; i < len(daily); i++ {
		os.Remove(daily[i].Path)
	}
	return path, nil
}

// Restore puts a copy back as the workspace's database. The database as
// it is now is kept first, beside the others, so a restore can itself be
// taken back. The workspace must not be running: its server holds the
// database open.
func (w *Workspace) Restore(from string) (kept string, err error) {
	if _, err := os.Stat(from); err != nil {
		return "", fmt.Errorf("there is no copy at %s", from)
	}
	if err := os.MkdirAll(w.SnapshotDir(), 0o755); err != nil {
		return "", err
	}
	kept = filepath.Join(w.SnapshotDir(), "before-restore-"+time.Now().Format("2006-01-02-150405")+".db")
	if _, err := os.Stat(w.DBPath()); err == nil {
		if err := copyFile(w.DBPath(), kept); err != nil {
			return "", err
		}
	} else {
		kept = ""
	}
	// What the live database was writing is part of it; with the copy put
	// back, the old journal would be applied on top, so it goes.
	for _, extra := range []string{"-wal", "-shm"} {
		if err := os.Remove(w.DBPath() + extra); err != nil && !errors.Is(err, os.ErrNotExist) {
			return kept, fmt.Errorf("the workspace is still running (its database is in use); stop it, then restore again")
		}
	}
	if err := copyFile(from, w.DBPath()); err != nil {
		return kept, fmt.Errorf("could not put the copy back (%v); if the workspace is running, stop it and restore again", err)
	}
	return kept, nil
}

func copyFile(from, to string) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := to + ".part"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, to)
}
