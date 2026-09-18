package server_test

import (
	"strings"
	"testing"
)

// TestUploadFormHasSubmitButton checks that the files list page renders an
// upload form with a visible "Upload" submit button, not "Add a file".
func TestUploadFormHasSubmitButton(t *testing.T) {
	_, h := newApp(t)
	rec := get(t, h, "/t/file")
	wantStatus(t, rec, 200)
	body := rec.Body.String()

	if !strings.Contains(body, `<button type="submit" class="sw-button sw-button--primary sw-pressable">Upload</button>`) {
		t.Errorf("upload form should have an Upload submit button\nbody: %s", truncate(body))
	}
	if strings.Contains(body, `>Add a file<`) {
		t.Error("upload button should say \"Upload\", not \"Add a file\"")
	}
}

// TestUploadFormHasAccessibleErrorRegion checks that the upload form contains
// an element with aria-live="assertive" so error messages are announced by
// screen readers when a file upload fails.
func TestUploadFormHasAccessibleErrorRegion(t *testing.T) {
	_, h := newApp(t)
	rec := get(t, h, "/t/file")
	wantStatus(t, rec, 200)
	body := rec.Body.String()

	if !strings.Contains(body, `aria-live="assertive"`) {
		t.Errorf("upload form should contain an aria-live=\"assertive\" region for screen-reader error announcements\nbody: %s", truncate(body))
	}
}
