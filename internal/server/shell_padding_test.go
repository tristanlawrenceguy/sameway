package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// TestShellStylesheetHasAdequateTopPadding checks that .sw-main's top padding is
// large enough to clear the sticky header, preventing overlap with controls like
// "Edit block" and "Delete note". The acceptance test requires 4rem (var(--sw-space-16))
// so content starts below the ~4.25rem tall header when scrolled to the page top.
// This covers backlog 0252: sticky header intercepts pointer events on detail pages.
func TestShellStylesheetHasAdequateTopPadding(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/design/sameway.css")
	wantStatus(t, rec, http.StatusOK)

	body := rec.Body.String()

	// The fix: .sw-main must use var(--sw-space-16) as its top padding value.
	if !strings.Contains(body, ".sw-main { padding-block: var(--sw-space-16) var(--sw-space-16)") {
		t.Errorf("stylesheet should set .sw-main padding-block to var(--sw-space-16) var(--sw-space-16)\n"+
			"so the sticky header does not overlap controls on note detail pages\n%s", truncate(body))
	}

	// The old value must be gone — a leftover var(--sw-space-8) here would mean
	// the fix was only partial.
	if strings.Contains(body, ".sw-main { padding-block: var(--sw-space-8)") {
		t.Errorf("stylesheet still has old .sw-main top padding of var(--sw-space-8)\n"+
			"which is too small to clear the sticky header\n%s", truncate(body))
	}

	// The sticky header must remain sticky — we are not changing its position.
	if !strings.Contains(body, ".sw-header") && !strings.Contains(body, "position: sticky") {
		t.Errorf("sticky header positioning should still be present in stylesheet")
	}
}
