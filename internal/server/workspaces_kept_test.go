package server_test

import (
	"strings"
	"testing"
)

// The Workspaces page says where this workspace is kept, with words saying
// what the path is, never a bare folder path.
func TestWorkspacesSayWhereTheyAreKept(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	page := get(t, h, "/workspaces").Body.String()
	if !strings.Contains(page, "Kept in "+a.Workspace.Dir) {
		t.Errorf("this workspace's folder is said as where it is kept")
	}
	if strings.Count(page, a.Workspace.Dir) != strings.Count(page, "Kept in "+a.Workspace.Dir)+strings.Count(page, `value="`+a.Workspace.Dir) {
		t.Errorf("the folder is never shown bare")
	}
}
