package chat_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/search"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

func searchTool(svc *chat.Service) string {
	for _, tool := range svc.Tools() {
		if tool.Name == "search" {
			return tool.Description
		}
	}
	return ""
}

// The assistant's search tool says it finds by meaning only when it does,
// and its answer reads as the page does: the same order said first, and
// the line found by meaning marked close in meaning.
func TestTheSearchToolSaysWhatIsCloseInMeaning(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "note.yaml"), []byte("name: note\ntitle: title\nfields:\n  title: {type: string, required: true}\n"), 0o644)
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
	svc := &chat.Service{Book: &records.Book{Store: st}, Registry: reg}
	boiler, _ := st.Create("note", map[string]any{"title": "Boiler repair, call Marek"})
	st.Create("note", map[string]any{"title": "Ask the plumber about the tap"})

	if d := searchTool(svc); strings.Contains(d, "close in meaning") {
		t.Errorf("no meaning, not said: %q", d)
	}
	search.UseMeaning(st, func(string) []string { return []string{"note/" + boiler.ID} })
	if d := searchTool(svc); !strings.Contains(d, "close in meaning") {
		t.Errorf("with meaning, said: %q", d)
	}
	args, _ := json.Marshal(map[string]any{"query": "plumber"})
	text, isErr := svc.Call("search", args)
	if isErr || !strings.HasPrefix(text, "First what has your words, then what is close in meaning.") {
		t.Fatalf("the order first, as on the page: %q", text)
	}
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, "Boiler repair") && !strings.Contains(line, "(close in meaning, not by its words)") {
			t.Errorf("the line found by meaning is marked: %q", line)
		}
	}
}
