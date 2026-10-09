package workspace

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Known workspaces: every workspace this machine has opened, with the
// address its server last listened on, kept in one small file in the
// person's config folder. A workspace can then offer the others: open
// one that is running, or start one that is not.

// Machine is where this computer keeps what is no one workspace's: the
// list of workspaces it has opened (Known), and beside it their daily
// copies (snapshots.go) and the deleted ones (trash.go). A workspace
// carries the one it was opened with, so a test gives it a folder of its
// own and never reads or writes the person's.
type Machine struct {
	Known string // the known list's file
	// Keys is the file pasted keys are kept in; "" is the person's
	// (llm.KeysPath).
	Keys string
}

// ThisMachine is the person's own: the known list in their config
// folder, or the file SAMEWAY_KNOWN names, for keeping it elsewhere.
func ThisMachine() Machine { return Machine{Known: KnownPath()} }

// Known is one workspace this machine has opened.
type Known struct {
	Dir  string `json:"dir"`
	Addr string `json:"addr,omitempty"`
}

// KnownPath is the file the list lives in.
func KnownPath() string {
	if p := os.Getenv("SAMEWAY_KNOWN"); p != "" {
		return p
	}
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "sameway", "workspaces.json")
}

// KnownWorkspaces lists the workspaces this machine has opened that are
// still there, most recently remembered first.
func (m Machine) KnownWorkspaces() []Known {
	var out []Known
	for _, k := range m.readKnown() {
		if _, err := os.Stat(filepath.Join(k.Dir, ConfigFile)); err == nil {
			out = append(out, k)
		}
	}
	return out
}

// Remember puts a workspace at the front of the list, with the address
// its server listens on when one is given; an empty addr keeps the one
// already known.
func (m Machine) Remember(dir, addr string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	entry := Known{Dir: abs, Addr: addr}
	rest := []Known{}
	for _, k := range m.readKnown() {
		if k.Dir == abs {
			if addr == "" {
				entry.Addr = k.Addr
			}
			continue
		}
		rest = append(rest, k)
	}
	return m.writeKnown(append([]Known{entry}, rest...))
}

// Forget drops a workspace from the list.
func (m Machine) Forget(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	kept := []Known{}
	for _, k := range m.readKnown() {
		if k.Dir != abs {
			kept = append(kept, k)
		}
	}
	return m.writeKnown(kept)
}

func (m Machine) readKnown() []Known {
	raw, err := os.ReadFile(m.Known)
	if err != nil {
		return nil
	}
	var out []Known
	if json.Unmarshal(raw, &out) != nil {
		return nil
	}
	return out
}

func (m Machine) writeKnown(list []Known) error {
	path := m.Known
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	raw, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}
