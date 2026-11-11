package server_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Photos kept before "sort my photos" was set up are sorted when the
// action is run: each read, named and tagged, in the background, as if
// it had just come. Run again, it does only what is left; and it does
// not ask the person, as it changes only what is in the workspace.
func TestSortingRunGoesThroughWhatIsAlreadyHere(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	model := &seesReceipt{}
	a.Chat.Provider = model
	var ids []string
	for _, name := range []string{"IMG_1.png", "IMG_2.png"} {
		body, ct := multipartFile(t, name, string(pngOf(t, 40, 30)), nil)
		ids = append(ids, strings.TrimPrefix(do(t, h, "POST", "/t/file/upload", body, ct).Header().Get("Location"), "/t/file/"))
	}
	a.Store.Create(records.TagType, map[string]any{"name": "receipts", "means": "a receipt for something bought"})
	act, _ := a.Store.Create(records.ActionType, map[string]any{"title": "Sort photos", "kind": "classify", "when": "added", "what": "file", "tags": []any{"receipts"}})
	a.Automate.StartWorker()
	said := a.Automate.Run(context.Background(), act.ID, "")
	if said.IsErr || !strings.Contains(said.Text, "2 files") {
		t.Fatalf("it says it goes through the two: %q", said.Text)
	}
	tagged := func() int {
		n := 0
		for _, id := range ids {
			if f, _ := a.Store.Get(records.FileType, id); f != nil && strings.Contains(strings.Join(records.StringList(f.Fields["tags"]), ","), "receipts") {
				n++
			}
		}
		return n
	}
	classified := func() int {
		cls, _ := a.Store.List(records.ClassificationType, store.ListOptions{})
		return len(cls)
	}
	for end := time.Now().Add(10 * time.Second); (tagged() < 2 || classified() < 2) && time.Now().Before(end); time.Sleep(20 * time.Millisecond) {
	}
	if tagged() != 2 {
		t.Fatalf("both photos already here are read and tagged: %d", tagged())
	}
	if again := a.Automate.Run(context.Background(), act.ID, ""); !strings.Contains(again.Text, "already done") {
		t.Errorf("run again, nothing is left: %q", again.Text)
	}
	if said, _ := call(t, a, "run_action", map[string]any{"id": act.ID}); strings.Contains(said, "Send something") {
		t.Errorf("sorting is not asked about as if it sent something away: %q", said)
	}
}
