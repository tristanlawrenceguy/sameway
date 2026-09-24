package server_test

// Tests for no "Please" prefix in inline error messages (task 0186).
// Inline scripts must not say "Please select a file." — just "Select a file.".
// Acceptance item 1.

import (
	"net/http"
	"strings"
	"testing"
)

// TestUploadFormErrorHasNoPleasePrefix checks that the upload form's inline
// validation script does not use "Please select a file." but instead says just
// "Select a file.". Acceptance item 1.
func TestUploadFormErrorHasNoPleasePrefix(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/t/file")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if strings.Contains(body, "Please select a file.") {
		t.Errorf("/t/file: inline script must not use \"Please\" prefix in error message;\nwant: \"Select a file.\" or similar without \"Please\"\nbody: %s", truncate(body))
	}

	// The essential message must still be present.
	if !strings.Contains(body, "select a file") {
		t.Errorf("/t/file: inline script should say to select a file;\nbody: %s", truncate(body))
	}
}

// TestImportFormErrorHasNoPleasePrefix checks that the import form's inline
// validation script does not use "Please select a file." but instead says just
// "Select a file.". Acceptance item 1.
func TestImportFormErrorHasNoPleasePrefix(t *testing.T) {
	_, h := newApp(t)

	for _, typ := range []string{"note", "action", "task"} {
		rec := get(t, h, "/t/"+typ+"/import")
		wantStatus(t, rec, http.StatusOK)
		body := rec.Body.String()

		if strings.Contains(body, "Please select a file.") {
			t.Errorf("/t/%s/import: inline script must not use \"Please\" prefix in error message;\nwant: \"Select a file.\" or similar without \"Please\"\nbody: %s", typ, truncate(body))
		}

		// The essential message must still be present.
		if !strings.Contains(body, "select a file") {
			t.Errorf("/t/%s/import: inline script should say to select a file;\nbody: %s", typ, truncate(body))
		}
	}
}
