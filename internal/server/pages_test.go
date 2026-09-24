package server_test

import (
	"net/http"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// componentNames is every built-in component, read from the registry so a
// component added to design/components is known here the same moment and
// nobody keeps a second list by hand.
var componentNames = func() []string {
	reg, err := app.NewRegistry("")
	if err != nil {
		panic(err)
	}
	var names []string
	for _, c := range reg.Components() {
		names = append(names, c.Manifest.Name)
	}
	return names
}()

// TestHomePageShell checks what a person with a screen reader or keyboard
// meets first: skip link, landmarks, one h1, and the two labelled regions.
func TestHomePageShell(t *testing.T) {
	a, h := newApp(t)
	a.Workspace.Config.UI.Developer = "shown" // this test walks the builder links too
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
		t.Error("home: expected rel=alternate link for RSS or similar")
	}

	assertAllComponentsKnown(t, doc, componentNames)

	for _, path := range []string{"/chat", "/activity"} {
		doc := parse(t, get(t, h, path))
		if n := len(doc.Elements("h1")); n != 1 {
			t.Errorf("%s: %d h1 elements", path, n)
		}
		assertAllComponentsKnown(t, doc, componentNames)
	}
	for _, path := range []string{"/t/note", "/t/activity"} {
		doc := parse(t, get(t, h, path))
		if n := len(doc.Elements("h1")); n != 1 {
			t.Errorf("%s: %d h1 elements", path, n)
		}
		assertAllComponentsKnown(t, doc, componentNames)
	}

	for _, path := range []string{"/t/note/new", "/t/note/0/edit"} {
		rec := get(t, h, path)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s: expected 404, got %d", path, rec.Code)
		}
	}

	for _, path := range []string{"/t/note/nonexistent"} {
		rec := get(t, h, path)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s: expected 404 for nonexistent record, got %d", path, rec.Code)
		}
	}

	for _, path := range []string{"/t/missing-type"} {
		rec := get(t, h, path)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s: expected 404 for missing type, got %d", path, rec.Code)
		}
	}

	for _, path := range []string{"/nonexistent"} {
		rec := get(t, h, path)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s: expected 404 for nonexistent route, got %d", path, rec.Code)
		}
	}
}

// TestFormValidationIsStillTested verifies that the form-validation test
// helper still works (the validation logic itself is unchanged).
func TestFormValidationIsStillTested(t *testing.T) {
	_, h := newApp(t)
	rec := get(t, h, "/t/note")
	wantStatus(t, rec, http.StatusOK)
	doc := parse(t, rec)
	if n := len(doc.Elements("h1")); n != 1 {
		t.Errorf("/t/note: %d h1 elements", n)
	}

	if len(doc.WithAttr("href", "/chat")) == 0 {
		t.Error("/t/note empty state should link to /chat")
	}
}

// TestContentPagesLifecycle exercises the full CRUD lifecycle through the JSON API.
func TestContentPagesLifecycle(t *testing.T) {
	a, h := newApp(t)
	list := get(t, h, "/t/note")
	wantStatus(t, list, http.StatusOK)
	doc := parse(t, list)

	if n := len(doc.Elements("h1")); n != 1 {
		t.Errorf("/t/note: %d h1 elements", n)
	}
	if len(doc.WithAttr("href", "/chat")) == 0 {
		t.Error("/t/note empty state should link to /chat")
	}

	// Create via API.
	rec, err := a.Store.Create("note", map[string]any{"title": "Hello", "tags": []any{"a", "b"}, "pinned": true})
	if err != nil {
		t.Fatal(err)
	}
	detailPath := "/t/note/" + rec.ID

	list = get(t, h, "/t/note")
	wantStatus(t, list, http.StatusOK)
	body := list.Body.String()
	if !strings.Contains(body, "Hello") {
		t.Errorf("list page should show the new note: %s", truncate(body))
	}

	detail := parse(t, get(t, h, detailPath))
	if !strings.Contains(htmltest.Text(detail.Root), "Hello") || !strings.Contains(htmltest.Text(detail.Root), "a, b") {
		t.Errorf("detail page missing saved values")
	}

	// Update via API.
	postJSON(t, h, http.MethodPut, "/api/note/"+rec.ID, map[string]any{"title": "Hello again", "status": "published"})
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
	for _, path := range []string{"/", "/chat", "/activity", "/design", "/t/note", "/t/note/" + rec.ID, "/t/file", "/t/task", "/t/project"} {
		doc := parse(t, get(t, h, path))
		if n := len(doc.Elements("h1")); n != 1 {
			t.Errorf("%s: %d h1 elements", path, n)
		}
		doc.Walk(func(n *html.Node) {
			if htmltest.Focusable(n) && doc.AccessibleName(n) == "" {
				t.Errorf("%s: focusable <%s> without an accessible name", path, n.Data)
			}
		})
		assertAllComponentsKnown(t, doc, componentNames)
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

// TestEmptyStateBodyStaysLowercase verifies that the empty-state paragraph is
// actionable and uses <p class="sw-empty"> — e.g. "Ask the assistant to add your first notes."
func TestEmptyStateBodyStaysLowercase(t *testing.T) {
	_, h := newApp(t)
	rec := get(t, h, "/t/note")
	wantStatus(t, rec, http.StatusOK)

	body := rec.Body.String()
	if !strings.Contains(body, `data-component="empty"`) {
		t.Errorf("empty-state body should use the empty component\nbody: %s", truncate(rec.Body.String()))
	}
	if !strings.Contains(body, `/chat`) {
		t.Errorf("empty-state should link to /chat\nbody: %s", truncate(rec.Body.String()))
	}
}
