package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestNewRouteReturns404 checks that the form-based "new record" page is gone.
// Covers acceptance item 1: no handler for GET /t/{type}/new.
func TestNewRouteReturns404(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/t/note/new")
	wantStatus(t, rec, http.StatusNotFound)

	t.Logf("status=%d bodyLen=%d first500=%s", rec.Code, len(rec.Body.String()), truncateLog(rec.Body.String()))
	doc := parse(t, rec)
	if n := len(doc.Elements("h1")); n != 1 {
		t.Errorf("/t/note/new: expected one h1 in the 404 page, got %d", n)
	}
}

func truncateLog(s string) string {
	if len(s) > 500 {
		return s[:500]
	}
	return s
}

// TestEditRouteReturns404 checks that the form-based "edit record" page is gone.
// Covers acceptance item 1: no handler for GET /t/{type}/{id}/edit.
func TestEditRouteReturns404(t *testing.T) {
	a, h := newApp(t)

	rec, _ := a.Store.Create("note", map[string]any{"title": "Seed"})
	editPath := "/t/note/" + rec.ID + "/edit"

	resp := get(t, h, editPath)
	wantStatus(t, resp, http.StatusNotFound)

	doc := parse(t, resp)
	if n := len(doc.Elements("h1")); n != 1 {
		t.Errorf("%s: expected one h1 in the 404 page, got %d", editPath, n)
	}
}

// TestCreatePOSTReturns404 checks that the form-based create handler is gone.
// Covers acceptance item 1: no POST /t/{type} handler.
func TestCreatePOSTReturns404(t *testing.T) {
	_, h := newApp(t)

	vals := url.Values{"title": {"Hello"}}
	rec := postForm(t, h, "/t/note", vals)
	wantStatus(t, rec, http.StatusNotFound)
}

// TestUpdatePOSTReturns404 checks that the form-based update handler is gone.
// Covers acceptance item 1: no POST /t/{type}/{id} handler.
func TestUpdatePOSTReturns404(t *testing.T) {
	a, h := newApp(t)

	rec, _ := a.Store.Create("note", map[string]any{"title": "Seed"})
	updatePath := "/t/note/" + rec.ID

	vals := url.Values{"title": {"Updated"}}
	resp := postForm(t, h, updatePath, vals)
	wantStatus(t, resp, http.StatusNotFound)
}

// TestListPageHasNoNewLink checks that the listing page no longer has a
// "New {Type}" link. Covers acceptance item 3.
func TestListPageHasNoNewLink(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/t/note")
	wantStatus(t, rec, http.StatusOK)

	body := rec.Body.String()
	if containsAny(body, []string{`href="/t/note/new"`, `href='/t/note/new'`, "New note"}) {
		t.Errorf("list page should not contain a 'New note' link:\n%s", truncate(rec.Body.String()))
	}

	doc := parse(t, rec)
	newLinks := doc.WithAttr("href", "/t/note/new")
	if len(newLinks) != 0 {
		t.Fatalf("expected zero links to /t/note/new on the list page, got %d", len(newLinks))
	}
}

// TestDetailPageHasNoEditLink checks that the detail page no longer has an
// "Edit" link but retains its Delete link. Covers acceptance item 4.
func TestDetailPageHasNoEditLink(t *testing.T) {
	a, h := newApp(t)

	rec, _ := a.Store.Create("note", map[string]any{"title": "Hello"})
	detailPath := "/t/note/" + rec.ID

	resp := get(t, h, detailPath)
	wantStatus(t, resp, http.StatusOK)

	body := resp.Body.String()
	if containsAny(body, []string{`href="/t/note/` + rec.ID + `/edit"`, `href='/t/note/` + rec.ID + `/edit'`, "Edit note"}) {
		t.Errorf("detail page should not contain an 'Edit note' link:\n%s", truncate(resp.Body.String()))
	}

	doc := parse(t, resp)
	editLinks := doc.WithAttr("href", detailPath+"/edit")
	if len(editLinks) != 0 {
		t.Fatalf("expected zero links to %s/edit on the detail page, got %d", detailPath, len(editLinks))
	}

	// Delete link must still be present.
	deleteLink := doc.WithAttr("href", detailPath+"/confirm-delete")
	if len(deleteLink) == 0 {
		t.Fatalf("detail page should still have a delete link to %s/confirm-delete", detailPath)
	}
}

