package server_test

import (
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
)

// A refused edit never costs the person what they typed. The server says
// why on the page they edited from and names the edit it answers; the
// page keeps the draft of that edit until it is said to be saved, and
// opens the editor again, holding the draft, when it was refused.

// refusedEdit posts a title too long to keep, from the note's own page.
func refusedEdit(t *testing.T) (h http.Handler, id string, page string, at string) {
	t.Helper()
	a, h := newApp(t)
	rec, err := a.Store.Create("note", map[string]any{"title": "Original title"})
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{"prop-title": {strings.Repeat("x", 300)}}
	r := withReferer(t, h, http.MethodPost, "/t/note/"+rec.ID+"/props", "/t/note/"+rec.ID, form.Encode(), "application/x-www-form-urlencoded")
	p, at := landed(t, h, r)
	return h, rec.ID, p.Body.String(), at
}

// TestValidationErrorPageShowsSubmittedValues: the person is back on the
// page they edited, told why, as an alert, and nothing was stored.
func TestValidationErrorPageShowsSubmittedValues(t *testing.T) {
	h, id, body, at := refusedEdit(t)
	if at != "/t/note/"+id {
		t.Errorf("a refused edit returns to the page it was made on, not an internal address, got %q", at)
	}
	if !strings.Contains(body, `data-outcome="failed"`) || !strings.Contains(body, `role="alert"`) || !strings.Contains(body, "Title ") {
		t.Errorf("the page says, as an alert, which field was refused; body starts with %q", truncate(body))
	}
	if !strings.Contains(get(t, h, "/api/note/"+id).Body.String(), "Original title") {
		t.Error("a refused edit stores nothing")
	}
}

// TestValidationErrorPageHasEditBlockWrapper: the outcome names the edit
// it answers exactly as the page's editable block does, which is how the
// page finds the draft to give back.
func TestValidationErrorPageHasEditBlockWrapper(t *testing.T) {
	_, id, body, _ := refusedEdit(t)
	action := "/t/note/" + id + "/props"
	if !strings.Contains(body, `data-outcome-for="`+action+`"`) || !strings.Contains(body, `data-edit-action="`+action+`"`) {
		t.Errorf("the outcome and the block both name %s; body starts with %q", action, truncate(body))
	}
	if !strings.Contains(body, `<script defer src="/design/base/08-edit.js"></script>`) {
		t.Error("the page carries the editor, to open again")
	}
}

// TestValidationErrorPageShowsEmptyTitleField: the draft is not let go
// when Save is pressed, only when the page says the edit was saved, and
// a refused edit opens again.
func TestValidationErrorPageShowsEmptyTitleField(t *testing.T) {
	src, err := os.ReadFile("../../design/base/16-drafts.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(src)
	if !strings.Contains(js, `form.addEventListener("submit", function () { set(k, fields(form)); });`) {
		t.Error("Save must not throw the draft away before the page says it was saved")
	}
	for _, want := range []string{`.sw-outcome[data-outcome-for]`, `getAttribute("data-outcome") === "done"`, `edit.click()`} {
		if !strings.Contains(js, want) {
			t.Errorf("16-drafts.js should settle drafts by the outcome: missing %s", want)
		}
	}
}
