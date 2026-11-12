package server_test

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
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

// tagsAll gives every record the tag receipts, once let go.
type tagsAll struct {
	gate  chan struct{}
	asked atomic.Int64
}

func (m *tagsAll) Name() string { return "tags all" }
func (m *tagsAll) Complete(context.Context, llm.Request) (*llm.Response, error) {
	<-m.gate
	m.asked.Add(1)
	return &llm.Response{Text: `{"tags": [{"tag": "receipts", "why": "a receipt"}]}`}, nil
}

// An action going through three hundred photos already here does not
// crowd out what comes meanwhile: a note shared then is sorted, where
// before it was dropped by a queue full of photos. Pressed again, it
// says it is going already, rather than doing each twice.
func TestWhatComesWhileSortingWhatIsHereIsSorted(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	model := &tagsAll{gate: make(chan struct{})}
	a.Chat.Provider = model
	for i := range 300 {
		a.Store.Create(records.FileType, map[string]any{"title": fmt.Sprintf("IMG_%d", i), "kind": "image"})
	}
	a.Store.Create(records.TagType, map[string]any{"name": "receipts", "means": "a receipt for something bought"})
	photos, _ := a.Store.Create(records.ActionType, map[string]any{"title": "Sort photos", "kind": "classify", "when": "added", "what": "file", "tags": []any{"receipts"}})
	a.Store.Create(records.ActionType, map[string]any{"title": "Sort notes", "kind": "classify", "when": "added", "what": "note", "tags": []any{"receipts"}})
	a.Automate.StartWorker()
	if said := a.Automate.Run(context.Background(), photos.ID, ""); !strings.Contains(said.Text, "300 files") {
		t.Fatalf("it goes through the 300: %q", said.Text)
	}
	if again := a.Automate.Run(context.Background(), photos.ID, ""); !strings.Contains(again.Text, "already going through") {
		t.Errorf("pressed again, it says it is going already: %q", again.Text)
	}
	time.Sleep(100 * time.Millisecond) // the photos queue up behind the first
	note, _, _ := records.WriteAs(a.Store, records.Who{Actor: "human"}, "created", "note", "", map[string]any{"title": "Receipt from Wickes"})
	close(model.gate)
	sorted := func() int {
		cls, _ := a.Store.List(records.ClassificationType, store.ListOptions{})
		return len(cls)
	}
	for end := time.Now().Add(30 * time.Second); sorted() < 301 && time.Now().Before(end); time.Sleep(20 * time.Millisecond) {
	}
	for n := int64(-1); model.asked.Load() != n; time.Sleep(300 * time.Millisecond) { // the last logged
		n = model.asked.Load()
	}
	if n, _ := a.Store.Get("note", note.ID); n == nil || !strings.Contains(strings.Join(records.StringList(n.Fields["tags"]), ","), "receipts") {
		t.Fatalf("the note shared while the photos were gone through is sorted (%d sorted)", sorted())
	}
}
