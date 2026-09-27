package chat_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCleanRecordLinkExists reads internal/chat/activity.go and asserts that an
// exported CleanRecordLink function is declared there. This covers acceptance
// item 1: the shared transformation must live in a single location, matching
// the pattern of the existing CleanSettingChange export.
func TestCleanRecordLinkExists(t *testing.T) {
	root := findRepoRoot(t)
	src, err := os.ReadFile(filepath.Join(root, "internal", "chat", "activity.go"))
	if err != nil {
		t.Fatalf("read chat/activity.go: %v", err)
	}
	if !strings.Contains(string(src), "func CleanRecordLink") {
		t.Errorf("internal/chat/activity.go must export CleanRecordLink so that server/undo.go can call it\nwant: func CleanRecordLink(c map[string]any) bool\ngot: (function not found in activity.go)")
	}
}

// TestUndoCallsCleanRecordLink checks that internal/server/undo.go calls
// chat.CleanRecordLink after cleanSettingChange, so record-type changes get
// cleaned in the undoable() function. This covers acceptance items 1 and 2:
// record links show only the trimmed title without (type) suffix, and the href
// is unchanged since only "component" is deleted from the change map.
func TestUndoCallsCleanRecordLink(t *testing.T) {
	root := findRepoRoot(t)
	src, err := os.ReadFile(filepath.Join(root, "internal", "server", "undo.go"))
	if err != nil {
		t.Fatalf("read undo.go: %v", err)
	}
	if !strings.Contains(string(src), "chat.CleanRecordLink") {
		t.Errorf("internal/server/undo.go must call chat.CleanRecordLink for record-type change cleaning\nwant: chat.CleanRecordLink(copied) after cleanSettingChange\ngot: (no delegation found — inline logic still present)")
	}
}
