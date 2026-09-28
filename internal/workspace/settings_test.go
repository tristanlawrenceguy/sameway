package workspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// Any setting in workspace.yaml can be changed by name: a line in its
// section changes or is added, the comments survive, a value that needs
// quoting gets it, and a secret is never written, only the name of the
// environment variable that holds it.
func TestAnySettingCanBeChangedByName(t *testing.T) {
	dir := t.TempDir()
	if err := workspace.Init(dir, examples.FS, examples.StarterRoot, false); err != nil {
		t.Fatal(err)
	}
	w, err := workspace.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	file := func() string { b, _ := os.ReadFile(filepath.Join(dir, "workspace.yaml")); return string(b) }

	if err := w.Set("ui.lists", "all"); err != nil {
		t.Fatal(err)
	}
	if w.Config.UI.Lists != "all" || !strings.Contains(file(), "  lists: all\n") || strings.Count(file(), "lists:") != 1 {
		t.Errorf("ui.lists should change its own line:\n%s", file())
	}
	if err := w.Set("name", "The Garden"); err != nil {
		t.Fatal(err)
	}
	if w.Config.Name != "The Garden" || !strings.Contains(file(), "name: The Garden\n") {
		t.Errorf("a top-level setting changes its line:\n%s", file())
	}
	if err := w.Set("chat.system_prompt", "Be brief: always."); err != nil {
		t.Fatal(err)
	}
	if w.Config.Chat.SystemPrompt != "Be brief: always." || !strings.Contains(file(), "system_prompt: 'Be brief: always.'") {
		t.Errorf("a value with a colon is quoted so the file still reads:\n%s", file())
	}
	if err := w.Set("chat.history_limit", "12"); err != nil || w.Config.Chat.HistoryLimit != 12 {
		t.Errorf("a number is read back as one: %v %d", err, w.Config.Chat.HistoryLimit)
	}
	if !strings.Contains(file(), "# auto:") {
		t.Error("the comments in the file should survive")
	}

	for key, bad := range map[string]string{"ui.lists": "some", "chat.history_limit": "many", "llm.api_key_env": "sk-abc123def456", "ui.font": "serif"} {
		if err := w.Set(key, bad); err == nil {
			t.Errorf("%s=%q should be refused", key, bad)
		}
	}
	if err := w.Set("llm.api_key_env", "MY_MODEL_KEY"); err != nil || w.Config.LLM.APIKeyEnv != "MY_MODEL_KEY" {
		t.Errorf("the name of the variable is fine: %v", err)
	}
	if reloaded, _ := workspace.Load(dir); reloaded.Config.Name != "The Garden" || reloaded.Config.UI.Lists != "all" {
		t.Error("the file reads back as set")
	}
}
