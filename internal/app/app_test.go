package app_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// A workspace made before blocks had a region must still let the assistant
// put a block in a side pane after the upgrade. Before this was fixed the
// region was dropped on save and the block landed in the main column while
// the assistant reported it had gone to the pane.
func TestOldWorkspaceStillPlacesBlocksInPanes(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "workspace.yaml"), []byte("name: old\n"), 0o644)
	os.MkdirAll(filepath.Join(dir, "schema"), 0o755)
	block := "name: block\ninternal: true\nfields:\n  component: {type: string, required: true}\n  props: {type: json}\n  position: {type: int, default: 0}\n"
	message := "name: message\ninternal: true\nfields:\n  role: {type: enum, values: [user, assistant, error], required: true}\n  content: {type: text, required: true}\n"
	os.WriteFile(filepath.Join(dir, "schema", "block.yaml"), []byte(block), 0o644)
	os.WriteFile(filepath.Join(dir, "schema", "message.yaml"), []byte(message), 0o644)

	a, err := app.Load(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	blk, _ := a.Types.Get("block")
	if _, ok := blk.Field("region"); !ok {
		t.Fatal("block should have gained region from the built-in definition")
	}
	rec, err := a.Store.Create(chat.BlockType, a.Chat.BlockFields(map[string]any{
		"component": "calendar", "props": map[string]any{"month": "2026-09"}, "region": "left",
	}))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := a.Store.Get(chat.BlockType, rec.ID)
	if got.Fields["region"] != "left" {
		t.Errorf("region should survive the round trip, got %v", got.Fields["region"])
	}
}
