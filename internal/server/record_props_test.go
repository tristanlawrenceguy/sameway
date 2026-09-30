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

// TestRecordPropsInvalidEnumReturns422 checks that an invalid enum value is
// refused, and the page the person is back on names the field and the
// values it takes. This is acceptance item 3.
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
	page := after(t, h, r)
	body := page.Body.String()
	if !strings.Contains(body, "Status ") {
		t.Errorf("the refusal should name the field, Status, got body starting with %q", truncate(body))
	}
	if !strings.Contains(body, "draft") && !strings.Contains(body, "published") {
		t.Errorf("error response should mention valid values, got body starting with %q", truncate(body))
	}

	doc := parse(t, page)
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
	page := after(t, h, r)
	body := page.Body.String()
	if !strings.Contains(body, "Title is required") {
		t.Errorf("error response should show 'Title is required', got body starting with %q", truncate(body))
	}
}

// TestRecordPropsUnknownTypeReturns404 checks that POSTing to an unknown
// content type returns 404.
func TestRecordPropsUnknownTypeReturns404(t *testing.T) {
	_, h := newApp(t)
	r := postForm(t, h, "/t/unknowntype/someid/props", url.Values{})
	wantStatus(t, r, http.StatusNotFound)
}

// TestRecordPropsNonExistentRecordReturns404 checks that editing a record
// that is gone says so on a page the person can go on from, not on a bare
// error page.
func TestRecordPropsNonExistentRecordReturns404(t *testing.T) {
	_, h := newApp(t)
	r := postForm(t, h, "/t/note/does-not-exist/props", url.Values{})
	page, at := landed(t, h, r)
	if at != "/t/note" || !strings.Contains(page.Body.String(), "It is not there any more") {
		t.Errorf("a gone record is said on the list, got %q: %s", at, truncate(page.Body.String()))
	}
}

// The log is kept by Sameway: an entry's page is not a way to edit it,
// as the API and the command line are not. It is undone instead.
func TestRecordPropsKeepsTheLog(t *testing.T) {
	a, h := newApp(t)
	rec, err := a.Store.Create("activity", map[string]any{"actor": "human", "action": "updated"})
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{}
	form.Set("prop-detail", "New detail text")
	postForm(t, h, "/t/activity/"+rec.ID+"/props", form)
	if updated, _ := a.Store.Get("activity", rec.ID); updated.Fields["detail"] == "New detail text" {
		t.Error("an entry in the log is not edited from its page")
	}
	page := get(t, h, "/t/activity/"+rec.ID).Body.String()
	if strings.Contains(page, "data-edit-action") || strings.Contains(page, "/delete\"") {
		t.Error("an entry's page offers no Edit and no Delete")
	}
}

// TestRecordPropsNoFieldsSaysSo checks that POSTing without any prop-
// prefixed values is refused in words, not passed over with an empty
// redirect: an agent that sent title=... got a 303 and nothing done.
func TestRecordPropsNoFieldsSaysSo(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title": "Unchanged note",
	})
	if err != nil {
		t.Fatal(err)
	}

	r := postForm(t, h, "/t/note/"+rec.ID+"/props", url.Values{"title": {"Changed"}})
	wantStatus(t, r, http.StatusBadRequest)
	if body := r.Body.String(); !strings.Contains(body, "prop-<name>") || !strings.Contains(body, "send prop-title") {
		t.Errorf("the answer says what the form takes: %q", body)
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
	page := after(t, h, r)
	body := page.Body.String()
	if !strings.Contains(body, "Bogus field") {
		t.Errorf("error response should show 'Bogus field', got body starting with %q", truncate(body))
	}
}
