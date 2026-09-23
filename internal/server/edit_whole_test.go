package server_test

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

// A person edits the whole record where it is: the page carries every
// field a person may change, in the type's order, for the editor to build
// from, including the title, the chips and the fields still empty. An
// empty record can be edited too.
func TestARecordCarriesEveryFieldAPersonMayEdit(t *testing.T) {
	a, h := newApp(t)
	project, _ := a.Store.Create("project", map[string]any{"title": "Garden"})
	task, _ := a.Store.Create("task", map[string]any{"title": "Repot the fern"})

	body := get(t, h, "/t/task/"+task.ID).Body.String()
	tmpl := regexp.MustCompile(`(?s)<template data-edit-fields>(.*?)</template>`).FindStringSubmatch(body)
	if tmpl == nil {
		t.Fatalf("an empty task still says what can be edited on it")
	}
	var order []string
	for _, m := range regexp.MustCompile(`data-prop="([a-z_]+)"`).FindAllStringSubmatch(tmpl[1], -1) {
		order = append(order, m[1])
	}
	if strings.Join(order, " ") != "title done due project notes tags" {
		t.Errorf("every field of a task, in its order, got %v", order)
	}
	for _, want := range []string{`data-prop="done" data-label="Done" data-kind="bool" data-source="false"`, `data-kind="datetime"`, project.ID, `&#34;label&#34;:&#34;None&#34;`, `data-source="Repot the fern"`} {
		if !strings.Contains(tmpl[1], want) {
			t.Errorf("the fields carry what the editor needs: missing %s", want)
		}
	}

	// A field the system keeps is not offered.
	file, _ := a.Store.Create("file", map[string]any{"title": "Plan", "path": "plan.pdf", "name": "plan.pdf"})
	files := regexp.MustCompile(`(?s)<template data-edit-fields>(.*?)</template>`).FindStringSubmatch(get(t, h, "/t/file/"+file.ID).Body.String())
	if files == nil || strings.Contains(files[1], `data-prop="path"`) || !strings.Contains(files[1], `data-prop="title"`) {
		t.Error("where a file is stored is kept by Sameway, not offered to edit")
	}
}

// A system-kept field sent by hand is refused by name, and nothing moves.
func TestAFieldTheSystemKeepsIsNotChangedByHand(t *testing.T) {
	a, h := newApp(t)
	file, _ := a.Store.Create("file", map[string]any{"title": "Plan", "path": "plan.pdf", "name": "plan.pdf"})
	r := postForm(t, h, "/t/file/"+file.ID+"/props", url.Values{"prop-path": {"../elsewhere"}})
	if body := after(t, h, r).Body.String(); !strings.Contains(body, "cannot be changed by hand") {
		t.Errorf("the refusal says why; body: %s", truncate(body))
	}
	if got, _ := a.Store.Get("file", file.ID); got.Fields["path"] != "plan.pdf" {
		t.Errorf("nothing moves, got %v", got.Fields["path"])
	}
}

// What a person wrote in the rich editor is what is kept, every time,
// though the Markdown box behind it goes with the form holding the old
// words: which of the two came first used to be chance.
func TestRichTextWinsOverTheMarkdownBehindIt(t *testing.T) {
	a, h := newApp(t)
	note, _ := a.Store.Create("note", map[string]any{"title": "Beds", "body": "Old words."})
	for i := 0; i < 40; i++ {
		form := url.Values{"prop-body": {"Old words."}, "html-body": {"<p>New words.</p>"}, "level-body": {"2"}}
		wantStatus(t, postForm(t, h, "/t/note/"+note.ID+"/props", form), http.StatusSeeOther)
		if got, _ := a.Store.Get("note", note.ID); !strings.Contains(got.Fields["body"].(string), "New words.") {
			t.Fatalf("try %d: the rich text is what is kept, got %q", i+1, got.Fields["body"])
		}
		a.Store.Update("note", note.ID, map[string]any{"title": "Beds", "body": "Old words."})
	}
}
