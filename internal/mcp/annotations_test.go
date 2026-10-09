package mcp_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/mcp"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// Every tool a client is offered says what it is like, so the client can
// ask its person before what changes or reaches outside the workspace.
func TestEveryToolSaysWhatItIs(t *testing.T) {
	t.Parallel()
	_, replies := drive(t, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	tools, _ := result(t, replies[0])["tools"].([]any)
	if len(tools) == 0 {
		t.Fatal("no tools listed")
	}
	for _, raw := range tools {
		tool := raw.(map[string]any)
		ann, _ := tool["annotations"].(map[string]any)
		if ann == nil || tool["title"] == "" || ann["title"] != tool["title"] {
			t.Errorf("%v has no title or annotations: give its Op a Title (internal/chat), or ownTools one in internal/mcp/annotations.go", tool["name"])
			continue
		}
		if ann["readOnlyHint"] == true && ann["destructiveHint"] == true {
			t.Errorf("%v cannot both only read and take away", tool["name"])
		}
	}
}

// What says it only reads is called, and the workspace on disk is the same
// afterwards, byte for byte.
func TestReadOnlyToolsChangeNothing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := workspace.Init(dir, examples.FS, examples.StarterRoot, false); err != nil {
		t.Fatal(err)
	}
	a, err := app.Load(dir, false) // on disk, so the database is hashed too
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	task, err := a.Store.Create("task", map[string]any{"title": "Call plumber"})
	if err != nil {
		t.Fatal(err)
	}
	listed := answers(t, a, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)

	args := map[string]string{
		"describe":     `{}`,
		"look":         `{"path":"/t/task"}`,
		"find_records": `{"type":"task"}`,
		"get_record":   `{"type":"task","id":"` + task.ID + `"}`,
		"search":       `{"query":"plumber"}`,
		"details":      `{"name":"note"}`,
		"try":          `{"name":"update_record","arguments":{"type":"task","id":"` + task.ID + `","fields":{"done":true}}}`,
	}
	var calls []string
	for _, raw := range result(t, listed[0])["tools"].([]any) {
		tool := raw.(map[string]any)
		ann, _ := tool["annotations"].(map[string]any)
		if ann["readOnlyHint"] != true {
			continue
		}
		name := tool["name"].(string)
		in, ok := args[name]
		if !ok {
			t.Errorf("%s says it only reads; give it arguments here so the test calls it", name)
			continue
		}
		calls = append(calls, `{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"`+name+`","arguments":`+in+`}}`)
	}

	before := fingerprint(t, dir)
	for i, reply := range answers(t, a, calls...) {
		if body, isErr := text(t, reply); isErr {
			t.Errorf("%s failed, so it proves nothing: %.200s", calls[i], body)
		}
	}
	if after := fingerprint(t, dir); after != before {
		t.Errorf("a tool that says it only reads changed the workspace")
	}
	// And the same check sees a change when there is one.
	answers(t, a, tick(task.ID))
	if fingerprint(t, dir) == before {
		t.Error("the fingerprint missed a change, so it proves nothing")
	}
}

// answers runs a stdio server over a and gives back its replies.
func answers(t *testing.T, a *app.App, lines ...string) []map[string]any {
	t.Helper()
	var out bytes.Buffer
	srv := &mcp.Server{App: a, Version: "test", In: strings.NewReader(strings.Join(lines, "\n") + "\n"), Out: &out}
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatal(err)
	}
	var replies []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("reply is not JSON: %q", line)
		}
		replies = append(replies, m)
	}
	return replies
}

// fingerprint hashes every file in the workspace, by path and content.
func fingerprint(t *testing.T, dir string) string {
	t.Helper()
	h := sha256.New()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		// SQLite's shared-memory index changes as it is read.
		if strings.HasSuffix(path, "-shm") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		h.Write([]byte(rel + "\x00"))
		h.Write(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}
