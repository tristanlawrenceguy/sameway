package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestNotFoundPageHasTitle checks that a GET to any non-existent path returns
// an HTML page with a <title> containing "404" and the site name.  Covers
// acceptance item 1 and 4.
func TestNotFoundPageHasTitle(t *testing.T) {
	_, h := newApp(t)
	rec := get(t, h, "/nonexistent-page-12345")
	wantStatus(t, rec, http.StatusNotFound)

	body := rec.Body.String()
	if !strings.Contains(body, "<title>") {
		t.Fatal("404 response should contain a <title> element")
	}
	if !strings.Contains(body, "404") {
		t.Errorf("404 page title should mention \"404\"\nbody: %s", truncate(body))
	}
	if !strings.Contains(body, "Sameway") {
		t.Errorf("404 page title should include the site name \"Sameway\"\nbody: %s", truncate(body))
	}

	doc := parse(t, rec)
	titleEls := doc.Elements("title")
	if len(titleEls) != 1 {
		t.Fatalf("expected exactly one <title>, got %d", len(titleEls))
	}
	titleText := htmltest.Text(titleEls[0])
	if titleText == "" {
		t.Errorf("<title> is empty on the 404 page")
	}
}

// TestNotFoundPageHasH1 checks that a GET to any non-existent path returns an
// HTML page with an h1 element whose text reads "Page not found".  Covers
// acceptance item 2.
func TestNotFoundPageHasH1(t *testing.T) {
	_, h := newApp(t)
	rec := get(t, h, "/nonexistent-page-12345")
	wantStatus(t, rec, http.StatusNotFound)

	doc := parse(t, rec)
	h1s := doc.Elements("h1")
	if len(h1s) != 1 {
		t.Fatalf("expected exactly one h1 on the 404 page, got %d", len(h1s))
	}
	text := htmltest.Text(h1s[0])
	if text == "" {
		t.Fatal("h1 on the 404 page has no text")
	}
	if !strings.Contains(text, "Page not found") {
		t.Errorf("h1 should say \"Page not found\", got %q", text)
	}
}

// TestNotFoundPageHasHomeLink checks that a GET to any non-existent path
// returns an HTML page with at least one navigation link to the home page (/)
// that is keyboard-accessible and has a visible label.  Covers acceptance item 3.
func TestNotFoundPageHasHomeLink(t *testing.T) {
	_, h := newApp(t)
	rec := get(t, h, "/nonexistent-page-12345")
	wantStatus(t, rec, http.StatusNotFound)

	body := rec.Body.String()
	if !strings.Contains(body, `href="/"`) && !strings.Contains(body, "href=\"/\"") {
		t.Fatal("404 page should contain a link to the home page /")
	}

	doc := parse(t, rec)
	var found bool
	for _, a := range doc.WithAttr("href", "/") {
		if htmltest.Focusable(a) {
			name := doc.AccessibleName(a)
			if name != "" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("404 page should have a keyboard-accessible link to / with a visible label\nbody: %s", truncate(body))
	}
}

// TestNotFoundPageHasLandmarks checks that the 404 response uses the same
// site layout as every other page — skip links, header, main, footer, and
// two nav landmarks.  This ensures screen-reader users get full context on an
// error page too.
func TestNotFoundPageHasLandmarks(t *testing.T) {
	_, h := newApp(t)
	rec := get(t, h, "/nonexistent-page-12345")
	wantStatus(t, rec, http.StatusNotFound)

	doc := parse(t, rec)

	if doc.ByID("main") == nil || len(doc.Elements("main")) != 1 {
		t.Error("404 page should have one <main id=main>")
	}
	for _, tag := range []string{"header", "footer"} {
		if len(doc.Elements(tag)) != 1 {
			t.Errorf("404 page should have one <%s>", tag)
		}
	}
	skips := doc.WithAttr("class", "sw-skip")
	if len(skips) == 0 || func() bool { h, _ := htmltest.Attr(skips[0], "href"); return h != "#main" }() {
		t.Errorf("404 page should have a skip link to #main")
	}
	navs := doc.Elements("nav")
	if len(navs) != 2 {
		var labels []string
		for _, n := range navs {
			l, _ := htmltest.Attr(n, "aria-label")
			labels = append(labels, l)
		}
		t.Errorf("404 page should have two <nav> landmarks; got %d: %v", len(navs), labels)
	}
}
