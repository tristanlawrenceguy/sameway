package server_test

import (
	"net/http"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// TestHomePageShell checks what a person with a screen reader or keyboard
// meets first: skip link, landmarks, one h1, and the two labelled regions.
func TestHomePageShell(t *testing.T) {
	_, h := newApp(t)
	rec := get(t, h, "/")
	wantStatus(t, rec, http.StatusOK)
	doc := parse(t, rec)

	if n := len(doc.Elements("h1")); n != 1 {
		t.Errorf("expected exactly one h1, got %d", n)
	}
	if lang, _ := htmltest.Attr(doc.Elements("html")[0], "lang"); lang != "en" {
		t.Errorf("html lang = %q", lang)
	}
	skips := doc.WithAttr("class", "sw-skip")
	if len(skips) == 0 || func() bool { h, _ := htmltest.Attr(skips[0], "href"); return h != "#main" }() {
		t.Errorf("first skip link must target #main")
	}
	if doc.ByID("main") == nil || len(doc.Elements("main")) != 1 {
		t.Errorf("expected one <main id=main>")
	}
	for _, tag := range []string{"header", "footer"} {
		if len(doc.Elements(tag)) != 1 {
			t.Errorf("expected one <%s>", tag)
		}
	}
	// Two navigation landmarks: the person's own content in the header, and
	// everything about the workspace itself tucked into the footer.
	navs := doc.Elements("nav")
	var labels []string
	for _, n := range navs {
		l, _ := htmltest.Attr(n, "aria-label")
		labels = append(labels, l)
	}
	if len(navs) != 2 || labels[0] != "Main" || labels[1] != "This workspace" {
		t.Errorf("expected a Main nav then a This workspace nav, got %v", labels)
	}
	for _, secondary := range []string{"/chat", "/activity", "/design"} {
		if len(doc.WithAttr("href", secondary)) == 0 {
			t.Errorf("%s should still be reachable from the footer", secondary)
		}
	}
	for _, s := range doc.Elements("section") {
		id, _ := htmltest.Attr(s, "aria-labelledby")
		if doc.ByID(id) == nil {
			t.Errorf("section aria-labelledby=%q has no heading", id)
		}
	}
	if len(doc.WithAttr("data-component", "alert")) == 0 {
		t.Errorf("with no model configured the page should show an alert explaining that")
	}
	alt := doc.WithAttr("rel", "alternate")
	if len(alt) == 0 {
		t.Error("expected an alternate link for JSON output")
	}
	if len(alt) > 0 {
		a := alt[0]
		href, _ := htmltest.Attr(a, "href")
		// Alternate links may end in .json (detail/list pages) or use /api/ paths (canvas/home).
		if !strings.HasSuffix(href, ".json") && !strings.HasPrefix(href, "/api/") {
			t.Errorf("alternate link should end in .json or start with /api/, got %q", href)
		}
	}

	assertAllComponentsKnown(t, doc)
}

func TestNavigationMarksCurrentPage(t *testing.T) {
	_, h := newApp(t)
	doc := parse(t, get(t, h, "/t/note"))

	var current []string
	for _, a := range doc.WithAttr("aria-current", "page") {
		current = append(current, htmltest.Text(a))
	}
	if len(current) != 1 || current[0] != "notes" {
		t.Errorf("expected exactly one aria-current link with text 'notes', got %v", current)
	}

	for _, a := range doc.WithAttr("href", "/t/activity") {
		text := htmltest.Text(a)
		if text != "activities" {
			t.Errorf("/t/activity nav link = %q, want 'activities'", text)
		}
		if cur, _ := htmltest.Attr(a, "aria-current"); cur == "page" {
			t.Errorf("/t/activity should not be current on the /t/note page")
		}
	}

	doc = parse(t, get(t, h, "/chat"))
	current = nil
	for _, a := range doc.WithAttr("aria-current", "page") {
		current = append(current, htmltest.Text(a))
	}
	if len(current) != 1 || current[0] != "Chat" {
		t.Errorf("expected exactly one aria-current link with text 'Chat', got %v", current)
	}

	doc = parse(t, get(t, h, "/activity"))
	current = nil
	for _, a := range doc.WithAttr("aria-current", "page") {
		current = append(current, htmltest.Text(a))
	}
	if len(current) != 1 || current[0] != "Activity" {
		t.Errorf("expected exactly one aria-current link with text 'Activity', got %v", current)
	}
}

// TestContentPagesLifecycle creates a note via JSON API, verifies the listing
// and detail pages, tests delete flow, and confirms removed routes return 404.
func TestContentPagesLifecycle(t *testing.T) {
	a, h := newApp(t)

	list := parse(t, get(t, h, "/t/note"))
	if len(list.WithAttr("href", "/api/note")) == 0 {
		t.Fatalf("list page needs a Create via API link to /api/note")
	}
	for _, lnk := range list.WithAttr("href", "/t/note/new") {
		href, _ := htmltest.Attr(lnk, "href")
		t.Errorf("list page should not still contain dead link %q", href)
	}

	// Create a note via JSON API.
	postRec := postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Hello", "tags": []any{"a", "b"}, "pinned": true})
	wantStatus(t, postRec, http.StatusCreated)

	recList := parse(t, get(t, h, "/t/note"))
	if !strings.Contains(htmltest.Text(recList.Root), "Hello") {
		t.Errorf("listing page should show created note title: %s", htmltest.Text(recList.Root))
	}

	// Get the record ID by listing notes.
	recs, _ := a.Store.List("note", store.ListOptions{})
	if len(recs) == 0 {
		t.Fatal("no notes found after create")
	}
	id := recs[0].ID
	detailPath := "/t/note/" + id

	detail := parse(t, get(t, h, detailPath))
	if !strings.Contains(htmltest.Text(detail.Root), "Hello") || !strings.Contains(htmltest.Text(detail.Root), "a, b") {
		t.Errorf("detail page missing saved values: %s", htmltest.Text(detail.Root))
	}

	// Verify no edit link on detail page.
	editLinks := detail.WithAttr("href", "/t/note/"+id+"/edit")
	if len(editLinks) > 0 {
		t.Errorf("detail page should not have an /edit link: %s", htmltest.Text(detail.Root))
	}

	// Verify confirm-delete link exists.
	deleteLinks := detail.WithAttr("href", "/t/note/"+id+"/confirm-delete")
	if len(deleteLinks) == 0 {
		t.Errorf("detail page should have a Delete → confirm-delete link: %s", htmltest.Text(detail.Root))
	}

	// Verify removed routes return 404.
	wantStatus(t, get(t, h, "/t/note/new"), http.StatusNotFound)
	wantStatus(t, get(t, h, detailPath+"/edit"), http.StatusNotFound)

	// Test delete flow: confirm-delete loads, POST delete removes it.
	confirm := parse(t, get(t, h, detailPath+"/confirm-delete"))
	if !strings.Contains(htmltest.Text(confirm.Root), "Delete") {
		t.Errorf("confirm-delete page should mention Delete: %s", htmltest.Text(confirm.Root))
	}

	delRec := postForm(t, h, detailPath+"/delete", nil)
	wantStatus(t, delRec, http.StatusSeeOther)
	wantStatus(t, get(t, h, detailPath), http.StatusNotFound)
}

// TestEveryPageHasOneH1AndLabelledControls runs the shell invariants on each page kind.
func TestEveryPageHasOneH1AndLabelledControls(t *testing.T) {
	a, h := newApp(t)
	rec, _ := a.Store.Create("note", map[string]any{"title": "Seed"})
	for _, path := range []string{"/", "/chat", "/activity", "/design", "/t/note", "/t/note/" + rec.ID, "/t/note/" + rec.ID + "/confirm-delete"} {
		doc := parse(t, get(t, h, path))
		if n := len(doc.Elements("h1")); n != 1 {
			t.Errorf("%s: %d h1 elements", path, n)
		}
		doc.Walk(func(n *html.Node) {
			if htmltest.Focusable(n) && doc.AccessibleName(n) == "" {
				t.Errorf("%s: focusable <%s> without an accessible name", path, n.Data)
			}
		})
		assertAllComponentsKnown(t, doc)
	}
}

func TestStylesheetRoute(t *testing.T) {
	_, h := newApp(t)
	rec := get(t, h, "/design/sameway.css")
	wantStatus(t, rec, http.StatusOK)
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/css") {
		t.Errorf("content type %q", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{"--sw-color-accent", "prefers-color-scheme: dark", "prefers-reduced-motion", ".sw-button", ".sw-message"} {
		if !strings.Contains(body, want) {
			t.Errorf("stylesheet missing %q", want)
		}
	}
}

// TestListingPageHeadingsAreCapitalized checks that every listing page shows
// a title-cased h1 — "Notes", "Activities", "Messages" — not lowercase.
func TestListingPageHeadingsAreCapitalized(t *testing.T) {
	_, h := newApp(t)
	for _, path := range []string{"/t/note", "/t/activity"} {
		rec := get(t, h, path)
		wantStatus(t, rec, http.StatusOK)
		doc := parse(t, rec)

		h1s := doc.Elements("h1")
		if len(h1s) != 1 {
			t.Fatalf("%s: expected one h1, got %d", path, len(h1s))
		}
		text := htmltest.Text(h1s[0])

		switch path {
		case "/t/note":
			if text != "Notes" {
				t.Errorf("/t/note h1 = %q, want %q", text, "Notes")
			}
		case "/t/activity":
			if text != "Activities" {
				t.Errorf("/t/activity h1 = %q, want %q", text, "Activities")
			}
		}
	}
}

// TestNavLinksStayLowercase ensures that only the page heading is capitalized;
// nav link labels remain lowercase as they call plural() directly.
func TestNavLinksStayLowercase(t *testing.T) {
	_, h := newApp(t)
	doc := parse(t, get(t, h, "/t/note"))

	var current []string
	for _, a := range doc.WithAttr("aria-current", "page") {
		current = append(current, htmltest.Text(a))
	}
	if len(current) != 1 || current[0] != "notes" {
		t.Errorf("nav link text should be lowercase \"notes\", got %v", current)
	}

	// The other nav links (non-current) must also be lowercase.
	for _, a := range doc.WithAttr("href", "/t/activity") {
		text := htmltest.Text(a)
		if text != "activities" {
			t.Errorf("/t/activity nav link = %q, want \"activities\"", text)
		}
	}
}

// TestEmptyStateBodyStaysLowercase verifies that the empty-state paragraph
// (body copy, not a heading) remains lowercase — e.g. "No notes yet."
func TestEmptyStateBodyStaysLowercase(t *testing.T) {
	_, h := newApp(t)
	rec := get(t, h, "/t/note")
	wantStatus(t, rec, http.StatusOK)

	body := rec.Body.String()
	if !strings.Contains(body, `<p>No notes yet.</p>`) {
		t.Errorf("empty-state body should say \"No notes yet.\" (lowercase)\nbody: %s", truncate(rec.Body.String()))
	}
}
