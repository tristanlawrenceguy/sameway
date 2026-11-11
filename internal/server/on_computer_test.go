package server_test

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A file on this computer that is not in Sameway is found from the search
// page by the words of its name, in the owner's Documents, Downloads and
// Desktop only, and brought in it is a file, opened on its own page. A
// path from anywhere else is refused, whatever the page was sent.
func TestAFileOnThisComputerIsFoundByNameAndBroughtIn(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	home := t.TempDir()
	write := func(parts ...string) string {
		t.Helper()
		p := filepath.Join(append([]string{home}, parts...)...)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte("Rent for the flat, paid each month."), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	lease := write("Documents", "House", "Flat Lease 2026.txt")
	write("Downloads", "lease form.txt") // one word, not both
	draft := write("Documents", ".old", "Flat lease draft.txt")
	outside := write("Elsewhere", "Flat lease copy.txt")
	h.(*server.Server).FindOnComputer(filepath.Join(home, "Documents"), filepath.Join(home, "Downloads"))

	page := get(t, h, "/search?q="+url.QueryEscape("lease FLAT")).Body.String()
	if !strings.Contains(page, "On this computer") || !strings.Contains(page, "Flat Lease 2026.txt") || !strings.Contains(page, "Bring it in") {
		t.Fatalf("the search finds it on this computer: %.3000s", page)
	}
	if strings.Contains(page, "No results") || !strings.Contains(page, "Nothing in Sameway matches") {
		t.Errorf("found here, the page does not say no results")
	}
	for _, not := range []string{"lease form.txt", "Flat lease draft.txt", "Flat lease copy.txt"} {
		if strings.Contains(page, not) {
			t.Errorf("%s is not offered", not)
		}
	}

	res := postForm(t, h, "/computer/bring", url.Values{"path": {lease}})
	wantStatus(t, res, 303)
	recs, _ := a.Store.List(records.FileType, store.ListOptions{})
	if len(recs) != 1 || recs[0].Fields["name"] != "Flat Lease 2026.txt" {
		t.Fatalf("it is kept as a file: %v", recs)
	}
	if to := res.Header().Get("Location"); !strings.HasPrefix(to, "/t/file/"+recs[0].ID) {
		t.Errorf("it opens on its own page: %s", to)
	}

	for _, p := range []string{outside, draft, filepath.Join(home, "Documents", "..", "Elsewhere", "Flat lease copy.txt"), filepath.Join(home, "Documents"), "Flat Lease 2026.txt"} {
		postForm(t, h, "/computer/bring", url.Values{"path": {p}})
	}
	if recs, _ := a.Store.List(records.FileType, store.ListOptions{}); len(recs) != 1 {
		t.Errorf("nothing outside the folders or hidden, no folder and no bare name comes in: %d files", len(recs))
	}
}
