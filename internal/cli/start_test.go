package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	workspace.ThisMachine().Remember(dir, addr)
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

// Opened again while it runs, its tab out of sight and the server slow to
// answer, the running one is shown in a new tab, whatever case or trailing
// separator its folder was written with; a second Sameway is not started.
func TestOpeningAgainShowsTheRunningOne(t *testing.T) {
	t.Setenv("SAMEWAY_KNOWN", filepath.Join(t.TempDir(), "known.json"))
	dir := filepath.Join(t.TempDir(), "Home")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, workspace.ConfigFile), []byte("name: Home\n"), 0o644)
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { time.Sleep(1500 * time.Millisecond) }))
	defer slow.Close()
	workspace.ThisMachine().Remember(dir, strings.TrimPrefix(slow.URL, "http://"))
	opened := ""
	openInBrowser = func(url string) error { opened = url; return nil }
	var out bytes.Buffer
	c := &ctx{Env: Env{Stdout: &out, Stderr: &out}}
	c.workspaceDir = strings.ToLower(dir) + string(filepath.Separator) // as Windows may write it
	if err := c.startCmd(); err != nil {
		t.Fatal(err)
	}
	if opened != slow.URL+"/" {
		t.Errorf("the running one is opened: %q\n%s", opened, out.String())
	}
}
