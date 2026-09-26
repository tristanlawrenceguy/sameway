package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// An edit refused comes back with its problems as a list whose title says
// Error in words, on a page whose window title starts Error:, and a
// problem with no field to lead to is plain words, not a link to nowhere.
func TestARefusedEditSaysErrorWhereItIsHeardFirst(t *testing.T) {
	a, h := newApp(t)
	rec, _ := a.Store.Create("note", map[string]any{"title": "Water the plants"})
	r := withReferer(t, h, http.MethodPost, "/t/note/"+rec.ID+"/props", "/t/note/"+rec.ID, url.Values{"prop-title": {""}, "prop-colour": {"red"}}.Encode(), "application/x-www-form-urlencoded")
	page, _ := landed(t, h, r)
	body := page.Body.String()
	if !strings.Contains(body, "<title>Error: ") {
		t.Errorf("the window title starts Error:\n%s", truncate(body))
	}
	if !strings.Contains(body, `<span class="sw-visually-hidden">Error: </span>Not saved</h2>`) {
		t.Errorf("the summary's title says Error in words, as a danger alert does\n%s", truncate(body))
	}
	if !strings.Contains(body, `<li id="error-summary-1">Colour is not a field of a note.</li>`) {
		t.Errorf("a problem with no field is said in plain words, not a link to nowhere\n%s", truncate(body))
	}
	if again := get(t, h, "/t/note/"+rec.ID).Body.String(); strings.Contains(again, "<title>Error: ") {
		t.Errorf("once said, the page's title is its own again")
	}
}
