package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// A double-click opens the person's workspace: a new one in Documents the
// first time, the one used last after that, and one already running is
// shown, not started twice.
func TestADoubleClickOpensYourWorkspace(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, "Documents"), 0o755)
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("SAMEWAY_KNOWN", filepath.Join(t.TempDir(), "known.json"))
	var out bytes.Buffer
	c := &ctx{Env: Env{Stdout: &out, Stderr: &out, Dir: t.TempDir()}}

	dir, made, err := c.yourWorkspace()
	if err != nil || !made || dir != filepath.Join(home, "Documents", "Sameway") {
		t.Fatalf("the first time, a new workspace in Documents: %q %v %v", dir, made, err)
	}
	if _, err := os.Stat(filepath.Join(dir, workspace.ConfigFile)); err != nil {
		t.Fatal("it is a workspace")
	}

	// Used before and running: shown in the browser, not started again.
	running := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer running.Close()
	addr := strings.TrimPrefix(running.URL, "http://")
	workspace.Remember(dir, addr)
	opened := ""
	openInBrowser = func(url string) error { opened = url; return nil }
	c = &ctx{Env: Env{Stdout: &out, Stderr: &out, Dir: t.TempDir()}}
	if err := c.startCmd(); err != nil {
		t.Fatal(err)
	}
	if opened != running.URL+"/" || !strings.Contains(out.String(), "already open") {
		t.Errorf("a running workspace is shown, not started twice: %q\n%s", opened, out.String())
	}
}

// An error waits to be read before the window closes.
func TestADoubleClickWaitsOnAnError(t *testing.T) {
	var out bytes.Buffer
	c := &ctx{Env: Env{Stdout: &out, Stderr: &out, Stdin: strings.NewReader("\n")}}
	c.holdOpen(os.ErrPermission)
	if !strings.Contains(out.String(), "Press Enter to close") {
		t.Errorf("the reason stays on screen: %s", out.String())
	}
}
