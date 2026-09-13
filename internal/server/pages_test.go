package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
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
		t.Errorf("page should link its JSON twin with rel=alternate")
	}
	assertAllComponentsKnown(t, doc)
}

// assertAllComponentsKnown checks every rendered component is one an agent
// can look up in /api/describe.
func assertAllComponentsKnown(t *testing.T, doc *htmltest.Doc) {
	t.Helper()
	known := map[string]bool{}
	for _, name := range []string{"alert", "badge", "button", "calendar", "card", "chat", "checkbox", "datepicker", "disclosure", "event", "heading", "link", "list", "message", "proposal", "select", "status", "table", "text", "text-field", "textarea"} {
		known[name] = true
	}
	for _, n := range doc.WithAttr("data-component", "") {
		name, _ := htmltest.Attr(n, "data-component")
		if !known[name] {
			t.Errorf("page renders unknown component %q", name)
		}
	}
}

// TestNavigationMarksCurrentPage covers aria-current for the nav links.
func TestNavigationMarksCurrentPage(t *testing.T) {
	_, h := newApp(t)
	doc := parse(t, get(t, h, "/t/note"))
	var current []string
	for _, a := range doc.WithAttr("aria-current", "page") {
		current = append(current, htmltest.Text(a))
	}
	if len(current) != 1 || current[0] != "notes" {
		t.Errorf("expected only the notes link to be current, got %v", current)
	}
	if len(doc.WithAttr("href", "/t/message")) != 0 {
		t.Errorf("internal types must not appear in the navigation")
	}
}

// TestContentPagesLifecycle walks the human path: list, new, create with an
// error, fix it, view, edit, delete.
func TestContentPagesLifecycle(t *testing.T) {
	_, h := newApp(t)

	list := parse(t, get(t, h, "/t/note"))
	if len(list.WithAttr("href", "/t/note/new")) == 0 {
		t.Fatalf("list page needs a New note link")
	}

	form := parse(t, get(t, h, "/t/note/new"))
	for _, name := range []string{"title", "body", "tags", "status", "pinned"} {
		if form.ByID(name) == nil || form.AccessibleName(form.ByID(name)) == "" {
			t.Errorf("new form: control %q missing or unlabelled", name)
		}
	}

	bad := postForm(t, h, "/t/note", url.Values{"title": {""}, "status": {"bogus"}})
	wantStatus(t, bad, http.StatusUnprocessableEntity)
	badDoc := parse(t, bad)
	for _, id := range []string{"title", "status"} {
		el := badDoc.ByID(id)
		if v, _ := htmltest.Attr(el, "aria-invalid"); v != "true" {
			t.Errorf("%s should be aria-invalid after a bad submit", id)
		}
		desc, _ := htmltest.Attr(el, "aria-describedby")
		if !strings.Contains(desc, id+"-error") || badDoc.ByID(id+"-error") == nil {
			t.Errorf("%s error text must be linked via aria-describedby", id)
		}
	}
	if len(badDoc.WithAttr("role", "alert")) == 0 {
		t.Errorf("a failed submit should announce an alert")
	}

	ok := postForm(t, h, "/t/note", url.Values{"title": {"Hello"}, "tags": {"a, b"}, "pinned": {"true"}})
	wantStatus(t, ok, http.StatusSeeOther)
	detailPath := ok.Header().Get("Location")
	if !strings.HasPrefix(detailPath, "/t/note/") {
		t.Fatalf("redirect to detail expected, got %q", detailPath)
	}
	detail := parse(t, get(t, h, detailPath))
	if !strings.Contains(htmltest.Text(detail.Root), "Hello") || !strings.Contains(htmltest.Text(detail.Root), "a, b") {
		t.Errorf("detail page missing saved values")
	}

	edit := parse(t, get(t, h, detailPath+"/edit"))
	if v, _ := htmltest.Attr(edit.ByID("title"), "value"); v != "Hello" {
		t.Errorf("edit form should be prefilled, title=%q", v)
	}
	if _, checked := htmltest.Attr(edit.ByID("pinned"), "checked"); !checked {
		t.Errorf("edit form should show pinned as checked")
	}
	upd := postForm(t, h, detailPath, url.Values{"title": {"Hello again"}, "status": {"published"}})
	wantStatus(t, upd, http.StatusSeeOther)
	after := parse(t, get(t, h, detailPath))
	if !strings.Contains(htmltest.Text(after.Root), "Hello again") || strings.Contains(htmltest.Text(after.Root), "Pinned yes") {
		t.Errorf("update should change title and clear the unchecked checkbox: %s", htmltest.Text(after.Root))
	}

	del := postForm(t, h, detailPath+"/delete", nil)
	wantStatus(t, del, http.StatusSeeOther)
	wantStatus(t, get(t, h, detailPath), http.StatusNotFound)
	wantStatus(t, get(t, h, "/t/nothing"), http.StatusNotFound)
}

// TestEveryPageHasOneH1AndLabelledControls runs the shell invariants on each page kind.
func TestEveryPageHasOneH1AndLabelledControls(t *testing.T) {
	a, h := newApp(t)
	rec, _ := a.Store.Create("note", map[string]any{"title": "Seed"})
	for _, path := range []string{"/", "/chat", "/activity", "/design", "/t/note", "/t/note/new", "/t/note/" + rec.ID, "/t/note/" + rec.ID + "/edit", "/t/note/" + rec.ID + "/confirm-delete"} {
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
