package server_test

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// An AI app found on this computer is a press from using the workspace,
// and says what to do after in its own way; one not found is a story of
// its own; the apps on the internet say plainly why not, and what instead.
func TestYourAIAppsAreAPressEach(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", filepath.Join(home, "AppData"))
	t.Setenv("PATH", "")
	os.MkdirAll(filepath.Join(home, ".cursor"), 0o755)
	_, h := newApp(t)

	page := get(t, h, "/apps").Body.String()
	on := strings.Index(page, "On this computer")
	other := strings.Index(page, "Another app")
	if on < 0 || other < on || !strings.Contains(page[on:other], "Connect Cursor") {
		t.Fatalf("Cursor is found, a press: %s", truncate(page))
	}
	if !strings.Contains(page[other:], "Claude Desktop") || !strings.Contains(page[other:], `href="https://claude.ai/download"`) {
		t.Errorf("Claude Desktop, not found, is a story of its own")
	}
	if !strings.Contains(page, "only reach Sameway over the internet") {
		t.Error("ChatGPT says plainly why not")
	}

	postForm(t, h, "/apps/connect", url.Values{"app": {"cursor"}})
	raw, err := os.ReadFile(filepath.Join(home, ".cursor", "mcp.json"))
	if err != nil || !strings.Contains(string(raw), `"sameway"`) || !strings.Contains(string(raw), `"mcp"`) {
		t.Fatalf("Cursor's file starts Sameway: %s %v", raw, err)
	}
	if page := get(t, h, "/apps").Body.String(); !strings.Contains(page, "Connected. Cursor finds it in Settings, MCP") {
		t.Errorf("connected, it says what to do next in Cursor's way: %s", truncate(page))
	}
	if help := get(t, h, "/help").Body.String(); !strings.Contains(help, `href="/apps"`) {
		t.Error("Help leads to it")
	}
}
