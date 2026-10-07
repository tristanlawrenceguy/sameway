package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// On Linux the first run puts Sameway in the applications menu with its
// icon, once; on a Mac an app run from Downloads is copied to the
// person's Applications folder, and one there already is left be.
func TestSamewayIsKeptWhereEachComputerFindsIt(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_DATA_HOME", "")
	var out bytes.Buffer
	keepOnLinux(&out, "/opt/sameway/sameway", home)
	entry, err := os.ReadFile(filepath.Join(home, ".local", "share", "applications", "sameway.desktop"))
	if err != nil || !strings.Contains(string(entry), `Exec="/opt/sameway/sameway"`) || !strings.Contains(string(entry), "Terminal=false") {
		t.Fatalf("a menu entry that runs it with no terminal: %v %s", err, entry)
	}
	if _, err := os.Stat(filepath.Join(home, ".local", "share", "icons", "hicolor", "256x256", "apps", "sameway.png")); err != nil {
		t.Error("with its icon")
	}
	if !strings.Contains(out.String(), "applications menu") {
		t.Errorf("and it says so: %q", out.String())
	}

	var copied [][2]string
	was := copyApp
	copyApp = func(from, to string) error { copied = append(copied, [2]string{from, to}); return os.MkdirAll(to, 0o755) }
	defer func() { copyApp = was }()
	app := filepath.Join(home, "Downloads", "Sameway.app", "Contents", "MacOS", "sameway")
	if !keepOnMac(&out, app, home) || len(copied) != 1 || copied[0][1] != filepath.Join(home, "Applications", "Sameway.app") {
		t.Errorf("an app from Downloads is copied to Applications: %v", copied)
	}
	if keepOnMac(&out, app, home) || len(copied) != 1 {
		t.Error("once")
	}
}
