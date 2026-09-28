package server_test

import (
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/look"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

type looked struct {
	Status  int
	Landed  string
	Outline look.Outline
	Scripts *server.Scripted
}

func notePage(t *testing.T, h http.Handler) string {
	t.Helper()
	created := postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Water the plants", "body": "Every Sunday."})
	wantStatus(t, created, http.StatusCreated)
	var note struct{ ID string }
	decode(t, created, &note)
	return "/t/note/" + note.ID
}

// A control says what it holds and which form it is in, and a long page
// can be asked for only what is wanted. What look is not asked in its own
// words is refused, not ignored.
func TestALookSaysValuesFormsAndOnlyWhatIsAsked(t *testing.T) {
	_, h := newApp(t)
	var seen looked
	rec := get(t, h, "/api/look?"+url.Values{"path": {"/chat?prompt=Create a note."}, "only": {"controls"}, "kind": {"textbox"}}.Encode())
	wantStatus(t, rec, http.StatusOK)
	decode(t, rec, &seen)
	if len(seen.Outline.Controls) == 0 || len(seen.Outline.Headings) != 0 || len(seen.Outline.Landmarks) != 0 {
		t.Fatalf("only=controls&kind=textbox keeps the textboxes and nothing else, got %+v", seen.Outline)
	}
	var message look.Control
	for _, c := range seen.Outline.Controls {
		if c.Kind != "textbox" {
			t.Errorf("kind=textbox keeps only textboxes, got %+v", c)
		}
		if strings.HasPrefix(c.Name, "Your message") {
			message = c
		}
	}
	if message.Value != "Create a note." || message.Form == 0 {
		t.Errorf("the composer says what it holds and which form it is in, got %+v", message)
	}

	rec = postJSON(t, h, http.MethodPost, "/api/look", map[string]any{"path": "/chat", "scripted": true})
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "scripted") {
		t.Errorf("a field look does not know is refused by name, got %d %s", rec.Code, rec.Body.String())
	}
	rec = get(t, h, "/api/look?path=/chat&only=everything")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("only names the sections it can keep, got %d", rec.Code)
	}
}

func needBrowser(t *testing.T) {
	t.Helper()
	if _, err := look.FindBrowser(); err != nil {
		t.Skip(err)
	}
}

// With its scripts run, a page is read as it stands after what a person
// does there: the editor a script builds is on it, with its values, Tab
// is pressed for real to say where focus goes, and whatever went wrong
// on the page is said.
func TestALookRunsThePageScriptsAndDoesWhatAPersonDoes(t *testing.T) {
	needBrowser(t)
	_, h := newApp(t)
	page := notePage(t, h)

	var seen looked
	rec := postJSON(t, h, http.MethodPost, "/api/look", map[string]any{"path": page, "steps": []map[string]any{{"press": "Edit"}}})
	wantStatus(t, rec, http.StatusOK)
	decode(t, rec, &seen)
	if seen.Scripts == nil || !slices.Equal(seen.Scripts.Did, []string{"pressed button: Edit block"}) {
		t.Fatalf("the step presses the Edit button, not the region it is in, got %+v", seen.Scripts)
	}
	var body *look.Control
	for i, c := range seen.Outline.Controls {
		// The Markdown source is there too, not shown until asked for.
		if c.Kind == "textbox" && c.Name == "Body" && !c.Hidden {
			body = &seen.Outline.Controls[i]
		}
	}
	if body == nil || body.Hidden || !strings.Contains(body.Value, "Every Sunday.") || body.Form == 0 {
		t.Errorf("the editor the script builds is read, its body shown and holding the words, got %+v", body)
	}
	if !slices.Contains(seen.Scripts.FocusOrder, "textbox: Body") || seen.Scripts.FocusOrder[0] != "link: Skip to main content" {
		t.Errorf("Tab goes from the top of the page and reaches the body, got %v", seen.Scripts.FocusOrder)
	}
	if len(seen.Scripts.Errors) != 0 {
		t.Errorf("nothing goes wrong on the page, got %v", seen.Scripts.Errors)
	}

	rec = postJSON(t, h, http.MethodPost, "/api/look", map[string]any{"path": page, "steps": []map[string]any{{"press": "Launch the rocket"}}})
	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "button: Edit block") {
		t.Errorf("a step that finds nothing says what the page has, got %d %s", rec.Code, rec.Body.String())
	}
}
