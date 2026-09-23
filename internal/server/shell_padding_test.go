package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// TestShellStylesheetHasAdequateTopPadding checks that where the header is
// sticky (the sidebar on a wide screen), .sw-main's top padding clears it,
// so controls like "Edit block" and "Delete note" are never under it
// (backlog 0252). On a narrow screen the header scrolls away with the page
// instead of pinning itself over most of a phone, so there it covers
// nothing and the page starts closer to the top.
func TestShellStylesheetHasAdequateTopPadding(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/design/sameway.css")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	wide := body[strings.Index(body, "@media (min-width: 64rem)"):]
	if !strings.Contains(wide, "position: sticky") {
		t.Errorf("the header should be sticky on a wide screen\n%s", truncate(body))
	}
	if !strings.Contains(wide, ".sw-main { padding-block: var(--sw-space-16); }") {
		t.Errorf("on a wide screen .sw-main should keep var(--sw-space-16) top padding\n%s", truncate(body))
	}
	shell := body[strings.Index(body, ".sw-header {"):]
	shell = shell[:strings.Index(shell, "}")]
	if strings.Contains(shell, "sticky") {
		t.Errorf("on a narrow screen the header should scroll away, not stick:\n%s", shell)
	}
}
