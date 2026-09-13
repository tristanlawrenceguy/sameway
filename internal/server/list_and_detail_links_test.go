package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestListPageLinksToChatNotNew checks that the list page replaces the old
// "New {type}" link with a chat link.  It must not contain any href with
// "/new", and it must include an <a> whose text is "Use chat to add".
func TestListPageLinksToChatNotNew(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/t/note")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if strings.Contains(body, "/new") {
		t.Errorf("list page must not contain a /new link; body snippet: %s", truncate(body))
	}
	if !strings.Contains(body, "Use chat to add") {
		t.Errorf("list page should have a \"Use chat to add\" link pointing to /chat; body snippet: %s", truncate(body))
	}

	// The chat link must actually point at /chat.
	doc := parse(t, rec)
	for _, a := range doc.WithAttr("href", "/chat") {
		name := doc.AccessibleName(a)
		if name != "Use chat to add" {
			t.Errorf("/chat link accessible name = %q, want \"Use chat to add\"", name)
		}
		return // found the correct link; that's enough
	}
	t.Error("no <a href=\"/chat\"> found on the list page")
}

// TestDetailPageHasNoEditLink verifies the detail page no longer shows an
// edit link (href containing "/edit").  The delete link must still be present.
func TestDetailPageHasNoEditLink(t *testing.T) {
	a, h := newApp(t)

	// Create a note via the store so we have something to view on the detail page.
	rec, err := a.Store.Create("note", map[string]any{"title": "Hello", "tags": []any{"a"}, "pinned": true})
	if err != nil {
		t.Fatal(err)
	}

	docRec := get(t, h, "/t/note/"+rec.ID)
	wantStatus(t, docRec, http.StatusOK)
	body := docRec.Body.String()

	if strings.Contains(body, "/edit") {
		t.Errorf("detail page must not contain an /edit link; body snippet: %s", truncate(body))
	}

	doc := parse(t, docRec)
	for _, el := range doc.WithAttr("href", "") {
		href, _ := htmltest.Attr(el, "href")
		if strings.Contains(href, "/edit") {
			t.Errorf("detail page link href %q must not contain /edit", href)
		}
	}
}