// TestDetailPageRetainsDeleteLink verifies the Delete link on the detail page
// is accessible and points at confirm-delete. Covers acceptance item 4.
func TestDetailPageRetainsDeleteLink(t *testing.T) {
	a, h := newApp(t)

	rec, _ := a.Store.Create("note", map[string]any{"title": "For deletion"})
	detailPath := "/t/note/" + rec.ID

	doc := parse(t, get(t, h, detailPath))
	deleteLinks := doc.WithAttr("href", detailPath+"/confirm-delete")
	if len(deleteLinks) != 1 {
		t.Fatalf("expected exactly one delete link at %s/confirm-delete, got %d", detailPath, len(deleteLinks))
	}

	link := deleteLinks[0]
	text := htmltest.Text(link)
	if text != "Delete note" {
		t.Errorf("delete link text = %q, want \"Delete note\"", text)
	}
}

// TestDescribeHasNoHtmlNew checks that the Describe() route map no longer
// contains an html_new entry. Covers acceptance item 2.
func TestDescribeHasNoHtmlNew(t *testing.T) {
	a, _ := newApp(t)

	desc := a.Describe()
	if _, ok := desc.Routes["html_new"]; ok {
		t.Error("describe routes should not contain 'html_new'")
	}
	_ = a // used above for Describe
}

// TestDescribeHasNoHtmlEdit checks that the Describe() route map no longer
// contains an html_edit entry (defensive: even if it was never added).
func TestDescribeHasNoHtmlEdit(t *testing.T) {
	a, _ := newApp(t)

	desc := a.Describe()
	if _, ok := desc.Routes["html_edit"]; ok {
		t.Error("describe routes should not contain 'html_edit'")
	}
	_ = a // used above for Describe
}

// TestDescribeRetainsApiRoutes checks that the JSON API CRUD routes are still
// present after removing HTML form surfaces. Covers acceptance item 2 (negative).
func TestDescribeRetainsApiRoutes(t *testing.T) {
	a, _ := newApp(t)

	desc := a.Describe()
	for _, key := range []string{"create", "get", "update", "delete"} {
		val, ok := desc.Routes[key]
		if !ok || val == "" {
			t.Errorf("describe routes should contain %q after form removal", key)
		}
		_ = val // avoid unused variable warning in some iterations
	}
	_ = a // used above for Describe
}

// TestEveryExistingPageHasOneH1AndLabelledControls checks only the pages that
// actually exist now (no /t/note/new, no /t/note/{id}/edit). Covers acceptance
// item 5 — these paths must all return 200 with valid HTML.
func TestEveryExistingPageHasOneH1AndLabelledControls(t *testing.T) {
	a, h := newApp(t)

	rec, _ := a.Store.Create("note", map[string]any{"title": "Seed"})
	paths := []string{
		"/", "/chat", "/activity", "/design",
		"/t/note",
		"/t/note/" + rec.ID,
		"/t/note/" + rec.ID + "/confirm-delete",
	}

	for _, path := range paths {
		resp := get(t, h, path)
		if resp.Code != http.StatusOK {
			t.Errorf("%s: expected 200, got %d", path, resp.Code)
			continue
		}
		doc := parse(t, resp)
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
	_ = a // used above for Create
}

// TestJsonApiCreateStillWorks verifies the JSON API create route still works
// after removing HTML form surfaces. Covers acceptance item 5 (negative).
func TestJsonApiCreateStillWorks(t *testing.T) {
	_, h := newApp(t)

	rec := postJSON(t, h, http.MethodPost, "/api/note", map[string]any{
		"title":  "API created",
		"status": "published",
	})
	wantStatus(t, rec, http.StatusCreated)

	var result struct {
		ID string
	}
	decode(t, rec, &result)
	if result.ID == "" {
		t.Fatal("expected an ID in the API create response")
	}
}

// TestJsonApiUpdateStillWorks verifies the JSON API update route still works.
func TestJsonApiUpdateStillWorks(t *testing.T) {
	a, h := newApp(t)

	createRec := postJSON(t, h, http.MethodPost, "/api/note", map[string]any{
		"title": "Before",
	})
	var created struct{ ID string }
	decode(t, createRec, &created)

	updateResp := do(t, h, http.MethodPut, "/api/note/"+created.ID,
		strings.NewReader(`{"title":"After"}`), "application/json")
	wantStatus(t, updateResp, http.StatusOK)

	var updated struct{ ID string }
	decode(t, updateResp, &updated)
	_ = updated // we verify via the HTML page below

	detail := parse(t, get(t, h, "/t/note/"+created.ID))
	if !contains(htmltest.Text(detail.Root), "After") {
		t.Errorf("detail page should show the updated title 'After'")
	}
	_ = a // used above for Create
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
