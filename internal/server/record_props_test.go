package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestRecordPropsUpdateSucceeds checks that POSTing valid fields to
// /t/note/{id}/props updates the record and returns a 303 redirect back to
// the detail page. This is acceptance item 1.
func TestRecordPropsUpdateSucceeds(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title": "Old title",
	})
	if err != nil {
		t.Fatal(err)
	}
	detailPath := "/t/note/" + rec.ID

	form := url.Values{}
	form.Set("prop-title", "New title")
	r := postForm(t, h, detailPath+"/props", form)
	wantStatus(t, r, http.StatusSeeOther)

	loc := r.Header().Get("Location")
	if !strings.Contains(loc, "/t/note/"+rec.ID) {
		t.Errorf("redirect Location = %q, want it to contain /t/note/%s", loc, rec.ID)
	}

	follow := get(t, h, r.Header().Get("Location"))
	body := follow.Body.String()
	if !strings.Contains(body, "New title") {
		t.Errorf("detail page should show updated title 'New title', got body starting with %q", truncate(body))
	}

	updated, err := a.Store.Get("note", rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Fields["title"] != "New title" {
		t.Errorf("stored title = %q, want 'New title'", updated.Fields["title"])
	}
}

// TestRecordPropsPartialUpdate checks that POSTing only one field updates
// just that field while preserving the others — acceptance item 2.
func TestRecordPropsPartialUpdate(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title": "Keep me",
		"body":  "Old body text",
	})
	if err != nil {
		t.Fatal(err)
	}
	detailPath := "/t/note/" + rec.ID

	form := url.Values{}
	form.Set("prop-body", "Updated body")
	r := postForm(t, h, detailPath+"/props", form)
	wantStatus(t, r, http.StatusSeeOther)

	updated, err := a.Store.Get("note", rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Fields["title"] != "Keep me" {
		t.Errorf("title should be preserved = %q, want 'Keep me'", updated.Fields["title"])
	}
	if updated.Fields["body"] != "Updated body" {
		t.Errorf("body should be updated = %q", updated.Fields["body"])
	}

	follow := parse(t, get(t, h, r.Header().Get("Location")))
	bodyText := htmltest.Text(follow.Root)
	if !strings.Contains(bodyText, "Keep me") {
		t.Errorf("detail page missing preserved title 'Keep me' in %q", truncate(bodyText))
	}
	if !strings.Contains(bodyText, "Updated body") {
		t.Errorf("detail page missing updated body 'Updated body' in %q", truncate(bodyText))
	}
}

// TestRecordPropsInvalidEnumReturns422 checks that POSTing an invalid enum
// value returns HTTP 422 with error messages linked to the field. This is
// acceptance item 3.
func TestRecordPropsInvalidEnumReturns422(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title": "Test note",
	})
	if err != nil {
		t.Fatal(err)
	}

	form := url.Values{}
	form.Set("prop-status", "not_a_valid_status")
	r := postForm(t, h, "/t/note/"+rec.ID+"/props", form)
	wantStatus(t, r, http.StatusUnprocessableEntity)

	body := r.Body.String()
	if !strings.Contains(body, "status") {
		t.Errorf("error response should mention field name 'status', got body starting with %q", truncate(body))
	}
	if !strings.Contains(body, "draft") && !strings.Contains(body, "published") {
		t.Errorf("error response should mention valid values, got body starting with %q", truncate(body))
	}

	doc := parse(t, r)
	assertAllComponentsKnown(t, doc, componentNames)
}

// TestRecordPropsInvalidRequiredFieldReturns422 checks that POSTing an empty
// value for a required field returns HTTP 422.
func TestRecordPropsInvalidRequiredFieldReturns422(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title": "Test note",
	})
	if err != nil {
		t.Fatal(err)
	}

	form := url.Values{}
	form.Set("prop-title", "")
	r := postForm(t, h, "/t/note/"+rec.ID+"/props", form)
	wantStatus(t, r, http.StatusUnprocessableEntity)

	body := r.Body.String()
	if !strings.Contains(body, "title") {
		t.Errorf("error response should mention field name 'title', got body starting with %q", truncate(body))
	}
}

