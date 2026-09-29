package server_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/mcp"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// everything is how many records of each kind the workspace holds, and
// the files in its folder, to tell whether anything was written.
func everything(t *testing.T, a *app.App) string {
	t.Helper()
	var b strings.Builder
	for _, typ := range a.Types.Types {
		recs, _ := a.Store.List(typ.Name, store.ListOptions{})
		for _, r := range recs {
			raw, _ := json.Marshal(r.Fields)
			b.WriteString(typ.Name + " " + r.ID + " " + r.UpdatedAt.String() + " " + string(raw) + "\n")
		}
	}
	filepath.Walk(a.Workspace.FilesDir(), func(p string, info os.FileInfo, err error) error {
		if err == nil {
			b.WriteString(p + "\n")
		}
		return nil
	})
	return b.String()
}

// Nothing anyone on the internet sends changes the workspace: no route,
// with any method, with any body, as a page's form or JSON, with a key or
// without; and the published MCP server has no tool that writes.
func TestThePublicCannotChangeAnything(t *testing.T) {
	a, h := newApp(t)
	srv := h.(*server.Server)
	note, _ := a.Store.Create("note", map[string]any{"title": "Sourdough"})
	a.Workspace.Config.Publish.Types = "note"
	pub := srv.Public(&mcp.Server{App: a, Version: "test", Published: func() map[string]bool { return srv.Published().Types }})
	before := everything(t, a)

	files, _ := filepath.Glob("*.go")
	pattern := regexp.MustCompile(`(?:HandleFunc|Handle)\("(?:([A-Z]+) )?(/[^"]*)"`)
	var routes []string
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, _ := os.ReadFile(f)
		for _, m := range pattern.FindAllStringSubmatch(string(src), -1) {
			path := strings.NewReplacer("{$}", "", "{type}", "note", "{id}", note.ID).Replace(m[2])
			path = regexp.MustCompile(`\{[^}]*\}`).ReplaceAllString(path, "x")
			routes = append(routes, path)
		}
	}
	bodies := map[string]string{
		"application/json":                  `{"title":"Changed from the internet","message":"hi","name":"x","prop-title":"Changed"}`,
		"application/x-www-form-urlencoded": "prop-title=Changed&title=Changed&message=hi&name=x",
	}
	for _, path := range routes {
		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
			for ct, body := range bodies {
				res := public(t, pub, method, path, body)
				_ = ct
				if res.Code < 400 && path != "/mcp" {
					t.Errorf("%s %s answered the internet %d", method, path, res.Code)
				}
			}
		}
	}
	for _, tool := range []string{"create_record", "update_record", "add_component", "undo_change", "set_setting", "import_records", "clear_conversation"} {
		body := public(t, pub, http.MethodPost, "/mcp", `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"`+tool+`","arguments":{"type":"note","id":"`+note.ID+`","fields":{"title":"Changed"},"component":"text","props":{"content":"x"}}}}`).Body.String()
		if !strings.Contains(body, "error") && !strings.Contains(body, `"isError":true`) {
			t.Errorf("published MCP ran %s: %s", tool, body)
		}
	}
	// Reading every page it may read writes nothing either.
	for _, path := range routes {
		public(t, pub, http.MethodGet, path, "")
	}
	if after := everything(t, a); after != before {
		t.Errorf("the internet changed the workspace:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// A recording on a published tab can be played, with its captions, but
// its sound is not copied out for anyone on the internet: that writes
// into the workspace's folder.
func TestThePublicPlaysARecordingButDoesNotCopyItsSound(t *testing.T) {
	a, h := newApp(t)
	srv := h.(*server.Server)
	os.MkdirAll(a.Workspace.FilesDir(), 0o755)
	os.WriteFile(filepath.Join(a.Workspace.FilesDir(), "talk.mp3"), []byte("ID3 not really"), 0o644)
	rec, _ := a.Store.Create("file", map[string]any{"title": "Talk", "name": "talk.mp3", "kind": "audio", "path": "talk.mp3", "text": "[0:00] Hello."})
	a.Store.Create("block", map[string]any{"component": "media", "canvas": "", "position": 1, "props": map[string]any{"src": "/files/" + rec.ID, "title": "Talk"}})
	a.Workspace.Config.Publish.Tabs = "Home"
	pub := srv.Public(nil)
	if res := public(t, pub, http.MethodGet, "/files/"+rec.ID, ""); res.Code != http.StatusOK {
		t.Fatalf("a published recording plays: %d", res.Code)
	}
	for _, path := range []string{"/files/" + rec.ID + "/sound", "/files/" + rec.ID + "/sound/0"} {
		if res := public(t, pub, http.MethodGet, path, ""); res.Code == http.StatusOK {
			t.Errorf("%s is the owner's to ask for: %d", path, res.Code)
		}
	}
	if _, err := os.Stat(filepath.Join(a.Workspace.FilesDir(), rec.ID+".sound")); err == nil {
		t.Error("nothing was copied out")
	}
}
