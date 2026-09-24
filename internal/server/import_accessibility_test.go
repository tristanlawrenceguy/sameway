package server_test

import (
	"strings"
	"testing"
)

// TestImportPageHasAccessibleErrorRegion checks that every import page
// renders an accessible error region next to the file input, linked via
// aria-describedby so a screen reader announces "select a file."
// when the user submits without choosing anything.  Covers backlog 0433.
func TestImportPageHasAccessibleErrorRegion(t *testing.T) {
	_, h := newApp(t)

	for _, typ := range []string{"note", "action", "task"} {
		rec := get(t, h, "/t/"+typ+"/import")
		wantStatus(t, rec, 200)
		body := rec.Body.String()

		if !strings.Contains(body, `id="import-file"`) {
			t.Errorf("/t/%s/import should have file input with id=import-file", typ)
		}
		if !strings.Contains(body, `aria-describedby="import-error"`) {
			t.Errorf("/t/%s/import: file input should link error via aria-describedby=\"import-error\"", typ)
		}
		if !strings.Contains(body, `id="import-error"`) {
			t.Errorf("/t/%s/import should contain an element with id=import-error", typ)
		}
		if !strings.Contains(body, `aria-live="assertive"`) {
			t.Errorf("/t/%s/import: error region should have aria-live=\"assertive\"", typ)
		}
		if !strings.Contains(body, "select a file.") {
			t.Errorf("/t/%s/import: inline script must include the text \"select a file.\"", typ)
		}
		if !strings.Contains(body, "reportValidity") {
			t.Errorf("/t/%s/import: inline script must call reportValidity() for sighted users", typ)
		}
	}
}
