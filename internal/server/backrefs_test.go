package server_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// A ref the schema marks listed lists its records on the page of what it
// points at, for any type: a recipe's steps, as a project's tasks. Asking
// for the same connection with ?show= does not list them twice. An older
// workspace's task, from before the flag, still lists on its project.
func TestAListedRefListsItsRecordsOnItsTarget(t *testing.T) {
	dir := t.TempDir()
	known := filepath.Join(t.TempDir(), "known.json")
	os.WriteFile(known, []byte("[]"), 0o644)
	t.Setenv("SAMEWAY_KNOWN", known)
	if err := workspace.Init(dir, examples.FS, examples.StarterRoot, false); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"recipe.yaml": "name: recipe\ntitle: title\nfields:\n  title:\n    type: string\n",
		"step.yaml":   "name: step\ntitle: title\nfields:\n  title:\n    type: string\n  recipe:\n    type: ref\n    to: recipe\n    listed: true\n",
		// A task as a workspace made before listed has it.
		"task.yaml": "name: task\ntitle: title\nprovided: true\nfields:\n  title:\n    type: string\n  done:\n    type: bool\n  project:\n    type: ref\n    to: project\n",
	}
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, "schema", name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	a, err := app.Load(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	h := server.New(a)

	var recipe, project struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/recipe", map[string]any{"title": "Bread"}), &recipe)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/step", map[string]any{"title": "Knead the dough", "recipe": recipe.ID}), http.StatusCreated)
	page := get(t, h, "/t/recipe/"+recipe.ID+"?show=points-here:step.recipe").Body.String()
	if !strings.Contains(page, `id="steps-`+recipe.ID+`"`) || !strings.Contains(page, ">Knead the dough</a>") {
		t.Errorf("a recipe's page lists its steps\n%s", truncate(page))
	}
	if n := strings.Count(page, ">Knead the dough</a>"); n != 1 {
		t.Errorf("listed once, not again as an opened connection; listed %d times", n)
	}
	// What a page lists, its glance counts: one rule for what a record holds.
	var view struct{ Glance string }
	decode(t, get(t, h, "/api/recipe/"+recipe.ID), &view)
	if view.Glance != "1 step" {
		t.Errorf("a recipe's glance is %q, want %q", view.Glance, "1 step")
	}

	decode(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{"title": "Garden"}), &project)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Dig the beds", "project": project.ID}), http.StatusCreated)
	if page := get(t, h, "/t/project/"+project.ID).Body.String(); !strings.Contains(page, `id="tasks-`+project.ID+`"`) || !strings.Contains(page, ">Dig the beds</a>") {
		t.Errorf("an older workspace's project lists its tasks\n%s", truncate(page))
	}
}
