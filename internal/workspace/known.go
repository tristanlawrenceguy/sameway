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
// one that is running, or start one that is not. SAMEWAY_KNOWN names a
// different file, for tests and for keeping the list somewhere else.

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
func KnownWorkspaces() []Known {
	var out []Known
	for _, k := range readKnown() {
		if _, err := os.Stat(filepath.Join(k.Dir, ConfigFile)); err == nil {
			out = append(out, k)
		}
	}
	return out
}

// Remember puts a workspace at the front of the list, with the address
// its server listens on when one is given; an empty addr keeps the one
// already known.
func Remember(dir, addr string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	entry := Known{Dir: abs, Addr: addr}
	rest := []Known{}
	for _, k := range readKnown() {
		if k.Dir == abs {
			if addr == "" {
				entry.Addr = k.Addr
			}
			continue
		}
		rest = append(rest, k)
	}
	return writeKnown(append([]Known{entry}, rest...))
}

// Forget drops a workspace from the list.
func Forget(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	kept := []Known{}
	for _, k := range readKnown() {
		if k.Dir != abs {
			kept = append(kept, k)
		}
	}
	return writeKnown(kept)
}

func readKnown() []Known {
	raw, err := os.ReadFile(KnownPath())
	if err != nil {
		return nil
	}
	var out []Known
	if json.Unmarshal(raw, &out) != nil {
		return nil
	}
	return out
}

func writeKnown(list []Known) error {
	path := KnownPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	raw, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}
