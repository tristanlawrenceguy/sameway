package server_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/cloudsync"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// Two computers keep one workspace in step through a cloud folder: the
// first turns it on and leaves a copy to start from; the second starts
// from it as a copy of its own; then a note made on either, a title
// changed on either, and a file added reach the other; a third workspace
// is offered it, the one that has it is not.
func TestTwoComputersKeepInStepThroughACloudFolder(t *testing.T) {
	cloud := t.TempDir()
	server.UseClouds([]workspace.Cloud{{Name: "OneDrive", Dir: cloud}})
	defer server.UseClouds(nil)

	a, _ := newApp(t)
	srvA := server.New(a)
	ha := srvA
	page := get(t, ha, "/workspaces").Body.String()
	if !strings.Contains(page, "Keep it in step through OneDrive") {
		t.Fatalf("Workspaces offers it: %s", truncate(page))
	}
	if rec := postForm(t, ha, "/cloud-sync", map[string][]string{"cloud": {"OneDrive"}}); rec.Code != 303 {
		t.Fatalf("%d %s", rec.Code, truncate(rec.Body.String()))
	}
	starts, _ := filepath.Glob(filepath.Join(cloud, cloudsync.Root, "*", cloudsync.Start))
	if len(starts) != 1 {
		t.Fatalf("a copy to start from: %v", starts)
	}

	// The second computer starts from the copy.
	data, _ := os.ReadFile(starts[0])
	dirB := filepath.Join(t.TempDir(), "laptop")
	if err := server.UnpackCopy(data, dirB); err != nil {
		t.Fatal(err)
	}
	b, err := app.Load(dirB, false)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	b.Chat.Provider, b.Chat.ProviderErr = nil, llm.ErrNotConfigured
	srvB := server.New(b)
	if b.Store.Origin() == a.Store.Origin() {
		t.Fatal("the copy names itself apart")
	}

	noteA, _ := a.Store.Create("note", map[string]any{"title": "Shopping"})
	srvA.SyncRound()
	srvB.SyncRound()
	if got, err := b.Store.Get("note", noteA.ID); err != nil || got.Fields["title"] != "Shopping" {
		t.Fatalf("A's note reaches B: %v %v", got, err)
	}
	noteB, _ := b.Store.Create("note", map[string]any{"title": "From the laptop"})
	b.Store.Update("note", noteA.ID, map[string]any{"title": "Shopping for Sunday"})
	os.MkdirAll(b.Workspace.FilesDir(), 0o755)
	os.WriteFile(filepath.Join(b.Workspace.FilesDir(), "photo.jpg"), []byte("jpeg"), 0o644)
	srvB.SyncRound()
	srvA.SyncRound()
	if got, err := a.Store.Get("note", noteB.ID); err != nil || got.Fields["title"] != "From the laptop" {
		t.Errorf("B's note reaches A: %v %v", got, err)
	}
	if got, _ := a.Store.Get("note", noteA.ID); got.Fields["title"] != "Shopping for Sunday" {
		t.Errorf("B's change reaches A: %v", got.Fields["title"])
	}
	if f, err := os.ReadFile(filepath.Join(a.Workspace.FilesDir(), "photo.jpg")); err != nil || string(f) != "jpeg" {
		t.Errorf("B's file reaches A: %v", err)
	}
	if page := get(t, ha, "/workspaces").Body.String(); !strings.Contains(page, "open on 1 other computer") || strings.Contains(page, "Open "+a.Workspace.Config.Name+" from OneDrive") {
		t.Errorf("A says it is open elsewhere, and is not offered itself: %s", truncate(page))
	}

	_, hc := newApp(t)
	if page := get(t, hc, "/workspaces").Body.String(); !strings.Contains(page, "Open "+a.Workspace.Config.Name+" from OneDrive") {
		t.Errorf("another workspace is offered it: %s", truncate(page))
	}
}
