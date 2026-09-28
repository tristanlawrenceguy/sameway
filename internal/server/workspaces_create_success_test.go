package server_test

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// TestWorkspacesNewStartFailsShowsSuccessAlert checks that when a new workspace
// is created but the auto-start fails, the response page confirms creation with
// a success alert and separately notes the start failure — it does not show
// "That did not go through". Covers acceptance item 1.
func TestWorkspacesNewStartFailsShowsSuccessAlert(t *testing.T) {
	known := filepath.Join(t.TempDir(), "workspaces.json")
	t.Setenv("SAMEWAY_KNOWN", known)

	a, h := newApp(t)
	h = server.New(a).WithFleet(&server.Fleet{Launch: func(dir, addr string) error {
		return fmt.Errorf("simulated start failure") // fleet refuses to launch
	}})
	workspace.Remember(a.Workspace.Dir, "")
	a.Workspace.Set("name", "Base")

	res := postForm(t, h, "/workspaces/new", url.Values{"name": {"New workspace"}})
	wantStatus(t, res, http.StatusOK) // not a redirect — start failed, so page is rendered

	body := res.Body.String()

	// Should NOT show the old misleading message.
	if strings.Contains(body, "That did not go through") {
		t.Errorf("success page must not say \"That did not go through\" when workspace was created\n%s", truncate(body))
	}

	// Should confirm creation succeeded with a success alert.
	if !strings.Contains(body, `data-kind="success"`) || !strings.Contains(body, "New workspace") {
		t.Errorf("response should show a success alert confirming creation; got:\n%s", truncate(body))
	}

	// Should note the auto-start failure separately (warning).
	if !strings.Contains(body, `data-kind="warning"`) {
		t.Errorf("response should also warn about the start failure\n%s", truncate(body))
	}
}

// TestWorkspacesCopyStartFailsShowsSuccessAlert checks that when a workspace copy
// is created but the auto-start fails, the response page confirms copying with
// a success alert and separately notes the start failure — it does not show
// "That did not go through". Covers acceptance item 2.
func TestWorkspacesCopyStartFailsShowsSuccessAlert(t *testing.T) {
	known := filepath.Join(t.TempDir(), "workspaces.json")
	t.Setenv("SAMEWAY_KNOWN", known)

	a, h := newApp(t)
	h = server.New(a).WithFleet(&server.Fleet{Launch: func(dir, addr string) error {
		return fmt.Errorf("simulated start failure") // fleet refuses to launch
	}})
	workspace.Remember(a.Workspace.Dir, "")
	a.Workspace.Set("name", "Base")

	res := postForm(t, h, "/workspaces/copy", url.Values{"name": {"Base copy"}})
	wantStatus(t, res, http.StatusOK) // not a redirect — start failed, so page is rendered

	body := res.Body.String()

	// Should NOT show the old misleading message.
	if strings.Contains(body, "That did not go through") {
		t.Errorf("success page must not say \"That did not go through\" when workspace was copied\n%s", truncate(body))
	}

	// Should confirm copy succeeded with a success alert.
	if !strings.Contains(body, `data-kind="success"`) || !strings.Contains(body, "copy") {
		t.Errorf("response should show a success alert confirming the copy; got:\n%s", truncate(body))
	}

	// Should note the auto-start failure separately (warning).
	if !strings.Contains(body, `data-kind="warning"`) {
		t.Errorf("response should also warn about the start failure\n%s", truncate(body))
	}
}
