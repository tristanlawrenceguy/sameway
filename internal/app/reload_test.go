package app_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// Two processes on one workspace (the server and `sameway mcp`) each take
// what the other did to the content types, and so do both when a person
// edits schema/ by hand: new types get their table, a file that does not
// read is left until it does, and a file taken away takes its type off
// the pages but keeps its records.
func TestSchemaChangedElsewhereIsTakenWhileRunning(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := workspace.Init(dir, examples.FS, examples.StarterRoot, false); err != nil {
		t.Fatal(err)
	}
	load := func() *app.App {
		a, err := app.Load(dir, false)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { a.Close() })
		return a
	}
	server, tools := load(), load()
	if changed, err := server.ReloadSchema(); changed || err != nil {
		t.Errorf("nothing changed yet, so nothing is taken: %v %v", changed, err)
	}

	// Either way round: the tools' process makes a type, the server a field.
	if _, err := tools.AddType(&schema.Type{Name: "recipe", Fields: []schema.Field{{Name: "title", Type: "string"}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := server.AddField("task", schema.Field{Name: "effort", Type: "int"}); err != nil {
		t.Fatal(err)
	}
	for name, a := range map[string]*app.App{"server": server, "tools": tools} {
		if changed, err := a.ReloadSchema(); !changed || err != nil {
			t.Errorf("%s takes the other's change: %v %v", name, changed, err)
		}
		if _, ok := a.Types.Get("recipe"); !ok {
			t.Errorf("%s has recipe", name)
		}
		task, _ := a.Types.Get("task")
		if _, ok := task.Field("effort"); !ok {
			t.Errorf("%s has task.effort", name)
		}
	}
	if _, err := server.Store.Create("task", map[string]any{"title": "Dig", "effort": 3}); err != nil {
		t.Fatal(err)
	}
	if got, _ := tools.Store.Count("task"); got != 1 {
		t.Errorf("records are shared through data.db as they always were: %d", got)
	}

	// By hand: a new file gets its table in whoever reads it.
	wine := filepath.Join(dir, "schema", "wine.yaml")
	os.WriteFile(wine, []byte("name: wine\nfields:\n  title: {type: string}\n  year: {type: int}\n"), 0o644)
	if changed, err := server.ReloadSchema(); !changed || err != nil {
		t.Fatalf("a file written by hand is taken: %v %v", changed, err)
	}
	if _, err := server.Store.Create("wine", map[string]any{"title": "Rioja", "year": 2019}); err != nil {
		t.Errorf("the new type has its table: %v", err)
	}

	// Half a file, or a mistake, changes nothing until it reads.
	os.WriteFile(wine, []byte("name: wine\nfields:\n  title: {type: strin"), 0o644)
	if changed, err := server.ReloadSchema(); changed || err == nil || !strings.Contains(err.Error(), "wine.yaml") {
		t.Errorf("a file that does not read is refused, naming it: %v %v", changed, err)
	}
	if w, ok := server.Types.Get("wine"); !ok || len(w.Fields) != 2 {
		t.Error("the types stay as they were meanwhile")
	}
	if changed, err := server.ReloadSchema(); changed || err != nil {
		t.Errorf("and it is not read again until it changes: %v %v", changed, err)
	}
	os.WriteFile(wine, []byte("name: wine\nfields:\n  title: {type: string}\n  year: {type: int}\n  grape: {type: string}\n"), 0o644)
	if changed, err := server.ReloadSchema(); !changed || err != nil {
		t.Errorf("once it reads it is taken: %v %v", changed, err)
	}
	if w, _ := server.Types.Get("wine"); len(w.Fields) != 3 {
		t.Errorf("wine has grape now: %v", w.Fields)
	}

	// Taken away: off the pages, records kept.
	os.Remove(wine)
	if changed, _ := server.ReloadSchema(); !changed {
		t.Error("a file taken away is noticed")
	}
	if _, ok := server.Types.Get("wine"); ok {
		t.Error("its type is gone from the pages")
	}
	os.WriteFile(wine, []byte("name: wine\nfields:\n  title: {type: string}\n  year: {type: int}\n"), 0o644)
	server.ReloadSchema()
	if n, _ := server.Store.Count("wine"); n != 1 {
		t.Errorf("its records were kept: %d", n)
	}

	// Files are written whole, through a file beside them that never stays.
	left, _ := filepath.Glob(filepath.Join(dir, "schema", ".*.tmp"))
	if len(left) > 0 {
		t.Errorf("no half-written files are left behind: %v", left)
	}
}