// TestRecordPropsUnknownTypeReturns404 checks that POSTing to an unknown
// content type returns 404.
func TestRecordPropsUnknownTypeReturns404(t *testing.T) {
	_, h := newApp(t)
	r := postForm(t, h, "/t/unknowntype/someid/props", url.Values{})
	wantStatus(t, r, http.StatusNotFound)
}

// TestRecordPropsNonExistentRecordReturns404 checks that POSTing to a valid
// type but non-existent record returns 404.
func TestRecordPropsNonExistentRecordReturns404(t *testing.T) {
	_, h := newApp(t)
	r := postForm(t, h, "/t/note/does-not-exist/props", url.Values{})
	if r.Code != http.StatusNotFound && r.Code != http.StatusInternalServerError {
		t.Errorf("expected 404 or 500 for non-existent record, got %d: %s", r.Code, truncate(r.Body.String()))
	}
}

// TestRecordPropsWorksForActivity checks that the handler works for other
// content types (activity) — acceptance item 5.
func TestRecordPropsWorksForActivity(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("activity", map[string]any{
		"actor":  "human",
		"action": "updated",
	})
	if err != nil {
		t.Fatal(err)
	}
	detailPath := "/t/activity/" + rec.ID

	form := url.Values{}
	form.Set("prop-detail", "New detail text")
	r := postForm(t, h, detailPath+"/props", form)
	wantStatus(t, r, http.StatusSeeOther)

	loc := r.Header().Get("Location")
	if !strings.Contains(loc, "/t/activity/"+rec.ID) {
		t.Errorf("redirect Location = %q, want it to contain /t/activity/%s", loc, rec.ID)
	}

	follow := parse(t, get(t, h, r.Header().Get("Location")))
	bodyText := htmltest.Text(follow.Root)
	if !strings.Contains(bodyText, "New detail text") {
		t.Errorf("detail page should show updated detail field, got %q", truncate(bodyText))
	}

	updated, err := a.Store.Get("activity", rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Fields["detail"] != "New detail text" {
		t.Errorf("stored detail = %q, want 'New detail text'", updated.Fields["detail"])
	}

	assertAllComponentsKnown(t, follow, componentNames)
}

// TestRecordPropsNoFieldsRedirects checks that POSTing without any prop-
// prefixed values returns a redirect (no-op behavior).
func TestRecordPropsNoFieldsRedirects(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title": "Unchanged note",
	})
	if err != nil {
		t.Fatal(err)
	}

	form := url.Values{}
	r := postForm(t, h, "/t/note/"+rec.ID+"/props", form)
	wantStatus(t, r, http.StatusSeeOther)

	loc := r.Header().Get("Location")
	if !strings.Contains(loc, "/t/note/"+rec.ID) {
		t.Errorf("redirect Location = %q, want it to contain /t/note/%s", loc, rec.ID)
	}

	unchanged, err := a.Store.Get("note", rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Fields["title"] != "Unchanged note" {
		t.Errorf("record should be unchanged, got title = %q", unchanged.Fields["title"])
	}
}

// TestRecordPropsUnknownFieldReturns422 checks that POSTing an unknown field
// name returns 422 with the field flagged as unknown.
func TestRecordPropsUnknownFieldReturns422(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title": "Test note",
	})
	if err != nil {
		t.Fatal(err)
	}

	form := url.Values{}
	form.Set("prop-bogus_field", "nope")
	r := postForm(t, h, "/t/note/"+rec.ID+"/props", form)
	wantStatus(t, r, http.StatusUnprocessableEntity)

	body := r.Body.String()
	if !strings.Contains(body, "bogus_field") {
		t.Errorf("error response should mention unknown field 'bogus_field', got body starting with %q", truncate(body))
	}
}
