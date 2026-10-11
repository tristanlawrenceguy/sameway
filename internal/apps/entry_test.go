package apps

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// An app's sameway entry is read for the workspace it names, so another
// workspace's is not taken for this one's, and taken out again leaving the
// other servers in the file as they were.
func TestAnEntryIsReadAndTakenOut(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ws, other := filepath.Join(dir, "home"), filepath.Join(dir, "work")
	for _, shape := range []string{"json", "toml"} {
		path := filepath.Join(dir, shape+".conf")
		app := App{Name: shape, Key: "mcpServers", Path: func(string) string { return path }, Shape: shape}
		if shape == "toml" {
			app.Key = "mcp_servers"
			os.WriteFile(path, []byte("model = \"x\"\n\n[mcp_servers.other]\ncommand = \"other\"\n"), 0o644)
		} else {
			os.WriteFile(path, []byte(`{"mcpServers":{"other":{"command":"other"}},"theme":"dark"}`), 0o644)
		}
		if Entry(app, ws) != "" || Connected(app, ws) {
			t.Fatalf("%s: no entry yet", shape)
		}
		if err := Write(path, app, Server(app, "sameway", other)); err != nil {
			t.Fatal(err)
		}
		if !Same(Entry(app, ws), other) || Connected(app, ws) {
			t.Errorf("%s: the entry is the other workspace's, not this one's: %q", shape, Entry(app, ws))
		}
		// Connecting this workspace replaces the entry, once.
		if err := Write(path, app, Server(app, "sameway", ws)); err != nil {
			t.Fatal(err)
		}
		raw, _ := os.ReadFile(path)
		if !Connected(app, ws) || strings.Count(string(raw), "--workspace") != 1 {
			t.Errorf("%s: one entry, this workspace's: %s", shape, raw)
		}
		if err := Remove(path, app); err != nil {
			t.Fatal(err)
		}
		raw, _ = os.ReadFile(path)
		if Entry(app, ws) != "" || strings.Contains(string(raw), "sameway") || !strings.Contains(string(raw), "other") {
			t.Errorf("%s: taken out, the other server kept: %s", shape, raw)
		}
		if shape == "json" && !strings.Contains(string(raw), `"theme": "dark"`) {
			t.Errorf("json: the rest of the file kept: %s", raw)
		}
	}
}
