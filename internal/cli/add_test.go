package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A file already on this computer is added from the command line, copied
// in as it is and read as an upload would be; each is said, or all as
// JSON; a folder or a missing file says what to do instead.
func TestAddCopiesFilesOnThisComputerIn(t *testing.T) {
	ws := initWorkspace(t)
	src := t.TempDir()
	plan := filepath.Join(src, "Plan.md")
	os.WriteFile(plan, []byte("# The plan\n\nDig the pond."), 0o644)
	memo := filepath.Join(src, "memo.m4a")
	os.WriteFile(memo, []byte("not really audio"), 0o644)

	r := run(t, ws, "add", plan, "--title", "Pond plan")
	if r.code != 0 || !strings.Contains(r.stdout, "added Plan.md as Pond plan (/t/file/") {
		t.Fatalf("add says what it added: %d %q %q", r.code, r.stdout, r.stderr)
	}
	r = run(t, ws, "add", memo, "--json")
	var added []struct {
		ID     string
		Fields map[string]any
	}
	if r.code != 0 || json.Unmarshal([]byte(r.stdout), &added) != nil || len(added) != 1 || added[0].Fields["kind"] != "audio" || added[0].Fields["status"] != "ready" {
		t.Fatalf("a recording is added as audio, ready: %d %q %q", r.code, r.stdout, r.stderr)
	}
	kept := filepath.Join(ws, "files", added[0].Fields["path"].(string))
	if b, err := os.ReadFile(kept); err != nil || string(b) != "not really audio" {
		t.Errorf("the file is copied into the workspace as it was: %v", err)
	}
	if b, _ := os.ReadFile(memo); string(b) != "not really audio" {
		t.Error("and the original is left where it was")
	}
	list := run(t, ws, "file", "list", "--json")
	if !strings.Contains(list.stdout, "Dig the pond") || !strings.Contains(list.stdout, "Pond plan") {
		t.Errorf("the document's words are read in: %s", list.stdout)
	}

	if r := run(t, ws, "add", src); r.code == 0 || !strings.Contains(r.stderr, "is a folder; name the files in it") {
		t.Errorf("a folder says to name its files: %q", r.stderr)
	}
	if r := run(t, ws, "add", filepath.Join(src, "gone.mp3")); r.code == 0 || !strings.Contains(r.stderr, "could not open") {
		t.Errorf("a missing file says so: %q", r.stderr)
	}
	if r := run(t, ws, "add"); r.code == 0 || !strings.Contains(r.stderr, "usage: sameway add") {
		t.Errorf("no file says how: %q", r.stderr)
	}
}
