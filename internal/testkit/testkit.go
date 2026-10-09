// Package testkit is what tests start from, made once per test binary
// and copied for each test rather than made again.
package testkit

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// The starter workspace, initialised and opened once: opening a new one
// makes every table, a commit each, which is most of what a test that
// builds an app spends before it starts. A copy opens with its tables
// there already.
var starter struct {
	once sync.Once
	dir  string
	err  error
}

// Starter is a fresh starter workspace for one test, in a folder of its
// own that goes when the test ends: a copy of the one made for the whole
// binary, as `sameway init` would make it and as an app leaves it once
// opened.
func Starter(t testing.TB) string {
	t.Helper()
	starter.once.Do(makeStarter)
	if starter.err != nil {
		t.Fatal(starter.err)
	}
	dir := t.TempDir()
	if err := copyDir(starter.dir, dir); err != nil {
		t.Fatal(err)
	}
	return dir
}

func makeStarter() {
	root, err := os.MkdirTemp("", "sameway-testkit-")
	if err != nil {
		starter.err = err
		return
	}
	dir := filepath.Join(root, "starter")
	if starter.err = workspace.Init(dir, examples.FS, examples.StarterRoot, false); starter.err != nil {
		return
	}
	m := workspace.Machine{Known: filepath.Join(root, "known.json"), Keys: filepath.Join(root, "keys.json")}
	a, err := app.Open(dir, app.Options{Machine: m})
	if err != nil {
		starter.err = err
		return
	}
	starter.err, starter.dir = a.Close(), dir
}

// Remove deletes what Starter made for the binary; a TestMain calls it
// after the tests have run.
func Remove() {
	if starter.dir != "" {
		os.RemoveAll(filepath.Dir(starter.dir))
	}
}

// copyDir copies the folder from into to, which exists.
func copyDir(from, to string) error {
	return filepath.WalkDir(from, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(from, p)
		target := filepath.Join(to, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(p, target)
	})
}

func copyFile(from, to string) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(to)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
