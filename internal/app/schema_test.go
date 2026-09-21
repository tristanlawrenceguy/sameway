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

// A property added while the workspace runs is in the type's file, in its
// table and in every record from then on, and is still there after a
// restart. A provided type without a file of its own gets one.
func TestAFieldOrATypeCanBeAddedWhileRunning(t *testing.T) {
	dir := t.TempDir()
	if err := workspace.Init(dir, examples.FS, examples.StarterRoot, false); err != nil {
		t.Fatal(err)
	}
	a, err := app.Load(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	if _, err := a.AddField("note", schema.Field{Name: "due", Type: "datetime", Description: "When it is needed by."}); err != nil {
		t.Fatal(err)
	}
	rec, err := a.Store.Create("note", map[string]any{"title": "Plan", "due": "2026-10-01T00:00:00Z"})
	if err != nil || rec.Fields["due"] != "2026-10-01T00:00:00Z" {
		t.Errorf("a note can carry the new field at once: %v %v", rec, err)
	}
	src, _ := os.ReadFile(filepath.Join(dir, "schema", "note.yaml"))
	if !strings.Contains(string(src), "# An ordinary content type") || !strings.Contains(string(src), "due:") || !strings.Contains(string(src), "When it is needed by.") {
		t.Errorf("the file keeps its comments and gains the field:\n%s", src)
	}
	if _, err := a.AddField("note", schema.Field{Name: "due", Type: "datetime"}); err == nil || !strings.Contains(err.Error(), "already has") {
		t.Errorf("adding a field twice is refused: %v", err)
	}
	if _, err := a.AddField("note", schema.Field{Name: "owner", Type: "ref", To: "unicorn"}); err == nil || !strings.Contains(err.Error(), `"unicorn"`) {
		t.Errorf("a ref to a type that is not there is refused: %v", err)
	}
	if _, err := a.AddField("note", schema.Field{Name: "Bad Name", Type: "string"}); err == nil {
		t.Error("a field name the schema would refuse at start-up is refused now")
	}

	// A provided type has no file of its own until it changes.
	if _, err := a.AddField("task", schema.Field{Name: "priority", Type: "enum", Values: []string{"low", "high"}, Default: "low"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "schema", "task.yaml")); err != nil {
		t.Error("the provided task type should now have its own file")
	}

	made, err := a.AddType(&schema.Type{Name: "contact", Description: "Someone to keep in touch with.", Fields: []schema.Field{{Name: "name", Type: "string", Required: true}, {Name: "email", Type: "string"}, {Name: "project", Type: "ref", To: "project"}}})
	if err != nil {
		t.Fatal(err)
	}
	if made.Title != "name" {
		t.Errorf("the first string field is the title, got %q", made.Title)
	}
	if _, err := a.Store.Create("contact", map[string]any{"name": "Ana"}); err != nil {
		t.Errorf("a record of the new type can be made at once: %v", err)
	}
	if _, err := a.AddType(&schema.Type{Name: "contact"}); err == nil {
		t.Error("a second type with the same name is refused")
	}

	// A restart sees all of it.
	a.Close()
	again, err := app.Load(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	defer again.Close()
	note, _ := again.Types.Get("note")
	if _, ok := note.Field("due"); !ok {
		t.Error("the note's new field survives a restart")
	}
	if _, ok := again.Types.Get("contact"); !ok {
		t.Error("the new type survives a restart")
	}
	task, _ := again.Types.Get("task")
	if f, ok := task.Field("priority"); !ok || f.Default != "low" {
		t.Errorf("the task's new field survives with its default: %v", f)
	}
}
