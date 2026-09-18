package server_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// One real turn through Claude Code, with a model named in the
// environment, for example SAMEWAY_CLAUDE_CODE_MODEL=haiku. It costs a
// little and takes a minute, so it runs only when asked for. Claude Code
// runs the tools itself over this workspace's MCP server, so the turn is
// the whole path: the prompt, the program, the tools landing in the
// store, the changes watched as they land, and the reply.
func TestARealTurnThroughClaudeCode(t *testing.T) {
	model := os.Getenv("SAMEWAY_CLAUDE_CODE_MODEL")
	if model == "" {
		t.Skip("set SAMEWAY_CLAUDE_CODE_MODEL=haiku to run one real turn through Claude Code")
	}
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude is not on PATH")
	}
	exe := buildSameway(t)
	dir := t.TempDir()
	if err := workspace.Init(dir, examples.FS, examples.StarterRoot, false); err != nil {
		t.Fatal(err)
	}
	a, err := app.Load(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	a.Chat.Provider, err = llm.New(llm.Config{Provider: "claude-code", Model: model, Workspace: dir, Executable: exe})
	if err != nil {
		t.Fatal(err)
	}
	a.Chat.ProviderErr = nil

	var kinds []string
	live := 0
	rec, err := a.Chat.SendLive(context.Background(), "", "Add a card block titled Shopping with the text: milk, bread. Then reply in one sentence.", "", func(e chat.Event) {
		kinds = append(kinds, e.Kind)
		if e.Kind == "change" {
			live++
		}
	})
	if err != nil {
		t.Fatalf("the turn failed: %v", err)
	}
	reply, _ := rec.Fields["content"].(string)
	if rec.Fields["role"] != "assistant" || strings.TrimSpace(reply) == "" {
		t.Errorf("the reply is the model's words, got %v: %q", rec.Fields["role"], reply)
	}
	changes, _ := rec.Fields["changes"].([]any)
	if len(changes) == 0 {
		t.Errorf("the reply carries what Claude Code changed over MCP, got none; events %v", kinds)
	}
	if live == 0 || kinds[len(kinds)-1] != "done" {
		t.Errorf("changes are told as they land, before the turn is done; events %v", kinds)
	}
	blocks, _ := a.Store.List(chat.BlockType, store.ListOptions{})
	if len(blocks) == 0 {
		t.Error("a block is on the canvas afterwards")
	}
	t.Logf("reply: %s", reply)
}

// buildSameway builds this program for the MCP server Claude Code is
// handed. Windows Smart App Control sometimes refuses a fresh build; a
// build with another build id is another file, so it tries a few.
func buildSameway(t *testing.T) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "sameway.exe")
	for try := 0; try < 6; try++ {
		cmd := exec.Command("go", "build", "-ldflags", fmt.Sprintf("-buildid=test-%d-%d", os.Getpid(), try), "-o", out, "../../cmd/sameway")
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("building sameway: %v\n%s", err, b)
		}
		if exec.Command(out, "--version").Run() == nil {
			return out
		}
	}
	t.Fatal("could not build a sameway this machine lets run")
	return ""
}
