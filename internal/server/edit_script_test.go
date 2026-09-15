package server_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEditScriptReadsDataEditAction verifies that the inline edit script reads
// a data-edit-action attribute from the block element when building its form,
// and falls back to /canvas/{id}/props when absent. This covers acceptance
// items 1 and 2 of task 0093.
func TestEditScriptReadsDataEditAction(t *testing.T) {
	script := readEditScript(t)

	// Acceptance item 1: the script must look up data-edit-action on the block.
	if !strings.Contains(script, `block.getAttribute("data-edit-action")`) &&
		!strings.Contains(script, "block.getAttribute('data-edit-action')") {
		t.Error("08-edit.js: expected block.getAttribute(\"data-edit-action\") to read a custom edit endpoint from the block element")
	}

	// Acceptance item 2: when no data-edit-action is set, it must fall back to /canvas/{id}/props.
	if !strings.Contains(script, "/canvas/"+"+ id + "+"/props") &&
		!strings.Contains(script, "\"/canvas/\"+id+\"/props\"") {
		t.Error("08-edit.js: expected a fallback to /canvas/{id}/props when data-edit-action is absent")
	}

	// The form action must be set from the resolved variable (not hardcoded).
	if !strings.Contains(script, "form.action = action") &&
		!strings.Contains(script, `form.action = action`) {
		t.Error("08-edit.js: expected form.action to be set from a variable, not hardcoded")
	}
}

// TestEditScriptDoesNotRemoveCanvasFallback ensures the script still builds
// forms targeting /canvas/{id}/props when no data-edit-action is present —
// matching acceptance item 3 so canvas blocks continue to work unchanged.
func TestEditScriptDoesNotRemoveCanvasFallback(t *testing.T) {
	script := readEditScript(t)

	// The fallback path string must still appear in the script body, because
	// when data-edit-action is absent the script falls back to /canvas/{id}/props.
	if !strings.Contains(script, "/canvas/") || !strings.Contains(script, "props") {
		t.Error("08-edit.js: the /canvas/.../props fallback path must still be present for canvas blocks that do not set data-edit-action")
	}

	// The form action must now come from a variable, not hardcoded on one line.
	if !strings.Contains(script, "form.action = action") {
		t.Error("08-edit.js: the hardcoded form.action should have been replaced with an 'action' variable that reads data-edit-action and falls back to /canvas/{id}/props")
	}

	// The fallback must explicitly set action when no attribute is found.
	if !strings.Contains(script, `!action`) {
		t.Error("08-edit.js: expected a check for missing data-edit-action (e.g., if (!action)) before falling back to the /canvas/ default")
	}
}

func readEditScript(t *testing.T) string {
	t.Helper()
	root := findRepoRoot(t)
	path := filepath.Join(root, "design", "base", "08-edit.js")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read 08-edit.js: %v", err)
	}
	return string(data)
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("could not find repository root (no .git directory)")
	return ""
}
