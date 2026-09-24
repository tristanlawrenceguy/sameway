package server_test

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
)

// TestARefusedEditListsEachProblem: an edit that cannot be taken comes back
// as an error summary at the top, one problem per field, each leading to its
// field, rather than one run-on sentence.
func TestARefusedEditListsEachProblem(t *testing.T) {
	a, h := newApp(t)
	habit, _ := a.Store.Create("habit", map[string]any{"name": "Water"})
	res := postForm(t, h, "/t/habit/"+habit.ID+"/props", url.Values{"prop-target": {"lots"}, "prop-cadence": {"fortnight"}})
	page, _ := landed(t, h, res)
	body := page.Body.String()
	for _, want := range []string{`data-component="error-summary"`, `data-field="target"`, `data-field="cadence"`, `href="#field-target"`, `>Not saved</h2>`} {
		if !strings.Contains(body, want) {
			t.Errorf("the refused edit should carry %s\n%.2000s", want, body[strings.Index(body, "<main"):])
		}
	}
}

// TestAFieldWithAMostSaysSo: a field the schema caps is marked for the
// editor to count down.
func TestAFieldWithAMostSaysSo(t *testing.T) {
	a, h := newApp(t)
	note, _ := a.Store.Create("note", map[string]any{"title": "Beds"})
	if body := get(t, h, "/t/note/"+note.ID).Body.String(); !strings.Contains(body, `data-prop="title" data-label="Title" data-max="200"`) {
		t.Errorf("a note's title, at most 200 characters, should say so to the editor")
	}
}

// TestALinkToOneOfManyIsLookedUp: past the most a list can hold, a link to
// another record is looked up by name, not dropped from the editor.
func TestALinkToOneOfManyIsLookedUp(t *testing.T) {
	a, h := newApp(t)
	var first string
	for i := 0; i < 501; i++ {
		p, err := a.Store.Create("project", map[string]any{"title": fmt.Sprintf("Project %d", i)})
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			first = p.ID
		}
	}
	task, _ := a.Store.Create("task", map[string]any{"title": "Repot the fern", "project": first})
	body := get(t, h, "/t/task/"+task.ID).Body.String()
	want := `data-kind="lookup" data-to="project" data-source="` + first + `" data-title="Project 0"`
	if !strings.Contains(body, want) {
		t.Errorf("the project, one of 501, should be looked up by name: want %s", want)
	}
}
