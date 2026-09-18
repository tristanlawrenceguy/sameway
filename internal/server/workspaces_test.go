package server_test

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// fleet stands in for the command line: it starts a workspace as a server
// in this process, on the address it is given, and notes when it is told
// to stop.
type fleet struct {
	t       *testing.T
	started []string
	exited  bool
}

func (f *fleet) launch(dir, addr string) error {
	a, err := app.Load(dir, false)
	if err != nil {
		return err
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	f.t.Cleanup(func() { l.Close(); a.Close() })
	go http.Serve(l, server.New(a))
	f.started = append(f.started, dir)
	return nil
}

// Every workspace this machine has opened is offered from any other: to
// open when it runs, to start when it does not. A new blank workspace and
// a copy of this one go beside it and open in servers of their own, and
// this one can be deleted once its name is typed, which sends the person
// on to another and stops this server.
func TestAWorkspaceOpensTheOthers(t *testing.T) {
	known := filepath.Join(t.TempDir(), "workspaces.json")
	t.Setenv("SAMEWAY_KNOWN", known)
	a, _ := newApp(t)
	f := &fleet{t: t}
	h := server.New(a).WithFleet(&server.Fleet{Launch: f.launch, Exit: func() { f.exited = true }})
	workspace.Remember(a.Workspace.Dir, "")
	a.Workspace.Set("name", "Home base")

	other := filepath.Join(filepath.Dir(a.Workspace.Dir), "garden")
	if err := workspace.Init(other, examples.FS, examples.StarterRoot, false); err != nil {
		t.Fatal(err)
	}
	ws, _ := workspace.Load(other)
	ws.Set("name", "Garden")
	workspace.Remember(other, "")

	page := get(t, h, "/workspaces").Body.String()
	if !strings.Contains(page, "Home base") || !strings.Contains(page, ">Garden<") || !strings.Contains(page, "Start and open") {
		t.Errorf("the page shows this workspace and offers to start the other\n%s", page)
	}

	// Starting the other workspace gives it a server of its own and sends
	// the person there; from then on it is open, with a link.
	res := postForm(t, h, "/workspaces/start", url.Values{"dir": {other}})
	wantStatus(t, res, http.StatusSeeOther)
	to := res.Header().Get("Location")
	if !strings.HasPrefix(to, "http://127.0.0.1:") || len(f.started) != 1 {
		t.Fatalf("the other workspace was started and the person sent to it, got %q after %v", to, f.started)
	}
	if page := get(t, h, "/workspaces").Body.String(); !strings.Contains(page, `href="`+to+`"`) {
		t.Errorf("a running workspace is a link to open\n%s", page)
	}
	if body, _ := os.ReadFile(known); !strings.Contains(string(body), "127.0.0.1:") {
		t.Error("the address a workspace runs at is remembered")
	}

	// A new blank workspace: beside this one, named, with no content, and
	// talking to the same model.
	res = postForm(t, h, "/workspaces/new", url.Values{"name": {"Kitchen plans"}})
	wantStatus(t, res, http.StatusSeeOther)
	made := filepath.Join(filepath.Dir(a.Workspace.Dir), "kitchen-plans")
	nws, err := workspace.Load(made)
	if err != nil || nws.Config.Name != "Kitchen plans" || nws.Config.LLM.Provider != a.Workspace.Config.LLM.Provider {
		t.Fatalf("the new workspace is beside this one, named, with the same model: %v %+v", err, nws)
	}
	na, err := app.Load(made, false)
	if err != nil {
		t.Fatal(err)
	}
	defer na.Close()
	if n, _ := na.Store.Count(chat.BlockType); n != 0 {
		t.Errorf("a new workspace starts blank, with none of the starter examples, got %d blocks", n)
	}
	if len(f.started) != 2 {
		t.Error("the new workspace was started")
	}

	// A copy: everything here, database included.
	a.Store.Create(chat.BlockType, a.Chat.BlockFields(map[string]any{"component": "heading", "props": map[string]any{"text": "Keep me"}}))
	res = postForm(t, h, "/workspaces/copy", url.Values{"name": {"Home base copy"}})
	wantStatus(t, res, http.StatusSeeOther)
	copied := filepath.Join(filepath.Dir(a.Workspace.Dir), "home-base-copy")
	ca, err := app.Load(copied, false)
	if err != nil {
		t.Fatal(err)
	}
	defer ca.Close()
	blocks, _ := ca.Store.List(chat.BlockType, store.ListOptions{})
	found := false
	for _, b := range blocks {
		if props, _ := b.Fields["props"].(map[string]any); props != nil && props["text"] == "Keep me" {
			found = true
		}
	}
	if !found || ca.Workspace.Config.Name != "Home base copy" {
		t.Errorf("the copy has the content and its own name, got %d blocks named %q", len(blocks), ca.Workspace.Config.Name)
	}

	// Deleting needs the name typed exactly.
	res = postForm(t, h, "/workspaces/delete", url.Values{"confirm": {"home base"}})
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "type its name exactly") {
		t.Errorf("a wrong name does not delete, got %d", res.Code)
	}
	if _, err := os.Stat(a.Workspace.Dir); err != nil {
		t.Fatal("the workspace is still there")
	}
	res = postForm(t, h, "/workspaces/delete", url.Values{"confirm": {"Home base"}})
	wantStatus(t, res, http.StatusSeeOther)
	if _, err := os.Stat(a.Workspace.Dir); err == nil {
		t.Error("the workspace folder is gone")
	}
	if !f.exited || !strings.HasPrefix(res.Header().Get("Location"), "http://127.0.0.1:") {
		t.Errorf("the person is sent to another workspace and this server stops, got %q exited %v", res.Header().Get("Location"), f.exited)
	}
	for _, k := range workspace.KnownWorkspaces() {
		if k.Dir == a.Workspace.Dir {
			t.Error("a deleted workspace is forgotten")
		}
	}
}
