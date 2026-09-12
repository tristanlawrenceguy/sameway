package render_test

import (
	"path/filepath"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// TestStarterWorkspaceLoads verifies that the starter workspace's component
// folder can be loaded without error.  The incomplete callout directory at
// examples/workspaces/starter/components/callout/ (missing manifest.json,
// template.html, style.css) must have been removed — otherwise LoadDir fails
// and blocks every agent workflow on a fresh init.
func TestStarterWorkspaceLoads(t *testing.T) {
	reg := render.New()
	starterComponents := filepath.Join("..", "..", "examples", "workspaces", "starter", "components")
	if err := reg.LoadDir(starterComponents, "workspace"); err != nil {
		t.Fatalf("starter workspace components must load cleanly: %v (this usually means an incomplete component folder exists)", err)
	}
}
