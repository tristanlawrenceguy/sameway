package server_test

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Photos taken on a phone come in by themselves: the folder iCloud,
// OneDrive or Google Drive copies them to on this computer is named on
// Workspaces, and each new photo or video there is kept as a file, once,
// after it has finished arriving. Those there before stay out, and so
// does anything that is not a photo or a video.
func TestNewPhotosInAFolderComeInByThemselves(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	dir := t.TempDir()
	write := func(name string, age time.Duration) {
		t.Helper()
		p := filepath.Join(dir, name)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, pngOf(t, 8, 8), 0o644); err != nil {
			t.Fatal(err)
		}
		at := time.Now().Add(-age)
		os.Chtimes(p, at, at)
	}
	write("Old.png", time.Hour)
	if page := get(t, h, "/workspaces").Body.String(); !strings.Contains(page, "Photos you take") || !strings.Contains(page, "Keep new photos from this folder") {
		t.Fatalf("Workspaces offers it: %.3000s", page)
	}
	res := postForm(t, h, "/photos/from", url.Values{"folder": {dir}})
	wantStatus(t, res, 303)

	write("2026/IMG_0002.png", time.Hour)
	write("IMG_0003.png", 0) // still being copied
	write("notes.txt", time.Hour)
	write(".trash/IMG_0004.png", time.Hour)
	run := func() {
		for _, j := range h.(*server.Server).BackgroundJobs() {
			if j.Name == "New photos kept from the phone's folder" {
				j.Run(context.Background(), time.Now())
			}
		}
	}
	names := func() []string {
		recs, _ := a.Store.List(records.FileType, store.ListOptions{})
		var out []string
		for _, r := range recs {
			out = append(out, r.Fields["name"].(string))
		}
		return out
	}
	run()
	if got := strings.Join(names(), ","); got != "IMG_0002.png" {
		t.Fatalf("only the new photo that has arrived whole: %s", got)
	}
	os.Chtimes(filepath.Join(dir, "IMG_0003.png"), time.Now().Add(-time.Minute), time.Now().Add(-time.Minute))
	run()
	run()
	if got := names(); len(got) != 2 {
		t.Errorf("each kept once, the other once it had arrived: %v", got)
	}
	if page := get(t, h, "/workspaces").Body.String(); !strings.Contains(page, "come into your files by themselves") {
		t.Errorf("Workspaces says it is on")
	}
	wantStatus(t, postForm(t, h, "/photos/from", url.Values{"folder": {""}}), 303)
	write("IMG_0005.png", time.Hour)
	run()
	if got := names(); len(got) != 2 {
		t.Errorf("stopped, nothing more comes: %v", got)
	}
}
