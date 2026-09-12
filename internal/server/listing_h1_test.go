package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestListingPageH1IsCapitalized checks that every content-type listing page
// shows a capitalized h1 — acceptance items 1 and 2.
func TestListingPageH1IsCapitalized(t *testing.T) {
	_, h := newApp(t)
	for _, path := range []string{"/t/note", "/t/message"} {
		rec := get(t, h, path)
		doc := parse(t, rec)
		h1s := doc.Elements("h1")
		if len(h1s) != 1 {
			t.Fatalf("%s: expected exactly one h1, got %d", path, len(h1s))
		}
		text := strings.TrimSpace(htmltest.Text(h1s[0]))
		if text == "" {
			t.Errorf("%s: h1 is empty", path)
			continue
		}
		first := text[:1]
		if first != strings.ToUpper(first) {
			t.Errorf("%s: h1 %q should start with a capital letter, got lowercase %q", path, text, first)
		}
	}
}

// TestEmptyStateTextStaysLowercase checks that the empty-state paragraph still
// uses lowercase — acceptance item 3.
func TestEmptyStateTextStaysLowercase(t *testing.T) {
	_, h := newApp(t)
	rec := get(t, h, "/t/block")
	doc := parse(t, rec)
	parts := doc.Elements("p")
	if len(parts) == 0 {
		t.Fatal("/t/block should show a paragraph when empty")
	}
	text := htmltest.Text(parts[0])
	if !strings.Contains(text, "No blocks yet.") {
		t.Errorf("expected lowercase 'No blocks yet.' in %q", text)
	}
}

// TestAriaLabelOnListIsLowercase checks that the ol's aria-label stays
// lowercase — acceptance item 4.
func TestAriaLabelOnListIsLowercase(t *testing.T) {
	a, h := newApp(t)
	_, err := a.Store.Create("note", map[string]any{"title": "Hi"})
	if err != nil {
		t.Fatal(err)
	}
	rec := get(t, h, "/t/note")
	doc := parse(t, rec)
	ols := doc.Elements("ol")
	if len(ols) == 0 {
		t.Fatal("/t/note should have an ol for the list when records exist")
	}
	label, _ := htmltest.Attr(ols[0], "aria-label")
	if label != "notes" {
		t.Errorf("expected aria-label 'notes', got %q", label)
	}
}
