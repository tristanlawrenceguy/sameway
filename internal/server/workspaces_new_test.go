package server_test

import (
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// TestWorkspacesNewGETPage checks that GET /workspaces/new returns a full
// accessible page with the "new workspace" form, not an HTTP 405 error.
func TestWorkspacesNewGETPage(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/workspaces/new")
	wantStatus(t, rec, http.StatusOK)

	body := rec.Body.String()

	if !strings.Contains(body, "New workspace") {
		t.Errorf("page should mention 'New workspace' in heading\n%s", truncate(body))
	}
	if !strings.Contains(body, `<label`) || !strings.Contains(body, "Name") {
		t.Errorf("page should have a <label> for the name input\n%s", truncate(body))
	}
	if !strings.Contains(body, `type="text"`) && !strings.Contains(body, `name="name"`) {
		t.Errorf("page should have an <input type=\"text\" name=\"name\">\n%s", truncate(body))
	}
	if !strings.Contains(body, "required") {
		t.Errorf("the name input should carry the required attribute\n%s", truncate(body))
	}
	if !strings.Contains(body, `type="submit"`) && !strings.Contains(body, "Create and open") {
		t.Errorf("page should have a submit button (\"Create and open\")\n%s", truncate(body))
	}
	if !strings.Contains(body, `action="/workspaces/new"`) {
		t.Errorf(`the form should post to /workspaces/new\n%s`, truncate(body))
	}
}

// TestWorkspacesNewPOSTRedirect checks that after creating a workspace via
// POST /workspaces/new the response redirects back to /workspaces, not to
// /workspaces/new or the new workspace URL.
func TestWorkspacesNewPOSTRedirect(t *testing.T) {
	known := filepath.Join(t.TempDir(), "workspaces.json")
	t.Setenv("SAMEWAY_KNOWN", known)

	a, h := newApp(t)
	f := &fleet{t: t}
	h = server.New(a).WithFleet(&server.Fleet{Launch: f.launch})
	workspace.Remember(a.Workspace.Dir, "")
	a.Workspace.Set("name", "Base")

	res := postForm(t, h, "/workspaces/new", url.Values{"name": {"Test redirect"}})
	wantStatus(t, res, http.StatusSeeOther)

	loc := res.Header().Get("Location")
	if !strings.HasSuffix(loc, "/workspaces") {
		t.Errorf("after creating a workspace the redirect should go to /workspaces, got %q", loc)
	}

	// The new workspace itself was still created and started.
	if strings.Contains(loc, "/workspaces/new") {
		t.Error("redirect must not land on /workspaces/new (the broken 405 path)")
	}

	if len(f.started) < 1 || f.started[0] != filepath.Join(filepath.Dir(a.Workspace.Dir), "test-redirect") {
		t.Errorf("the new workspace was created and started: %v", f.started)
	}
}
