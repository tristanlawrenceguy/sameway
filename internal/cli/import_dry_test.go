package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sameway import --dry-run says what it would make, change and remove,
// and changes nothing; a real import is one entry in the log.
func TestImportSaysWhatItWouldDoFirst(t *testing.T) {
	dir := initWorkspace(t)
	run(t, dir, "note", "create", "--set", "title=Keep me")
	os.RemoveAll(filepath.Join(dir, "content", "note"))

	dry := run(t, dir, "import", "--dry-run")
	if dry.code != 0 || !strings.Contains(dry.stdout, "would make 0, change 0, remove 1") || !strings.Contains(dry.stdout, "remove note/") || !strings.Contains(dry.stdout, "nothing was changed") {
		t.Fatalf("a dry run says what it would remove: %+v", dry)
	}
	if list := run(t, dir, "note", "list"); !strings.Contains(list.stdout, "Keep me") {
		t.Error("a dry run removes nothing")
	}
	if log := run(t, dir, "activity", "list"); strings.Contains(log.stdout, "synced") {
		t.Error("a dry run logs nothing")
	}
}
