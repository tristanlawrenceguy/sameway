package server_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/testkit"
)

// TestMain points this computer's folders (the known list, copies,
// deleted workspaces, pasted keys) at a temp folder for the whole run, so
// an app a test opens with app.Load, which takes the person's own, never
// reads or writes theirs. newApp gives each test a folder of its own.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "sameway-server-test-")
	if err != nil {
		panic(err)
	}
	os.Setenv("SAMEWAY_KNOWN", filepath.Join(dir, "known.json"))
	os.Setenv("SAMEWAY_KEYS", filepath.Join(dir, "keys.json"))
	code := m.Run()
	testkit.Remove()
	os.RemoveAll(dir)
	os.Exit(code)
}
