// Package testhome keeps tests out of the person's own folders. Sameway
// keeps what belongs to this computer, not to a workspace, in the
// person's config folder (os.UserConfigDir: the known list, snapshots,
// pasted keys, tailnet keys, meeting sign-ins) and puts things in their
// home folder (a first workspace, what starts at sign-in). A test that
// reaches any of it without a folder of its own would read or write the
// developer's real ones, so a test binary that can reach them starts in
// TestMain with Run, which points them all at a temp folder for the run.
// tools/check fails a package whose tests can reach them without it.
package testhome

import (
	"os"
	"path/filepath"
	"testing"
)

// Run points this computer's folders at a temp folder, runs the tests and
// removes it. It returns the code for os.Exit.
func Run(m *testing.M) int {
	dir, err := os.MkdirTemp("", "sameway-home-")
	if err != nil {
		panic(err)
	}
	keepGo()
	// Laid out as Windows lays out a home, which a browser a test starts
	// needs to open at all.
	home := filepath.Join(dir, "home")
	for name, p := range map[string]string{
		"USERPROFILE": home, "HOME": home,
		"APPDATA": filepath.Join(home, "AppData", "Roaming"), "LOCALAPPDATA": filepath.Join(home, "AppData", "Local"),
		"XDG_CONFIG_HOME": filepath.Join(home, ".config"), "XDG_DATA_HOME": filepath.Join(home, ".local", "share"),
	} {
		os.MkdirAll(p, 0o755)
		os.Setenv(name, p)
	}
	os.Setenv("SAMEWAY_KNOWN", filepath.Join(dir, "known.json"))
	os.Setenv("SAMEWAY_KEYS", filepath.Join(dir, "keys.json"))
	code := m.Run()
	os.RemoveAll(dir)
	return code
}

// keepGo pins Go's own caches and settings where they are now, before the
// folders move: a test that builds the program with `go build` would
// otherwise start from an empty cache and fetch every module again.
func keepGo() {
	keep := func(name, dir string, err error, rest ...string) {
		if os.Getenv(name) == "" && err == nil {
			os.Setenv(name, filepath.Join(append([]string{dir}, rest...)...))
		}
	}
	cache, err := os.UserCacheDir()
	keep("GOCACHE", cache, err, "go-build")
	home, err := os.UserHomeDir()
	keep("GOPATH", home, err, "go")
	config, err := os.UserConfigDir()
	keep("GOENV", config, err, "go", "env")
}
