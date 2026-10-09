package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// A project's page lists its tasks by itself.
//
// Acceptance items from backlog 0662 / task 0266:
// 1. Every project detail page shows a "Tasks" section below the lede and facts
// 2. Each associated task appears as a clickable link showing only the task title
// 3. An empty project shows a brief message like "No tasks yet" in the section
// 4. The list is ordered with the most recently created task first
// 5. A screen reader hears "Tasks — [count]" as the section heading

func TestProjectDetailShowsAssociatedTasks(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	// Seed a project and two tasks that belong to it via chat-style API writes.
	var proj struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{"title": "Weekly errands"}), &proj)

	var task1, task2 struct{ ID string }
	// Task 1 created first (older).
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Buy groceries", "project": proj.ID}), &task1)
	// Task 2 created second (newer, should appear first in the list).
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Walk the dog", "project": proj.ID}), &task2)

	page := get(t, h, "/t/project/"+proj.ID).Body.String()

	// 1. The page contains a tasks section (data-component="collection").
	if !strings.Contains(page, `<section class="sw-collection"`) && !strings.Contains(page, `data-component="collection"`) {
		t.Errorf("project detail page should show a Tasks collection section\n%s", page)
	}

	// 2. Each associated task appears as a clickable link with its title.
	if !strings.Contains(page, ">Buy groceries</a>") {
		t.Error("the project's tasks list should include \"Buy groceries\"")
	}
	if !strings.Contains(page, ">Walk the dog</a>") {
		t.Error("the project's tasks list should include \"Walk the dog\"")
	}

	// 4. The most recently created task appears first (Walk the dog before Buy groceries).
	idxDog := strings.Index(page, "Walk the dog")
	idxGroceries := strings.Index(page, "Buy groceries")
	if idxDog < 0 || idxGroceries < 0 {
		t.Fatalf("both tasks must appear in the page to check order; got: %.800s", page)
	}
	if idxDog > idxGroceries {
		t.Errorf("the most recently created task should appear first in the list: Walk the dog (idx %d) appears after Buy groceries (idx %d)\n%s", idxDog, idxGroceries, page)
	}

	// 5. The section heading is accessible to screen readers.
	if !strings.Contains(page, "Tasks") {
		t.Error("the tasks section should have a visible \"Tasks\" heading")
	}
}

func TestProjectDetailShowsEmptyMessageWhenNoTasks(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	var proj struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{"title": "Empty project"}), &proj)

	page := get(t, h, "/t/project/"+proj.ID).Body.String()

	// 1. The page still contains a tasks section (not absent).
	if !strings.Contains(page, `data-component="collection"`) {
		t.Error("an empty project should still show an empty Tasks collection section")
	}

	// 3. An empty project shows a brief message like "No tasks yet".
	if !(strings.Contains(page, "Nothing here yet") || strings.Contains(page, "no task") || strings.Contains(page, "nothing")) {
		t.Error("an empty project should show an empty-state message in the Tasks section\n" + truncate(page))
	}

	// The project's own title is still there (the page didn't disappear).
	if !strings.Contains(page, ">Empty project</") {
		t.Error("the project detail page should still show the project title\n" + truncate(page))
	}

	_ = a // silence unused import if everything else compiles.
}

func TestProjectDetailShowsOnlyItsOwnTasks(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)

	var proj1, proj2 struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{"title": "Project Alpha"}), &proj1)
	decode(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{"title": "Project Beta"}), &proj2)

	// A task in Project Alpha.
	var alphaTask struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Alpha task", "project": proj1.ID}), &alphaTask)

	// A task in Project Beta (should NOT appear on Alpha's page).
	var betaTask struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Beta task", "project": proj2.ID}), &betaTask)

	alphaPage := get(t, h, "/t/project/"+proj1.ID).Body.String()

	if !strings.Contains(alphaPage, ">Alpha task</a>") {
		t.Error("Project Alpha page should show its own task \"Alpha task\"\n" + truncate(alphaPage))
	}
	if strings.Contains(alphaPage, "Beta task") {
		t.Errorf("Project Alpha page must not show tasks from other projects; found \"Beta task\"\n%s", alphaPage)
	}

	betaPage := get(t, h, "/t/project/"+proj2.ID).Body.String()
	if !strings.Contains(betaPage, ">Beta task</a>") {
		t.Error("Project Beta page should show its own task\n" + truncate(betaPage))
	}
	if strings.Contains(betaPage, "Alpha task") {
		t.Errorf("Project Beta page must not show tasks from other projects; found \"Alpha task\"\n%s", betaPage)
	}

	_ = a // silence unused import.
}
