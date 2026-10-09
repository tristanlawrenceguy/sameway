package render_test

import (
	"strings"
	"testing"
)

// Links in a message are found in the order they are written, an address
// and a page path side by side or one inside the other.

// TestMessageMixedLinksOrdered renders a paragraph containing both an external
// URL and an internal path where the URL appears first, verifying anchors are
// emitted in text order (not regex-loop order). Addresses reviewer finding 3.
func TestMessageMixedLinksOrdered(t *testing.T) {
	t.Parallel()
	reg := builtins(t)

	out, err := reg.Render("message", map[string]any{
		"role":    "assistant",
		"content": "Visit https://example.com or see /t/note/abc123.",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	got := string(out)

	// Both anchors must be present.
	wantURL := `<a class="sw-link" href="https://example.com">example.com</a>`
	wantPath := `<a class="sw-link" href="/t/note/abc123">/t/note/abc123</a>`
	if !strings.Contains(got, wantURL) {
		t.Errorf("missing external URL anchor:\ngot:\n%s", got)
	}
	if !strings.Contains(got, wantPath) {
		t.Errorf("missing internal path anchor:\ngot:\n%s", got)
	}

	// The URL anchor must appear before the internal path anchor (text order).
	urlPos := strings.Index(got, wantURL)
	pathPos := strings.Index(got, wantPath)
	if urlPos < 0 || pathPos < 0 {
		t.Fatalf("could not find both anchors to check order:\ngot:\n%s", got)
	}
	if urlPos > pathPos {
		t.Errorf("anchors must appear in text order; URL anchor should come before internal path\nwant: %s … %s\ngot:\n%s", wantURL, wantPath, got)
	}
}

// TestMessageOverlappingURLAndPath renders content where an external URL
// contains a substring matching /t/<type>/<id>, verifying only one anchor for
// the full URL is emitted (no nested anchors). Addresses reviewer finding 4.
func TestMessageOverlappingURLAndPath(t *testing.T) {
	t.Parallel()
	reg := builtins(t)

	out, err := reg.Render("message", map[string]any{
		"role":    "assistant",
		"content": "See https://example.com/t/note/abc123.",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	got := string(out)

	// The full URL anchor must be present.
	wantURL := `<a class="sw-link" href="https://example.com/t/note/abc123">example.com/t/note/abc123</a>`
	if !strings.Contains(got, wantURL) {
		t.Errorf("full URL anchor missing:\ngot:\n%s", got)
	}

	// There must be exactly one <a class="sw-link"> in the output — no nested
	// or overlapping anchors for the /t/note/abc123 substring.
	count := strings.Count(got, `<a class="sw-link"`)
	if count != 1 {
		t.Errorf("expected exactly one sw-link anchor, got %d:\ngot:\n%s", count, got)
	}
}
