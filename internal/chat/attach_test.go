package chat_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// withFiles is a service whose workspace has files and messages that can
// carry one.
func withFiles(t *testing.T, steps ...*llm.Response) (*chat.Service, *scripted) {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"message.yaml": "name: message\nfields:\n  role: {type: enum, values: [user, assistant, error], required: true}\n  content: {type: text, required: true}\n  file: {type: string}\n",
		"block.yaml":   "name: block\nfields:\n  component: {type: string, required: true}\n  props: {type: json}\n  position: {type: int, default: 0}\n",
		"file.yaml":    "name: file\nfields:\n  title: {type: string, required: true}\n  kind: {type: string}\n  text: {type: markdown}\n  description: {type: text}\n  status: {type: enum, values: [ready, converting, failed], default: ready}\n  note: {type: text}\n",
	}
	for name, src := range files {
		os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644)
	}
	types, err := schema.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(":memory:", types)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	reg, err := app.NewRegistry("")
	if err != nil {
		t.Fatal(err)
	}
	m := &scripted{steps: steps}
	return &chat.Service{Store: st, Registry: reg, Provider: m, HistoryLimit: 10}, m
}

// The model gets what a file says with the message it came with, and is
// told what it cannot get: a picture's description, or text still on its way.
func TestTheModelReadsWhatCameWithTheMessage(t *testing.T) {
	svc, m := withFiles(t)
	ready, _ := svc.Store.Create("file", map[string]any{"title": "Lease", "kind": "document", "text": "# Lease\n\nRent is due on the first."})
	if _, err := svc.SendFile(context.Background(), "", "", ready.ID); err != nil {
		t.Fatal(err)
	}
	got := m.seen[0].Messages[0].Content
	if !strings.HasPrefix(got, "Here is a file.") || !strings.Contains(got, "Rent is due") || !strings.Contains(got, "filed at /t/file/"+ready.ID) {
		t.Errorf("an empty message with a file still says something, and carries the file: %q", got)
	}

	picture, _ := svc.Store.Create("file", map[string]any{"title": "Garden", "kind": "image"})
	svc.SendFile(context.Background(), "", "Look", picture.ID)
	if got := m.seen[1].Messages[len(m.seen[1].Messages)-1].Content; !strings.Contains(got, "no description yet") {
		t.Errorf("a picture without a description asks for one: %q", got)
	}

	later, _ := svc.Store.Create("file", map[string]any{"title": "Scan", "kind": "document", "status": "converting"})
	svc.SendFile(context.Background(), "", "And this", later.ID)
	if got := m.seen[2].Messages[len(m.seen[2].Messages)-1].Content; !strings.Contains(got, "still being read") {
		t.Errorf("text on its way is said to be: %q", got)
	}

	long := strings.Repeat("word ", 3000)
	big, _ := svc.Store.Create("file", map[string]any{"title": "Big", "kind": "document", "text": long})
	svc.SendFile(context.Background(), "", "Big one", big.ID)
	if got := m.seen[3].Messages[len(m.seen[3].Messages)-1].Content; len(got) > 9000 || !strings.Contains(got, "the rest is on its page") {
		t.Errorf("a long file is cut with a pointer to the rest: %d chars", len(got))
	}
}
