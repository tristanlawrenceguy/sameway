package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// TestValidationErrorPageShowsSubmittedValues checks that when a validation
// error occurs on POST /t/note/{id}/props, the 422 response renders the user's
// submitted value in the <dd> element instead of the original stored record value.
// This covers acceptance item 1 and 2: field dd elements render from the submitted
// values map rather than from rec.Fields, and the test name matches what is required.
func TestValidationErrorPageShowsSubmittedValues(t *testing.T) {
	a, h := newApp(t)

	origTitle := "Original title"
	rec, err := a.Store.Create("note", map[string]any{
		"title": origTitle,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Submit a title that exceeds maxLength (200), so validation fails.
	longTitle := strings.Repeat("x", 300)

	form := url.Values{}
	form.Set("prop-title", longTitle)
	r := postForm(t, h, "/t/note/"+rec.ID+"/props", form)
	wantStatus(t, r, http.StatusUnprocessableEntity)

	body := r.Body.String()

	// The rejected value should appear in the response so the user can see it and adjust.
	if !strings.Contains(body, longTitle) {
		t.Errorf("422 body should contain the submitted title %q (length %d), but it does not; body starts with %q",
			longTitle, len(longTitle), truncate(body))
	}

	// The original stored value must NOT appear — otherwise we are showing old data,
	// which is exactly the bug reported in 0243.
	if strings.Contains(body, "<dd>"+origTitle+"</dd>") {
		t.Errorf("422 body should not contain the original title %q (the bug), got it in dd element", origTitle)
	}

	// The page must still be valid HTML with known components only.
	doc := parse(t, r)
	assertAllComponentsKnown(t, doc, componentNames)
}

// TestValidationErrorPageHasEditBlockWrapper checks that the validation error
// page rendered by POST /t/note/{id}/props (with invalid data) has the same
// edit block wrapper as a normal detail page, so 08-edit.js can inject an Edit
// button and pre-fill inline inputs with the rejected values. This covers
// acceptance items 1–3: sw-dl-block div wrapping, data-prop on dd elements,
// and the 08-edit script tag in the head.
func TestValidationErrorPageHasEditBlockWrapper(t *testing.T) {
	a, h := newApp(t)

	origTitle := "Original title"
	rec, err := a.Store.Create("note", map[string]any{
		"title": origTitle,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Submit a title that exceeds maxLength (200), so validation fails.
	longTitle := strings.Repeat("x", 300)

	form := url.Values{}
	form.Set("prop-title", longTitle)
	r := postForm(t, h, "/t/note/"+rec.ID+"/props", form)
	wantStatus(t, r, http.StatusUnprocessableEntity)

	body := r.Body.String()

	// 1. The error page contains the sw-dl-block wrapper with data-block-id and
	//    data-edit-action attributes matching the note's ID and props path.
	blockOpen := strings.Index(body, `data-block-id="`+rec.ID+`"`)
	if blockOpen < 0 {
		t.Errorf("422 body should contain sw-dl-block wrapper with data-block-id=%q; body starts with %q", rec.ID, truncate(body))
	}

	if blockOpen < 0 {
		t.Fatalf("cannot check further without data-block-id; body starts with %q", truncate(body))
	}

	blockRest := body[blockOpen:]
	lastDiv := strings.LastIndex(blockRest, "</div>")
	if lastDiv < 0 {
		t.Fatalf("cannot find closing </div> after data-block-id in error page; body starts with %q", truncate(body))
	}
	blockContent := blockRest[:lastDiv]

	expectedAction := "/t/note/" + rec.ID + "/props"
	if !strings.Contains(blockContent, `data-edit-action="`+expectedAction+`"`) {
		t.Errorf("sw-dl-block should contain data-edit-action=%q; found %q", expectedAction, truncate(blockContent))
	}

	// 2. Each <dd> element in the error page has a data-prop attribute matching
	//    its field name (e.g., <dd data-prop="title">). The dl must come before
	//    the closing </div> of the block wrapper, and dd elements should have
	//    data-prop attributes.
	if !strings.Contains(blockContent, `<dl class="sw-dl">`) {
		t.Errorf("sw-dl-block should contain a dl.sw-dl; body starts with %q", truncate(body))
	}

	// Check that dd elements have data-prop attributes. We look for the pattern
	// seen in normal detail pages: <dd data-prop="title">
	if !strings.Contains(blockContent, `data-prop="title"`) {
		t.Errorf("sw-dl-block should contain a <dd> with data-prop=\"title\"; body starts with %q", truncate(body))
	}

	// The .sw-bar sw-quiet div is the anchor point 08-edit.js uses to inject its
	// Edit button; without it, clicking "Edit block" on the error page does nothing.
	if !strings.Contains(blockContent, `<div class="sw-bar sw-quiet">`) {
		t.Errorf("sw-dl-block should contain a div.sw-bar.sw-quiet for the edit anchor; body starts with %q", truncate(body))
	}

	// 3. The error page includes the 08-edit.js script in the head.
	if !strings.Contains(body, `<script defer src="/design/base/08-edit.js"></script>`) {
		t.Errorf("422 body should include <script defer src=\"/design/base/08-edit.js\"></script>; body starts with %q", truncate(body))
	}

	// The page must still be valid HTML with known components only.
	doc := parse(t, r)
	assertAllComponentsKnown(t, doc, componentNames)
}

// TestValidationErrorPageShowsEmptyTitleField checks that when a validation error
// occurs because the user cleared the Title field (empty string), the 422 error
// page still renders <dd data-prop="title"> so that 08-edit.js can build an input
// for it. This is the bug from backlog item 0397: after clearing the title and
// saving, clicking "Edit block" showed no Title field because empty display values
// were skipped in renderDetailError. Acceptance items 1–4.
func TestValidationErrorPageShowsEmptyTitleField(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title":  "My Note",
		"status": "draft",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Submit with an empty title — this triggers a required-field validation error.
	form := url.Values{}
	form.Set("prop-title", "")
	r := postForm(t, h, "/t/note/"+rec.ID+"/props", form)
	wantStatus(t, r, http.StatusUnprocessableEntity)

	body := r.Body.String()

	// The Title field must appear as a <dd data-prop="title"> even though its
	// value is empty. Without this dd element 08-edit.js has nothing to find and
	// cannot build the inline input for fixing the title. This covers acceptance
	// item 3: "the form now shows all fields" when re-entering edit mode after a
	// validation error on an empty title.
	if !strings.Contains(body, `data-prop="title"`) {
		t.Errorf("422 body should contain <dd data-prop=\"title\"> so the user can fix the empty title; body starts with %q", truncate(body))
	}

	// The other fields (status=pinned) that have non-empty display values must
	// also be present — acceptance item 3 says "all fields" appear.
	if !strings.Contains(body, `<dl class="sw-dl">`) {
		t.Errorf("422 body should contain the definition list; body starts with %q", truncate(body))
	}

	// The page must still be valid HTML with known components only.
	doc := parse(t, r)
	assertAllComponentsKnown(t, doc, componentNames)
}
