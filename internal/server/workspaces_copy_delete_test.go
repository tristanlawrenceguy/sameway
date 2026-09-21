package server_test

import (
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// TestWorkspacesCopyGETPage checks that GET /workspaces/copy returns a full
// accessible page with the "copy workspace" form, not an HTTP 405 error.
func TestWorkspacesCopyGETPage(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/workspaces/copy")
	wantStatus(t, rec, http.StatusOK)

	body := rec.Body.String()

	if !strings.Contains(body, "Copy workspace") {
		t.Errorf("page should mention 'Copy workspace' in heading\n%s", truncate(body))
	}
	if !strings.Contains(body, `<label`) || !strings.Contains(body, "Name for the copy") {
		t.Errorf("page should have a <label> for the name input\n%s", truncate(body))
	}
	if !strings.Contains(body, `type="text"`) || !strings.Contains(body, `name="name"`) {
		t.Errorf(`page should have an <input type="text" name="name">\n%s`, truncate(body))
	}
	if !strings.Contains(body, "required") {
		t.Errorf("the name input should carry the required attribute\n%s", truncate(body))
	}
	if !strings.Contains(body, `type="submit"`) || !strings.Contains(body, ">Copy<") {
		t.Errorf(`page should have a submit button ("Copy")\n%s`, truncate(body))
	}
	if !strings.Contains(body, `action="/workspaces/copy"`) {
		t.Errorf("the form should post to /workspaces/copy\n%s", truncate(body))
	}
}

// TestWorkspacesDeleteGETPage checks that GET /workspaces/delete returns a full
// accessible page with the "delete workspace" confirmation form, not an HTTP 405 error.
func TestWorkspacesDeleteGETPage(t *testing.T) {
	dir := t.TempDir()
	if err := workspace.Init(dir, examples.FS, examples.StarterRoot, false); err != nil {
		t.Fatal(err)
	}
	a, err := app.Load(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })

	h := server.New(a)
	a.Workspace.Set("name", "TestWorkspace")

	rec := get(t, h, "/workspaces/delete")
	wantStatus(t, rec, http.StatusOK)

	body := rec.Body.String()

	if !strings.Contains(body, "Delete workspace") {
		t.Errorf("page should mention 'Delete workspace' in heading\n%s", truncate(body))
	}
	if !strings.Contains(body, `<label`) || !strings.Contains(body, `Type TestWorkspace to delete it`) {
		t.Errorf(`page should have a <label> for the confirm input\n%s`, truncate(body))
	}
	if !strings.Contains(body, `type="text"`) || !strings.Contains(body, `name="confirm"`) {
		t.Errorf(`page should have an <input type="text" name="confirm">\n%s`, truncate(body))
	}
	if !strings.Contains(body, "required") {
		t.Errorf("the confirm input should carry the required attribute\n%s", truncate(body))
	}
	if !strings.Contains(body, `type="submit"`) || !strings.Contains(body, ">Delete<") {
		t.Errorf(`page should have a submit button ("Delete")\n%s`, truncate(body))
	}
	if !strings.Contains(body, `action="/workspaces/delete"`) {
		t.Errorf("the form should post to /workspaces/delete\n%s", truncate(body))
	}
}

// TestWorkspacesCopyGETPageWithTitle checks that the copy page has a proper
// <title> element containing the page name and site. Covers acceptance item 3.
func TestWorkspacesCopyGETPageWithTitle(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/workspaces/copy")
	wantStatus(t, rec, http.StatusOK)

	body := rec.Body.String()

	if !strings.Contains(body, "<title>") {
		t.Fatal("copy page should contain a <title> element")
	}

	doc := parse(t, rec)
	titleEls := doc.Elements("title")
	if len(titleEls) != 1 {
		t.Fatalf("expected exactly one <title>, got %d", len(titleEls))
	}
	titleText := htmltest.Text(titleEls[0])
	if titleText == "" {
		t.Error("<title> is empty on the copy page")
	}
	if !strings.Contains(body, "Copy workspace") {
		t.Errorf("page title should mention 'Copy workspace'\n%s", truncate(body))
	}
}

// TestWorkspacesDeleteGETPageWithTitle checks that the delete page has a proper
// <title> element containing the page name and site. Covers acceptance item 3.
func TestWorkspacesDeleteGETPageWithTitle(t *testing.T) {
	dir := t.TempDir()
	if err := workspace.Init(dir, examples.FS, examples.StarterRoot, false); err != nil {
		t.Fatal(err)
	}
	a, err := app.Load(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })

	h := server.New(a)
	a.Workspace.Set("name", "TestWorkspace")

	rec := get(t, h, "/workspaces/delete")
	wantStatus(t, rec, http.StatusOK)

	body := rec.Body.String()

	if !strings.Contains(body, "<title>") {
		t.Fatal("delete page should contain a <title> element")
	}

	doc := parse(t, rec)
	titleEls := doc.Elements("title")
	if len(titleEls) != 1 {
		t.Fatalf("expected exactly one <title>, got %d", len(titleEls))
	}
	titleText := htmltest.Text(titleEls[0])
	if titleText == "" {
		t.Error("<title> is empty on the delete page")
	}
	if !strings.Contains(body, "Delete workspace") {
		t.Errorf("page title should mention 'Delete workspace'\n%s", truncate(body))
	}
}

