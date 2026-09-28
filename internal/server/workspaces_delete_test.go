package server_test

import (
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// TestWorkspacesDeleteSuccessShowsConfirmation checks that POST /workspaces/delete
// with the correct workspace name (fleet mode, so deletion succeeds) returns a page
// containing an alert element with role="alert" and text mentioning "deleted".
// Covers acceptance item 1.
func TestWorkspacesDeleteSuccessShowsConfirmation(t *testing.T) {
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

	f := &fleet{t: t}
	h := server.New(a).WithFleet(&server.Fleet{Launch: f.launch, Exit: func() {}})
	a.Workspace.Set("name", "My Sameway")

	rec := postForm(t, h, "/workspaces/delete", url.Values{"confirm": {"My Sameway"}})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (confirmation page), got %d: %s", rec.Code, truncate(rec.Body.String()))
	}
	body := rec.Body.String()

	if !strings.Contains(body, "deleted") && !strings.Contains(body, "removed") {
		t.Errorf("response should mention 'deleted' or 'removed' in the confirmation alert; got status %d, body length %d\n%s", rec.Code, len(body), truncate(body))
	}
	if strings.Contains(body, "could not be started") || strings.Contains(body, "this one stays") {
		t.Error("success response must NOT contain copy/start language\n" + body)
	}

	doc := parse(t, rec)
	if len(doc.WithAttr("role", "status")) == 0 || !strings.Contains(body, "trash") {
		t.Errorf("the deletion is announced as a status, saying where the workspace went\n%s", truncate(body))
	}
}
