package app

import (
	"net/http"
	"os"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// Options are what an app takes from where it runs rather than from its
// workspace: what a test sets to have an app of its own, which shares
// nothing with another app in the same program. The zero value is the
// real program's: this computer's clock, its folders (SAMEWAY_KNOWN and
// SAMEWAY_KEYS still name others), the internet.
type Options struct {
	// MemoryDB keeps the records in memory: tests, read-only commands.
	MemoryDB bool
	// Clock is the time it is for the person, for everything said and
	// worked out by the day (today, overdue, rings at); nil is this
	// computer's clock. A test fixes it, so "tomorrow" is never said a
	// minute before midnight about a minute after.
	Clock func() time.Time
	// Machine is where this computer keeps what is no one workspace's:
	// the known list, copies, deleted workspaces, pasted keys. Zero is
	// the person's own (workspace.ThisMachine); a test gives a temp one.
	Machine workspace.Machine
	// HTTP sends what the assistant's webhook actions send; nil is a
	// client with a ten second limit. A test points it at its own server.
	HTTP *http.Client
	// Ntfy is where phone topics are made; "" is notify.NtfyServer.
	Ntfy string
}

// Load opens the workspace at dir. Pass memoryDB to use an in-memory store
// (tests and read-only commands).
func Load(dir string, memoryDB bool) (*App, error) {
	return Open(dir, Options{MemoryDB: memoryDB})
}

// Options are what the app was opened with, defaults filled in.
func (a *App) Options() Options { return a.opts }

// Now is the time it is for the person (Options.Clock).
func (a *App) Now() time.Time { return a.Records.Now() }

// LLMConfig is the workspace's model settings with what the app adds:
// the workspace and program a command provider's MCP points at, and the
// keys file.
func (a *App) LLMConfig() llm.Config {
	cfg := a.Workspace.Config.LLM
	cfg.Workspace = a.Workspace.Dir
	cfg.Executable, _ = os.Executable()
	cfg.Keys = llm.Keys(a.Workspace.Machine.Keys)
	return cfg
}
