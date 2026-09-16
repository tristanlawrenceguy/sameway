package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestRecordPropsWorksForProposal checks that the handler works for proposal
// records — acceptance item 5.
func TestRecordPropsWorksForProposal(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("proposal", map[string]any{
		"summary": "Test proposal?",
		"action":  `{"tool":"foo"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	detailPath := "/t/proposal/" + rec.ID

	form := url.Values{}
	form.Set("prop-state", "accepted")
	r := postForm(t, h, detailPath+"/props", form)
	wantStatus(t, r, http.StatusSeeOther)

	follow := parse(t, get(t, h, r.Header().Get("Location")))
	bodyText := htmltest.Text(follow.Root)
	if !strings.Contains(bodyText, "accepted") {
		t.Errorf("detail page should show updated state 'accepted', got %q", truncate(bodyText))
	}

	assertAllComponentsKnown(t, follow, componentNames)
}

// TestRecordPropsPreservesExistingFields checks that fields not included in the
// POST body keep their original values after a successful update.
func TestRecordPropsPreservesExistingFields(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title":  "My note",
		"body":   "Some body text",
		"status": "draft",
		"pinned": true,
	})
	if err != nil {
		t.Fatal(err)
	}

	form := url.Values{}
	form.Set("prop-title", "Updated title")
	r := postForm(t, h, "/t/note/"+rec.ID+"/props", form)
	wantStatus(t, r, http.StatusSeeOther)

	follow := parse(t, get(t, h, r.Header().Get("Location")))
	bodyText := htmltest.Text(follow.Root)

	if !strings.Contains(bodyText, "Updated title") {
		t.Errorf("detail page missing new title 'Updated title' in %q", truncate(bodyText))
	}
	if !strings.Contains(bodyText, "Some body text") {
		t.Errorf("detail page missing preserved body 'Some body text' in %q", truncate(bodyText))
	}
	if !strings.Contains(bodyText, "draft") {
		t.Errorf("detail page missing preserved status 'draft' in %q", truncate(bodyText))
	}

	assertAllComponentsKnown(t, follow, componentNames)
}

// TestRecordPropsRedirectsToCorrectType verifies the redirect goes to the same
// type that was requested — e.g. POST /t/activity/... returns 303 to
// /t/activity/{id}. This ensures cross-type correctness.
func TestRecordPropsRedirectsToCorrectType(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("activity", map[string]any{
		"actor":  "assistant",
		"action": "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	form := url.Values{}
	form.Set("prop-detail", "Redirect check")
	r := postForm(t, h, "/t/activity/"+rec.ID+"/props", form)
	wantStatus(t, r, http.StatusSeeOther)

	loc := r.Header().Get("Location")
	if !strings.Contains(loc, "/t/activity/") {
		t.Errorf("redirect Location = %q, want it to contain /t/activity/", loc)
	}
	if !strings.Contains(loc, rec.ID) {
		t.Errorf("redirect Location = %q, want it to contain the record ID %s", loc, rec.ID)
	}
}
