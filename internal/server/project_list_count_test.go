package server_test

// The project list page shows a task count next to each row title, so a
// person scanning multiple projects can see at a glance how many tasks
// belong to each. Acceptance items 1–4 of task 0267 (backlog 0689).

import (
	"net/http"
	"strings"
	"testing"
)

func findRow(body, id string) (string, bool) {
	linkPrefix := `/t/project/` + id
	idx := strings.Index(body, linkPrefix)
	if idx < 0 {
		return "", false
	}
	start := strings.LastIndex(body[:idx], "<li class=") + 4
	endAbs := idx + strings.Index(body[idx:], "</li>")
	return body[start : endAbs+5], true
}

// TestProjectListShowsTaskCount verifies that the project list page shows a
// task count next to each project row's title. A project with tasks shows
// "5 tasks" (or "1 task") in its row; an empty project says none, as a
// state where every record rests is not said (design/foundations/glance.md).
// This covers acceptance items 1 and 2 of task 0267.
func TestProjectListShowsTaskCount(t *testing.T) {
	_, h := newApp(t)

	var projWithTasks struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{"title": "Garden projects"}), &projWithTasks)

	var t1, t2 struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Plant tomatoes", "project": projWithTasks.ID}), &t1)
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Build shed", "project": projWithTasks.ID}), &t2)

	var projOneTask struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{"title": "House tasks"}), &projOneTask)

	var t3 struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Paint fence", "project": projOneTask.ID}), &t3)

	var projEmpty struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{"title": "No tasks here"}), &projEmpty)

	body := get(t, h, "/t/project").Body.String()

	for _, tc := range []struct {
		id    string
		count string
	}{
		{projWithTasks.ID, "2 tasks"},
		{projOneTask.ID, "1 task"},
	} {
		row, ok := findRow(body, tc.id)
		if !ok {
			t.Fatalf("could not find project row for %s in page\n%s", tc.id, truncate(body))
		}
		if !strings.Contains(row, tc.count) {
			t.Errorf("project row should show %q; found in:\n%s", tc.count, truncate(row))
		}
	}
	if row, _ := findRow(body, projEmpty.ID); strings.Contains(row, "0 tasks") {
		t.Errorf("an empty project row should say no count; found in:\n%s", truncate(row))
	}
}

// TestProjectListTaskCountIsVisible verifies that the task count appears in the
// row's meta area (sw-row__meta) rather than hidden or only in visually-hidden
// text. The count is plain visible text next to the project link, consistent with
// how other list pages display counts. Acceptance item 3 of task 0267.
func TestProjectListTaskCountIsVisible(t *testing.T) {
	_, h := newApp(t)

	var proj struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{"title": "Visible count test"}), &proj)

	var taskOne struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Task one", "project": proj.ID}), &taskOne)

	body := get(t, h, "/t/project").Body.String()

	row, ok := findRow(body, proj.ID)
	if !ok {
		t.Fatalf("could not find project row in page\n%s", truncate(body))
	}

	// The count must be inside the sw-row__meta element.
	if !strings.Contains(row, `class="sw-row__meta"`) {
		t.Errorf("project row should contain a sw-row__meta element\n%s", truncate(row))
		return
	}

	metaStart := strings.Index(row, `class="sw-row__meta"`)
	metaContent := row[metaStart:]
	closeMeta := strings.Index(metaContent, "</p>")
	if closeMeta > 0 {
		metaContent = metaContent[:closeMeta]
	}

	if !strings.Contains(metaContent, "1 task") {
		t.Errorf("sw-row__meta should contain visible task count \"1 task\"; found:\n%s", truncate(metaContent))
	}

	// The count must NOT be inside a sw-visually-hidden span.
	if strings.Contains(row, `class="sw-visually-hidden"`) && strings.Contains(row, "1 task") {
		t.Errorf("task count must not be hidden visually; found in:\n%s", truncate(row))
	}
}

// TestProjectListTaskCountScreenReader verifies that the task count appears
// in the natural reading order for a screen reader — it is inside the row's
// <li> and comes after the project link, with no sw-visually-hidden wrapper.
// Acceptance item 4 of task 0267.
func TestProjectListTaskCountScreenReader(t *testing.T) {
	_, h := newApp(t)

	var proj struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{"title": "Screen reader test"}), &proj)

	var taskOne struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "A task", "project": proj.ID}), &taskOne)

	body := get(t, h, "/t/project").Body.String()

	row, ok := findRow(body, proj.ID)
	if !ok {
		t.Fatalf("could not find project row in page\n%s", truncate(body))
	}

	// The link must come before the count in reading order.
	linkIdx := strings.Index(row, `<a class="sw-row__link"`)
	countIdx := strings.Index(row, "1 task")
	if linkIdx < 0 {
		t.Errorf("project row should contain a sw-row__link\n%s", truncate(row))
		return
	}
	if countIdx < 0 {
		t.Fatalf("project row must show \"1 task\" for screen readers; found in:\n%s", truncate(row))
	}
	if linkIdx > countIdx {
		t.Errorf("task count should appear after the project link in reading order (link at %d, count at %d)\n%s", linkIdx, countIdx, truncate(row))
	}

	// The count text must not be inside a sw-visually-hidden span.
	vhStart := strings.Index(row, `class="sw-visually-hidden"`)
	if vhStart >= 0 && strings.Contains(row[vhStart:], "1 task") {
		t.Errorf("task count \"1 task\" should not be wrapped in sw-visually-hidden; found in:\n%s", truncate(row))
	}
}

// TestProjectListSaysNothingForZeroTasks verifies that a project with no
// tasks says no count: the count is a glance fact (glance_count.go), and
// none is where every project starts, so "0 tasks" on every empty project
// (and "0 interactions" on every person) would be read and say nothing.
// Acceptance item 2 of task 0267 asked for "0 tasks"; changed deliberately.
func TestProjectListSaysNothingForZeroTasks(t *testing.T) {
	_, h := newApp(t)

	var proj struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{"title": "Empty project"}), &proj)

	body := get(t, h, "/t/project").Body.String()

	row, ok := findRow(body, proj.ID)
	if !ok {
		t.Fatalf("could not find project row in page\n%s", truncate(body))
	}

	if strings.Contains(row, "task") {
		t.Errorf("an empty project row should say no task count; found in:\n%s", truncate(row))
	}
}
