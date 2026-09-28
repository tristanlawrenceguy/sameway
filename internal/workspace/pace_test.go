package workspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// The pace is kept in workspace.yaml, edited as one line so the comments a
// person reads there survive, and read back with a safe default.
func TestPaceIsKeptInTheWorkspaceFile(t *testing.T) {
	dir := t.TempDir()
	if err := workspace.Init(dir, examples.FS, examples.StarterRoot, false); err != nil {
		t.Fatal(err)
	}
	w, err := workspace.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if w.Config.UI.Pace != "calm" {
		t.Errorf("the default pace is calm, got %q", w.Config.UI.Pace)
	}
	before, _ := os.ReadFile(filepath.Join(dir, "workspace.yaml"))
	if err := w.SetPace("quick"); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(dir, "workspace.yaml"))
	if strings.Count(string(after), "\n")-strings.Count(string(before), "\n") != 1 || !strings.Contains(string(after), "  pace: quick\n") {
		t.Errorf("setting a pace should add one line under ui:\n%s", after)
	}
	if !strings.Contains(string(after), "# auto:") {
		t.Error("the comments in the file should survive")
	}
	if err := w.SetPace("still"); err != nil {
		t.Fatal(err)
	}
	again, _ := os.ReadFile(filepath.Join(dir, "workspace.yaml"))
	if strings.Count(string(again), "pace:") != 1 || !strings.Contains(string(again), "  pace: still\n") {
		t.Errorf("setting it again should change the line, not add one:\n%s", again)
	}
	reloaded, _ := workspace.Load(dir)
	if reloaded.Config.UI.Pace != "still" {
		t.Errorf("the pace should be read back, got %q", reloaded.Config.UI.Pace)
	}
	if err := w.SetPace("frantic"); err == nil {
		t.Error("an unknown pace should be refused")
	}

	// A file without a ui block gets one.
	os.WriteFile(filepath.Join(dir, "workspace.yaml"), []byte("name: Bare\n"), 0o644)
	bare, _ := workspace.Load(dir)
	if err := bare.SetPace("quick"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "workspace.yaml")); !strings.HasSuffix(string(got), "ui:\n  pace: quick") {
		t.Errorf("a file without ui: should gain the block:\n%s", got)
	}
}