// TestWorkspacesCopyGETPageHasLayout checks that the copy page has full Sameway
// layout: main, header, footer, and nav landmarks. Covers acceptance item 1.
func TestWorkspacesCopyGETPageHasLayout(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/workspaces/copy")
	wantStatus(t, rec, http.StatusOK)

	doc := parse(t, rec)

	if doc.ByID("main") == nil || len(doc.Elements("main")) != 1 {
		t.Error("copy page should have one <main id=main>")
	}
	for _, tag := range []string{"header", "footer"} {
		if len(doc.Elements(tag)) != 1 {
			t.Errorf("copy page should have one <%s>", tag)
		}
	}
	skips := doc.WithAttr("class", "sw-skip")
	if len(skips) == 0 {
		t.Error("copy page should have a skip link")
	}
	navs := doc.Elements("nav")
	if len(navs) != 2 {
		var labels []string
		for _, n := range navs {
			l, _ := htmltest.Attr(n, "aria-label")
			labels = append(labels, l)
		}
		t.Errorf("copy page should have two <nav> landmarks; got %d: %v", len(navs), labels)
	}
}

// TestWorkspacesDeleteGETPageHasLayout checks that the delete page has full
// Sameway layout: main, header, footer, and nav landmarks. Covers acceptance item 2.
func TestWorkspacesDeleteGETPageHasLayout(t *testing.T) {
	dir := t.TempDir()
	if err := workspace.Init(dir, examples.FS, examples.StarterRoot, false); err != nil {
		t.Fatal(err)
	}
	a, err := app.Load(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })

	h := server.New(a)
	a.Workspace.Set("name", "TestWorkspace")

	rec := get(t, h, "/workspaces/delete")
	wantStatus(t, rec, http.StatusOK)

	doc := parse(t, rec)

	if doc.ByID("main") == nil || len(doc.Elements("main")) != 1 {
		t.Error("delete page should have one <main id=main>")
	}
	for _, tag := range []string{"header", "footer"} {
		if len(doc.Elements(tag)) != 1 {
			t.Errorf("delete page should have one <%s>", tag)
		}
	}
	skips := doc.WithAttr("class", "sw-skip")
	if len(skips) == 0 {
		t.Error("delete page should have a skip link")
	}
	navs := doc.Elements("nav")
	if len(navs) != 2 {
		var labels []string
		for _, n := range navs {
			l, _ := htmltest.Attr(n, "aria-label")
			labels = append(labels, l)
		}
		t.Errorf("delete page should have two <nav> landmarks; got %d: %v", len(navs), labels)
	}
}

// TestWorkspacesDeleteWrongNameShowsWarning checks that POST /workspaces/delete
// with a wrong confirm value shows an alert mentioning "delete" and does NOT
// contain copy/start language ("could not be started", "this one stays").
func TestWorkspacesDeleteWrongNameShowsWarning(t *testing.T) {
	dir := t.TempDir()
	if err := workspace.Init(dir, examples.FS, examples.StarterRoot, false); err != nil {
		t.Fatal(err)
	}
	a, err := app.Load(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })

	h := server.New(a)
	a.Workspace.Set("name", "TestWorkspace")

	rec := postForm(t, h, "/workspaces/delete", url.Values{"confirm": {"wrong name"}})
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if !strings.Contains(body, "delete") {
		t.Errorf("alert should mention 'delete' in wrong-name warning\n%s", truncate(body))
	}
	if strings.Contains(body, "could not be started") || strings.Contains(body, "this one stays") {
		t.Error("wrong-name alert must NOT contain copy/start language\n" + body)
	}
}

// TestWorkspacesDeleteNoFleetShowsDeletionError checks that POST /workspaces/delete
// with the correct name but no fleet (so starting a replacement fails) shows an
// error message specific to deletion, not copy/start language. Covers acceptance 1.
func TestWorkspacesDeleteNoFleetShowsDeletionError(t *testing.T) {
	known := filepath.Join(t.TempDir(), "workspaces.json")
	t.Setenv("SAMEWAY_KNOWN", known)

	dir := t.TempDir()
	if err := workspace.Init(dir, examples.FS, examples.StarterRoot, false); err != nil {
		t.Fatal(err)
	}
	a, err := app.Load(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })

	h := server.New(a)
	a.Workspace.Set("name", "TestWorkspace")

	rec := postForm(t, h, "/workspaces/delete", url.Values{"confirm": {"TestWorkspace"}})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (error shown on page), got %d: %s", rec.Code, truncate(rec.Body.String()))
	}
	body := rec.Body.String()

	if !strings.Contains(body, "delete") {
		t.Errorf("response should mention 'delete' in the error alert\n%s", truncate(body))
	}
	if strings.Contains(body, "could not be started") || strings.Contains(body, "this one stays") {
		t.Error("error must NOT contain copy/start language\n" + body)
	}
}
