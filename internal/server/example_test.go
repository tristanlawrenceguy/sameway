package server_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// Try it with an example makes a workspace of its own with a week in it,
// and its home page shows every block it was given, none set up wrong.
func TestAnExampleWorkspaceShowsAWeek(t *testing.T) {
	t.Setenv("SAMEWAY_KNOWN", filepath.Join(t.TempDir(), "known.json"))
	a, _ := newApp(t)
	h := server.New(a).WithFleet(&server.Fleet{Launch: func(dir, addr string) error { return nil }, Exit: func() {}})
	if page := get(t, h, "/workspaces").Body.String(); !strings.Contains(page, ">Open an example<") {
		t.Fatalf("Workspaces offers it: %s", truncate(page))
	}
	postForm(t, h, "/workspaces/example", nil)
	ex, err := app.Load(filepath.Join(filepath.Dir(a.Workspace.Dir), "Example"), false)
	if err != nil {
		t.Fatalf("the example is a workspace beside this one: %v", err)
	}
	defer ex.Close()
	for typ, want := range map[string]int{"task": 6, "event": 3, "note": 2, "habit": 1, "entry": 5, "person": 1} {
		if n, _ := ex.Store.Count(typ); n != want {
			t.Errorf("%d %s, want %d", n, typ, want)
		}
	}
	if n, _ := a.Store.Count("task"); n != 0 {
		t.Error("this workspace is left as it was")
	}
	home := get(t, server.New(ex), "/").Body.String()
	for _, want := range []string{"Due this week", "Call the bank about the card", "Water"} {
		if !strings.Contains(home, want) {
			t.Errorf("its home shows %s", want)
		}
	}
	if strings.Contains(home, `data-component="problem"`) {
		t.Errorf("no block is set up wrong: %s", truncate(home))
	}
}
