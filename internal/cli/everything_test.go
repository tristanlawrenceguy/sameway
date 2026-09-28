package cli_test

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A whole workspace goes out in one zip and comes back into another: its
// records by `sameway import`, its files as they were added.
func TestAWholeWorkspaceGoesElsewhere(t *testing.T) {
	from := initWorkspace(t)
	run(t, from, "task", "create", "--set", "title=Order compost", "--set", "due=2026-10-05")
	run(t, from, "note", "create", "--set", "title=Pond plan", "--set", "body=Dig it.")
	src := filepath.Join(t.TempDir(), "Plan.md")
	os.WriteFile(src, []byte("# The plan"), 0o644)
	run(t, from, "add", src)
	out := filepath.Join(t.TempDir(), "everything.zip")
	if r := run(t, from, "export", "--zip", out); r.code != 0 || !strings.Contains(r.stdout, "wrote everything to") {
		t.Fatalf("export --zip: %d %q %q", r.code, r.stdout, r.stderr)
	}
	z, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	to := initWorkspace(t)
	names := map[string]bool{}
	for _, f := range z.File {
		names[strings.SplitN(f.Name, "/", 2)[0]] = true
		if strings.HasPrefix(f.Name, "content/") || strings.HasPrefix(f.Name, "files/") {
			dst := filepath.Join(to, filepath.FromSlash(f.Name))
			os.MkdirAll(filepath.Dir(dst), 0o755)
			r, _ := f.Open()
			b, _ := io.ReadAll(r)
			r.Close()
			os.WriteFile(dst, b, 0o644)
		}
	}
	for _, want := range []string{"README.txt", "content", "files", "spreadsheets"} {
		if !names[want] {
			t.Errorf("the zip has %s", want)
		}
	}
	if r := run(t, to, "import"); r.code != 0 {
		t.Fatalf("import: %q %q", r.stdout, r.stderr)
	}
	tasks := run(t, to, "task", "list", "--json").stdout
	notes := run(t, to, "note", "list", "--json").stdout
	files := run(t, to, "file", "list", "--json").stdout
	if !strings.Contains(tasks, "Order compost") || !strings.Contains(notes, "Pond plan") || !strings.Contains(files, "Plan") {
		t.Errorf("the records come back: %s %s %s", tasks, notes, files)
	}
	var stored string
	for _, f := range z.File {
		if strings.HasPrefix(f.Name, "files/") {
			stored = f.Name
		}
	}
	if b, err := os.ReadFile(filepath.Join(to, filepath.FromSlash(stored))); err != nil || string(b) != "# The plan" {
		t.Errorf("and the file as it was added: %v", err)
	}
}
