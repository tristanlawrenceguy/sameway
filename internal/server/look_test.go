package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/look"
)

// An agent reads a page the way a screen reader gets it, does what a
// person does and reads where they land, and reads one component from
// props before adding it, all without a browser.
func TestAnAgentLooksAtAPageWithoutABrowser(t *testing.T) {
	_, h := newApp(t)
	created := postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Water the plants", "body": "Every Sunday."})
	wantStatus(t, created, http.StatusCreated)
	var note struct{ ID string }
	decode(t, created, &note)

	var seen struct {
		Path    string
		Status  int
		Landed  string
		Outline look.Outline
	}
	rec := get(t, h, "/api/look?path=/t/note/"+note.ID)
	wantStatus(t, rec, http.StatusOK)
	decode(t, rec, &seen)
	if seen.Status != 200 || len(seen.Outline.Headings) == 0 || seen.Outline.Headings[0].Level != 1 {
		t.Errorf("the note page should read with its h1, got %+v", seen)
	}
	if len(seen.Outline.Problems) != 0 {
		t.Errorf("the note page should have no structural problems, got %v", seen.Outline.Problems)
	}
	var crumbs bool
	for _, l := range seen.Outline.Landmarks {
		crumbs = crumbs || (l.Role == "navigation" && l.Label == "You are here")
	}
	if !crumbs {
		t.Errorf("the crumbs are a labelled navigation landmark, got %v", seen.Outline.Landmarks)
	}

	// Doing what a person does: an over-long title is refused on the page,
	// and a good one lands them back on the note.
	rec = postJSON(t, h, http.MethodPost, "/api/look", map[string]any{
		"path": "/t/note/" + note.ID + "/props", "form": map[string]string{"prop-title": strings.Repeat("x", 250)},
	})
	wantStatus(t, rec, http.StatusOK)
	decode(t, rec, &seen)
	refused := false
	for _, l := range seen.Outline.Live {
		refused = refused || (l.Politeness == "assertive" && strings.Contains(l.Text, "Not saved"))
	}
	if !strings.HasPrefix(seen.Landed, "/t/note/"+note.ID) || !refused {
		t.Errorf("an invalid edit is refused on the note, said as an alert, got landed %q live %v", seen.Landed, seen.Outline.Live)
	}
	rec = postJSON(t, h, http.MethodPost, "/api/look", map[string]any{
		"path": "/t/note/" + note.ID + "/props", "method": "POST", "form": map[string]string{"prop-title": "Water the garden"},
	})
	decode(t, rec, &seen)
	if !strings.HasPrefix(seen.Landed, "/t/note/"+note.ID) || seen.Outline.Headings[0].Text != "Water the garden" {
		t.Errorf("a good edit lands back on the note with the new title, got landed %q headings %v", seen.Landed, seen.Outline.Headings)
	}

	// One component from props, and a component that does not exist.
	rec = postJSON(t, h, http.MethodPost, "/api/look", map[string]any{"component": "card", "props": map[string]any{"title": "Plan"}})
	wantStatus(t, rec, http.StatusOK)
	var frag struct {
		Component string
		HTML      string
		Outline   look.Outline
	}
	decode(t, rec, &frag)
	if !strings.Contains(frag.HTML, `data-component="card"`) || len(frag.Outline.Problems) != 0 || strings.Join(frag.Outline.Components, ",") != "card" {
		t.Errorf("a card from props should read clean, got %+v", frag)
	}
	rec = postJSON(t, h, http.MethodPost, "/api/look", map[string]any{"component": "nope"})
	wantStatus(t, rec, http.StatusNotFound)
	if !strings.Contains(rec.Body.String(), "card") {
		t.Error("an unknown component should be answered with the ones that exist")
	}

	// A path that is not a page here is refused in the API's own words.
	wantStatus(t, get(t, h, "/api/look?path=http://elsewhere"), http.StatusBadRequest)
}
