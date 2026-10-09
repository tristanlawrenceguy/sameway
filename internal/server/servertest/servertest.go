// Package servertest is what the server's tests and its features' tests
// share: an app served on a fresh starter workspace, requests made to it
// the way a browser or an agent makes them, and what a page says. A
// feature's tests (internal/server/media, ...) use the whole server, as a
// person does, so they test what the person gets.
package servertest

import (
	"os"
	"path/filepath"
	"testing"

	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/testkit"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// Main points this computer's folders (the known list, copies, deleted
// workspaces, pasted keys) at a temp folder for the whole run, so an app a
// test opens with app.Load, which takes the person's own, never reads or
// writes theirs. New gives each test a folder of its own. A test package's
// TestMain calls it.
func Main(m *testing.M) {
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

// New is an app on a fresh copy of the starter workspace (testkit), in a
// temp dir of its own, with no model attached, and the server serving it.
func New(t *testing.T) (*app.App, http.Handler) {
	t.Helper()
	return NewWith(t, app.Options{})
}

// NewWith is New opened with options: a fixed clock, say.
func NewWith(t *testing.T, o app.Options) (*app.App, http.Handler) {
	t.Helper()
	// A machine of its own, so the workspaces this computer has opened, its
	// copies and its pasted keys are never read or written by a test.
	if o.Machine == (workspace.Machine{}) {
		o.Machine = Machine(t)
	}
	a, err := app.Open(testkit.Starter(t), o)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	a.Chat.Provider, a.Chat.ProviderErr = nil, llm.ErrNotConfigured
	return a, server.New(a)
}

// Machine is a computer's folders for one test: an empty known list, no
// keys, no copies (workspace.Machine).
func Machine(t *testing.T) workspace.Machine {
	dir := t.TempDir()
	m := workspace.Machine{Known: filepath.Join(dir, "known.json"), Keys: filepath.Join(dir, "keys.json")}
	os.WriteFile(m.Known, []byte("[]"), 0o644)
	return m
}
