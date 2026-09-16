package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// TestRecordPropsValidationErrorShowsRejectedValue checks that when a validation
// error occurs on POST /t/note/{id}/props, the 422 response renders the user's
// submitted value in the <dd> element instead of the original stored record value.
// This covers acceptance item 3: field dd elements render from the submitted values
// map rather than from rec.Fields.
func TestRecordPropsValidationErrorShowsRejectedValue(t *testing.T) {
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
