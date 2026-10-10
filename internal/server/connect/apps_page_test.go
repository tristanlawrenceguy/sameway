package connect_test

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// An AI app found on this computer is a press from using the workspace,
// and says what to do after in its own way; one not found is a story of
// its own; the apps on the internet say plainly why not, and what instead.
// What a connected app may do is said once before the press, exactly what
// it starts is there to read, and disconnecting is a press too, leaving
// the app's other servers as they were.
func TestYourAIAppsAreAPressEach(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", filepath.Join(home, "AppData"))
	t.Setenv("PATH", "")
	cursorFile := filepath.Join(home, ".cursor", "mcp.json")
	os.MkdirAll(filepath.Dir(cursorFile), 0o755)
	// Cursor already starts Sameway for another workspace, beside another server.
	os.WriteFile(cursorFile, []byte(`{"mcpServers":{"other":{"command":"x"},"sameway":{"command":"sameway","args":["--workspace","/elsewhere/Work","mcp"]}}}`), 0o644)
	a, h := newApp(t)

	page := get(t, h, "/apps").Body.String()
	on := strings.Index(page, "On this computer")
	other := strings.Index(page, "Another app")
	if on < 0 || other < on || !strings.Contains(page[on:other], "Connect Cursor here") {
		t.Fatalf("Cursor is found, a press: %s", truncate(page))
	}
	if !strings.Contains(said(page[on:other]), "Connected to another workspace, Work.") {
		t.Errorf("another workspace's entry is not taken for this one's: %s", said(page[on:other]))
	}
	if may := strings.Index(page, "can read and change everything in this workspace"); may < on || may > strings.Index(page, "Connect Cursor") {
		t.Error("what a connected app may do is said once, before the press")
	}
	if !strings.Contains(page, "What Cursor starts") || !strings.Contains(said(page), "--workspace "+a.Workspace.Dir+" mcp") && !strings.Contains(said(page), `--workspace "`+a.Workspace.Dir+`" mcp`) {
		t.Errorf("exactly what Cursor starts is there to read: %s", said(page[on:other]))
	}
	if !strings.Contains(page[other:], "Claude Desktop") || !strings.Contains(page[other:], `href="https://claude.ai/download"`) {
		t.Errorf("Claude Desktop, not found, is a story of its own")
	}
	if !strings.Contains(page, "only reach Sameway over the internet") {
		t.Error("ChatGPT says plainly why not")
	}

	postForm(t, h, "/apps/connect", url.Values{"app": {"cursor"}})
	raw, err := os.ReadFile(cursorFile)
	if err != nil || !strings.Contains(string(raw), `"other"`) || strings.Contains(string(raw), "elsewhere") {
		t.Fatalf("Cursor's file starts Sameway for this workspace, its other server kept: %s %v", raw, err)
	}
	page = get(t, h, "/apps").Body.String()
	if !strings.Contains(page, "Connected. Cursor finds it in Settings, MCP") || !strings.Contains(page, "Disconnect Cursor") {
		t.Errorf("connected, it says what to do next in Cursor's way, and how to take it back: %s", truncate(page))
	}

	done := after(t, h, postForm(t, h, "/apps/disconnect", url.Values{"app": {"cursor"}})).Body.String()
	raw, _ = os.ReadFile(cursorFile)
	if strings.Contains(string(raw), "sameway") || !strings.Contains(string(raw), `"other"`) {
		t.Errorf("disconnected, Sameway is out of Cursor's file and the rest kept: %s", raw)
	}
	if !strings.Contains(done, "Cursor is disconnected") || !strings.Contains(done, "Connect Cursor") {
		t.Errorf("it says so, and connecting is a press again: %s", truncate(done))
	}
	if help := get(t, h, "/help").Body.String(); !strings.Contains(help, `href="/apps"`) {
		t.Error("Help leads to it")
	}
}

// The agents let in with keys are on the page beside the apps, each with
// what it may do in plain words and when it was last used, and one press
// takes its key away, with an Undo that lets it back in.
func TestAgentsWithKeysAreSeenAndTakenAway(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	if page := get(t, h, "/apps").Body.String(); strings.Contains(page, "Agents with a key") {
		t.Error("no agent has a key, so nothing is said of them")
	}
	me := records.Who{Actor: "human"}
	key, _, err := records.LetAgentIn(a.Store, me, "Reader", records.View)
	if err != nil {
		t.Fatal(err)
	}
	records.LetAgentIn(a.Store, me, "Helper", records.Edit)

	page := said(get(t, h, "/apps").Body.String())
	for _, want := range []string{"Agents with a key", "May look at everything, and change nothing. Never used.", "May look at and change your lists and notes.", "Take its key away Reader", "Take its key away Helper"} {
		if !strings.Contains(page, want) {
			t.Errorf("the page says %q: %s", want, page)
		}
	}
	done := after(t, h, postForm(t, h, "/apps/agents/remove", url.Values{"name": {"Reader"}})).Body.String()
	if records.AgentByKey(a.Store, key) != nil {
		t.Fatal("Reader's key still works")
	}
	if !strings.Contains(done, "Reader&#39;s key is taken away") || !strings.Contains(done, "Undo") {
		t.Errorf("it says so, with an Undo: %s", truncate(done))
	}
	if strings.Contains(said(done), "away Reader") || !strings.Contains(said(done), "away Helper") {
		t.Errorf("Reader is gone from the list, Helper stays: %s", said(done))
	}
}
