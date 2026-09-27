package chat_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEventDelegatesToCleanSettingChange checks that server/activity.go event()
// delegates to chat.CleanSettingChange instead of having its own inline setting-
// change logic. This covers acceptance item 3.
func TestEventDelegatesToCleanSettingChange(t *testing.T) {
	root := findRepoRoot(t)
	src, err := os.ReadFile(filepath.Join(root, "internal", "server", "activity.go"))
	if err != nil {
		t.Fatalf("read activity.go: %v", err)
	}
	if !strings.Contains(string(src), "chat.CleanSettingChange") {
		t.Errorf("internal/server/activity.go event() must call chat.CleanSettingChange for human-readable transformation\nwant: chat.CleanSettingChange in the event function\ngot: (no delegation found — inline logic still present)")
	}
}

// TestUndoDelegatesToCleanSettingChange checks that server/undo.go cleanSettingChange
// delegates to chat.CleanSettingChange instead of having its own inline logic. This
// covers acceptance item 3.
func TestUndoDelegatesToCleanSettingChange(t *testing.T) {
	root := findRepoRoot(t)
	src, err := os.ReadFile(filepath.Join(root, "internal", "server", "undo.go"))
	if err != nil {
		t.Fatalf("read undo.go: %v", err)
	}
	if !strings.Contains(string(src), "chat.CleanSettingChange") {
		t.Errorf("internal/server/undo.go cleanSettingChange must delegate to chat.CleanSettingChange\nwant: chat.CleanSettingChange in the function body\ngot: (no delegation found — inline logic still present)")
	}
}

// TestNoDuplicateSettingChangeLogic checks that there is no second copy of the
// setting-change transformation logic outside internal/chat/activity.go. After
// sharing via CleanSettingChange, the only place with this logic should be the
// shared function itself. This covers acceptance item 3.
func TestNoDuplicateSettingChangeLogic(t *testing.T) {
	root := findRepoRoot(t)

	// event() should not have inline ui./llm. prefix checks.
	actSrc, err := os.ReadFile(filepath.Join(root, "internal", "server", "activity.go"))
	if err != nil {
		t.Fatalf("read server/activity.go: %v", err)
	}
	if strings.Contains(string(actSrc), `HasPrefix(target, "ui.")`) {
		t.Errorf("internal/server/activity.go still has inline setting-change logic instead of delegating to chat.CleanSettingChange\nwant: no HasPrefix(ui./llm.) checks — delegation only\ngot: inline transformation still present")
	}

	// cleanSettingChange() should not have its own inline transformation.
	undoSrc, err := os.ReadFile(filepath.Join(root, "internal", "server", "undo.go"))
	if err != nil {
		t.Fatalf("read server/undo.go: %v", err)
	}
	if strings.Contains(string(undoSrc), `HasPrefix(component, "ui.")`) {
		t.Errorf("internal/server/undo.go still has inline setting-change logic instead of delegating to chat.CleanSettingChange\nwant: no HasPrefix(ui./llm.) checks — delegation only\ngot: inline transformation still present")
	}
}

// TestSharedTransformationFunctionExists reads internal/chat/activity.go and
// asserts that an exported CleanSettingChange function is declared there. This
// covers acceptance item 3: the shared logic must live in a single location.
func TestSharedTransformationFunctionExists(t *testing.T) {
	root := findRepoRoot(t)
	src, err := os.ReadFile(filepath.Join(root, "internal", "chat", "activity.go"))
	if err != nil {
		t.Fatalf("read chat/activity.go: %v", err)
	}
	if !strings.Contains(string(src), "func CleanSettingChange") {
		t.Errorf("internal/chat/activity.go must export CleanSettingChange so that server/activity.go event() and server/undo.go cleanSettingChange can share the logic\nwant: func CleanSettingChange(c map[string]any) bool\ngot: (function not found in activity.go)")
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("could not find repo root (no go.mod)")
	return ""
}
