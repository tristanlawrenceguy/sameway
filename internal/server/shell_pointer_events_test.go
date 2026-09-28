package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// TestStickyHeaderHasPointerEventsNone checks that .sw-header sets
// pointer-events: none so it does not intercept clicks on controls below
// it — specifically "Edit block" buttons and "Delete note" links.
// This is the real fix for backlog 0252, complementing the top padding
// added in TestShellStylesheetHasAdequateTopPadding.
func TestStickyHeaderHasPointerEventsNone(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/design/sameway.css")
	wantStatus(t, rec, http.StatusOK)

	body := rec.Body.String()

	if !strings.Contains(body, ".sw-header") || !strings.Contains(body, "pointer-events: none") {
		t.Errorf("stylesheet should set pointer-events: none on .sw-header\n"+
			"so the sticky header does not block clicks on elements below it\n%s", truncate(body))
	}
}

// TestStickyHeaderInteractiveChildrenHavePointerEventsAuto checks that
// interactive elements inside .sw-header — the brand link and navigation links —
// still receive pointer events (pointer-events: auto), because they must remain
// clickable even though the header itself is transparent to clicks.
func TestStickyHeaderInteractiveChildrenHavePointerEventsAuto(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/design/sameway.css")
	wantStatus(t, rec, http.StatusOK)

	body := rec.Body.String()

	if !strings.Contains(body, ".sw-header__brand") || !strings.Contains(body, "pointer-events: auto") {
		t.Errorf("stylesheet should set pointer-events: auto on .sw-header__brand\n"+
			"so the brand link remains clickable even though the header has pointer-events: none\n%s", truncate(body))
	}

	if !strings.Contains(body, ".sw-nav .sw-link") || !strings.Contains(body, "pointer-events: auto") {
		t.Errorf("stylesheet should set pointer-events: auto on .sw-nav .sw-link\n"+
			"so navigation links remain clickable even though the header has pointer-events: none\n%s", truncate(body))
	}
}

// TestStickyHeaderRemainsStickyAfterPointerEventsFix checks that adding
// pointer-events to the header does not remove its sticky positioning.
// The header must still be visible when scrolling.
func TestStickyHeaderRemainsStickyAfterPointerEventsFix(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/design/sameway.css")
	wantStatus(t, rec, http.StatusOK)

	body := rec.Body.String()

	if !strings.Contains(body, "position: sticky") {
		t.Errorf("stylesheet must still set position: sticky on the header\n"+
			"so it remains visible when scrolling down a page\n%s", truncate(body))
	}

	if !strings.Contains(body, ".sw-header") || !strings.Contains(body, "top: 0") {
		t.Errorf("stylesheet must keep top: 0 on .sw-header\n"+
			"so the sticky header stays at the viewport top\n%s", truncate(body))
	}
}
