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

// TestUploadFormHasValidationScript checks that the files list page renders an
// inline script after the upload form which intercepts submission when no file
// is selected, sets "select a file." on the aria-live region, and
// then calls reportValidity() so the browser still shows its native tooltip.
// This covers acceptance item 2: the status element with id upload-error and
// aria-live="assertive" receives the error text when validation fails.
func TestUploadFormHasValidationScript(t *testing.T) {
	_, h := newApp(t)
	rec := get(t, h, "/t/file")
	wantStatus(t, rec, 200)
	body := rec.Body.String()

	// The page must contain an inline script tag.
	if !strings.Contains(body, "<script>") || !strings.Contains(body, "</script>") {
		t.Error("files list page should include an inline <script> for upload validation")
	}

	// The script must target the sw-upload form.
	if !strings.Contains(body, "sw-upload") {
		t.Error(`inline script should reference the "sw-upload" form class`)
	}

	// The script must set text on an element with role="status".
	if !strings.Contains(body, `role="status"`) {
		t.Error(`inline script should target an element with role="status" for aria-live announcements`)
	}

	// The script must include the exact error message text.
	if !strings.Contains(body, "select a file.") {
		t.Error(`inline script must set the text "select a file." on the live region`)
	}

	// The script must call reportValidity() so sighted users still see the
	// browser-native tooltip (acceptance item 3).
	if !strings.Contains(body, "reportValidity") {
		t.Error(`inline script must call reportValidity() to preserve the native tooltip for sighted users`)
	}

	// The upload-error element id must exist in the page.
	if !strings.Contains(body, `id="upload-error"`) {
		t.Error("page should contain an element with id=\"upload-error\"")
	}
}

// TestUploadValidationScriptOnlyOnFilesList ensures the inline upload
// validation script is rendered only on /t/file, not on other listing pages.
// This keeps the change scoped to acceptance item 4 (no changes outside /t/file).
func TestUploadValidationScriptOnlyOnFilesList(t *testing.T) {
	_, h := newApp(t)

	// The files page must have the script.
	fileRec := get(t, h, "/t/file")
	wantStatus(t, fileRec, 200)
	fileBody := fileRec.Body.String()
	if !strings.Contains(fileBody, "reportValidity") {
		t.Error("/t/file should include the upload validation inline script")
	}

	// A non-file listing page must NOT have it.
	noteRec := get(t, h, "/t/note")
	wantStatus(t, noteRec, 200)
	noteBody := noteRec.Body.String()
	if strings.Contains(noteBody, "reportValidity") {
		t.Error("/t/note should not contain the upload validation script (out of scope)")
	}

	if strings.Contains(noteBody, `id="upload-error"`) {
		t.Error("/t/note should not contain the upload-error element id")
	}
}
