package workspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/update"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// A new version arrives on its own unless a person says otherwise, and
// saying otherwise is one line of workspace.yaml, changed by asking.
func TestUpdatesArriveOnTheirOwnUntilTurnedToManual(t *testing.T) {
	dir := t.TempDir()
	if err := workspace.Init(dir, examples.FS, examples.StarterRoot, false); err != nil {
		t.Fatal(err)
	}
	w, err := workspace.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if w.Config.Update.Mode != update.Auto {
		t.Errorf("a new workspace should update itself, got %q", w.Config.Update.Mode)
	}
	if err := w.Set("update.mode", update.Manual); err != nil {
		t.Fatal(err)
	}
	file, _ := os.ReadFile(filepath.Join(dir, "workspace.yaml"))
	if !strings.Contains(string(file), "  mode: manual\n") {
		t.Errorf("the mode should be one line under update:\n%s", file)
	}
	reloaded, err := workspace.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Config.Update.Mode != update.Manual {
		t.Errorf("manual should read back as manual, got %q", reloaded.Config.Update.Mode)
	}
	if err := w.Set("update.mode", "sometimes"); err == nil {
		t.Error("only auto and manual are modes")
	}
	// A workspace made before the setting existed, or one with nonsense in
	// it, updates itself: that is the default, not a failure to start.
	os.WriteFile(filepath.Join(dir, "workspace.yaml"), []byte("name: Old\nupdate:\n  mode: whenever\n"), 0o644)
	old, err := workspace.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if old.Config.Update.Mode != update.Auto {
		t.Errorf("an unreadable mode falls back to auto, got %q", old.Config.Update.Mode)
	}
}

// The assistant is told about every setting, so the one that turns
// updates to manual has to be in the list it reads.
func TestTheSettingIsOfferedToTheAssistant(t *testing.T) {
	var found bool
	for _, key := range workspace.SettingKeys() {
		if key == "update.mode" {
			found = true
		}
	}
	if !found {
		t.Fatalf("update.mode is missing from %v", workspace.SettingKeys())
	}
	doc := workspace.SettingsDoc()
	if !strings.Contains(doc, "update.mode") || !strings.Contains(doc, "(auto, manual)") {
		t.Errorf("the settings the assistant reads should say what update.mode takes: %s", doc)
	}
}
