package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// A ref is a record pointing at another: a task belongs to a project. The
// task's page shows the project as the way there, the project's page
// lists its tasks by itself, and a ref to nothing is refused with the
// field named.
func TestARefIsARecordPointingAtAnother(t *testing.T) {
	_, h := newApp(t)
	var garden struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{"title": "Garden", "notes": "The back garden this autumn."}), &garden)
	var task struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Dig the pond", "project": garden.ID}), &task)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Paint the hall"}), http.StatusCreated)

	bad := postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Nowhere", "project": "zzzzzzzzzzzzzzzz"})
	if bad.Code < 400 || !strings.Contains(bad.Body.String(), "no project with id zzzzzzzzzzzzzzzz") {
		t.Errorf("a ref to nothing is refused and names the field: %d %s", bad.Code, truncate(bad.Body.String()))
	}

	page := get(t, h, "/t/task/"+task.ID).Body.String()
	if !strings.Contains(page, `<dt>Project</dt><dd data-prop="project" data-source="`+garden.ID+`" data-options="`) || !strings.Contains(page, `"><a class="sw-link" href="/t/project/`+garden.ID+`">Garden</a></dd>`) {
		t.Errorf("the task's page shows its project as a link: %.700s", page[strings.Index(page, "<dl"):])
	}
	// The project's page is the project. Not its tasks, and not a count of
	// them either: the page says what the record is, and what else exists
	// is asked for.
	project := get(t, h, "/t/project/"+garden.ID).Body.String()
	for _, gone := range []string{"Dig the pond", ">1 task</a>", ">Related</h2>"} {
		if strings.Contains(project, gone) {
			t.Errorf("the project's page at rest says nothing of its tasks, found %q: %.600s", gone, project)
		}
	}
	open := get(t, h, "/t/project/"+garden.ID+"?show=points-here:task.project").Body.String()
	if !strings.Contains(open, `data-component="collection"`) || !strings.Contains(open, "Dig the pond") || strings.Contains(open, "Paint the hall") {
		t.Errorf("opening it lists the project's own tasks: %.900s", open[strings.Index(open, "</dl>"):])
	}
	if !strings.Contains(open, "/t/task?order=-updated_at&amp;where=project%3D"+garden.ID) {
		t.Error("the list of tasks leads on to the list page with the same query")
	}

	// On the canvas, a record block shows the project by its title.
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "record", "props": map[string]any{"type": "task", "record": task.ID}}), http.StatusCreated)
	if canvas := get(t, h, "/").Body.String(); !strings.Contains(canvas, "<dt>Project</dt><dd>Garden</dd>") {
		t.Errorf("a record block names the project, not its id: %.500s", canvas[strings.Index(canvas, "Dig the pond"):])
	}
}
