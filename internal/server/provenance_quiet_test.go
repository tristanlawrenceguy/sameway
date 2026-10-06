package server_test

import (
	"strings"
	"testing"
)

// A screen reader and an agent reading the page heard "Added by the
// workspace" before each block a new workspace begins with, which says
// nothing a person knows. Such a block says nothing about who added it.
func TestWhatTheWorkspaceBeganWithIsNotSaidToBeAdded(t *testing.T) {
	_, h := newApp(t)
	if page := get(t, h, "/").Body.String(); strings.Contains(page, "Added by the workspace") {
		t.Errorf("a new workspace's blocks say who added them: %s", truncate(page))
	}
}
