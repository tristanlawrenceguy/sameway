package server_test

import (
	"strings"
	"testing"
)

// TestUploadFormHasSubmitButton checks that the files list page renders an
// upload form with a visible "Upload" submit button, not "Add a file".
func TestUploadFormHasSubmitButton(t *testing.T) {
	t.Parallel()
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

// The upload form says what it takes before anyone chooses, and has a place
// above the field for a problem, tied to the field and heard politely.
func TestUploadFormSaysWhatItTakesAndWhereProblemsGo(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	body := get(t, h, "/t/file").Body.String()
	for _, want := range []string{"Up to 4 GB.", `class="sw-upload__error" id="upload-error" aria-live="polite"`, `aria-describedby="upload-hint upload-error"`, `data-busy-target="upload-status"`} {
		if !strings.Contains(body, want) {
			t.Errorf("the upload form should carry %s: %s", want, truncate(body))
		}
	}
	if strings.Contains(body, "<script>(function(){var f=document.querySelector('.sw-upload__field')") {
		t.Error("the page no longer pastes a script for the form; the component's own script serves every upload form")
	}
}

// What the upload script says of a missing, empty or picture file is
// checked in a browser: tools/a11y-runner/behave-arm.mjs.
