package server_test

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// What Windows marks hidden or the system's own, as desktop.ini in every
// folder is, is passed over as one named with a dot is: not found, and
// not brought in when asked for by its path.
func TestWhatWindowsHidesIsNotFound(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	docs := filepath.Join(t.TempDir(), "Documents")
	mark := func(path string, attrs uint32) {
		t.Helper()
		p, _ := syscall.UTF16PtrFromString(path)
		if err := syscall.SetFileAttributes(p, attrs); err != nil {
			t.Fatal(err)
		}
	}
	os.MkdirAll(filepath.Join(docs, "Kept away"), 0o755)
	write := func(parts ...string) string {
		p := filepath.Join(append([]string{docs}, parts...)...)
		os.WriteFile(p, []byte("x"), 0o644)
		return p
	}
	ini, away, plain := write("desktop.ini"), write("Kept away", "desktop notes.txt"), write("desktop plan.txt")
	mark(ini, syscall.FILE_ATTRIBUTE_HIDDEN|syscall.FILE_ATTRIBUTE_SYSTEM)
	mark(filepath.Join(docs, "Kept away"), syscall.FILE_ATTRIBUTE_HIDDEN|syscall.FILE_ATTRIBUTE_DIRECTORY)
	h.(*server.Server).FindOnComputer(docs)

	page := get(t, h, "/search?q=desktop").Body.String()
	if !strings.Contains(page, "desktop plan.txt") {
		t.Fatalf("a plain file is found: %.3000s", page)
	}
	if strings.Contains(page, "desktop.ini") || strings.Contains(page, "desktop notes.txt") {
		t.Errorf("nothing Windows hides is found: %.3000s", page)
	}
	for _, p := range []string{ini, away} {
		postForm(t, h, "/computer/bring", url.Values{"path": {p}})
	}
	postForm(t, h, "/computer/bring", url.Values{"path": {plain}})
	if recs, _ := a.Store.List(records.FileType, store.ListOptions{}); len(recs) != 1 || recs[0].Fields["name"] != "desktop plan.txt" {
		t.Errorf("only the plain file comes in: %v", recs)
	}
}
