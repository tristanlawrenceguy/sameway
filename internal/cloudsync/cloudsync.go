// Package cloudsync keeps copies of one workspace on several computers the
// same through a folder their cloud app already keeps in step (OneDrive,
// Dropbox, iCloud Drive, Google Drive): no network setup, no account of
// ours. Each copy writes what it changed, as stamps (store/state.go), in
// files of its own under the folder, and reads the files the others wrote.
// A file is only ever written whole and never changed, so a cloud app that
// copies it late or twice does no harm, and stamps applied twice change
// nothing. Files people added go there too, under files/.
//
//	Sameway sync/<id>/start.zip           a whole copy, to start a new computer from
//	Sameway sync/<id>/<origin>/<clock>.json  one copy's changes, a batch at a time
//	Sameway sync/<id>/files/<name>        the files records point to
package cloudsync

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Root is the folder under a cloud folder that synced workspaces live in.
const Root = "Sameway sync"

// Start is the whole copy a new computer starts from.
const Start = "start.zip"

const (
	metaOrigin = "cloudsync:origin" // the copy the marks below are for
	metaOut    = "cloudsync:out"    // the last of this copy's stamps written
	metaIn     = "cloudsync:in:"    // + origin: the last batch of theirs read
)

// fresh forgets how far this copy wrote when it is a new copy of the
// workspace (a database opened elsewhere takes a new origin): it has
// written nothing yet. What it read stays true: the copy it came from had
// read that much, and holds it.
func fresh(st *store.Store) {
	if st.Meta(metaOrigin) == st.Origin() {
		return
	}
	st.SetMeta(metaOut, "")
	st.SetMeta(metaOrigin, st.Origin())
}

// Out writes this copy's stamps not written yet as one new batch, and the
// files of dir not in the folder yet. It says how many stamps it wrote.
func Out(st *store.Store, folder, filesDir string) (int, error) {
	fresh(st)
	if err := st.Seed(); err != nil {
		return 0, err
	}
	me := st.Origin()
	all, err := st.Since(map[string]string{me: st.Meta(metaOut)})
	if err != nil {
		return 0, err
	}
	var mine []store.Stamp
	for _, s := range all {
		if store.OriginOf(s.Clock) == me {
			mine = append(mine, s)
		}
	}
	if err := copyFiles(filesDir, filepath.Join(folder, "files")); err != nil {
		return 0, err
	}
	if len(mine) == 0 {
		return 0, nil
	}
	last := mine[len(mine)-1].Clock
	for _, s := range mine {
		if s.Clock > last {
			last = s.Clock
		}
	}
	raw, err := json.Marshal(mine)
	if err != nil {
		return 0, err
	}
	if err := writeWhole(filepath.Join(folder, me, last+".json"), raw); err != nil {
		return 0, err
	}
	st.SetMeta(metaOut, last)
	return len(mine), nil
}

// In reads the batches the other copies wrote since this copy last read,
// and the files not here yet. It says how many records changed.
func In(st *store.Store, folder, filesDir string) (int, error) {
	fresh(st)
	if err := copyFiles(filepath.Join(folder, "files"), filesDir); err != nil {
		return 0, err
	}
	dirs, err := os.ReadDir(folder)
	if err != nil {
		return 0, err
	}
	changed := 0
	for _, d := range dirs {
		o := d.Name()
		if !d.IsDir() || o == "files" || o == st.Origin() {
			continue
		}
		names, _ := filepath.Glob(filepath.Join(folder, o, "*.json"))
		sort.Strings(names)
		read := st.Meta(metaIn + o)
		for _, name := range names {
			batch := filepath.Base(name)
			if batch <= read {
				continue
			}
			raw, err := os.ReadFile(name)
			if err != nil {
				break // still arriving; the next round reads it
			}
			var stamps []store.Stamp
			if err := json.Unmarshal(raw, &stamps); err != nil {
				break // half written by the cloud app; read it next time
			}
			n, err := st.Apply(stamps)
			if err != nil {
				return changed, err
			}
			changed += n
			read = batch
			st.SetMeta(metaIn+o, read)
		}
	}
	return changed, nil
}

// Others are the copies that wrote to the folder, besides this one.
func Others(st *store.Store, folder string) []string {
	dirs, _ := os.ReadDir(folder)
	var out []string
	for _, d := range dirs {
		if d.IsDir() && d.Name() != "files" && d.Name() != st.Origin() {
			out = append(out, d.Name())
		}
	}
	return out
}

// writeWhole writes a file under another name first, so no one reads it
// half written.
func writeWhole(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".part"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// copyFiles copies each file of from that to lacks, or has smaller.
func copyFiles(from, to string) error {
	entries, err := os.ReadDir(from)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || strings.HasSuffix(e.Name(), ".part") {
			continue
		}
		src := filepath.Join(from, e.Name())
		info, err := e.Info()
		if err != nil {
			continue
		}
		if have, err := os.Stat(filepath.Join(to, e.Name())); err == nil && have.Size() >= info.Size() {
			continue
		}
		if err := copyFile(src, filepath.Join(to, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst + ".part")
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst + ".part")
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Rename(dst+".part", dst)
}
