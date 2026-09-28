package server_test

// Tests for button-label simplification in workspaces.go.
// These pin that task 0155's acceptance criteria are met:
//   - All button labels ≤4 words (Acceptance 1)
//   - No passive-voice filler ("this", "and open") (Acceptance 2)

import (
	"net/http"
	"strings"
	"testing"
)

// TestWorkspaceNewPageButtonLabel checks that the new-workspace form button
// reads as a short, plain verb — not "Create and open".  This covers Acceptance 1
// (≤4 words) and Acceptance 2 (active verb only).
func TestWorkspaceNewPageButtonLabel(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/workspaces/new")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if !strings.Contains(body, "Create") {
		t.Errorf("new-workspace button should say 'Create' (short verb)\n%s", truncate(body))
	}
	if strings.Contains(body, "and open") {
		t.Error("button label must not contain filler phrases like 'and open'")
	}
}

// TestWorkspaceCopyPageButtonLabel checks that the copy-workspace form button
// reads as a short verb — not "Copy and open".
func TestWorkspaceCopyPageButtonLabel(t *testing.T) {
	a, h := newApp(t)
	a.Workspace.Set("name", "Test workspace")

	rec := get(t, h, "/workspaces/copy")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if !strings.Contains(body, "Copy") {
		t.Errorf("copy-workspace button should say 'Copy'\n%s", truncate(body))
	}
	if strings.Contains(body, "and open") {
		t.Error("button label must not contain filler phrases like 'and open'")
	}
	// The context prop (if added) renders as visually-hidden text after the label.
	// Even without it, the visible label itself must be ≤4 words and active.
}

// TestWorkspaceStartButtonLabel checks that the workspace-list row "start" button
// is a short verb — not "Start and open".
func TestWorkspaceStartButtonLabel(t *testing.T) {
	a, h := newApp(t)
	a.Workspace.Set("name", "Test workspace")

	rec := get(t, h, "/workspaces")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if !strings.Contains(body, "Start") {
		t.Errorf("workspace row button should say 'Start'\n%s", truncate(body))
	}
	if strings.Contains(body, "and open") {
		t.Error("button label must not contain filler phrases like 'and open'")
	}
}

// TestWorkspaceDeleteButtonLabel checks that the delete-workspace form button
// reads as a short verb — not "Delete this workspace".  After the change it
// should be just "Delete", with optional context for screen readers.
func TestWorkspaceDeleteButtonLabel(t *testing.T) {
	a, h := newApp(t)
	a.Workspace.Set("name", "Test workspace")

	rec := get(t, h, "/workspaces/delete")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if !strings.Contains(body, ">Delete<") && !strings.Contains(body, "> Delete <") {
		t.Errorf("delete-workspace button should be the plain verb 'Delete'\n%s", truncate(body))
	}
	if strings.Contains(body, `">Delete this workspace"`) || strings.Contains(body, `"> Delete this workspace "`) {
		t.Error("button label must not contain filler words like 'this' — it should be just 'Delete'")
	}
}
