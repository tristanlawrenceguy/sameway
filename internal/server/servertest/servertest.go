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
	"time"

	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/testhome"
	"github.com/tristanlawrenceguy/sameway/internal/testkit"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// Main points this computer's folders (the known list, copies, deleted
// workspaces, pasted keys) at a temp folder for the whole run (testhome),
// so an app a test opens with app.Load, which takes the person's own,
// never reads or writes theirs. New gives each test a folder of its own.
// A test package's TestMain calls it.
func Main(m *testing.M) {
	code := testhome.Run(m)
	testkit.Remove()
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
	// No model server this computer runs is offered: a test gives its own.
	if o.LocalModels == nil {
		o.LocalModels = []llm.Candidate{}
	}
	a, err := app.Open(testkit.Starter(t), o)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	a.Chat.Provider, a.Chat.ProviderErr = nil, llm.ErrNotConfigured
	return a, server.New(a)
}

// At is New with the app's clock stopped at now, so what the app says by
// the day (today, tomorrow, overdue) is worked out from the moment the
// test works out what it expects from, not from a second clock reading.
func At(t *testing.T, now time.Time) (*app.App, http.Handler) {
	t.Helper()
	return NewWith(t, app.Options{Clock: func() time.Time { return now }})
}

// Tonight is where a test of the day stops the clock: a tenth of a second
// before midnight, when a day read twice is likeliest to be two days. A
// fixed date, since the store stamps what is made by the app's clock too.
var Tonight = time.Date(2026, 3, 11, 23, 59, 59, 9e8, time.Local) // wall clock: a test workspace keeps the computer's zone

// Machine is a computer's folders for one test: an empty known list, no
// keys, no copies (workspace.Machine).
func Machine(t *testing.T) workspace.Machine {
	dir := t.TempDir()
	m := workspace.Machine{Known: filepath.Join(dir, "known.json"), Keys: filepath.Join(dir, "keys.json")}
	os.WriteFile(m.Known, []byte("[]"), 0o644)
	return m
}
