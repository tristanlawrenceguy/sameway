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
	"github.com/tristanlawrenceguy/sameway/internal/server/servertest"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// computers are two copies of one workspace kept in step through one
// cloud folder, as on a desktop and a laptop.
type computers struct {
	cloud, folder string
	a, b          *app.App
	srvA, srvB    *server.Server
}

// twoComputers turns keeping in step on in a first workspace, which
// leaves a copy to start from, and starts a second from that copy.
func twoComputers(t *testing.T) computers {
	t.Helper()
	c := computers{cloud: t.TempDir()}
	machine := func() workspace.Machine {
		m := servertest.Machine(t)
		m.Cloud = workspace.Cloud{Name: "OneDrive", Dir: c.cloud}
		return m
	}
	c.a, _ = newAppWith(t, app.Options{Machine: machine()})
	c.srvA = server.New(c.a)
	if rec := postForm(t, c.srvA, "/cloud-sync", map[string][]string{"cloud": {"OneDrive"}}); rec.Code != 303 {
		t.Fatalf("%d %s", rec.Code, truncate(rec.Body.String()))
	}
	starts, _ := filepath.Glob(filepath.Join(c.cloud, cloudsync.Root, "*", cloudsync.Start))
	if len(starts) != 1 {
		t.Fatalf("a copy to start from: %v", starts)
	}
	c.folder = filepath.Dir(starts[0])
	data, _ := os.ReadFile(starts[0])
	dirB := filepath.Join(t.TempDir(), "laptop")
	if err := server.UnpackCopy(data, dirB); err != nil {
		t.Fatal(err)
	}
	b, err := app.Open(dirB, app.Options{Machine: machine()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { b.Close() })
	b.Chat.Provider, b.Chat.ProviderErr = nil, llm.ErrNotConfigured
	c.b, c.srvB = b, server.New(b)
	return c
}

// round passes changes both ways: B, then A, then B again.
func (c computers) round() {
	c.srvB.SyncRound()
	c.srvA.SyncRound()
	c.srvB.SyncRound()
}

// Two computers keep one workspace in step through a cloud folder: the
// first turns it on and leaves a copy to start from; the second starts
// from it as a copy of its own; then a note made on either, a title
// changed on either, and a file added reach the other; a third workspace
// is offered it, the one that has it is not.
func TestTwoComputersKeepInStepThroughACloudFolder(t *testing.T) {
	t.Parallel()
	c := twoComputers(t)
	a, b := c.a, c.b
	page := get(t, c.srvA, "/workspaces").Body.String()
	if !strings.Contains(page, "No other computer has opened it yet") {
		t.Fatalf("Workspaces says it is on: %s", truncate(page))
	}
	if b.Store.Origin() == a.Store.Origin() {
		t.Fatal("the copy names itself apart")
	}

	noteA, _ := a.Store.Create("note", map[string]any{"title": "Shopping"})
	c.srvA.SyncRound()
	c.srvB.SyncRound()
	if got, err := b.Store.Get("note", noteA.ID); err != nil || got.Fields["title"] != "Shopping" {
		t.Fatalf("A's note reaches B: %v %v", got, err)
	}
	noteB, _ := b.Store.Create("note", map[string]any{"title": "From the laptop"})
	b.Store.Update("note", noteA.ID, map[string]any{"title": "Shopping for Sunday"})
	os.MkdirAll(b.Workspace.FilesDir(), 0o755)
	os.WriteFile(filepath.Join(b.Workspace.FilesDir(), "photo.jpg"), []byte("jpeg"), 0o644)
	c.srvB.SyncRound()
	c.srvA.SyncRound()
	if got, err := a.Store.Get("note", noteB.ID); err != nil || got.Fields["title"] != "From the laptop" {
		t.Errorf("B's note reaches A: %v %v", got, err)
	}
	if got, _ := a.Store.Get("note", noteA.ID); got.Fields["title"] != "Shopping for Sunday" {
		t.Errorf("B's change reaches A: %v", got.Fields["title"])
	}
	if f, err := os.ReadFile(filepath.Join(a.Workspace.FilesDir(), "photo.jpg")); err != nil || string(f) != "jpeg" {
		t.Errorf("B's file reaches A: %v", err)
	}
	page = get(t, c.srvA, "/workspaces").Body.String()
	if !strings.Contains(page, "open on 1 other computer") || strings.Contains(page, "Open "+a.Workspace.Config.Name+" from OneDrive") {
		t.Errorf("A says it is open elsewhere, and is not offered itself: %s", truncate(page))
	}
	if !strings.Contains(page, "Last in step with OneDrive today at") {
		t.Errorf("A says when it was last in step: %s", truncate(page))
	}

	m := servertest.Machine(t)
	m.Cloud = workspace.Cloud{Name: "OneDrive", Dir: c.cloud}
	_, hc := newAppWith(t, app.Options{Machine: m})
	if page := get(t, hc, "/workspaces").Body.String(); !strings.Contains(page, "Open "+a.Workspace.Config.Name+" from OneDrive") {
		t.Errorf("another workspace is offered it: %s", truncate(page))
	}
}
